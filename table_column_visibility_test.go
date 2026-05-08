package swiftui

import (
	"slices"
	"testing"
)

func TestTableColumnVisibilityState(t *testing.T) {
	state := NewTableColumnVisibilityState("owner")
	defer state.Release()

	if state.Visible("owner") {
		t.Fatal("owner should start hidden")
	}
	if !state.Visible("status") {
		t.Fatal("status should start visible")
	}

	state.Toggle("owner")
	if !state.Visible("owner") {
		t.Fatal("owner should become visible")
	}

	state.SetVisible("status", false)
	if state.Visible("status") {
		t.Fatal("status should become hidden")
	}

	hidden := state.HiddenIDs()
	slices.Sort(hidden)
	if got, want := len(hidden), 1; got != want || hidden[0] != "status" {
		t.Fatalf("HiddenIDs() = %v, want [status]", hidden)
	}
}

func TestTableColumnVisibilityStateBulkHelpers(t *testing.T) {
	state := NewTableColumnVisibilityState("owner", "status")
	defer state.Release()

	state.SetVisibleAll(true, "owner", "status")
	if !state.Visible("owner") || !state.Visible("status") {
		t.Fatalf("SetVisibleAll(true) should show owner and status")
	}

	state.ShowOnly([]string{"service", "owner", "status", "load"}, "service", "load")
	if !state.Visible("service") || !state.Visible("load") {
		t.Fatalf("ShowOnly should keep service and load visible")
	}
	if state.Visible("owner") || state.Visible("status") {
		t.Fatalf("ShowOnly should hide owner and status")
	}
	if got, want := state.HiddenIDs(), []string{"owner", "status"}; !slices.Equal(got, want) {
		t.Fatalf("HiddenIDs() = %v, want %v", got, want)
	}
}

func TestTableColumnVisibilityStateReplaceHiddenIDs(t *testing.T) {
	state := NewTableColumnVisibilityState("owner")
	defer state.Release()

	state.ReplaceHiddenIDs("status", "load", "", "load")
	if got, want := state.HiddenIDs(), []string{"load", "status"}; !slices.Equal(got, want) {
		t.Fatalf("HiddenIDs() = %v, want %v", got, want)
	}
	if got, want := state.VisibleIDs([]string{"service", "owner", "status", "load"}), []string{"service", "owner"}; !slices.Equal(got, want) {
		t.Fatalf("VisibleIDs() = %v, want %v", got, want)
	}

	state.ReplaceHiddenIDs("status", "load")
	if got, want := state.HiddenIDs(), []string{"load", "status"}; !slices.Equal(got, want) {
		t.Fatalf("HiddenIDs() after no-op replace = %v, want %v", got, want)
	}
}

func TestTableColumnVisibilityStateApply(t *testing.T) {
	state := NewTableColumnVisibilityState("owner")
	defer state.Release()

	columns := []TableModelColumn[tableModelRow]{
		TextTableModelColumn("Name", func(row tableModelRow) string { return row.Name }, nil).WithID("name"),
		TextTableModelColumn("Owner", func(row tableModelRow) string { return row.Name }, nil).WithID("owner"),
	}
	got := ApplyTableColumnVisibility(state, columns)
	if got[0].Hidden {
		t.Fatal("name should stay visible")
	}
	if !got[1].Hidden {
		t.Fatal("owner should be hidden")
	}
	if columns[1].Hidden {
		t.Fatal("Apply should not mutate the input slice")
	}
}

func TestTableColumnPresetState(t *testing.T) {
	visibility := NewTableColumnVisibilityState("owner", "load")
	defer visibility.Release()
	presets := NewTableColumnPresetState(
		TableColumnPreset{ID: "ops", Label: "Ops", HiddenIDs: []string{"owner"}},
		TableColumnPreset{ID: "compact", HiddenIDs: []string{"load", "owner", "owner"}},
	)
	defer presets.Release()

	if got, want := presets.PresetIDs(), []string{"compact", "ops"}; !slices.Equal(got, want) {
		t.Fatalf("PresetIDs() = %v, want %v", got, want)
	}
	if preset, ok := presets.Preset("compact"); !ok || preset.Label != "compact" || !slices.Equal(preset.HiddenIDs, []string{"load", "owner"}) {
		t.Fatalf("Preset(compact) = %+v %v, want normalized preset true", preset, ok)
	}

	if !presets.ApplyPreset("ops", visibility) {
		t.Fatal("ApplyPreset(ops) = false, want true")
	}
	if got, want := visibility.HiddenIDs(), []string{"owner"}; !slices.Equal(got, want) {
		t.Fatalf("HiddenIDs() after ApplyPreset = %v, want %v", got, want)
	}
	if got, want := presets.CurrentPresetID(), "ops"; got != want {
		t.Fatalf("CurrentPresetID() = %q, want %q", got, want)
	}

	visibility.ReplaceHiddenIDs("status", "load")
	if !presets.SavePreset("custom", "Custom", visibility) {
		t.Fatal("SavePreset(custom) = false, want true")
	}
	if got, want := presets.CurrentPresetID(), "custom"; got != want {
		t.Fatalf("CurrentPresetID() after save = %q, want %q", got, want)
	}
	if preset, ok := presets.Preset("custom"); !ok || !slices.Equal(preset.HiddenIDs, []string{"load", "status"}) {
		t.Fatalf("Preset(custom) = %+v %v, want hidden [load status] true", preset, ok)
	}

	if !presets.DeletePreset("custom") {
		t.Fatal("DeletePreset(custom) = false, want true")
	}
	if _, ok := presets.Preset("custom"); ok {
		t.Fatal("Preset(custom) should be deleted")
	}
	if got := presets.CurrentPresetID(); got != "" {
		t.Fatalf("CurrentPresetID() after delete = %q, want empty", got)
	}
}

func TestTableColumnPresetSnapshotRoundTrip(t *testing.T) {
	presets := NewTableColumnPresetState(
		TableColumnPreset{ID: "ops", Label: "Ops", HiddenIDs: []string{"owner"}},
	)
	defer presets.Release()
	visibility := NewTableColumnVisibilityState("load", "status")
	defer visibility.Release()
	if !presets.SavePreset("custom", "Custom", visibility) {
		t.Fatal("SavePreset(custom) = false, want true")
	}

	snapshot := presets.Snapshot()
	restored := NewTableColumnPresetState()
	defer restored.Release()
	if !restored.ReplaceSnapshot(snapshot) {
		t.Fatal("ReplaceSnapshot(snapshot) = false, want true")
	}
	if got, want := restored.PresetIDs(), []string{"custom", "ops"}; !slices.Equal(got, want) {
		t.Fatalf("PresetIDs() after restore = %v, want %v", got, want)
	}
	if got, want := restored.CurrentPresetID(), "custom"; got != want {
		t.Fatalf("CurrentPresetID() after restore = %q, want %q", got, want)
	}
	if preset, ok := restored.Preset("custom"); !ok || !slices.Equal(preset.HiddenIDs, []string{"load", "status"}) {
		t.Fatalf("Preset(custom) after restore = %+v %v, want hidden [load status] true", preset, ok)
	}
}

func TestTableColumnLayoutSnapshotRoundTrip(t *testing.T) {
	visibility := NewTableColumnVisibilityState("owner", "status")
	defer visibility.Release()
	widths := NewTableColumnWidthState(map[string]float64{"service": 240, "owner": 120})
	defer widths.Release()
	presets := NewTableColumnPresetState(
		TableColumnPreset{ID: "all", Label: "All"},
		TableColumnPreset{ID: "compact", Label: "Compact", HiddenIDs: []string{"owner", "status"}},
	)
	defer presets.Release()
	if !presets.ApplyPreset("compact", visibility) {
		t.Fatal("ApplyPreset(compact) = false, want true")
	}

	snapshot := CaptureTableColumnLayoutSnapshot(visibility, widths, presets)

	visibility.ReplaceHiddenIDs("load")
	widths.ReplaceWidths(map[string]float64{"service": 999})
	presets.ReplaceSnapshot(TableColumnPresetSnapshot{
		CurrentPresetID: "temporary",
		Presets: []TableColumnPreset{{
			ID:        "temporary",
			Label:     "Temporary",
			HiddenIDs: []string{"load"},
		}},
	})

	ApplyTableColumnLayoutSnapshot(snapshot, visibility, widths, presets)

	if got, want := visibility.HiddenIDs(), []string{"owner", "status"}; !slices.Equal(got, want) {
		t.Fatalf("HiddenIDs() after snapshot restore = %v, want %v", got, want)
	}
	if got, want := widths.Widths()["service"], 240.0; got != want {
		t.Fatalf("Widths()[service] after snapshot restore = %v, want %v", got, want)
	}
	if got, want := presets.CurrentPresetID(), "compact"; got != want {
		t.Fatalf("CurrentPresetID() after snapshot restore = %q, want %q", got, want)
	}
	if got, want := presets.PresetIDs(), []string{"all", "compact"}; !slices.Equal(got, want) {
		t.Fatalf("PresetIDs() after snapshot restore = %v, want %v", got, want)
	}
}
