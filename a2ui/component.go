package a2ui

// Component represents a single UI component in the A2UI tree.
type Component struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Children   []string       `json:"children,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}
