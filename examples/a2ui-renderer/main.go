//go:build darwin

// Command a2ui-renderer connects to an A2UI server via SSE and renders
// the component tree as native macOS views using SwiftUI.
//
// Usage:
//
//	go run .
//	go run . -server http://localhost:8090/sse
//	go run . -components components.json -data data.json
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/a2ui"
)

func init() { runtime.LockOSThread() }

// surfaceState holds the current A2UI surface state, protected by a mutex.
type surfaceState struct {
	mu         sync.Mutex
	surfaceID  string
	title      string
	theme      *a2ui.Theme
	components map[string]a2ui.Component
	rootID     string
	dataModel  *a2ui.DataModel
	status     string // "disconnected", "connecting", "connected"
	revision   int
}

// stateCache holds SwiftUI states keyed by JSON Pointer path.
type stateCache struct {
	mu          sync.Mutex
	strings     map[string]*swiftui.StringState
	ints        map[string]*swiftui.IntState
	floats      map[string]*swiftui.FloatState
	bools       map[string]*swiftui.BoolState
	dates       map[string]*swiftui.DateState
	modalStates map[string]*swiftui.BoolState
}

func newStateCache() *stateCache {
	return &stateCache{
		strings:     make(map[string]*swiftui.StringState),
		ints:        make(map[string]*swiftui.IntState),
		floats:      make(map[string]*swiftui.FloatState),
		bools:       make(map[string]*swiftui.BoolState),
		dates:       make(map[string]*swiftui.DateState),
		modalStates: make(map[string]*swiftui.BoolState),
	}
}

func (sc *stateCache) getString(path, initial string) *swiftui.StringState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.strings[path]; ok {
		return s
	}
	s := swiftui.NewStringState(initial)
	sc.strings[path] = s
	return s
}

func (sc *stateCache) getInt(path string, initial int) *swiftui.IntState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.ints[path]; ok {
		return s
	}
	s := swiftui.NewIntState(initial)
	sc.ints[path] = s
	return s
}

func (sc *stateCache) getDate(path string, initial float64) *swiftui.DateState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.dates[path]; ok {
		return s
	}
	s := swiftui.NewDateState(initial)
	sc.dates[path] = s
	return s
}

func (sc *stateCache) getModal(compID string) *swiftui.BoolState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.modalStates[compID]; ok {
		return s
	}
	s := swiftui.NewBoolState(false)
	sc.modalStates[compID] = s
	return s
}

func (sc *stateCache) syncFromDataModel(dm *a2ui.DataModel) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for path, s := range sc.strings {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		if str, ok := v.(string); ok {
			s.Set(str)
		}
	}
	for path, s := range sc.ints {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			s.Set(int(n))
		case int:
			s.Set(n)
		case bool:
			if n {
				s.Set(1)
			} else {
				s.Set(0)
			}
		}
	}
	for path, s := range sc.floats {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			s.Set(n)
		case int:
			s.Set(float64(n))
		}
	}
	for path, s := range sc.bools {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		if b, ok := v.(bool); ok {
			s.Set(b)
		}
	}
	for path, s := range sc.dates {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			s.Set(n)
		case string:
			if t, err := time.Parse(time.RFC3339, n); err == nil {
				s.Set(float64(t.Unix()))
			}
		}
	}
}

var (
	serverFlag     = flag.String("server", "http://localhost:8090/sse", "A2UI SSE server URL")
	componentsFlag = flag.String("components", "", "path to components JSON file (offline mode)")
	dataFlag       = flag.String("data", "", "path to data model JSON file (offline mode)")
)

func main() {
	flag.Parse()

	state := &surfaceState{
		components: make(map[string]a2ui.Component),
		dataModel:  a2ui.NewDataModel(),
		status:     "disconnected",
	}
	cache := newStateCache()

	revision := swiftui.NewIntState(0)
	urlState := swiftui.NewStringState(*serverFlag)
	statusRevision := swiftui.NewIntState(0)

	connect := func() {
		url := urlState.Get()
		if url == "" {
			return
		}
		state.mu.Lock()
		state.status = "connecting"
		state.mu.Unlock()
		statusRevision.Set(statusRevision.Get() + 1)

		go sseLoop(url, state, cache, revision, statusRevision)
	}

	// File-based offline mode: load components and data from JSON files.
	if *componentsFlag != "" {
		if err := loadFromFiles(state, cache, *componentsFlag, *dataFlag); err != nil {
			log.Fatalf("load files: %v", err)
		}
		state.mu.Lock()
		state.status = "file"
		state.mu.Unlock()
		revision.Set(1)
		statusRevision.Set(1)
	} else {
		go connect()
	}

	// Header: URL field + connect button (pinned to top).
	header := swiftui.HStackSpaced(8,
		swiftui.TextField("Server URL", urlState, connect).
			TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
		swiftui.Button("Connect", connect).
			ButtonStyle(swiftui.ButtonStyleBorderedProminent),
	)

	// Main content area (fills middle, scrollable) with animated transitions.
	content := swiftui.AnimatedDynamicView(revision, swiftui.TransitionOpacity, func(rev int) swiftui.View {
		state.mu.Lock()
		rootID := state.rootID
		comps := make(map[string]a2ui.Component, len(state.components))
		maps.Copy(comps, state.components)
		dm := state.dataModel
		surfaceID := state.surfaceID
		theme := state.theme
		state.mu.Unlock()

		if rootID == "" {
			return swiftui.VStackSpaced(8,
				swiftui.Spacer(),
				swiftui.Image("antenna.radiowaves.left.and.right").
					ForegroundStyleNamed("secondary").
					ImageScale(swiftui.ImageScaleLarge),
				swiftui.Text("Waiting for A2UI surface...").
					Font(swiftui.FontBody).
					ForegroundStyleNamed("secondary"),
				swiftui.Text("Connect to an A2UI server to render its UI.").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("tertiary"),
				swiftui.Spacer(),
			).Padding(36)
		}

		return renderComponent(comps, dm, cache, surfaceID, rootID, theme)
	})

	// Footer: status bar (pinned to bottom) with animated status transitions.
	footer := swiftui.AnimatedDynamicView(statusRevision, swiftui.TransitionOpacity, func(_ int) swiftui.View {
		state.mu.Lock()
		status := state.status
		sid := state.surfaceID
		rev := state.revision
		state.mu.Unlock()

		iconName := "circle.fill"
		var iconR, iconG, iconB float64
		statusText := "Disconnected"
		switch status {
		case "connected":
			iconR, iconG, iconB = 0.3, 0.8, 0.4
			statusText = "Connected"
			if sid != "" {
				statusText = fmt.Sprintf("Connected to %s", sid)
			}
		case "connecting":
			iconR, iconG, iconB = 0.9, 0.7, 0.2
			statusText = "Connecting..."
		case "file":
			iconR, iconG, iconB = 0.4, 0.6, 0.9
			statusText = "Loaded from file"
		default:
			iconR, iconG, iconB = 0.8, 0.3, 0.3
		}

		return swiftui.HStackSpaced(6,
			swiftui.Image(iconName).
				ForegroundStyle(iconR, iconG, iconB, 1.0).
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Text(statusText).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Text(fmt.Sprintf("Rev: %d", rev)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("tertiary").
				MonospacedDigit(),
		)
	})

	rootView := swiftui.VStackSpaced(0,
		header.Padding(12),
		swiftui.Divider(),
		swiftui.ScrollView(
			content.Padding(16),
		).MaxFrame(-1, -1),
		swiftui.Divider(),
		footer.PaddingEdge(swiftui.EdgeHorizontal, 12).PaddingEdge(swiftui.EdgeVertical, 6),
	)

	swiftui.Run(swiftui.AppConfig{
		Title:  "A2UI Renderer",
		Width:  700,
		Height: 600,
	}, rootView)
}

// sseLoop connects to the SSE endpoint and processes A2UI messages.
// On disconnect it retries with exponential backoff.
func sseLoop(url string, state *surfaceState, cache *stateCache, revision, statusRevision *swiftui.IntState) {
	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			log.Printf("sse reconnect: attempt %d, waiting %v", attempt+1, delay)
			time.Sleep(delay)
			state.mu.Lock()
			state.status = "connecting"
			state.mu.Unlock()
			statusRevision.Set(statusRevision.Get() + 1)
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("sse connect: %v", err)
			continue
		}

		state.mu.Lock()
		state.status = "connected"
		state.mu.Unlock()
		statusRevision.Set(statusRevision.Get() + 1)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

		var dataBuf bytes.Buffer
		for scanner.Scan() {
			line := scanner.Text()
			if data, ok := strings.CutPrefix(line, "data: "); ok {
				dataBuf.WriteString(data)
				continue
			}
			if line == "" && dataBuf.Len() > 0 {
				processSSEData(dataBuf.Bytes(), state, cache, revision, statusRevision)
				dataBuf.Reset()
			}
		}
		resp.Body.Close()

		if err := scanner.Err(); err != nil {
			log.Printf("sse read: %v", err)
		}

		state.mu.Lock()
		state.status = "disconnected"
		state.mu.Unlock()
		statusRevision.Set(statusRevision.Get() + 1)
	}

	log.Printf("sse: exhausted %d reconnect attempts", maxAttempts)
}

func processSSEData(data []byte, state *surfaceState, cache *stateCache, revision, statusRevision *swiftui.IntState) {
	var env a2ui.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		log.Printf("sse unmarshal envelope: %v", err)
		return
	}

	msg, err := env.DecodePayload()
	if err != nil {
		log.Printf("sse decode payload: %v", err)
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	// structural tracks whether the component tree changed (requiring a full view rebuild).
	// Data-model-only updates sync caches without rebuilding the view hierarchy,
	// preserving SwiftUI internal state like TabView selection.
	structural := false

	switch m := msg.(type) {
	case a2ui.CreateSurface:
		state.surfaceID = m.SurfaceID
		state.title = m.Title
		state.theme = m.Theme
		state.components = make(map[string]a2ui.Component, len(m.InitialComponents))
		for _, c := range m.InitialComponents {
			state.components[c.ID] = c
			if state.rootID == "" {
				state.rootID = c.ID
			}
		}
		if m.InitialDataModel != nil {
			state.dataModel.Data = m.InitialDataModel
			cache.syncFromDataModel(state.dataModel)
		}
		structural = true

	case a2ui.UpdateComponents:
		for _, c := range m.Components {
			state.components[c.ID] = c
		}
		// Set rootID if not yet established.
		if state.rootID == "" {
			if _, ok := state.components["root"]; ok {
				state.rootID = "root"
			} else {
				for _, c := range m.Components {
					state.rootID = c.ID
					break
				}
			}
		}
		structural = true

	case a2ui.UpdateDataModel:
		if m.Value != nil {
			if err := state.dataModel.Set(m.Path, m.Value); err != nil {
				log.Printf("data model set %q: %v", m.Path, err)
			}
		} else if m.Path != "" {
			if err := state.dataModel.Remove(m.Path); err != nil {
				log.Printf("data model remove %q: %v", m.Path, err)
			}
		}
		cache.syncFromDataModel(state.dataModel)

	case a2ui.DeleteSurface:
		state.surfaceID = ""
		state.rootID = ""
		state.theme = nil
		state.components = make(map[string]a2ui.Component)
		state.dataModel = a2ui.NewDataModel()
		structural = true
	}

	if structural {
		state.revision++
		revision.SetAnimated(state.revision)
	}
	statusRevision.SetAnimated(statusRevision.Get() + 1)
}

// renderComponent dispatches on component type and returns a SwiftUI view.
func renderComponent(comps map[string]a2ui.Component, dm *a2ui.DataModel, cache *stateCache, surfaceID, id string, theme *a2ui.Theme) swiftui.View {
	comp, ok := comps[id]
	if !ok {
		return swiftui.Text(fmt.Sprintf("[missing: %s]", id)).
			ForegroundStyleNamed("secondary").AsView()
	}

	props := comp.Properties
	if props == nil {
		props = make(map[string]any)
	}

	var view swiftui.View

	switch comp.Type {
	case a2ui.ComponentText:
		text := resolveText(props, "text", dm)
		view = applyTextVariant(swiftui.Text(text), propString(props, "variant"))

	case a2ui.ComponentButton:
		childID := propString(props, "child")
		if childID == "" && len(comp.Children) > 0 {
			childID = comp.Children[0]
		}
		variant := propString(props, "variant")
		actionName, actionCtx := resolveAction(props)
		compID := comp.ID

		var label swiftui.View
		if childID != "" {
			label = renderComponent(comps, dm, cache, surfaceID, childID, theme)
		} else {
			labelText := propString(props, "label")
			label = swiftui.Text(labelText).AsView()
		}

		btn := swiftui.ButtonView(label, func() {
			go postAction(surfaceID, compID, actionName, actionCtx)
		})
		switch variant {
		case "primary":
			btn = btn.ButtonStyle(swiftui.ButtonStyleBorderedProminent)
			if theme != nil && theme.PrimaryColor != "" {
				if r, g, b, a, ok := parseHexColor(theme.PrimaryColor); ok {
					btn = btn.Tint(r, g, b, a)
				}
			}
		case "borderless":
			btn = btn.ButtonStyle(swiftui.ButtonStyleBorderless)
		default:
			btn = btn.ButtonStyle(swiftui.ButtonStyleBordered)
		}
		view = btn

	case a2ui.ComponentTextField:
		label := propString(props, "label")
		if label == "" {
			label = propString(props, "placeholder")
		}
		binding := resolveBinding(props)
		actionName, actionCtx := resolveAction(props)
		variant := propString(props, "variant")
		compID := comp.ID
		s := cache.getString(binding, "")

		onSubmit := func() {
			ctx := actionCtx
			if ctx == nil {
				ctx = map[string]any{}
			}
			ctx["value"] = s.Get()
			go postAction(surfaceID, compID, actionName, ctx)
		}

		switch variant {
		case "longText":
			view = swiftui.TextEditor(s)
		case "obscured":
			view = swiftui.SecureField(label, s, onSubmit)
		default:
			view = swiftui.TextField(label, s, onSubmit).
				TextFieldStyle(swiftui.TextFieldStyleRoundedBorder)
		}

	case a2ui.ComponentCheckBox:
		label := propString(props, "label")
		binding := resolveBinding(props)
		actionName, actionCtx := resolveAction(props)
		compID := comp.ID
		s := cache.getInt(binding, 0)
		view = swiftui.Toggle(label, s, func() {
			ctx := actionCtx
			if ctx == nil {
				ctx = map[string]any{}
			}
			ctx["value"] = s.Get() != 0
			go postAction(surfaceID, compID, actionName, ctx)
		}).ToggleStyle(swiftui.ToggleStyleCheckbox)

	case a2ui.ComponentSlider:
		label := propString(props, "label")
		binding := resolveBinding(props)
		actionName, actionCtx := resolveAction(props)
		compID := comp.ID
		minVal := propFloat(props, "min", 0)
		maxVal := propFloat(props, "max", 100)
		s := cache.getInt(binding, 0)
		view = swiftui.Slider(label, s, minVal, maxVal, func() {
			ctx := actionCtx
			if ctx == nil {
				ctx = map[string]any{}
			}
			ctx["value"] = s.Get()
			go postAction(surfaceID, compID, actionName, ctx)
		})

	case a2ui.ComponentRow:
		align := propString(props, "align")
		justify := propString(props, "justify")
		spacing := propFloat(props, "spacing", 8)
		children := renderChildren(comps, dm, cache, surfaceID, comp.Children, theme)
		va := swiftui.VerticalAlignmentCenter
		switch align {
		case "start":
			va = swiftui.VerticalAlignmentTop
		case "end":
			va = swiftui.VerticalAlignmentBottom
		}
		if justify == "spaceBetween" {
			spaced := make([]swiftui.Viewable, 0, len(children)*2)
			for i, c := range children {
				if i > 0 {
					spaced = append(spaced, swiftui.Spacer())
				}
				spaced = append(spaced, c)
			}
			view = swiftui.HStackAlignedSpaced(va, spacing, spaced...)
		} else {
			view = swiftui.HStackAlignedSpaced(va, spacing, children...)
		}

	case a2ui.ComponentColumn:
		align := propString(props, "align")
		spacing := propFloat(props, "spacing", 8)
		children := renderChildren(comps, dm, cache, surfaceID, comp.Children, theme)
		ha := swiftui.HorizontalAlignmentCenter
		switch align {
		case "start":
			ha = swiftui.HorizontalAlignmentLeading
		case "stretch":
			ha = swiftui.HorizontalAlignmentLeading
		case "end":
			ha = swiftui.HorizontalAlignmentTrailing
		}
		view = swiftui.VStackAlignedSpaced(ha, spacing, children...)

	case a2ui.ComponentCard:
		child := renderFirstChild(comps, dm, cache, surfaceID, comp, theme)
		view = swiftui.VStack(child).
			Padding(16).
			BackgroundStyle("regularMaterial").
			CornerRadius(12).
			Shadow(0, 0, 0, 0.15, 6, 0, 3)

	case a2ui.ComponentList:
		direction := propString(props, "direction")
		children := renderChildren(comps, dm, cache, surfaceID, comp.Children, theme)
		if direction == "horizontal" {
			view = swiftui.ScrollView(swiftui.LazyHStack(children...))
		} else {
			view = swiftui.ScrollView(swiftui.LazyVStack(children...))
		}

	case a2ui.ComponentDivider:
		d := swiftui.Divider()
		if propString(props, "axis") == "vertical" {
			d = d.RotationEffect(90)
		}
		view = d

	case a2ui.ComponentIcon:
		name := propString(props, "name")
		if name == "" {
			name = propString(props, "systemName")
		}
		name = mapIconName(name)
		view = swiftui.Image(name)

	case a2ui.ComponentImage:
		name := propString(props, "name")
		src := propString(props, "src")
		if src == "" {
			src = propString(props, "url")
		}
		if name != "" {
			view = swiftui.Image(name)
		} else if src != "" {
			img := swiftui.AsyncImage(src)
			// Constrain async images based on variant hint.
			switch propString(props, "variant") {
			case "smallFeature":
				img = img.Frame(60, 60).CornerRadius(8)
			case "mediumFeature":
				img = img.Frame(120, 120).CornerRadius(12)
			case "largeFeature":
				img = img.Frame(240, 240).CornerRadius(16)
			default:
				img = img.Frame(120, 120).CornerRadius(8)
			}
			view = img
		} else {
			view = swiftui.Image("photo").ForegroundStyleNamed("secondary")
		}

	case a2ui.ComponentTabs:
		tabs := propSlice(props, "tabs")
		if len(tabs) > 0 {
			tabViews := make([]swiftui.Viewable, 0, len(tabs))
			for _, raw := range tabs {
				tab, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				title, _ := tab["title"].(string)
				childID, _ := tab["child"].(string)
				if childID == "" {
					continue
				}
				tabView := renderComponent(comps, dm, cache, surfaceID, childID, theme).
					TabItem(title, "rectangle.grid.1x2")
				tabViews = append(tabViews, tabView)
			}
			view = swiftui.TabView(tabViews...)
		} else {
			// Fall back to children as tabs.
			children := renderChildren(comps, dm, cache, surfaceID, comp.Children, theme)
			view = swiftui.TabView(children...)
		}

	case a2ui.ComponentModal:
		triggerID := propString(props, "trigger")
		contentID := propString(props, "content")
		modalState := cache.getModal(comp.ID)

		var triggerView swiftui.View
		if triggerID != "" {
			triggerView = renderComponent(comps, dm, cache, surfaceID, triggerID, theme)
		} else {
			triggerView = swiftui.Text("Open").AsView()
		}

		var contentView swiftui.View
		if contentID != "" {
			contentView = renderComponent(comps, dm, cache, surfaceID, contentID, theme)
		} else {
			contentView = swiftui.Text("").AsView()
		}

		view = triggerView.
			OnTapGesture(func() { modalState.Set(true) }).
			SheetPresented(modalState, contentView)

	case a2ui.ComponentChoicePicker:
		label := propString(props, "label")
		binding := resolveBinding(props)
		actionName, actionCtx := resolveAction(props)
		variant := propString(props, "variant")
		displayStyle := propString(props, "displayStyle")
		compID := comp.ID
		options := resolveOptions(props)
		s := cache.getInt(binding, 0)

		onChange := func() {
			ctx := actionCtx
			if ctx == nil {
				ctx = map[string]any{}
			}
			ctx["value"] = s.Get()
			go postAction(surfaceID, compID, actionName, ctx)
		}

		optionViews := make([]swiftui.Viewable, len(options))
		for i, opt := range options {
			optionViews[i] = swiftui.Text(opt)
		}
		optionsView := swiftui.VStack(optionViews...)

		if variant == "multipleSelection" {
			// VStack of toggles for multi-select.
			toggles := make([]swiftui.Viewable, len(options))
			for i, opt := range options {
				toggleState := cache.getInt(fmt.Sprintf("%s/%d", binding, i), 0)
				toggles[i] = swiftui.Toggle(opt, toggleState, func() {
					go postAction(surfaceID, compID, actionName, actionCtx)
				})
			}
			view = swiftui.VStackSpaced(4, toggles...)
		} else if displayStyle == "chips" {
			view = swiftui.PickerSegmented(label, s, optionsView, onChange)
		} else {
			view = swiftui.PickerMenu(label, s, optionsView, onChange)
		}

	case a2ui.ComponentDateTimeInput:
		label := propString(props, "label")
		binding := resolveBinding(props)
		actionName, actionCtx := resolveAction(props)
		compID := comp.ID
		s := cache.getDate(binding, float64(time.Now().Unix()))
		view = swiftui.DatePicker(label, s, func() {
			ctx := actionCtx
			if ctx == nil {
				ctx = map[string]any{}
			}
			ctx["value"] = s.Get()
			go postAction(surfaceID, compID, actionName, ctx)
		})

	case a2ui.ComponentVideo:
		url := propString(props, "src")
		if url == "" {
			url = propString(props, "url")
		}
		view = swiftui.VStackSpaced(8,
			swiftui.Image("play.rectangle.fill").
				ImageScale(swiftui.ImageScaleLarge).
				ForegroundStyleNamed("secondary"),
			swiftui.Text(url).Font(swiftui.FontCaption),
		)

	case a2ui.ComponentAudioPlayer:
		description := propString(props, "description")
		if description == "" {
			description = propString(props, "label")
		}
		view = swiftui.HStackSpaced(8,
			swiftui.Image("waveform").
				ForegroundStyleNamed("secondary"),
			swiftui.Text(description).Font(swiftui.FontCaption),
		)

	case a2ui.ComponentProgress:
		if v, ok := props["value"]; ok {
			val := resolveNumericValue(v, dm)
			maxVal := propFloat(props, "max", 1.0)
			view = swiftui.ProgressLinear(val, maxVal)
		} else {
			view = swiftui.ProgressSpinning()
		}

	default:
		view = swiftui.Text(fmt.Sprintf("[unsupported: %s]", comp.Type)).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").AsView()
	}

	// Apply accessibility if present.
	if acc, ok := props["accessibility"].(map[string]any); ok {
		if label, ok := acc["label"].(string); ok && label != "" {
			view = view.AccessibilityLabel(label)
		}
		if desc, ok := acc["description"].(string); ok && desc != "" {
			view = view.AccessibilityHint(desc)
		}
	}

	return view
}

// applyTextVariant applies font and style modifiers based on the text variant.
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

// resolveText resolves a text property that may be a string literal or a DynamicValue binding.
// It also strips simple markdown heading markers (e.g. "# Title" → "Title").
func resolveText(props map[string]any, key string, dm *a2ui.DataModel) string {
	v, ok := props[key]
	if !ok {
		return ""
	}
	var text string
	switch t := v.(type) {
	case string:
		text = t
	case map[string]any:
		if path, ok := t["path"].(string); ok {
			if val, err := dm.Get(path); err == nil {
				text = fmt.Sprintf("%v", val)
			}
		}
	default:
		text = fmt.Sprintf("%v", v)
	}
	// Strip markdown heading markers ("# Title" → "Title").
	for strings.HasPrefix(text, "#") {
		text = strings.TrimPrefix(text, "#")
	}
	text = strings.TrimLeft(text, " ")
	return text
}

func renderChildren(comps map[string]a2ui.Component, dm *a2ui.DataModel, cache *stateCache, surfaceID string, childIDs []string, theme *a2ui.Theme) []swiftui.Viewable {
	views := make([]swiftui.Viewable, 0, len(childIDs))
	for _, cid := range childIDs {
		views = append(views, renderComponent(comps, dm, cache, surfaceID, cid, theme))
	}
	if len(views) == 0 {
		views = append(views, swiftui.Text("").AsView())
	}
	return views
}

func renderFirstChild(comps map[string]a2ui.Component, dm *a2ui.DataModel, cache *stateCache, surfaceID string, comp a2ui.Component, theme *a2ui.Theme) swiftui.View {
	if len(comp.Children) > 0 {
		return renderComponent(comps, dm, cache, surfaceID, comp.Children[0], theme)
	}
	// v0.9 wire format uses "child" property for single-child containers (e.g. Card).
	if childID := propString(comp.Properties, "child"); childID != "" {
		return renderComponent(comps, dm, cache, surfaceID, childID, theme)
	}
	return swiftui.Text("").AsView()
}

// postAction sends a ClientAction to the server's /action endpoint.

// loadFromFiles loads components and data from files for offline rendering.
// Supports three formats:
//   - JSON array of Component objects (-components file.json)
//   - JSONL with A2UI envelope messages (-components file.jsonl)
//   - Separate data model file (-data data.json)
func loadFromFiles(state *surfaceState, cache *stateCache, componentsPath, dataPath string) error {
	compData, err := os.ReadFile(componentsPath)
	if err != nil {
		return fmt.Errorf("read components: %w", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	state.surfaceID = "file"
	state.title = "A2UI (file)"
	state.components = make(map[string]a2ui.Component)

	// Detect format: JSONL (line-delimited envelopes) vs JSON array.
	trimmed := bytes.TrimSpace(compData)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		// JSONL: each line is an A2UI envelope message.
		if err := loadJSONL(state, cache, compData); err != nil {
			return err
		}
	} else {
		// JSON array of components.
		var comps []a2ui.Component
		if err := json.Unmarshal(compData, &comps); err != nil {
			return fmt.Errorf("parse components: %w", err)
		}
		for _, c := range comps {
			state.components[c.ID] = c
		}
	}

	// Set root ID.
	if _, ok := state.components["root"]; ok {
		state.rootID = "root"
	} else {
		// Find the first component (any will do).
		for id := range state.components {
			state.rootID = id
			break
		}
	}

	if dataPath != "" {
		dataBytes, err := os.ReadFile(dataPath)
		if err != nil {
			return fmt.Errorf("read data: %w", err)
		}
		var dm map[string]any
		if err := json.Unmarshal(dataBytes, &dm); err != nil {
			return fmt.Errorf("parse data: %w", err)
		}
		state.dataModel.Data = dm
		cache.syncFromDataModel(state.dataModel)
	}

	state.revision = 1
	return nil
}

// loadJSONL processes line-delimited A2UI envelope messages.
func loadJSONL(state *surfaceState, cache *stateCache, data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var env a2ui.Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			log.Printf("jsonl unmarshal: %v", err)
			continue
		}
		msg, err := env.DecodePayload()
		if err != nil {
			log.Printf("jsonl decode: %v", err)
			continue
		}
		switch m := msg.(type) {
		case a2ui.CreateSurface:
			state.surfaceID = m.SurfaceID
			state.title = m.Title
			state.theme = m.Theme
		case a2ui.UpdateComponents:
			for _, c := range m.Components {
				state.components[c.ID] = c
			}
		case a2ui.UpdateDataModel:
			if m.Value != nil {
				state.dataModel.Set(m.Path, m.Value)
			}
			cache.syncFromDataModel(state.dataModel)
		case a2ui.DeleteSurface:
			// Skip in file mode.
		}
	}
	return scanner.Err()
}

func postAction(surfaceID, componentID, actionName string, ctx map[string]any) {
	if actionName == "" {
		return
	}
	action := a2ui.ClientAction{
		Name:              actionName,
		SurfaceID:         surfaceID,
		SourceComponentID: componentID,
		Context:           ctx,
	}
	body, err := json.Marshal(action)
	if err != nil {
		log.Printf("marshal action: %v", err)
		return
	}

	actionURL := strings.TrimSuffix(*serverFlag, "/sse") + "/action"
	resp, err := http.Post(actionURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("post action: %v", err)
		return
	}
	resp.Body.Close()
}

// Property helpers.

func propString(props map[string]any, key string) string {
	v, ok := props[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func propFloat(props map[string]any, key string, def float64) float64 {
	v, ok := props[key]
	if !ok {
		return def
	}
	return toFloat(v)
}

func propSlice(props map[string]any, key string) []any {
	v, ok := props[key]
	if !ok {
		return nil
	}
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	return s
}

// resolveOptions extracts option labels from either []string or []{label,value} format.
func resolveOptions(props map[string]any) []string {
	raw := propSlice(props, "options")
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch opt := v.(type) {
		case string:
			result = append(result, opt)
		case map[string]any:
			if label, ok := opt["label"].(string); ok {
				result = append(result, label)
			}
		}
	}
	return result
}

// resolveAction extracts an action name and context from the "action" property.
// Supports: string "actionName", or {"event":{"name":"...", "context":{...}}}.
func resolveAction(props map[string]any) (string, map[string]any) {
	v, ok := props["action"]
	if !ok {
		return "", nil
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	if m, ok := v.(map[string]any); ok {
		if ev, ok := m["event"].(map[string]any); ok {
			name, _ := ev["name"].(string)
			ctx, _ := ev["context"].(map[string]any)
			return name, ctx
		}
	}
	return "", nil
}

// resolveBinding extracts a data model path from a "value" property.
// Supports: {"path":"/foo"} or a plain string path "/foo" or legacy "binding" key.
func resolveBinding(props map[string]any) string {
	if v, ok := props["value"]; ok {
		if m, ok := v.(map[string]any); ok {
			if p, ok := m["path"].(string); ok {
				return p
			}
		}
		if s, ok := v.(string); ok {
			return s
		}
	}
	// Legacy: "binding" key.
	if b, ok := props["binding"].(string); ok {
		return b
	}
	return ""
}

// resolveNumericValue resolves a value that may be a literal number or a data model binding.
func resolveNumericValue(v any, dm *a2ui.DataModel) float64 {
	if m, ok := v.(map[string]any); ok {
		if path, ok := m["path"].(string); ok {
			if val, err := dm.Get(path); err == nil {
				return toFloat(val)
			}
		}
	}
	return toFloat(v)
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f
		}
	}
	return 0
}

// mapIconName maps A2UI icon names to SF Symbol names.
func mapIconName(name string) string {
	m := map[string]string{
		"flight":    "airplane",
		"airplane":  "airplane",
		"search":    "magnifyingglass",
		"settings":  "gearshape",
		"home":      "house",
		"person":    "person",
		"mail":      "envelope",
		"phone":     "phone",
		"calendar":  "calendar",
		"clock":     "clock",
		"location":  "location",
		"heart":     "heart",
		"star":      "star",
		"bookmark":  "bookmark",
		"share":     "square.and.arrow.up",
		"delete":    "trash",
		"edit":      "pencil",
		"add":       "plus",
		"remove":    "minus",
		"close":     "xmark",
		"check":     "checkmark",
		"warning":   "exclamationmark.triangle",
		"error":     "xmark.circle",
		"info":      "info.circle",
		"help":      "questionmark.circle",
		"refresh":   "arrow.clockwise",
		"download":  "arrow.down.circle",
		"upload":    "arrow.up.circle",
		"play":      "play.fill",
		"pause":     "pause.fill",
		"stop":      "stop.fill",
		"forward":   "forward.fill",
		"backward":  "backward.fill",
		"menu":      "line.3.horizontal",
		"more":      "ellipsis",
		"filter":    "line.3.horizontal.decrease",
		"sort":      "arrow.up.arrow.down",
		"list":      "list.bullet",
		"grid":      "square.grid.2x2",
		"lock":      "lock",
		"unlock":    "lock.open",
		"eye":       "eye",
		"eye-off":   "eye.slash",
		"camera":    "camera",
		"image":     "photo",
		"video":     "video",
		"mic":       "mic",
		"speaker":   "speaker.wave.2",
		"wifi":      "wifi",
		"bluetooth": "antenna.radiowaves.left.and.right",
		"battery":   "battery.100",
		"bolt":      "bolt",
		"sun":       "sun.max",
		"moon":      "moon",
		"cloud":     "cloud",
		"rain":      "cloud.rain",
		"snow":      "cloud.snow",
		"wind":      "wind",
		"fog":       "cloud.fog",
	}
	if sfName, ok := m[name]; ok {
		return sfName
	}
	return name
}

// parseHexColor parses a hex color string (#RRGGBB or #RRGGBBAA) into RGBA floats.
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
		return float64(rv) / 255.0, float64(gv) / 255.0, float64(bv) / 255.0, 1.0, true
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
		return float64(rv) / 255.0, float64(gv) / 255.0, float64(bv) / 255.0, float64(av) / 255.0, true
	}
	return 0, 0, 0, 0, false
}
