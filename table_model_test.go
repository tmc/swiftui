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
