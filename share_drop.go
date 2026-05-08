package swiftui

import "runtime"

// ShareItem describes one concrete payload that can be handed to ShareLinkItem.
//
// Bridge surface.
//
// Exactly one of Text, URL, or FilePath should be set.
type ShareItem struct {
	Title    string
	Text     string
	URL      string
	FilePath string
}

// PastePayload describes one concrete payload read from the clipboard.
//
// Bridge surface.
//
// Exactly one of Text, URL, or FilePath should be set.
type PastePayload struct {
	Text     string
	URL      string
	FilePath string
}

// DropPayload describes one concrete payload delivered from a drop destination.
//
// Bridge surface.
//
// Exactly one of Text, URL, or FilePath should be set.
type DropPayload struct {
	Text     string
	URL      string
	FilePath string
}

// Kind reports the normalized payload kind.
func (item ShareItem) Kind() string {
	kind, _, ok := singlePayload(item.Text, item.URL, item.FilePath)
	if !ok {
		return ""
	}
	return kind
}

// Value reports the normalized payload value.
func (item ShareItem) Value() string {
	_, value, ok := singlePayload(item.Text, item.URL, item.FilePath)
	if !ok {
		return ""
	}
	return value
}

// Kind reports the normalized payload kind.
func (item PastePayload) Kind() string {
	kind, _, ok := singlePayload(item.Text, item.URL, item.FilePath)
	if !ok {
		return ""
	}
	return kind
}

// Value reports the normalized payload value.
func (item PastePayload) Value() string {
	_, value, ok := singlePayload(item.Text, item.URL, item.FilePath)
	if !ok {
		return ""
	}
	return value
}

// Kind reports the normalized payload kind.
func (item DropPayload) Kind() string {
	kind, _, ok := singlePayload(item.Text, item.URL, item.FilePath)
	if !ok {
		return ""
	}
	return kind
}

// Value reports the normalized payload value.
func (item DropPayload) Value() string {
	_, value, ok := singlePayload(item.Text, item.URL, item.FilePath)
	if !ok {
		return ""
	}
	return value
}

func shareItemPayload(item ShareItem) (kind, value string, ok bool) {
	return singlePayload(item.Text, item.URL, item.FilePath)
}

func dropPayloadPayload(item DropPayload) (kind, value string, ok bool) {
	return singlePayload(item.Text, item.URL, item.FilePath)
}

func singlePayload(text, url, filePath string) (kind, value string, ok bool) {
	set := 0
	if text != "" {
		set++
	}
	if url != "" {
		set++
	}
	if filePath != "" {
		set++
	}
	if set != 1 {
		return "", "", false
	}
	switch {
	case text != "":
		return "text", text, true
	case url != "":
		return "url", url, true
	default:
		return "file", filePath, true
	}
}

// ShareLinkItem creates a share button from a concrete share payload.
// Exactly one payload field must be set on item.
func ShareLinkItem(title string, item ShareItem) View {
	if title == "" {
		title = item.Title
	}
	if title == "" {
		title = "Share"
	}
	kind, value, ok := shareItemPayload(item)
	if !ok {
		return Button(title, func() {})
	}
	return ShareLinkItemRaw(title, kind, value, item.FilePath)
}

// Draggable makes a view draggable with a concrete payload.
func (v View) Draggable(item ShareItem) View {
	switch item.Kind() {
	case "text":
		return v.DraggableText(item.Text)
	case "url":
		return v.DraggableURL(item.URL)
	case "file":
		return v.DraggableFileURL(item.FilePath)
	default:
		runtime.KeepAlive(v.retained)
		return v
	}
}

// DropDestination accepts a concrete payload from a drop interaction.
func (v View) DropDestination(action func(DropPayload) bool) View {
	if action == nil {
		runtime.KeepAlive(v.retained)
		return v
	}
	withText := func(text string) bool {
		return action(DropPayload{Text: text})
	}
	withURL := func(url string) bool {
		return action(DropPayload{URL: url})
	}
	withFile := func(path string) bool {
		return action(DropPayload{FilePath: path})
	}
	ret := v.DropDestinationText(withText)
	ret = ret.DropDestinationURL(withURL)
	ret = ret.DropDestinationFileURL(withFile)
	runtime.KeepAlive(v.retained)
	return ret
}
