package swiftui

// PhotosPicker renders a concrete picker surface backed by PhotosPickerSelectionState.
//
// When items are omitted, it uses the native PhotosPicker bridge with a
// concrete selection model. When items are provided, it falls back to the
// curated sample-item menu used by the example surfaces.
func PhotosPicker(label string, selection *PhotosPickerSelectionState, items ...PhotosPickerItem) View {
	if len(items) == 0 {
		return PhotosPickerNative(label, selection, PhotosPickerConfig{})
	}
	return PhotosPickerMenu(label, selection, items...)
}

// PhotosPickerMenu renders the curated sample-item picker surface.
//
// This remains useful in examples and tests where deterministic local sample
// items are preferable to opening the system photo library. Sample items may
// also carry lazy file handles for deterministic file-backed preview flows.
func PhotosPickerMenu(label string, selection *PhotosPickerSelectionState, items ...PhotosPickerItem) View {
	if label == "" {
		label = "Choose Photos"
	}
	if selection == nil {
		return Button(label, func() {})
	}
	options := make([]Viewable, 0, len(items)+1)
	for _, item := range items {
		item := item
		if item.ID == "" {
			continue
		}
		title := item.Filename
		if title == "" {
			title = item.ID
		}
		icon := "circle"
		if selection.Has(item.ID) {
			icon = "checkmark.circle.fill"
		}
		options = append(options, ButtonWithLabel(title, icon, func() {
			selection.Toggle(item)
		}))
	}
	if len(options) == 0 {
		options = append(options, Text("No photo options configured").Font(FontCaption).ForegroundStyleNamed("secondary").AsView())
	}
	options = append(options, Divider(), ButtonWithLabel("Clear Selection", "trash", func() {
		selection.Clear()
	}))
	return Menu(label, VStackSpaced(8, options...))
}
