package a2uiruntime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/a2ui"
)

func renderComponent(rt *Runtime, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID, id string, theme *a2ui.Theme) swiftui.View {
	comp, ok := comps[id]
	if !ok {
		return swiftui.Text(fmt.Sprintf("[missing: %s]", id)).
			ForegroundStyleNamed("secondary").AsView()
	}
	return renderResolvedComponent(rt, comps, dm, surfaceID, comp, theme)
}

func renderResolvedComponent(rt *Runtime, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID string, comp a2ui.Component, theme *a2ui.Theme) swiftui.View {
	var view swiftui.View

	switch comp.ComponentType() {
	case a2ui.ComponentText:
		text := resolveTextValue(comp.Text.Text, dm)
		tv := swiftui.Text(text)
		if comp.Strikethrough != nil && *comp.Strikethrough {
			tv = tv.Strikethrough()
		}
		view = applyTextVariant(tv, string(comp.Text.Variant))

	case a2ui.ComponentButton:
		view = renderButton(rt, comp, comps, dm, surfaceID, theme, func() {
			rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
		})

	case a2ui.ComponentTextField:
		label := resolveTextValue(comp.TextField.Label, dm)
		binding := dynamicStringBinding(comp.TextField.Value)
		s := rt.cache.getString(binding, "")
		validState := rt.cache.getBool(comp.ID+"/valid", true)
		onChange := func() {
			rt.cache.setStringValue(binding, s.Get())
			setValidationState(validState, validateComponent(rt, dm, comp, binding, s.Get()))
		}
		onSubmit := func() {
			rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
		}

		policy := swiftui.TextInputPolicy{
			ValidationPattern: comp.TextField.ValidationRegexp,
			ValidState:        validState,
		}
		if comp.TextField.Variant == a2ui.TextFieldVariantNumber {
			policy.AllowedPattern = `[-0-9.]*`
			if policy.ValidationPattern == "" {
				policy.ValidationPattern = `^-?[0-9]*([.][0-9]+)?$`
			}
		}

		switch comp.TextField.Variant {
		case a2ui.TextFieldVariantLongText:
			if usesTextInputPolicy(policy) {
				view = renderValidatedControl(label, swiftui.TextEditorPolicy(s, policy, onChange).
					FrameAligned(0, 76, swiftui.HorizontalAlignmentLeading, swiftui.VerticalAlignmentCenter), validState)
			} else {
				view = renderLabeledControl(label, swiftui.TextEditorOnChange(s, onChange).
					FrameAligned(0, 76, swiftui.HorizontalAlignmentLeading, swiftui.VerticalAlignmentCenter))
			}
		case a2ui.TextFieldVariantObscured:
			if usesTextInputPolicy(policy) {
				view = renderValidatedControl(label, swiftui.SecureFieldPolicy(label, s, policy, onChange, onSubmit), validState)
			} else {
				view = renderLabeledControl(label, swiftui.SecureFieldCallbacks(label, s, onChange, onSubmit))
			}
		default:
			if usesTextInputPolicy(policy) {
				view = renderValidatedControl(label, swiftui.TextFieldPolicy(label, s, policy, onChange, onSubmit), validState)
			} else {
				view = renderLabeledControl(label, swiftui.TextFieldCallbacks(label, s, onChange, onSubmit).
					TextFieldStyle(swiftui.TextFieldStyleRoundedBorder))
			}
		}
		setValidationState(validState, validateComponent(rt, dm, comp, binding, s.Get()))

	case a2ui.ComponentCheckBox:
		label := resolveTextValue(comp.CheckBox.Label, dm)
		binding := dynamicBooleanBinding(comp.CheckBox.Value)
		initial, _ := a2ui.ResolveDynamicBoolean(comp.CheckBox.Value, dm)
		s := rt.cache.getInt(binding, boolToInt(initial))
		view = swiftui.Toggle(label, s, func() {
			rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
		}).ToggleStyle(swiftui.ToggleStyleCheckbox)

	case a2ui.ComponentSlider:
		label := resolveDynamicStringPtr(comp.Slider.Label, dm)
		binding := dynamicNumberBinding(comp.Slider.Value)
		initial, _ := a2ui.ResolveDynamicNumber(comp.Slider.Value, dm)
		s := rt.cache.getInt(binding, int(initial))
		view = swiftui.Slider(label, s, floatFromPtr(comp.Slider.Min, 0), comp.Slider.Max, func() {
			rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
		})

	case a2ui.ComponentRow:
		spacing := floatFromPtr(comp.Spacing, 8)
		children := justifyRowChildren(renderChildList(rt, comps, dm, surfaceID, comp.Row.Children, theme), comp.Row.Justify)
		va := swiftui.VerticalAlignmentCenter
		switch comp.Row.Align {
		case a2ui.LayoutAlignStart:
			va = swiftui.VerticalAlignmentTop
		case a2ui.LayoutAlignEnd:
			va = swiftui.VerticalAlignmentBottom
		}
		view = swiftui.HStackAlignedSpaced(va, spacing, children...)

	case a2ui.ComponentColumn:
		spacing := floatFromPtr(comp.Spacing, 8)
		children := justifyColumnChildren(renderChildList(rt, comps, dm, surfaceID, comp.Column.Children, theme), comp.Column.Justify)
		ha := horizontalAlignment(comp.Column.Align)
		view = swiftui.VStackAlignedSpaced(ha, spacing, children...)

	case a2ui.ComponentCard:
		child := renderFirstChild(rt, comps, dm, surfaceID, comp.Card.Child, theme)
		view = swiftui.VStack(child).
			Padding(16).
			BackgroundStyle("regularMaterial").
			CornerRadius(12).
			Shadow(0, 0, 0, 0.15, 6, 0, 3)

	case a2ui.ComponentList:
		children := renderChildList(rt, comps, dm, surfaceID, comp.List.Children, theme)
		spacing := floatFromPtr(comp.Spacing, 8)
		if comp.List.Direction == a2ui.ListDirectionHorizontal {
			view = swiftui.ScrollView(swiftui.HStackAlignedSpaced(verticalAlignment(comp.List.Align), spacing, children...))
		} else {
			view = swiftui.ScrollView(swiftui.VStackAlignedSpaced(horizontalAlignment(comp.List.Align), spacing, children...))
		}

	case a2ui.ComponentDivider:
		d := swiftui.Divider()
		if comp.Divider.Axis == a2ui.DividerAxisVertical {
			d = d.RotationEffect(90)
		}
		view = d

	case a2ui.ComponentIcon:
		view = swiftui.Image(mapIconName(iconName(comp.Icon.Name)))

	case a2ui.ComponentImage:
		src, _ := a2ui.ResolveDynamicString(comp.Image.URL, dm)
		if src == "" {
			view = swiftui.Image("photo").ForegroundStyleNamed("secondary")
		} else {
			view = renderImageComponent(comp, src)
		}

	case a2ui.ComponentTabs:
		tabViews := make([]swiftui.Viewable, 0, len(comp.Tabs.Tabs))
		for _, tab := range comp.Tabs.Tabs {
			if tab.Child == "" {
				continue
			}
			tabViews = append(tabViews, renderComponent(rt, comps, dm, surfaceID, tab.Child, theme).
				TabItem(resolveTextValue(tab.Title, dm), "rectangle.grid.1x2"))
		}
		view = swiftui.TabView(tabViews...)

	case a2ui.ComponentModal:
		modalState := rt.cache.getModal(comp.ID)
		triggerView := renderFirstChild(rt, comps, dm, surfaceID, comp.Modal.Trigger, theme)
		if trigger, ok := comps[comp.Modal.Trigger]; ok && trigger.Button != nil {
			triggerView = renderButton(rt, trigger, comps, dm, surfaceID, theme, func() {
				modalState.Set(true)
				rt.HandleAction(surfaceID, trigger.ID, componentAction(trigger))
			})
		}
		contentView := renderModalContent(rt, comps, dm, surfaceID, comp.Modal.Content, modalState, theme)
		view = triggerView.
			OnTapGesture(func() { modalState.Set(true) }).
			SheetPresented(modalState, contentView)

	case a2ui.ComponentChoicePicker:
		view = renderChoicePicker(rt, comp, comps, dm, surfaceID, theme)

	case a2ui.ComponentDateTimeInput:
		label := resolveDynamicStringPtr(comp.DateTimeInput.Label, dm)
		binding := dynamicStringBinding(&comp.DateTimeInput.Value)
		initial := float64(time.Now().Unix())
		if raw, _ := a2ui.ResolveDynamicString(comp.DateTimeInput.Value, dm); raw != "" {
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				initial = float64(t.Unix())
			}
		}
		s := rt.cache.getDate(binding, initial)
		min, max := resolveDateBounds(comp.DateTimeInput, dm)
		view = swiftui.DatePickerBounded(dateInputLabel(label, comp.DateTimeInput), s, swiftui.DateBounds{Min: min, Max: max}, func() {
			rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
		})

	case a2ui.ComponentVideo:
		url, _ := a2ui.ResolveDynamicString(comp.Video.URL, dm)
		view = swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 8,
			playerView(rt.cache, url, 320, 180),
			swiftui.Text(url).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
		)

	case a2ui.ComponentAudioPlayer:
		description := resolveDynamicStringPtr(comp.AudioPlayer.Description, dm)
		url, _ := a2ui.ResolveDynamicString(comp.AudioPlayer.URL, dm)
		if description == "" {
			description = "Audio"
		}
		view = swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 8,
			swiftui.HStackSpaced(8,
				swiftui.Image("waveform").ForegroundStyleNamed("secondary"),
				swiftui.Text(description).Font(swiftui.FontCaption),
			),
			playerView(rt.cache, url, 320, 52),
		)

	case a2ui.ComponentProgress:
		if comp.Progress != nil && comp.Progress.Value != nil {
			val, _ := a2ui.ResolveDynamicNumber(*comp.Progress.Value, dm)
			view = swiftui.ProgressLinear(val, floatFromPtr(comp.Progress.Max, 1))
		} else {
			view = swiftui.ProgressSpinning()
		}

	default:
		view = swiftui.Text(fmt.Sprintf("[unsupported: %s]", comp.ComponentType())).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").AsView()
	}

	if comp.Accessibility != nil {
		if label := resolveDynamicStringPtr(comp.Accessibility.Label, dm); label != "" {
			view = view.AccessibilityLabel(label)
		}
		if desc := resolveDynamicStringPtr(comp.Accessibility.Description, dm); desc != "" {
			view = view.AccessibilityHint(desc)
		}
	}
	return applyComponentLayout(view, comp)
}

func renderChoicePicker(rt *Runtime, comp a2ui.Component, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID string, theme *a2ui.Theme) swiftui.View {
	label := resolveDynamicStringPtr(comp.ChoicePicker.Label, dm)
	binding := dynamicStringListBinding(comp.ChoicePicker.Value)
	labels, values := resolveChoiceOptions(comp.ChoicePicker.Options, dm)

	if comp.ChoicePicker.Filterable != nil && *comp.ChoicePicker.Filterable {
		options := make([]swiftui.ChoiceOption, 0, len(labels))
		for i, opt := range labels {
			options = append(options, swiftui.ChoiceOption{Label: opt, Value: values[i]})
		}
		if comp.ChoicePicker.Variant == a2ui.ChoicePickerVariantMultipleSelection {
			selection := rt.cache.getStringList(binding, selectedValueList(binding, dm))
			return swiftui.SearchableMultiPicker(label, "Filter options", selection, options, func() {
				rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
			})
		}
		selected := ""
		if idx := selectedChoiceIndex(binding, values, dm); idx >= 0 {
			selected = values[idx]
		}
		selection := rt.cache.getString(binding, selected)
		return swiftui.SearchablePicker(label, "Filter options", selection, options, func() {
			rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
		})
	}

	s := rt.cache.getInt(binding, 0)
	if idx := selectedChoiceIndex(binding, values, dm); idx >= 0 {
		s.Set(idx)
	}

	optionViews := make([]swiftui.Viewable, len(labels))
	for i, opt := range labels {
		optionViews[i] = swiftui.Text(opt).AsView().Tag(int32(i))
	}
	optionsView := swiftui.VStack(optionViews...)

	if comp.ChoicePicker.Variant == a2ui.ChoicePickerVariantMultipleSelection {
		toggles := make([]swiftui.Viewable, len(labels))
		selected := selectedChoiceValues(binding, values, dm)
		for i, opt := range labels {
			toggleState := rt.cache.getInt(fmt.Sprintf("%s/%d", binding, i), 0)
			if selected[values[i]] {
				toggleState.Set(1)
			} else {
				toggleState.Set(0)
			}
			toggles[i] = swiftui.Toggle(opt, toggleState, func() {
				rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
			})
		}
		return swiftui.VStackSpaced(4, toggles...)
	}

	if comp.ChoicePicker.DisplayStyle == a2ui.ChoicePickerDisplayStyleChips {
		return swiftui.DynamicView(s, func(selected int) swiftui.View {
			chips := make([]swiftui.Viewable, 0, len(labels)+1)
			if label != "" {
				chips = append(chips, swiftui.Text(label).Font(swiftui.FontBody).AsView())
			}
			for i, opt := range labels {
				idx := i
				btn := swiftui.Button(opt, func() {
					s.Set(idx)
					rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
				})
				if idx == selected {
					btn = btn.ButtonStyle(swiftui.ButtonStyleBorderedProminent)
					if theme != nil && theme.PrimaryColor != "" {
						if r, g, b, a, ok := parseHexColor(theme.PrimaryColor); ok {
							btn = btn.Tint(r, g, b, a)
						}
					}
				} else {
					btn = btn.ButtonStyle(swiftui.ButtonStyleBordered)
				}
				chips = append(chips, btn)
			}
			return swiftui.HStackSpaced(8, chips...)
		})
	}

	return swiftui.PickerMenu(label, s, optionsView, func() {
		rt.HandleAction(surfaceID, comp.ID, componentAction(comp))
	})
}

func renderChildList(rt *Runtime, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID string, list a2ui.ChildList, theme *a2ui.Theme) []swiftui.Viewable {
	if list.Template != nil {
		return renderTemplateChildren(rt, comps, dm, surfaceID, *list.Template, theme)
	}
	views := make([]swiftui.Viewable, 0, len(list.IDs))
	for _, cid := range list.IDs {
		views = append(views, renderComponent(rt, comps, dm, surfaceID, cid, theme))
	}
	if len(views) == 0 {
		return []swiftui.Viewable{swiftui.Text("").AsView()}
	}
	return views
}

func renderTemplateChildren(rt *Runtime, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID string, tmpl a2ui.ChildTemplate, theme *a2ui.Theme) []swiftui.Viewable {
	template, ok := comps[tmpl.ComponentID]
	if !ok {
		rt.report(RuntimeError{Code: "missing_component", SurfaceID: surfaceID, ComponentID: tmpl.ComponentID, Message: "template component not found"})
		return []swiftui.Viewable{swiftui.Text("[missing template]").AsView()}
	}
	raw, err := dm.Get(tmpl.Path)
	if err != nil {
		rt.report(RuntimeError{Code: "invalid_template", SurfaceID: surfaceID, ComponentID: tmpl.ComponentID, Path: tmpl.Path, Message: err.Error()})
		return []swiftui.Viewable{swiftui.Text("[invalid template]").AsView()}
	}
	items, ok := raw.([]any)
	if !ok {
		rt.report(RuntimeError{Code: "invalid_template", SurfaceID: surfaceID, ComponentID: tmpl.ComponentID, Path: tmpl.Path, Message: "template path must resolve to a list"})
		return []swiftui.Viewable{swiftui.Text("[invalid template]").AsView()}
	}
	views := make([]swiftui.Viewable, 0, len(items))
	for i, item := range items {
		override, ok := item.(map[string]any)
		if !ok {
			rt.report(RuntimeError{Code: "invalid_template", SurfaceID: surfaceID, ComponentID: tmpl.ComponentID, Path: tmpl.Path, Message: "template item must be an object"})
			continue
		}
		component, err := instantiateTemplate(template, override, i)
		if err != nil {
			rt.report(RuntimeError{Code: "invalid_template", SurfaceID: surfaceID, ComponentID: tmpl.ComponentID, Path: tmpl.Path, Message: err.Error()})
			continue
		}
		views = append(views, renderResolvedComponent(rt, comps, dm, surfaceID, component, theme))
	}
	if len(views) == 0 {
		return []swiftui.Viewable{swiftui.Text("").AsView()}
	}
	return views
}

func instantiateTemplate(base a2ui.Component, override map[string]any, index int) (a2ui.Component, error) {
	baseData, err := json.Marshal(base)
	if err != nil {
		return a2ui.Component{}, err
	}
	var merged map[string]any
	if err := json.Unmarshal(baseData, &merged); err != nil {
		return a2ui.Component{}, err
	}
	deepMerge(merged, override)
	merged["id"] = fmt.Sprintf("%s[%d]", base.ID, index)
	data, err := json.Marshal(merged)
	if err != nil {
		return a2ui.Component{}, err
	}
	var out a2ui.Component
	if err := json.Unmarshal(data, &out); err != nil {
		return a2ui.Component{}, err
	}
	return out, nil
}

func deepMerge(dst, src map[string]any) {
	for key, value := range src {
		if child, ok := value.(map[string]any); ok {
			if existing, ok := dst[key].(map[string]any); ok {
				deepMerge(existing, child)
				continue
			}
		}
		dst[key] = value
	}
}

func renderFirstChild(rt *Runtime, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID, childID string, theme *a2ui.Theme) swiftui.View {
	if childID == "" {
		return swiftui.Text("").AsView()
	}
	return renderComponent(rt, comps, dm, surfaceID, childID, theme)
}

func renderButton(rt *Runtime, comp a2ui.Component, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID string, theme *a2ui.Theme, onPress func()) swiftui.View {
	childID := ""
	variant := a2ui.ButtonVariantDefault
	if comp.Button != nil {
		childID = comp.Button.Child
		variant = comp.Button.Variant
	}
	label := swiftui.Text("Button").AsView()
	if childID != "" {
		label = renderComponent(rt, comps, dm, surfaceID, childID, theme)
	}
	btn := swiftui.ButtonView(label, onPress)
	switch variant {
	case a2ui.ButtonVariantPrimary:
		btn = btn.ButtonStyle(swiftui.ButtonStyleBorderedProminent)
		if theme != nil && theme.PrimaryColor != "" {
			if r, g, b, a, ok := parseHexColor(theme.PrimaryColor); ok {
				btn = btn.Tint(r, g, b, a)
			}
		}
	case a2ui.ButtonVariantBorderless:
		btn = btn.ButtonStyle(swiftui.ButtonStyleBorderless)
	default:
		btn = btn.ButtonStyle(swiftui.ButtonStyleBordered)
	}
	return btn
}

type modalPresentation struct {
	Title        string
	Icon         string
	Width        float64
	Height       float64
	DismissLabel string
}

func renderModalContent(rt *Runtime, comps map[string]a2ui.Component, dm *a2ui.DataModel, surfaceID, contentID string, modalState *swiftui.BoolState, theme *a2ui.Theme) swiftui.View {
	content := renderFirstChild(rt, comps, dm, surfaceID, contentID, theme)
	presentation := resolveModalPresentation(comps, contentID, dm)
	return renderModalSheet(content, modalState, presentation)
}

func renderLabeledControl(label string, control swiftui.View) swiftui.View {
	if label == "" || strings.HasSuffix(label, "...") {
		return control
	}
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4,
		swiftui.Text(label).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
		control,
	)
}

func renderValidatedControl(label string, control swiftui.View, valid *swiftui.BoolState) swiftui.View {
	base := renderLabeledControl(label, control)
	return swiftui.DynamicBoolView(valid, func(ok bool) swiftui.View {
		if ok {
			return base
		}
		return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4,
			base,
			swiftui.Text("Invalid value").Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
		)
	})
}

func usesTextInputPolicy(policy swiftui.TextInputPolicy) bool {
	return policy.AllowedPattern != "" || policy.ValidationPattern != "" || policy.ValidState != nil
}

func renderImageComponent(comp a2ui.Component, src string) swiftui.View {
	img := swiftui.AsyncImageFit(src, imageFit(comp.Image.Fit))
	switch comp.Image.Variant {
	case a2ui.ImageVariantIcon:
		return img.Frame(28, 28).Padding(8).BackgroundStyle("thinMaterial").CornerRadius(10)
	case a2ui.ImageVariantAvatar:
		return img.Frame(64, 64).Padding(5).BackgroundStyle("ultraThinMaterial").CornerRadius(37).Shadow(0, 0, 0, 0.10, 6, 0, 2)
	case a2ui.ImageVariantSmallFeature:
		return img.Frame(54, 54).BackgroundStyle("thinMaterial").CornerRadius(10)
	case a2ui.ImageVariantMediumFeature:
		return img.Frame(112, 78).BackgroundStyle("thinMaterial").CornerRadius(12).Shadow(0, 0, 0, 0.10, 8, 0, 3)
	case a2ui.ImageVariantLargeFeature:
		return img.Frame(208, 138).BackgroundStyle("regularMaterial").CornerRadius(16).Shadow(0, 0, 0, 0.14, 10, 0, 4)
	case a2ui.ImageVariantHeader:
		return img.Frame(280, 96).BackgroundStyle("regularMaterial").CornerRadius(14).Shadow(0, 0, 0, 0.12, 8, 0, 3)
	default:
		return img.Frame(120, 120).CornerRadius(10)
	}
}

func resolveModalPresentation(comps map[string]a2ui.Component, contentID string, dm *a2ui.DataModel) modalPresentation {
	p := modalPresentation{Title: "Details", Icon: "info.circle.fill", Width: 420, Height: 220, DismissLabel: "Done"}
	if title := modalTitle(comps, contentID, dm); title != "" {
		p.Title = title
	}
	return p
}

func resolveDateBounds(input *a2ui.DateTimeInputComponent, dm *a2ui.DataModel) (min, max *float64) {
	if input == nil {
		return nil, nil
	}
	parse := func(value *a2ui.DynamicString) *float64 {
		if value == nil {
			return nil
		}
		raw, err := a2ui.ResolveDynamicString(*value, dm)
		if err != nil || raw == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil
		}
		unix := float64(t.Unix())
		return &unix
	}
	return parse(input.Min), parse(input.Max)
}

func renderModalSheet(content swiftui.View, modalState *swiftui.BoolState, presentation modalPresentation) swiftui.View {
	header := swiftui.HStackAlignedSpaced(swiftui.VerticalAlignmentCenter, 12,
		swiftui.HStackAlignedSpaced(swiftui.VerticalAlignmentCenter, 8,
			swiftui.Image(presentation.Icon).ImageScale(swiftui.ImageScaleLarge).ForegroundStyleNamed("secondary"),
			swiftui.Text(presentation.Title).Font(swiftui.FontTitle3).FontWeight(swiftui.WeightSemibold),
		),
		swiftui.Spacer(),
		swiftui.Button(presentation.DismissLabel, func() { modalState.Set(false) }).ButtonStyle(swiftui.ButtonStyleBorderedProminent).ControlSize(swiftui.ControlSizeSmall),
	)
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 16,
		header,
		swiftui.Divider(),
		content,
	).Padding(20).
		Frame(presentation.Width, presentation.Height).
		BackgroundStyle("windowBackground").
		CornerRadius(14).
		Shadow(0, 0, 0, 0.12, 10, 0, 4).
		FixedSize()
}

func modalTitle(comps map[string]a2ui.Component, contentID string, dm *a2ui.DataModel) string {
	content, ok := comps[contentID]
	if !ok || content.Column == nil || len(content.Column.Children.IDs) == 0 {
		return ""
	}
	child, ok := comps[content.Column.Children.IDs[0]]
	if !ok || child.Text == nil {
		return ""
	}
	return resolveTextValue(child.Text.Text, dm)
}

func resolveDynamicStringPtr(value *a2ui.DynamicString, dm *a2ui.DataModel) string {
	if value == nil {
		return ""
	}
	return resolveTextValue(*value, dm)
}

func resolveTextValue(value a2ui.DynamicString, dm *a2ui.DataModel) string {
	text, err := a2ui.ResolveDynamicString(value, dm)
	if err != nil {
		return ""
	}
	for strings.HasPrefix(text, "#") {
		text = strings.TrimPrefix(text, "#")
	}
	return strings.TrimLeft(text, " ")
}

func dynamicStringBinding(value *a2ui.DynamicString) string {
	if value != nil && value.Binding != nil {
		return value.Binding.Path
	}
	return ""
}

func dynamicBooleanBinding(value a2ui.DynamicBoolean) string {
	if value.Binding != nil {
		return value.Binding.Path
	}
	return ""
}

func dynamicNumberBinding(value a2ui.DynamicNumber) string {
	if value.Binding != nil {
		return value.Binding.Path
	}
	return ""
}

func dynamicStringListBinding(value a2ui.DynamicStringList) string {
	if value.Binding != nil {
		return value.Binding.Path
	}
	return ""
}

func componentAction(comp a2ui.Component) *a2ui.Action {
	if comp.Action != nil {
		return comp.Action
	}
	if comp.Button != nil {
		action := comp.Button.Action
		if action.Event != nil || action.FunctionCall != nil {
			return &action
		}
	}
	return nil
}

func resolveChoiceOptions(options []a2ui.ChoiceOption, dm *a2ui.DataModel) ([]string, []string) {
	labels := make([]string, 0, len(options))
	values := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, resolveTextValue(option.Label, dm))
		values = append(values, option.Value)
	}
	return labels, values
}

func justifyRowChildren(children []swiftui.Viewable, justify a2ui.LayoutJustify) []swiftui.Viewable {
	switch justify {
	case a2ui.LayoutJustifyEnd:
		return append([]swiftui.Viewable{swiftui.Spacer()}, children...)
	case a2ui.LayoutJustifyCenter:
		out := make([]swiftui.Viewable, 0, len(children)+2)
		out = append(out, swiftui.Spacer())
		out = append(out, children...)
		return append(out, swiftui.Spacer())
	case a2ui.LayoutJustifySpaceBetween, a2ui.LayoutJustifyStretch:
		out := make([]swiftui.Viewable, 0, len(children)*2)
		for i, child := range children {
			if i > 0 {
				out = append(out, swiftui.Spacer())
			}
			out = append(out, child)
		}
		return out
	case a2ui.LayoutJustifySpaceAround, a2ui.LayoutJustifySpaceEvenly:
		out := make([]swiftui.Viewable, 0, len(children)*2+2)
		out = append(out, swiftui.Spacer())
		for _, child := range children {
			out = append(out, child, swiftui.Spacer())
		}
		return out
	default:
		return children
	}
}

func justifyColumnChildren(children []swiftui.Viewable, justify a2ui.LayoutJustify) []swiftui.Viewable {
	switch justify {
	case a2ui.LayoutJustifyEnd:
		return append([]swiftui.Viewable{swiftui.Spacer()}, children...)
	case a2ui.LayoutJustifyCenter:
		out := make([]swiftui.Viewable, 0, len(children)+2)
		out = append(out, swiftui.Spacer())
		out = append(out, children...)
		return append(out, swiftui.Spacer())
	case a2ui.LayoutJustifySpaceBetween, a2ui.LayoutJustifyStretch:
		out := make([]swiftui.Viewable, 0, len(children)*2)
		for i, child := range children {
			if i > 0 {
				out = append(out, swiftui.Spacer())
			}
			out = append(out, child)
		}
		return out
	case a2ui.LayoutJustifySpaceAround, a2ui.LayoutJustifySpaceEvenly:
		out := make([]swiftui.Viewable, 0, len(children)*2+2)
		out = append(out, swiftui.Spacer())
		for _, child := range children {
			out = append(out, child, swiftui.Spacer())
		}
		return out
	default:
		return children
	}
}

func selectedChoiceIndex(binding string, values []string, dm *a2ui.DataModel) int {
	if binding == "" {
		return -1
	}
	value, err := dm.Get(binding)
	if err != nil {
		return -1
	}
	switch v := value.(type) {
	case string:
		for i, candidate := range values {
			if candidate == v {
				return i
			}
		}
	case []any:
		if len(v) > 0 {
			first := fmt.Sprintf("%v", v[0])
			for i, candidate := range values {
				if candidate == first {
					return i
				}
			}
		}
	}
	return -1
}

func selectedChoiceValues(binding string, values []string, dm *a2ui.DataModel) map[string]bool {
	selected := make(map[string]bool, len(values))
	if binding == "" {
		return selected
	}
	value, err := dm.Get(binding)
	if err != nil {
		return selected
	}
	switch v := value.(type) {
	case []string:
		for _, item := range v {
			selected[item] = true
		}
	case []any:
		for _, item := range v {
			selected[fmt.Sprintf("%v", item)] = true
		}
	}
	return selected
}

func selectedValueList(binding string, dm *a2ui.DataModel) []string {
	if binding == "" {
		return nil
	}
	value, err := dm.Get(binding)
	if err != nil {
		return nil
	}
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return nil
	}
}

func imageFit(fit a2ui.ImageFit) swiftui.ImageFit {
	switch fit {
	case a2ui.ImageFitCover:
		return swiftui.ImageFitCover
	case a2ui.ImageFitFill:
		return swiftui.ImageFitFill
	case a2ui.ImageFitNone:
		return swiftui.ImageFitNone
	case a2ui.ImageFitScaleDown:
		return swiftui.ImageFitScaleDown
	default:
		return swiftui.ImageFitContain
	}
}

func iconName(name a2ui.IconNameOrPath) string {
	if name.Name != nil {
		return string(*name.Name)
	}
	if name.Path != nil {
		return *name.Path
	}
	return ""
}

func floatFromPtr(value *float64, def float64) float64 {
	if value == nil {
		return def
	}
	return *value
}

func applyPadding(view swiftui.View, comp a2ui.Component) swiftui.View {
	if comp.Padding != nil {
		return view.Padding(*comp.Padding)
	}
	return view
}

func applyComponentLayout(view swiftui.View, comp a2ui.Component) swiftui.View {
	view = applyPadding(view, comp)
	if comp.Weight != nil {
		view = view.LayoutPriority(*comp.Weight)
		if *comp.Weight > 0 {
			view = view.MaxFrameAligned(-1, 0, swiftui.HorizontalAlignmentLeading, swiftui.VerticalAlignmentCenter)
		}
	}
	return view
}

func applyTextVariant(text swiftui.TextView, variant string) swiftui.View {
	switch variant {
	case "h1":
		return text.Font(swiftui.FontLargeTitle).FontWeight(swiftui.WeightBold).AsView()
	case "h2":
		return text.Font(swiftui.FontTitle).AsView()
	case "h3":
		return text.Font(swiftui.FontTitle2).AsView()
	case "h4":
		return text.Font(swiftui.FontTitle3).AsView()
	case "h5":
		return text.Font(swiftui.FontHeadline).AsView()
	case "caption":
		return text.Font(swiftui.FontCaption).AsView()
	default:
		return text.Font(swiftui.FontBody).AsView()
	}
}

func horizontalAlignment(align a2ui.LayoutAlign) swiftui.HorizontalAlignment {
	switch align {
	case a2ui.LayoutAlignEnd:
		return swiftui.HorizontalAlignmentTrailing
	default:
		return swiftui.HorizontalAlignmentLeading
	}
}

func verticalAlignment(align a2ui.LayoutAlign) swiftui.VerticalAlignment {
	switch align {
	case a2ui.LayoutAlignStart:
		return swiftui.VerticalAlignmentTop
	case a2ui.LayoutAlignEnd:
		return swiftui.VerticalAlignmentBottom
	default:
		return swiftui.VerticalAlignmentCenter
	}
}

func dateInputLabel(label string, input *a2ui.DateTimeInputComponent) string {
	if input == nil {
		return label
	}
	enableDate := input.EnableDate == nil || *input.EnableDate
	enableTime := input.EnableTime != nil && *input.EnableTime
	switch {
	case enableDate && enableTime:
		return label
	case enableDate:
		return label + " (date)"
	case enableTime:
		return label + " (time)"
	default:
		return label
	}
}

func validateComponent(rt *Runtime, dm *a2ui.DataModel, comp a2ui.Component, binding string, value any) string {
	if binding == "" {
		return ""
	}
	overlay, err := overlayDataModel(dm, binding, value)
	if err != nil {
		return ""
	}
	for _, check := range comp.Checks {
		ok, err := a2ui.ResolveDynamicBoolean(check.Condition, overlay)
		if err != nil || !ok {
			return check.Message
		}
	}
	return ""
}

func overlayDataModel(dm *a2ui.DataModel, path string, value any) (*a2ui.DataModel, error) {
	data, err := json.Marshal(dm.Data)
	if err != nil {
		return nil, err
	}
	clone := a2ui.NewDataModel()
	if err := json.Unmarshal(data, &clone.Data); err != nil {
		return nil, err
	}
	if err := clone.Set(path, value); err != nil {
		return nil, err
	}
	return clone, nil
}

func setValidationState(state *swiftui.BoolState, message string) {
	state.Set(message == "")
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func mapIconName(name string) string {
	m := map[string]string{
		"flight": "airplane", "airplane": "airplane", "search": "magnifyingglass", "settings": "gearshape", "home": "house",
		"person": "person", "mail": "envelope", "phone": "phone", "calendar": "calendar", "clock": "clock", "location": "location",
		"heart": "heart", "star": "star", "bookmark": "bookmark", "share": "square.and.arrow.up", "delete": "trash", "edit": "pencil",
		"add": "plus", "remove": "minus", "close": "xmark", "check": "checkmark", "warning": "exclamationmark.triangle", "error": "xmark.circle",
		"info": "info.circle", "help": "questionmark.circle", "refresh": "arrow.clockwise", "download": "arrow.down.circle", "upload": "arrow.up.circle",
		"play": "play.fill", "pause": "pause.fill", "stop": "stop.fill", "forward": "forward.fill", "backward": "backward.fill", "menu": "line.3.horizontal",
		"more": "ellipsis", "filter": "line.3.horizontal.decrease", "sort": "arrow.up.arrow.down", "list": "list.bullet", "grid": "square.grid.2x2",
		"lock": "lock", "unlock": "lock.open", "eye": "eye", "eye-off": "eye.slash", "camera": "camera", "image": "photo", "video": "video",
		"mic": "mic", "speaker": "speaker.wave.2", "wifi": "wifi", "bluetooth": "antenna.radiowaves.left.and.right", "battery": "battery.100",
		"bolt": "bolt", "sun": "sun.max", "moon": "moon", "cloud": "cloud", "rain": "cloud.rain", "snow": "cloud.snow", "wind": "wind", "fog": "cloud.fog",
	}
	if sf, ok := m[name]; ok {
		return sf
	}
	return name
}

func parseHexColor(hex string) (r, g, b, a float64, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	switch len(hex) {
	case 6:
		rv, err := strconv.ParseUint(hex[0:2], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		gv, err := strconv.ParseUint(hex[2:4], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		bv, err := strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		return float64(rv) / 255, float64(gv) / 255, float64(bv) / 255, 1, true
	case 8:
		rv, err := strconv.ParseUint(hex[0:2], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		gv, err := strconv.ParseUint(hex[2:4], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		bv, err := strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		av, err := strconv.ParseUint(hex[6:8], 16, 8)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		return float64(rv) / 255, float64(gv) / 255, float64(bv) / 255, float64(av) / 255, true
	default:
		return 0, 0, 0, 0, false
	}
}
