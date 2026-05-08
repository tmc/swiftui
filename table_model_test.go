package swiftui

import (
	"reflect"
	"testing"
)

type tableModelRow struct {
	ID   string
	Name string
	Rank int
}

func TestTableModelSelectionAndSort(t *testing.T) {
	rows := []tableModelRow{
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "c", Name: "Alpha", Rank: 3},
	}
	model := NewTableModel(rows, func(r tableModelRow) string { return r.ID })

	if got, want := model.RowCount(), len(rows); got != want {
		t.Fatalf("row count = %d, want %d", got, want)
	}
	if got := model.Revision(); got == 0 {
		t.Fatalf("revision = %d, want non-zero after initialization", got)
	}
	if got, ok := model.RowAt(1); !ok || got.ID != "a" {
		t.Fatalf("row at 1 = %+v, %v, want a, true", got, ok)
	}
	if got, ok := model.RowAt(99); ok || got != (tableModelRow{}) {
		t.Fatalf("row at 99 = %+v, %v, want zero, false", got, ok)
	}

	model.SelectID("c")
	if got, ok := model.SelectedID(); !ok || got != "c" {
		t.Fatalf("selected id = %q, %v, want c, true", got, ok)
	}
	if got, ok := model.SelectedIndex(); !ok || got != 2 {
		t.Fatalf("selected index = %d, %v, want 2, true", got, ok)
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "c" {
		t.Fatalf("selected row = %+v, %v, want c, true", got, ok)
	}

	model.Sort(func(a, b tableModelRow) bool { return a.Name < b.Name })
	if got, want := model.RowIDs(), []string{"a", "c", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("row ids after sort = %v, want %v", got, want)
	}
	if got, ok := model.SelectedID(); !ok || got != "c" {
		t.Fatalf("selected id after sort = %q, %v, want c, true", got, ok)
	}
	if got, ok := model.SelectedIndex(); !ok || got != 1 {
		t.Fatalf("selected index after sort = %d, %v, want 1, true", got, ok)
	}
	if got, ok := model.SelectedRow(); !ok || got.ID != "c" {
		t.Fatalf("selected row after sort = %+v, %v, want c, true", got, ok)
	}
	if revState := model.RevisionState(); revState != nil && revState.Get() != model.Revision() {
		t.Fatalf("revision state = %d, want %d", revState.Get(), model.Revision())
	}
}

func TestTableModelMultiSelection(t *testing.T) {
	rows := []tableModelRow{
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "c", Name: "Gamma", Rank: 3},
	}
	model := NewTableModel(rows, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	model.SelectIDs("c", "missing", "a", "a")
	if got, want := model.SelectedIDs(), []string{"a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected ids = %v, want %v", got, want)
	}
	if got, ok := model.SelectedID(); !ok || got != "a" {
		t.Fatalf("primary selection = %q, %v, want a true", got, ok)
	}
	if got, want := model.SelectedRows(), []tableModelRow{
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "c", Name: "Gamma", Rank: 3},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected rows = %v, want %v", got, want)
	}
	if !model.HasSelectedID("a") || !model.HasSelectedID("c") || model.HasSelectedID("b") {
		t.Fatalf("selected membership incorrect: a=%v c=%v b=%v", model.HasSelectedID("a"), model.HasSelectedID("c"), model.HasSelectedID("b"))
	}

	model.ToggleSelectedID("a")
	if got, want := model.SelectedIDs(), []string{"c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected ids after toggle = %v, want %v", got, want)
	}
	if got, ok := model.SelectedID(); !ok || got != "c" {
		t.Fatalf("primary selection after toggle = %q, %v, want c true", got, ok)
	}

	model.Sort(func(a, b tableModelRow) bool { return a.Name < b.Name })
	if got, want := model.SelectedIDs(), []string{"c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected ids after sort = %v, want %v", got, want)
	}
	if got, ok := model.SelectedID(); !ok || got != "c" {
		t.Fatalf("primary selection after sort = %q, %v, want c true", got, ok)
	}
}

func TestTableModelSelectAllAndClearSelection(t *testing.T) {
	rows := []tableModelRow{
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "c", Name: "Gamma", Rank: 3},
	}
	model := NewTableModel(rows, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	if !model.SelectAll() {
		t.Fatal("SelectAll() = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"b", "a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected ids after select all = %v, want %v", got, want)
	}
	if got, ok := model.SelectedID(); !ok || got != "b" {
		t.Fatalf("primary selection after select all = %q, %v, want b true", got, ok)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "b" {
		t.Fatalf("selection anchor after select all = %q, %v, want b true", got, ok)
	}
	model.ClearSelection()
	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("selected id after clear = %q, %v, want empty false", got, ok)
	}
	if got, ok := model.SelectionAnchorID(); ok || got != "" {
		t.Fatalf("selection anchor after clear = %q, %v, want empty false", got, ok)
	}
}

func TestTableModelSetRowsCopiesAndDropsMissingSelection(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "x", Name: "X", Rank: 1},
		{ID: "y", Name: "Y", Rank: 2},
	}, func(r tableModelRow) string { return r.ID })

	rows := model.Rows()
	rows[0].ID = "mutated"
	if got, ok := model.RowAt(0); !ok || got.ID != "x" {
		t.Fatalf("row copy mutation leaked: %+v, %v", got, ok)
	}

	model.SelectID("y")
	model.SetRows([]tableModelRow{{ID: "z", Name: "Z", Rank: 3}})
	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("selection after replacement = %q, %v, want empty, false", got, ok)
	}
	if got, ok := model.SelectedRow(); ok || got != (tableModelRow{}) {
		t.Fatalf("selected row after replacement = %+v, %v, want zero, false", got, ok)
	}
	if got, want := model.RowCount(), 1; got != want {
		t.Fatalf("row count after replacement = %d, want %d", got, want)
	}

	model.SelectID("z")
	if got, ok := model.SelectedIndex(); !ok || got != 0 {
		t.Fatalf("selected index for z = %d, %v, want 0, true", got, ok)
	}
	model.ClearSelection()
	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("selection after clear = %q, %v, want empty, false", got, ok)
	}

	model.Release()
	if got, want := model.RowCount(), 0; got != want {
		t.Fatalf("row count after release = %d, want %d", got, want)
	}
	if got, ok := model.SelectedID(); ok || got != "" {
		t.Fatalf("selection after release = %q, %v, want empty, false", got, ok)
	}
}

func TestTableModelSetRowsPreservesActiveSort(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "a", Name: "Alpha", Rank: 1},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	model.SetSortColumn("rank", false, func(a, b tableModelRow) bool { return a.Rank < b.Rank })
	model.SetRows([]tableModelRow{
		{ID: "x", Name: "Gamma", Rank: 4},
		{ID: "y", Name: "Delta", Rank: 9},
		{ID: "z", Name: "Epsilon", Rank: 1},
	})
	if got, want := model.RowIDs(), []string{"y", "x", "z"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("row ids after SetRows with active sort = %v, want %v", got, want)
	}
	if id, ascending, ok := model.SortColumn(); !ok || id != "rank" || ascending {
		t.Fatalf("sort after SetRows = %q %v %v, want rank false true", id, ascending, ok)
	}
}

func TestTableModelClearSort(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "a", Name: "Alpha", Rank: 1},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	model.SetSortColumn("rank", true, func(a, b tableModelRow) bool { return a.Rank < b.Rank })
	model.ClearSort()
	if id, ascending, ok := model.SortColumn(); ok || id != "" || ascending {
		t.Fatalf("sort after clear = %q %v %v, want empty false false", id, ascending, ok)
	}
}

func TestTableModelActivate(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "a", Name: "Alpha", Rank: 1},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	var got string
	model.SetOnActivate(func(row tableModelRow) {
		got = row.ID
	})
	row, ok := model.RowAt(0)
	if !ok {
		t.Fatal("RowAt(0) = false, want true")
	}
	model.ActivateRow(row)
	if got != "a" {
		t.Fatalf("activated row = %q, want a", got)
	}

	model.SelectRow(row)
	if !model.ActivateSelected() {
		t.Fatal("ActivateSelected() = false, want true")
	}
	if got != "a" {
		t.Fatalf("activated selected row = %q, want a", got)
	}
	model.ClearSelection()
	if model.ActivateSelected() {
		t.Fatal("ActivateSelected() = true, want false after clear")
	}
}

func TestTableModelSelectRow(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "b", Name: "Beta", Rank: 2},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	row, ok := model.RowAt(1)
	if !ok {
		t.Fatal("RowAt(1) = false, want true")
	}
	model.SelectRow(row)
	if got, ok := model.SelectedID(); !ok || got != "b" {
		t.Fatalf("SelectedID() = %q %v, want b true", got, ok)
	}
}

func TestTableModelLookupAndReveal(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "b", Name: "Beta", Rank: 2},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	if got, ok := model.IndexOfID("b"); !ok || got != 1 {
		t.Fatalf("IndexOfID(b) = %d %v, want 1 true", got, ok)
	}
	if got, ok := model.RowByID("a"); !ok || got.ID != "a" {
		t.Fatalf("RowByID(a) = %+v %v, want a true", got, ok)
	}
	if !model.RevealID("b") {
		t.Fatal("RevealID(b) = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "b" {
		t.Fatalf("SelectedID() after RevealID = %q %v, want b true", got, ok)
	}
	row, ok := model.RowByID("a")
	if !ok {
		t.Fatal("RowByID(a) = false, want true")
	}
	if !model.RevealRow(row) {
		t.Fatal("RevealRow(a) = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "a" {
		t.Fatalf("SelectedID() after RevealRow = %q %v, want a true", got, ok)
	}
	if model.RevealID("missing") {
		t.Fatal("RevealID(missing) = true, want false")
	}
}

func TestTableModelSelectNextPrevious(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "c", Name: "Gamma", Rank: 3},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	if !model.SelectNext() {
		t.Fatal("SelectNext() = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "a" {
		t.Fatalf("SelectedID() after SelectNext = %q %v, want a true", got, ok)
	}
	if !model.SelectNext() {
		t.Fatal("SelectNext() second = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "b" {
		t.Fatalf("SelectedID() after second SelectNext = %q %v, want b true", got, ok)
	}
	if !model.SelectPrevious() {
		t.Fatal("SelectPrevious() = false, want true")
	}
	if got, ok := model.SelectedID(); !ok || got != "a" {
		t.Fatalf("SelectedID() after SelectPrevious = %q %v, want a true", got, ok)
	}
	model.SelectID("c")
	if model.SelectNext() {
		t.Fatal("SelectNext() at end = true, want false")
	}
	model.SelectID("a")
	if model.SelectPrevious() {
		t.Fatal("SelectPrevious() at start = true, want false")
	}
	if got, want := model.SelectedCount(), 1; got != want {
		t.Fatalf("SelectedCount() = %d, want %d", got, want)
	}
}

func TestTableModelSelectionAnchorAndExtend(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "c", Name: "Gamma", Rank: 3},
		{ID: "d", Name: "Delta", Rank: 4},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	if _, ok := model.SelectionAnchorID(); ok {
		t.Fatal("SelectionAnchorID() before selection = true, want false")
	}
	model.SelectID("b")
	if got, ok := model.SelectionAnchorID(); !ok || got != "b" {
		t.Fatalf("SelectionAnchorID() after SelectID = %q %v, want b true", got, ok)
	}
	if !model.ExtendSelectionNext() {
		t.Fatal("ExtendSelectionNext() = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after ExtendSelectionNext = %v, want %v", got, want)
	}
	if got, ok := model.SelectionAnchorID(); !ok || got != "b" {
		t.Fatalf("SelectionAnchorID() after extend = %q %v, want b true", got, ok)
	}
	if !model.ExtendSelectionPrevious() {
		t.Fatal("ExtendSelectionPrevious() = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after ExtendSelectionPrevious = %v, want %v", got, want)
	}
	model.ClearSelection()
	if _, ok := model.SelectionAnchorID(); ok {
		t.Fatal("SelectionAnchorID() after clear = true, want false")
	}
	if !model.ExtendSelectionNext() {
		t.Fatal("ExtendSelectionNext() from empty = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() from empty extend = %v, want %v", got, want)
	}
}

func TestTableModelRangeSelection(t *testing.T) {
	model := NewTableModel([]tableModelRow{
		{ID: "a", Name: "Alpha", Rank: 1},
		{ID: "b", Name: "Beta", Rank: 2},
		{ID: "c", Name: "Gamma", Rank: 3},
		{ID: "d", Name: "Delta", Rank: 4},
	}, func(r tableModelRow) string { return r.ID })
	defer model.Release()

	if !model.AddSelectedID("b") {
		t.Fatal("AddSelectedID(b) = false, want true")
	}
	if !model.AddSelectedID("d") {
		t.Fatal("AddSelectedID(d) = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"b", "d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after add = %v, want %v", got, want)
	}
	if model.AddSelectedID("missing") {
		t.Fatal("AddSelectedID(missing) = true, want false")
	}

	if !model.SelectRangeByIndex(1, 3) {
		t.Fatal("SelectRangeByIndex(1, 3) = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SelectRangeByIndex = %v, want %v", got, want)
	}

	model.SelectID("b")
	if !model.SelectRangeToID("d") {
		t.Fatal("SelectRangeToID(d) = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"b", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SelectRangeToID = %v, want %v", got, want)
	}

	model.SetSortColumn("rank", false, func(a, b tableModelRow) bool { return a.Rank < b.Rank })
	if got, want := model.RowIDs(), []string{"d", "c", "b", "a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("RowIDs() after descending sort = %v, want %v", got, want)
	}
	model.SelectID("d")
	if !model.SelectRangeToID("a") {
		t.Fatal("SelectRangeToID(a) after sort = false, want true")
	}
	if got, want := model.SelectedIDs(), []string{"d", "c", "b", "a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectedIDs() after SelectRangeToID(a) = %v, want %v", got, want)
	}
}
