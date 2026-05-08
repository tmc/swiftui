package swiftui

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNativeSelectionState(t *testing.T) {
	s := NewNativeSelectionState("b", "a")
	defer s.Release()

	if got, ok := s.SelectedID(); !ok || got != "b" {
		t.Fatalf("SelectedID() = %q %v, want b true", got, ok)
	}
	if got, want := s.SelectedIDs(), []string{"b", "a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() = %v, want %v", got, want)
	}
	if got, ok := s.AnchorID(); !ok || got != "b" {
		t.Fatalf("AnchorID() = %q %v, want b true", got, ok)
	}
	if !s.Add("c") {
		t.Fatal("Add(c) = false, want true")
	}
	if got, want := s.SelectedCount(), 3; got != want {
		t.Fatalf("SelectedCount() = %d, want %d", got, want)
	}
	s.Clear()
	if _, ok := s.SelectedID(); ok {
		t.Fatal("SelectedID() after clear = true, want false")
	}
}

func TestNativeRowAccessibilityMetadataHelpers(t *testing.T) {
	if got, want := tableRowAccessibilityID("service-2"), "table-row-service-2"; got != want {
		t.Fatalf("table row accessibility id = %q, want %q", got, want)
	}
	if got, want := outlineRowAccessibilityID("node-2"), "outline-row-node-2"; got != want {
		t.Fatalf("outline row accessibility id = %q, want %q", got, want)
	}
	if got, want := rowStateSummary(true, false, false), "selected"; got != want {
		t.Fatalf("rowStateSummary(selected) = %q, want %q", got, want)
	}
	if got, want := rowStateSummary(true, true, true), "selected, expanded"; got != want {
		t.Fatalf("rowStateSummary(branch) = %q, want %q", got, want)
	}
	if got, want := rowStateSummary(false, false, true), "collapsed"; got != want {
		t.Fatalf("rowStateSummary(collapsed) = %q, want %q", got, want)
	}
	if got, want := rowStateSummary(false, true, true), "expanded"; got != want {
		t.Fatalf("rowStateSummary(expanded) = %q, want %q", got, want)
	}
}

func TestNativeSelectedRowStateSummaries(t *testing.T) {
	table := NewNativeTableModel([]NativeTableRow{
		{ID: "edge"},
		{ID: "search"},
	}, NewNativeTableSelectionState("search"))
	defer table.Release()
	defer table.SelectionState().Release()

	if got, want := table.SelectedRowStateSummary(), "selected"; got != want {
		t.Fatalf("SelectedRowStateSummary(table) = %q, want %q", got, want)
	}

	outline := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "branch",
			Label: Text("Branch").AsView(),
			Children: []NativeOutlineNode{{
				ID:    "leaf",
				Label: Text("Leaf").AsView(),
			}},
		}},
	}}, NewNativeOutlineSelectionState("branch"), NewNativeOutlineExpansionState("root", "branch"))
	defer outline.Release()
	defer outline.SelectionState().Release()
	defer outline.ExpansionState().Release()

	if got, want := outline.SelectedRowStateSummary(), "selected, expanded"; got != want {
		t.Fatalf("SelectedRowStateSummary(outline) = %q, want %q", got, want)
	}
}

func TestNativeTableModel(t *testing.T) {
	model := NewNativeTableModel([]NativeTableRow{
		{ID: "a"},
		{ID: "b"},
	}, nil)
	defer model.Release()
	defer model.SelectionState().Release()

	if !model.SelectID("b") {
		t.Fatal("SelectID(b) = false, want true")
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "b" {
		t.Fatalf("SelectedRow() = %+v %v, want b true", got, ok)
	}
	if !model.RevealID("a") {
		t.Fatal("RevealID(a) = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "a" {
		t.Fatalf("SelectedID() after reveal = %q %v, want a true", got, ok)
	}
	var activated string
	model.SetOnActivate(func(row NativeTableRow) {
		activated = row.ID
	})
	if !model.ActivateSelected() {
		t.Fatal("ActivateSelected() = false, want true")
	}
	if activated != "a" {
		t.Fatalf("activated = %q, want a", activated)
	}
	model.SetRows([]NativeTableRow{{ID: "z"}})
	if _, ok := model.SelectionState().SelectedID(); ok {
		t.Fatal("SelectedID() after SetRows prune = true, want false")
	}
}

func TestNativeTableModelSelectionSurvivesSetRowsInRowOrder(t *testing.T) {
	selection := NewNativeTableSelectionState("ops", "api")
	defer selection.Release()
	model := NewNativeTableModel([]NativeTableRow{
		{ID: "api"},
		{ID: "ops"},
		{ID: "worker"},
	}, selection)
	defer model.Release()

	model.SetRows([]NativeTableRow{
		{ID: "worker"},
		{ID: "api"},
		{ID: "ops"},
	})

	if got, want := model.SelectionState().SelectedIDs(), []string{"ops", "api"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after reorder = %v, want %v", got, want)
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "api" {
		t.Fatalf("SelectedID() after reorder = %q %v, want api true", got, ok)
	}
	if got, want := model.SelectedRows(), []NativeTableRow{{ID: "api"}, {ID: "ops"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedRows() after reorder = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "api" {
		t.Fatalf("SelectionAnchorID() after reorder = %q %v, want api true", got, ok)
	}
}

func TestNativeTableModelNavigation(t *testing.T) {
	model := NewNativeTableModel([]NativeTableRow{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}, nil)
	defer model.Release()
	defer model.SelectionState().Release()

	if !model.SelectNext() {
		t.Fatal("SelectNext() = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "a" {
		t.Fatalf("SelectedID() after first SelectNext = %q %v, want a true", got, ok)
	}
	if !model.SelectNext() {
		t.Fatal("SelectNext() second = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "b" {
		t.Fatalf("SelectedID() after second SelectNext = %q %v, want b true", got, ok)
	}
	if !model.SelectPrevious() {
		t.Fatal("SelectPrevious() = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "a" {
		t.Fatalf("SelectedID() after SelectPrevious = %q %v, want a true", got, ok)
	}
}

func TestNativeTableModelRangeSelection(t *testing.T) {
	model := NewNativeTableModel([]NativeTableRow{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
		{ID: "d"},
	}, nil)
	defer model.Release()
	defer model.SelectionState().Release()

	if !model.SelectID("b") {
		t.Fatal("SelectID(b) = false, want true")
	}
	if !model.SelectRangeToID("d") {
		t.Fatal("SelectRangeToID(d) = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after range = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "b" {
		t.Fatalf("SelectionAnchorID() after range = %q %v, want b true", got, ok)
	}
	if !model.ExtendSelectionPrevious() {
		t.Fatal("ExtendSelectionPrevious() = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"a", "b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after extend previous = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "b" {
		t.Fatalf("SelectionAnchorID() after extend previous = %q %v, want b true", got, ok)
	}
	if !model.SelectID("b") {
		t.Fatal("SelectID(b) after extend previous = false, want true")
	}
	if !model.ExtendSelectionNext() {
		t.Fatal("ExtendSelectionNext() = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after extend next = %v, want %v", got, want)
	}
}

func TestNativeTableModelSelectAllAndClearSelection(t *testing.T) {
	model := NewNativeTableModel([]NativeTableRow{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}, nil)
	defer model.Release()
	defer model.SelectionState().Release()

	if !model.SelectAll() {
		t.Fatal("SelectAll() = false, want true")
	}
	if got, want := model.SelectedRows(), []NativeTableRow{{ID: "a"}, {ID: "b"}, {ID: "c"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedRows() after SelectAll = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "a" {
		t.Fatalf("SelectionAnchorID() after SelectAll = %q %v, want a true", got, ok)
	}
	model.ClearSelection()
	if got, ok := model.SelectionState().SelectedID(); ok || got != "" {
		t.Fatalf("SelectedID() after ClearSelection = %q %v, want empty false", got, ok)
	}
}

func TestNativeTableColumnLayoutSnapshotRoundTrip(t *testing.T) {
	visibility := NewTableColumnVisibilityState("owner")
	defer visibility.Release()
	widths := NewTableColumnWidthState(map[string]float64{
		"name": 220,
		"load": 72,
	})
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

	visibility.ReplaceHiddenIDs("status")
	widths.ReplaceWidths(map[string]float64{"name": 999})
	presets.ReplaceSnapshot(TableColumnPresetSnapshot{
		CurrentPresetID: "temporary",
		Presets: []TableColumnPreset{{
			ID:        "temporary",
			Label:     "Temporary",
			HiddenIDs: []string{"load"},
		}},
	})

	ApplyTableColumnLayoutSnapshot(snapshot, visibility, widths, presets)

	if got, want := visibility.HiddenIDs(), []string{"owner", "status"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("HiddenIDs() after restore = %v, want %v", got, want)
	}
	if got, want := widths.Widths()["name"], 220.0; got != want {
		t.Fatalf("Widths()[name] after restore = %v, want %v", got, want)
	}
	if got, want := presets.CurrentPresetID(), "compact"; got != want {
		t.Fatalf("CurrentPresetID() after restore = %q, want %q", got, want)
	}
	if got, want := presets.PresetIDs(), []string{"all", "compact"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PresetIDs() after restore = %v, want %v", got, want)
	}
}

func TestNativeTableModelToggleSelectedID(t *testing.T) {
	model := NewNativeTableModel([]NativeTableRow{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}, nil)
	defer model.Release()
	defer model.SelectionState().Release()

	if !model.ToggleSelectedID("b") {
		t.Fatal("ToggleSelectedID(b) = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after toggle on = %v, want %v", got, want)
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "b" {
		t.Fatalf("SelectedID() after toggle on = %q %v, want b true", got, ok)
	}
	if !model.ToggleSelectedID("c") {
		t.Fatal("ToggleSelectedID(c) = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after second toggle on = %v, want %v", got, want)
	}
	if !model.ToggleSelectedID("b") {
		t.Fatal("ToggleSelectedID(b) off = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after toggle off = %v, want %v", got, want)
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "c" {
		t.Fatalf("SelectedID() after toggle off = %q %v, want c true", got, ok)
	}
}

func TestNativeTableModelSetRowsPreservesReachableSelection(t *testing.T) {
	model := NewNativeTableModel([]NativeTableRow{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}, nil)
	defer model.Release()
	defer model.SelectionState().Release()

	if !model.SelectID("b") {
		t.Fatal("SelectID(b) = false, want true")
	}
	if !model.SelectionState().Add("c") {
		t.Fatal("SelectionState().Add(c) = false, want true")
	}

	model.SetRows([]NativeTableRow{
		{ID: "c"},
		{ID: "b"},
		{ID: "z"},
	})

	if got, want := model.SelectionState().SelectedIDs(), []string{"b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SetRows preserve = %v, want %v", got, want)
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "c" {
		t.Fatalf("SelectedID() after SetRows preserve = %q %v, want c true", got, ok)
	}
	if got, ok := model.SelectionState().AnchorID(); !ok || got != "c" {
		t.Fatalf("AnchorID() after SetRows preserve = %q %v, want c true", got, ok)
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "c" {
		t.Fatalf("SelectedRow() after SetRows preserve = %+v %v, want c true", got, ok)
	}
}

func TestNativeOutlineModelRevealAndExpand(t *testing.T) {
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child").AsView(),
			Children: []NativeOutlineNode{{
				ID:    "leaf",
				Label: Text("Leaf").AsView(),
			}},
		}},
	}}, nil, nil)
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	if !model.RevealID("leaf") {
		t.Fatal("RevealID(leaf) = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "leaf" {
		t.Fatalf("SelectedID() = %q %v, want leaf true", got, ok)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() = %v, want %v", got, want)
	}
	var activated string
	model.SetOnActivate(func(row NativeOutlineNode) {
		activated = row.ID
	})
	if !model.ActivateSelected() {
		t.Fatal("ActivateSelected() = false, want true")
	}
	if activated != "leaf" {
		t.Fatalf("activated = %q, want leaf", activated)
	}
	model.SetRoots([]NativeOutlineNode{{ID: "next", Label: Text("Next").AsView()}})
	if _, ok := model.SelectionState().SelectedID(); ok {
		t.Fatal("SelectedID() after SetRoots prune = true, want false")
	}
}

func TestNativeOutlineModelSelectionAndExpansionPrune(t *testing.T) {
	selection := NewNativeOutlineSelectionState("leaf", "child")
	expansion := NewNativeOutlineExpansionState("root", "child", "stale")
	defer selection.Release()
	defer expansion.Release()

	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child").AsView(),
			Children: []NativeOutlineNode{{
				ID:    "leaf",
				Label: Text("Leaf").AsView(),
			}},
		}},
	}, {
		ID:    "other",
		Label: Text("Other").AsView(),
	}}, selection, expansion)
	defer model.Release()

	model.SetRoots([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child").AsView(),
		}},
	}})

	if got, want := model.SelectionState().SelectedIDs(), []string{"child"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after prune = %v, want %v", got, want)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after prune = %v, want %v", got, want)
	}
}

func TestNativeOutlineModelVisibleNavigation(t *testing.T) {
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child").AsView(),
		}},
	}, {
		ID:    "tail",
		Label: Text("Tail").AsView(),
	}}, nil, nil)
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	model.SetExpandedID("root", true)
	if !model.SelectNextVisible() {
		t.Fatal("SelectNextVisible() = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "root" {
		t.Fatalf("SelectedID() after first SelectNextVisible = %q %v, want root true", got, ok)
	}
	if !model.SelectNextVisible() {
		t.Fatal("SelectNextVisible() second = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "child" {
		t.Fatalf("SelectedID() after second SelectNextVisible = %q %v, want child true", got, ok)
	}
	if !model.SelectPreviousVisible() {
		t.Fatal("SelectPreviousVisible() = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "root" {
		t.Fatalf("SelectedID() after SelectPreviousVisible = %q %v, want root true", got, ok)
	}
}

func TestNativeOutlineModelVisibleRangeSelection(t *testing.T) {
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child").AsView(),
			Children: []NativeOutlineNode{{
				ID:    "leaf",
				Label: Text("Leaf").AsView(),
			}},
		}},
	}, {
		ID:    "tail",
		Label: Text("Tail").AsView(),
	}}, nil, nil)
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	if !model.SelectID("root") {
		t.Fatal("SelectID(root) = false, want true")
	}
	if !model.SelectVisibleRangeToID("leaf") {
		t.Fatal("SelectVisibleRangeToID(leaf) = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"root", "child", "leaf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after visible range = %v, want %v", got, want)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after visible range = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "root" {
		t.Fatalf("SelectionAnchorID() after visible range = %q %v, want root true", got, ok)
	}
	if !model.ExtendSelectionNextVisible() {
		t.Fatal("ExtendSelectionNextVisible() = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"root", "child", "leaf", "tail"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after extend next visible = %v, want %v", got, want)
	}
}

func TestNativeOutlineModelSelectAllAndClearSelection(t *testing.T) {
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child").AsView(),
		}},
	}, {
		ID:    "tail",
		Label: Text("Tail").AsView(),
	}}, nil, nil)
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	if !model.SelectAll() {
		t.Fatal("SelectAll() = false, want true")
	}
	selectedRows := model.SelectedRows()
	if got, want := len(selectedRows), 3; got != want {
		t.Fatalf("SelectedRows() len after SelectAll = %d, want %d", got, want)
	}
	if selectedRows[0].ID != "root" || selectedRows[1].ID != "child" || selectedRows[2].ID != "tail" {
		t.Fatalf("SelectedRows() after SelectAll = %v, want root child tail", selectedRows)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "root" {
		t.Fatalf("SelectionAnchorID() after SelectAll = %q %v, want root true", got, ok)
	}
	model.ClearSelection()
	if got, ok := model.SelectionState().SelectedID(); ok || got != "" {
		t.Fatalf("SelectedID() after ClearSelection = %q %v, want empty false", got, ok)
	}
}

func TestNativeOutlineModelToggleSelectedID(t *testing.T) {
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child").AsView(),
			Children: []NativeOutlineNode{{
				ID:    "leaf",
				Label: Text("Leaf").AsView(),
			}},
		}},
	}}, nil, nil)
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	if !model.ToggleSelectedID("leaf") {
		t.Fatal("ToggleSelectedID(leaf) = false, want true")
	}
	if got, want := model.SelectionState().SelectedIDs(), []string{"leaf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after toggle on = %v, want %v", got, want)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after toggle on = %v, want %v", got, want)
	}
	if !model.ToggleSelectedID("leaf") {
		t.Fatal("ToggleSelectedID(leaf) off = false, want true")
	}
	if got, ok := model.SelectionState().SelectedID(); ok || got != "" {
		t.Fatalf("SelectedID() after toggle off = %q %v, want empty false", got, ok)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after toggle off = %v, want %v", got, want)
	}
}

func TestNativeOutlineModelSetRootsPreservesReachableSelectionAndPrunesExpansion(t *testing.T) {
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
		Children: []NativeOutlineNode{
			{
				ID:    "child",
				Label: Text("Child").AsView(),
				Children: []NativeOutlineNode{{
					ID:    "leaf",
					Label: Text("Leaf").AsView(),
				}},
			},
			{
				ID:    "stale",
				Label: Text("Stale").AsView(),
			},
		},
	}}, nil, nil)
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	if !model.RevealID("leaf") {
		t.Fatal("RevealID(leaf) = false, want true")
	}
	if !model.SelectionState().Add("stale") {
		t.Fatal("SelectionState().Add(stale) = false, want true")
	}
	model.SetExpandedID("stale", true)

	model.SetRoots([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root v2").AsView(),
		Children: []NativeOutlineNode{{
			ID:    "child",
			Label: Text("Child v2").AsView(),
			Children: []NativeOutlineNode{{
				ID:    "leaf",
				Label: Text("Leaf v2").AsView(),
			}},
		}},
	}})

	if got, want := model.SelectionState().SelectedIDs(), []string{"leaf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SetRoots preserve/prune = %v, want %v", got, want)
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "leaf" {
		t.Fatalf("SelectedID() after SetRoots preserve/prune = %q %v, want leaf true", got, ok)
	}
	if got, ok := model.SelectionState().AnchorID(); !ok || got != "leaf" {
		t.Fatalf("AnchorID() after SetRoots preserve/prune = %q %v, want leaf true", got, ok)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after SetRoots preserve/prune = %v, want %v", got, want)
	}
	if _, ok := model.RowByID("stale"); ok {
		t.Fatal("RowByID(stale) = true, want false after SetRoots")
	}
}

func TestNativeOutlineModelSetExpandedIDIgnoresUnknownID(t *testing.T) {
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:    "root",
		Label: Text("Root").AsView(),
	}}, nil, nil)
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	model.SetExpandedID("missing", true)
	if got := model.ExpandedIDs(); len(got) != 0 {
		t.Fatalf("ExpandedIDs() after unknown expand = %v, want empty", got)
	}

	model.SetExpandedID("root", true)
	if got, want := model.ExpandedIDs(), []string{"root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after root expand = %v, want %v", got, want)
	}
}

func TestNativeTableModelLargeSetRowsPreservesSelectionAndRowOrder(t *testing.T) {
	rows := make([]NativeTableRow, 0, 512)
	for i := 0; i < 512; i++ {
		rows = append(rows, NativeTableRow{ID: fmt.Sprintf("svc-%03d", i)})
	}
	selection := NewNativeTableSelectionState("svc-100", "svc-250", "svc-400")
	defer selection.Release()
	model := NewNativeTableModel(rows, selection)
	defer model.Release()

	reversed := append([]NativeTableRow(nil), rows...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	model.SetRows(reversed)

	if got, want := model.SelectionState().SelectedIDs(), []string{"svc-100", "svc-250", "svc-400"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SetRows = %v, want %v", got, want)
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "svc-400" {
		t.Fatalf("SelectedRow() after SetRows = %+v %v, want svc-400 true", got, ok)
	}
	selectedRows := model.SelectedRows()
	if got, want := len(selectedRows), 3; got != want {
		t.Fatalf("SelectedRows() len = %d, want %d", got, want)
	}
	if selectedRows[0].ID != "svc-400" || selectedRows[1].ID != "svc-250" || selectedRows[2].ID != "svc-100" {
		t.Fatalf("SelectedRows() order = %v, want descending row order", selectedRows)
	}
}

func TestNativeOutlineModelLargeSetRootsPreservesSelectionAndExpansion(t *testing.T) {
	children := make([]NativeOutlineNode, 0, 512)
	for i := 0; i < 512; i++ {
		children = append(children, NativeOutlineNode{
			ID:    fmt.Sprintf("item-%03d", i),
			Label: Text(fmt.Sprintf("Item %03d", i)).AsView(),
		})
	}
	model := NewNativeOutlineModel([]NativeOutlineNode{{
		ID:       "root",
		Label:    Text("Root").AsView(),
		Children: children,
	}}, NewNativeOutlineSelectionState("item-100", "item-250", "item-400"), NewNativeOutlineExpansionState("root"))
	defer model.Release()
	defer model.SelectionState().Release()
	defer model.ExpansionState().Release()

	reversed := append([]NativeOutlineNode(nil), children...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	model.SetRoots([]NativeOutlineNode{{
		ID:       "root",
		Label:    Text("Root v2").AsView(),
		Children: reversed,
	}})

	if got, want := model.SelectionState().SelectedIDs(), []string{"item-100", "item-250", "item-400"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SetRoots = %v, want %v", got, want)
	}
	if got, ok := model.SelectionState().SelectedID(); !ok || got != "item-400" {
		t.Fatalf("SelectedID() after SetRoots = %q %v, want item-400 true", got, ok)
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "item-400" {
		t.Fatalf("SelectedRow() after SetRoots = %+v %v, want item-400 true", got, ok)
	}
	selectedRows := model.SelectedRows()
	if got, want := len(selectedRows), 3; got != want {
		t.Fatalf("SelectedRows() len = %d, want %d", got, want)
	}
	if selectedRows[0].ID != "item-400" || selectedRows[1].ID != "item-250" || selectedRows[2].ID != "item-100" {
		t.Fatalf("SelectedRows() order = %v, want descending row order", selectedRows)
	}
	if got, want := model.ExpandedIDs(), []string{"root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after SetRoots = %v, want %v", got, want)
	}
}
