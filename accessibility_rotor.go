package swiftui

import (
	"encoding/json"
	"strings"
)

// AccessibilityRotorEntry identifies one rotor stop by stable ID and label.
//
// Curated surface.
//
// The zero value is not usable. Use NewAccessibilityRotorEntry or build entries
// through NewAccessibilityRotorModel.
type AccessibilityRotorEntry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// NewAccessibilityRotorEntry creates a normalized rotor entry.
func NewAccessibilityRotorEntry(label, id string) AccessibilityRotorEntry {
	return AccessibilityRotorEntry{
		ID:    strings.TrimSpace(id),
		Label: strings.TrimSpace(label),
	}
}

// AccessibilityRotorModel owns the rotor label and entries used by the bridge.
//
// Curated surface.
//
// The zero value is not usable. Use NewAccessibilityRotorModel.
type AccessibilityRotorModel struct {
	Name    string                    `json:"name"`
	Entries []AccessibilityRotorEntry `json:"entries"`
}

// NewAccessibilityRotorModel creates a normalized rotor model.
func NewAccessibilityRotorModel(name string, entries ...AccessibilityRotorEntry) AccessibilityRotorModel {
	model := AccessibilityRotorModel{Name: strings.TrimSpace(name)}
	for _, entry := range entries {
		if normalized, ok := normalizeAccessibilityRotorEntry(entry); ok {
			model.Entries = append(model.Entries, normalized)
		}
	}
	return model
}

// Valid reports whether the model contains enough data for the bridge.
func (m AccessibilityRotorModel) Valid() bool {
	return strings.TrimSpace(m.Name) != "" && len(m.Entries) > 0
}

func normalizeAccessibilityRotorEntry(entry AccessibilityRotorEntry) (AccessibilityRotorEntry, bool) {
	entry.ID = strings.TrimSpace(entry.ID)
	entry.Label = strings.TrimSpace(entry.Label)
	return entry, entry.ID != "" && entry.Label != ""
}

func encodeAccessibilityRotorModel(model AccessibilityRotorModel) (string, bool) {
	if !model.Valid() {
		return "", false
	}
	data, err := json.Marshal(model)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// AccessibilityRotor attaches a rotor definition to the current view.
func (v View) AccessibilityRotor(model AccessibilityRotorModel) View {
	payload, ok := encodeAccessibilityRotorModel(model)
	if !ok {
		return v
	}
	return v.AccessibilityRotorJSON(payload)
}
