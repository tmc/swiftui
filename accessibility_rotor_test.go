package swiftui

import (
	"encoding/json"
	"testing"
)

func TestAccessibilityRotorModelNormalize(t *testing.T) {
	model := NewAccessibilityRotorModel("  Transcript navigation  ",
		NewAccessibilityRotorEntry("  Overview  ", " overview "),
		AccessibilityRotorEntry{Label: "   ", ID: "skip"},
		AccessibilityRotorEntry{Label: "Summary", ID: " summary "},
	)

	if got, want := model.Name, "Transcript navigation"; got != want {
		t.Fatalf("model.Name = %q, want %q", got, want)
	}
	if got, want := len(model.Entries), 2; got != want {
		t.Fatalf("len(model.Entries) = %d, want %d", got, want)
	}
	if got, want := model.Entries[0], (AccessibilityRotorEntry{Label: "Overview", ID: "overview"}); got != want {
		t.Fatalf("model.Entries[0] = %#v, want %#v", got, want)
	}
	if got, want := model.Entries[1], (AccessibilityRotorEntry{Label: "Summary", ID: "summary"}); got != want {
		t.Fatalf("model.Entries[1] = %#v, want %#v", got, want)
	}

	payload, ok := encodeAccessibilityRotorModel(model)
	if !ok {
		t.Fatal("encodeAccessibilityRotorModel(model) = false, want true")
	}

	var decoded struct {
		Name    string                    `json:"name"`
		Entries []AccessibilityRotorEntry `json:"entries"`
	}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(payload) failed: %v", err)
	}
	if decoded.Name != model.Name {
		t.Fatalf("decoded.Name = %q, want %q", decoded.Name, model.Name)
	}
	if len(decoded.Entries) != len(model.Entries) {
		t.Fatalf("decoded entries = %d, want %d", len(decoded.Entries), len(model.Entries))
	}
}

func TestAccessibilityRotorModelInvalid(t *testing.T) {
	if got := (AccessibilityRotorModel{}).Valid(); got {
		t.Fatal("zero AccessibilityRotorModel should be invalid")
	}
	if got, ok := encodeAccessibilityRotorModel(AccessibilityRotorModel{}); ok || got != "" {
		t.Fatalf("encodeAccessibilityRotorModel(zero) = %q, %v, want empty, false", got, ok)
	}
}
