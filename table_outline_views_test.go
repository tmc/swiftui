package swiftui

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTextTableModelColumn(t *testing.T) {
	column := TextTableModelColumn("Name",
		func(row tableModelRow) string { return row.Name },
		func(a, b tableModelRow) bool { return a.Name < b.Name },
	)
	if column.Cell == nil {
		t.Fatalf("column cell is nil")
	}
	if column.Less == nil {
		t.Fatalf("column less is nil")
	}
	if !column.Less(tableModelRow{Name: "A"}, tableModelRow{Name: "B"}) {
		t.Fatalf("column less did not sort names")
	}
	if column.ID != "Name" {
		t.Fatalf("column id = %q, want Name", column.ID)
	}
	if got := column.WithID("name").WithMaxWidth(160); got.ID != "name" || got.MaxWidth != 160 {
		t.Fatalf("column options = %+v, want id name width 160", got)
	}
	if got := column.WithHidden(true); got.IsVisible() {
		t.Fatalf("column with hidden=true should not be visible")
	}
	if got := column.WithVisible(false); got.IsVisible() {
		t.Fatalf("column with visible=false should not be visible")
	}
	if got := column.WithVisible(true); !got.IsVisible() {
		t.Fatalf("column with visible=true should be visible")
	}
}

func TestTableModelVisibleColumns(t *testing.T) {
	columns := []TableModelColumn[tableModelRow]{
		TextTableModelColumn("Name", func(row tableModelRow) string { return row.Name }, nil).WithID("name"),
		TextTableModelColumn("Rank", func(row tableModelRow) string { return "" }, nil).WithID("rank").WithHidden(true),
	}
	visible := tableModelVisibleColumns(columns)
	if got, want := len(visible), 1; got != want {
		t.Fatalf("visible column count = %d, want %d", got, want)
	}
	if got, want := visible[0].ID, "name"; got != want {
		t.Fatalf("visible column id = %q, want %q", got, want)
	}
}

func TestRowAccessibilityMetadataHelpers(t *testing.T) {
	if got, want := tableRowAccessibilityID("service-1"), "table-row-service-1"; got != want {
		t.Fatalf("table row accessibility id = %q, want %q", got, want)
	}
	if got, want := outlineRowAccessibilityID("branch-1"), "outline-row-branch-1"; got != want {
		t.Fatalf("outline row accessibility id = %q, want %q", got, want)
	}

	cases := []struct {
		name        string
		selected    bool
		expanded    bool
		hasChildren bool
		want        string
	}{
		{name: "empty"},
		{name: "selected", selected: true, want: "selected"},
		{name: "collapsed branch", hasChildren: true, want: "collapsed"},
		{name: "selected expanded branch", selected: true, expanded: true, hasChildren: true, want: "selected, expanded"},
	}
	for _, tc := range cases {
		if got := rowStateSummary(tc.selected, tc.expanded, tc.hasChildren); got != tc.want {
			t.Fatalf("%s: rowStateSummary() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestApplyTableColumnLayout(t *testing.T) {
	visibility := NewTableColumnVisibilityState("rank")
	defer visibility.Release()
	widths := NewTableColumnWidthState(map[string]float64{
		"name": 220,
		"rank": 96,
	})
	defer widths.Release()

	columns := []TableModelColumn[tableModelRow]{
		TextTableModelColumn("Name", func(row tableModelRow) string { return row.Name }, nil).WithID("name").WithMaxWidth(160),
		TextTableModelColumn("Rank", func(row tableModelRow) string { return "" }, nil).WithID("rank").WithMaxWidth(72),
	}
	got := ApplyTableColumnLayout(visibility, widths, columns)
	if got[0].Hidden {
		t.Fatal("name should stay visible")
	}
	if got[0].MaxWidth != 220 {
		t.Fatalf("name width = %v, want 220", got[0].MaxWidth)
	}
	if !got[1].Hidden {
		t.Fatal("rank should be hidden")
	}
	if got[1].MaxWidth != 96 {
		t.Fatalf("rank width = %v, want 96", got[1].MaxWidth)
	}
	if columns[0].MaxWidth != 160 || columns[1].MaxWidth != 72 {
		t.Fatalf("ApplyTableColumnLayout mutated input columns: %+v", columns)
	}
}

func TestTableModelToggleSortAndSelectIndex(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "a", Name: "Alpha", Rank: 1},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	model.ToggleSortColumn("rank", func(a, b tableModelRow) bool { return a.Rank < b.Rank })
	if got, want := model.RowIDs(), []string{"a", "b"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("row ids after ascending sort = %v, want %v", got, want)
	}
	if id, ascending, ok := model.SortColumn(); !ok || id != "rank" || !ascending {
		t.Fatalf("sort = %q %v %v, want rank true true", id, ascending, ok)
	}

	model.ToggleSortColumn("rank", func(a, b tableModelRow) bool { return a.Rank < b.Rank })
	if got, want := model.RowIDs(), []string{"b", "a"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("row ids after descending sort = %v, want %v", got, want)
	}
	if id, ascending, ok := model.SortColumn(); !ok || id != "rank" || ascending {
		t.Fatalf("sort = %q %v %v, want rank false true", id, ascending, ok)
	}

	model.SelectIndex(1)
	if id, ok := model.SelectedID(); !ok || id != "a" {
		t.Fatalf("selected id = %q %v, want a true", id, ok)
	}
	model.SelectIndex(99)
	if id, ok := model.SelectedID(); ok || id != "" {
		t.Fatalf("selected id after invalid index = %q %v, want empty false", id, ok)
	}
}

func TestOutlineModelExpansionAndRoots(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{
			{ID: "child"},
		},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	roots := model.Roots()
	roots[0].ID = "mutated"
	if got := model.Roots()[0].ID; got != "root" {
		t.Fatalf("root mutation leaked: %q", got)
	}

	state := model.ExpandedState(root)
	if state == nil {
		t.Fatalf("expanded state is nil")
	}
	model.SetExpanded(root, true)
	if state.Get() != true {
		t.Fatalf("expanded state = false, want true")
	}
	model.ToggleExpanded(root)
	if state.Get() != false {
		t.Fatalf("expanded state after toggle = true, want false")
	}
	model.SetExpandedAll(true)
	if state.Get() != true {
		t.Fatalf("expanded state after expand all = false, want true")
	}
	model.SelectRow(root)
	if got, ok := model.SelectedID(); !ok || got != "root" {
		t.Fatalf("selected id = %q %v, want root true", got, ok)
	}
	model.ClearSelection()
	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("selected id after clear = %q %v, want empty false", got, ok)
	}

	model.SetRoots([]node{{ID: "next"}})
	if got := model.Roots()[0].ID; got != "next" {
		t.Fatalf("root after set = %q, want next", got)
	}
	if model.revision == 0 {
		t.Fatalf("revision = 0, want non-zero")
	}
}

func TestOutlineModelSetRootsDropsMissingSelectionAndExpansion(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{
			{ID: "child"},
		},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SetExpanded(root, true)
	model.SelectID("child")
	model.SetRoots([]node{{ID: "next"}})

	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("selected id after roots replace = %q %v, want empty false", got, ok)
	}
	if got := model.ExpandedIDs(); len(got) != 0 {
		t.Fatalf("expanded ids after roots replace = %v, want empty", got)
	}
}

func TestOutlineModelSelectIDRejectsUnknownID(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	model := NewOutlineModel([]node{{ID: "root"}},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SelectID("missing")
	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("selected id for missing row = %q %v, want empty false", got, ok)
	}
}

func TestOutlineModelMultiSelectionAndPrune(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{{
			ID:       "child",
			Children: []node{{ID: "leaf"}},
		}},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SelectIDs("leaf", "root", "missing")
	if got, want := model.SelectedIDs(), []string{"root", "leaf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected ids = %v, want %v", got, want)
	}
	if got, ok := model.SelectedID(); !ok || got != "root" {
		t.Fatalf("primary selection = %q, %v, want root true", got, ok)
	}
	if got, want := model.SelectedRows(), []node{
		root,
		root.Children[0].Children[0],
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected rows = %v, want %v", got, want)
	}
	if !model.HasSelectedID("root") || !model.HasSelectedID("leaf") || model.HasSelectedID("child") {
		t.Fatalf("selected membership incorrect")
	}

	model.ToggleSelectedID("root")
	if got, want := model.SelectedIDs(), []string{"leaf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected ids after toggle = %v, want %v", got, want)
	}

	model.SetRoots([]node{{ID: "next"}})
	if got := model.SelectedIDs(); len(got) != 0 {
		t.Fatalf("selected ids after roots replace = %v, want empty", got)
	}
}

func TestOutlineModelSelectAllAndClearSelection(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	model := NewOutlineModel([]node{{
		ID: "root",
		Children: []node{{
			ID: "child",
		}},
	}, {
		ID: "tail",
	}},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	if !model.SelectAll() {
		t.Fatal("SelectAll() = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"root", "child", "tail"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SelectAll = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "root" {
		t.Fatalf("SelectionAnchorID() after SelectAll = %q %v, want root true", got, ok)
	}
	model.ClearSelection()
	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("SelectedID() after ClearSelection = %q %v, want empty false", got, ok)
	}
	if got, ok := model.SelectionAnchorID(); ok || got != "" {
		t.Fatalf("SelectionAnchorID() after ClearSelection = %q %v, want empty false", got, ok)
	}
}

func TestOutlineModelExpandedIDs(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{
			{ID: "child"},
		},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SetExpanded(root, true)
	if !model.IsExpanded(root) {
		t.Fatal("root should be expanded")
	}
	if got, want := model.ExpandedIDs(), []string{"root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded ids = %v, want %v", got, want)
	}
}

func TestOutlineModelRevealID(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{{
			ID: "child",
			Children: []node{{
				ID: "leaf",
			}},
		}},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	if !model.RevealID("leaf") {
		t.Fatal("RevealID(leaf) = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "leaf" {
		t.Fatalf("SelectedID() = %q %v, want leaf true", got, ok)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() = %v, want %v", got, want)
	}
	if model.RevealID("missing") {
		t.Fatal("RevealID(missing) = true, want false")
	}
}

func TestOutlineModelRevealRowAndActivateRow(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{{
			ID: "child",
			Children: []node{{
				ID: "leaf",
			}},
		}},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	if !model.RevealRow(root.Children[0].Children[0]) {
		t.Fatal("RevealRow(leaf) = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "leaf" {
		t.Fatalf("SelectedID() = %q %v, want leaf true", got, ok)
	}
	if got, want := model.ExpandedIDs(), []string{"child", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() = %v, want %v", got, want)
	}

	var got string
	model.SetOnActivate(func(row node) {
		got = row.ID
	})
	model.ActivateRow(root)
	if got != "root" {
		t.Fatalf("activated row = %q, want root", got)
	}

	model.SelectID("leaf")
	if !model.ActivateSelected() {
		t.Fatal("ActivateSelected() = false, want true")
	}
	if got != "leaf" {
		t.Fatalf("activated selected row = %q, want leaf", got)
	}
	model.ClearSelection()
	if model.ActivateSelected() {
		t.Fatal("ActivateSelected() = true, want false after clear")
	}
}

func TestOutlineModelRowByIDAndSelectedRow(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{{
			ID: "child",
		}},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	if got, ok := model.RowByID("child"); !ok || got.ID != "child" {
		t.Fatalf("RowByID(child) = %+v %v, want child true", got, ok)
	}
	if _, ok := model.SelectedRow(); ok {
		t.Fatal("SelectedRow() = true, want false without selection")
	}
	model.SelectID("child")
	if got, ok := model.SelectedRow(); !ok || got.ID != "child" {
		t.Fatalf("SelectedRow() = %+v %v, want child true", got, ok)
	}
}

func TestOutlineModelSelectNextPreviousVisible(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{
			{ID: "child-a"},
			{ID: "child-b"},
		},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SetExpanded(root, true)
	if !model.SelectNextVisible() {
		t.Fatal("SelectNextVisible() = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "root" {
		t.Fatalf("SelectedID() after SelectNextVisible = %q %v, want root true", got, ok)
	}
	if !model.SelectNextVisible() {
		t.Fatal("second SelectNextVisible() = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "child-a" {
		t.Fatalf("SelectedID() after second SelectNextVisible = %q %v, want child-a true", got, ok)
	}
	if !model.SelectPreviousVisible() {
		t.Fatal("SelectPreviousVisible() = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "root" {
		t.Fatalf("SelectedID() after SelectPreviousVisible = %q %v, want root true", got, ok)
	}
	model.SelectID("child-b")
	if model.SelectNextVisible() {
		t.Fatal("SelectNextVisible() at end = true, want false")
	}
	model.SelectID("root")
	if model.SelectPreviousVisible() {
		t.Fatal("SelectPreviousVisible() at start = true, want false")
	}
	if got, want := model.SelectedCount(), 1; got != want {
		t.Fatalf("SelectedCount() = %d, want %d", got, want)
	}
}

func TestOutlineModelSelectionAnchorAndExtendVisible(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{
			{ID: "child-a"},
			{ID: "child-b"},
		},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SetExpanded(root, true)
	if _, ok := model.SelectionAnchorID(); ok {
		t.Fatal("SelectionAnchorID() before selection = true, want false")
	}
	model.SelectID("child-a")
	if got, ok := model.SelectionAnchorID(); !ok || got != "child-a" {
		t.Fatalf("SelectionAnchorID() after SelectID = %q %v, want child-a true", got, ok)
	}
	if !model.ExtendSelectionNextVisible() {
		t.Fatal("ExtendSelectionNextVisible() = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"child-a", "child-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after ExtendSelectionNextVisible = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "child-a" {
		t.Fatalf("SelectionAnchorID() after extend = %q %v, want child-a true", got, ok)
	}
	if !model.ExtendSelectionPreviousVisible() {
		t.Fatal("ExtendSelectionPreviousVisible() = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"root", "child-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after ExtendSelectionPreviousVisible = %v, want %v", got, want)
	}
	model.ClearSelection()
	if _, ok := model.SelectionAnchorID(); ok {
		t.Fatal("SelectionAnchorID() after clear = true, want false")
	}
	if !model.ExtendSelectionNextVisible() {
		t.Fatal("ExtendSelectionNextVisible() from empty = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() from empty extend = %v, want %v", got, want)
	}
}

func TestOutlineModelAddSelectedIDAndVisibleRange(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	root := node{
		ID: "root",
		Children: []node{
			{ID: "child-a"},
			{
				ID: "child-b",
				Children: []node{
					{ID: "leaf"},
				},
			},
		},
	}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SetExpanded(root, true)
	if !model.AddSelectedID("child-a") {
		t.Fatal("AddSelectedID(child-a) = false, want true")
	}
	if !model.AddSelectedID("child-b") {
		t.Fatal("AddSelectedID(child-b) = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"child-a", "child-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after add = %v, want %v", got, want)
	}
	if model.AddSelectedID("missing") {
		t.Fatal("AddSelectedID(missing) = true, want false")
	}

	model.SelectID("child-a")
	if !model.SelectVisibleRangeToID("leaf") {
		t.Fatal("SelectVisibleRangeToID(leaf) = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"child-a", "child-b", "leaf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SelectVisibleRangeToID = %v, want %v", got, want)
	}
	if got, want := model.ExpandedIDs(), []string{"child-b", "root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after SelectVisibleRangeToID = %v, want %v", got, want)
	}
}

func TestTableModelLargeSetRowsPreservesSelectionAndSort(t *testing.T) {
	type row struct {
		ID   string
		Load int
	}
	rows := make([]row, 0, 512)
	for i := 0; i < 512; i++ {
		rows = append(rows, row{
			ID:   fmt.Sprintf("svc-%03d", i),
			Load: i,
		})
	}
	model := NewTableModel(rows, func(r row) string { return r.ID })
	defer model.Release()

	model.SetSortColumn("load", true, func(a, b row) bool { return a.Load < b.Load })
	model.SelectID("svc-100")
	if !model.SelectRangeToID("svc-400") {
		t.Fatal("SelectRangeToID(svc-400) = false, want true")
	}

	reversed := append([]row(nil), rows...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	model.SetRows(reversed)

	if got, want := model.RowIDs(), []string{
		"svc-000", "svc-001", "svc-002",
	}; len(got) < len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("row ids after resort = %v, want prefix %v", got[:3], want)
	}
	if got, want := model.SelectedCount(), 301; got != want {
		t.Fatalf("SelectedCount() = %d, want %d", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "svc-100" {
		t.Fatalf("SelectionAnchorID() = %q %v, want svc-100 true", got, ok)
	}
	selected := model.SelectedIDs()
	if got, want := selected[0], "svc-100"; got != want {
		t.Fatalf("first selected id = %q, want %q", got, want)
	}
	if got, want := selected[len(selected)-1], "svc-400"; got != want {
		t.Fatalf("last selected id = %q, want %q", got, want)
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "svc-100" {
		t.Fatalf("SelectedRow() = %+v %v, want svc-100 true", got, ok)
	}
}

func TestOutlineModelLargeVisibleRangeSelectionSurvivesSetRoots(t *testing.T) {
	type node struct {
		ID       string
		Children []node
	}
	children := make([]node, 0, 512)
	for i := 0; i < 512; i++ {
		children = append(children, node{ID: fmt.Sprintf("item-%03d", i)})
	}
	root := node{ID: "root", Children: children}
	model := NewOutlineModel([]node{root},
		func(n node) string { return n.ID },
		func(n node) []node { return n.Children },
	)
	defer model.Release()

	model.SetExpanded(root, true)
	model.SelectID("item-100")
	if !model.SelectVisibleRangeToID("item-400") {
		t.Fatal("SelectVisibleRangeToID(item-400) = false, want true")
	}

	reversed := append([]node(nil), children...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	model.SetRoots([]node{{ID: "root", Children: reversed}})

	if got, want := model.ExpandedIDs(), []string{"root"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedIDs() after SetRoots = %v, want %v", got, want)
	}
	if got, want := model.SelectedCount(), 301; got != want {
		t.Fatalf("SelectedCount() = %d, want %d", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "item-100" {
		t.Fatalf("SelectionAnchorID() = %q %v, want item-100 true", got, ok)
	}
	selected := model.SelectedIDs()
	if got, want := selected[0], "item-400"; got != want {
		t.Fatalf("first selected id = %q, want %q", got, want)
	}
	if got, want := selected[len(selected)-1], "item-100"; got != want {
		t.Fatalf("last selected id = %q, want %q", got, want)
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "item-400" {
		t.Fatalf("SelectedRow() = %+v %v, want item-400 true", got, ok)
	}
}
