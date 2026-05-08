package swiftui

import (
	"maps"
	"testing"
)

func TestTableColumnWidthState(t *testing.T) {
	state := NewTableColumnWidthState(map[string]float64{
		"service": 240,
		"load":    96,
	})
	defer state.Release()

	if got, ok := state.Width("service"); !ok || got != 240 {
		t.Fatalf("Width(service) = %v %v, want 240 true", got, ok)
	}
	if got, ok := state.Width("owner"); ok || got != 0 {
		t.Fatalf("Width(owner) = %v %v, want 0 false", got, ok)
	}

	state.SetWidth("owner", 180)
	if got, ok := state.Width("owner"); !ok || got != 180 {
		t.Fatalf("Width(owner) after set = %v %v, want 180 true", got, ok)
	}
	state.ClearWidth("load")
	if got, ok := state.Width("load"); ok || got != 0 {
		t.Fatalf("Width(load) after clear = %v %v, want 0 false", got, ok)
	}
	if got, want := state.WidthIDs(), []string{"owner", "service"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("WidthIDs() = %v, want %v", got, want)
	}
}

func TestTableColumnWidthStateReplaceWidths(t *testing.T) {
	state := NewTableColumnWidthState(map[string]float64{"service": 240})
	defer state.Release()

	state.ReplaceWidths(map[string]float64{
		"owner":  180,
		"status": 144,
		"load":   -1,
		"":       99,
	})
	if got, want := state.Widths(), map[string]float64{"owner": 180, "status": 144}; !maps.Equal(got, want) {
		t.Fatalf("Widths() = %v, want %v", got, want)
	}
}
