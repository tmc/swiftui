package swiftui

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

type deltaRow struct {
	ID    string
	Value int
}

func deltaRows(n int) []deltaRow {
	rows := make([]deltaRow, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, deltaRow{ID: fmt.Sprintf("r-%03d", i), Value: i})
	}
	return rows
}

func deltaRowID(r deltaRow) string { return r.ID }

func deltaIDs(rows []deltaRow) []string {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

// TestTableModelAppend covers append of 0, 1, and N rows.
func TestTableModelAppend(t *testing.T) {
	base := deltaRows(4)
	model := NewTableModel(base, deltaRowID)
	defer model.Release()

	start := model.Revision()
	model.Append()
	if model.Revision() != start {
		t.Fatalf("empty Append should not bump revision; got %d want %d", model.Revision(), start)
	}

	model.Append(deltaRow{ID: "r-new", Value: 99})
	if got := model.RowCount(); got != 5 {
		t.Fatalf("after Append(1) got %d rows, want 5", got)
	}
	if id, _, _ := model.idAtIndex(4); id != "r-new" {
		t.Fatalf("appended row id = %q, want r-new", id)
	}

	more := []deltaRow{{ID: "r-a", Value: 1}, {ID: "r-b", Value: 2}}
	model.Append(more...)
	if got := model.RowCount(); got != 7 {
		t.Fatalf("after Append(N) got %d rows, want 7", got)
	}
}

// idAtIndex is a test helper that reaches into TableModel without holding
// the public lock assumptions captured by Rows().
func (m *TableModel[T]) idAtIndex(i int) (string, T, bool) {
	if row, ok := m.RowAt(i); ok {
		return m.RowID(row), row, true
	}
	var zero T
	return "", zero, false
}

// TestTableModelInsert covers insert at 0, middle, end, beyond-end.
func TestTableModelInsert(t *testing.T) {
	base := deltaRows(3)
	model := NewTableModel(base, deltaRowID)
	defer model.Release()

	model.Insert(0, deltaRow{ID: "r-head", Value: -1})
	if id, _, _ := model.idAtIndex(0); id != "r-head" {
		t.Fatalf("Insert(0) id = %q, want r-head", id)
	}

	model.Insert(2, deltaRow{ID: "r-mid", Value: -2})
	if id, _, _ := model.idAtIndex(2); id != "r-mid" {
		t.Fatalf("Insert(2) id = %q, want r-mid", id)
	}

	model.Insert(model.RowCount(), deltaRow{ID: "r-end", Value: -3})
	last := model.RowCount() - 1
	if id, _, _ := model.idAtIndex(last); id != "r-end" {
		t.Fatalf("Insert(end) id = %q, want r-end", id)
	}

	beforeCount := model.RowCount()
	model.Insert(1000, deltaRow{ID: "r-tail", Value: -4})
	if model.RowCount() != beforeCount+1 {
		t.Fatalf("Insert(beyond) row count = %d, want %d", model.RowCount(), beforeCount+1)
	}
	last = model.RowCount() - 1
	if id, _, _ := model.idAtIndex(last); id != "r-tail" {
		t.Fatalf("Insert(beyond) appended id = %q, want r-tail", id)
	}
}

// TestTableModelRemove covers removal from 0, middle, and end.
func TestTableModelRemove(t *testing.T) {
	base := deltaRows(5)
	model := NewTableModel(base, deltaRowID)
	defer model.Release()

	model.Remove(0, 1)
	if id, _, _ := model.idAtIndex(0); id != "r-001" {
		t.Fatalf("after Remove(0,1) index 0 = %q, want r-001", id)
	}

	model.Remove(1, 1) // middle
	if got := deltaIDs(model.Rows()); !reflect.DeepEqual(got, []string{"r-001", "r-003", "r-004"}) {
		t.Fatalf("after middle remove got %v", got)
	}

	model.Remove(model.RowCount()-1, 1) // end
	if got := deltaIDs(model.Rows()); !reflect.DeepEqual(got, []string{"r-001", "r-003"}) {
		t.Fatalf("after end remove got %v", got)
	}
}

// TestTableModelApplyDeltaRoundTrip asserts the fast-path ApplyDelta
// produces the same terminal state as the slower SetRows-based rebuild.
func TestTableModelApplyDeltaRoundTrip(t *testing.T) {
	base := deltaRows(8)
	model := NewTableModel(base, deltaRowID)
	defer model.Release()

	d := Delta[deltaRow]{
		Remove: []int{1, 5},
		Insert: []Indexed[deltaRow]{{Index: 2, Value: deltaRow{ID: "r-ins", Value: 99}}},
		Update: []Indexed[deltaRow]{{Index: 0, Value: deltaRow{ID: "r-000", Value: 100}}},
		Move:   []Move{{From: 3, To: 0}},
	}
	model.ApplyDelta(d)

	// Compute the expected state using the standalone helper, then compare.
	expected := applyDeltaToSlice(append([]deltaRow(nil), base...), d)
	if got := model.Rows(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("ApplyDelta produced %v, want %v", got, expected)
	}
}

// TestTableModelMoveSelfNoop verifies Move{From: i, To: i} is a true no-op.
func TestTableModelMoveSelfNoop(t *testing.T) {
	base := deltaRows(4)
	model := NewTableModel(base, deltaRowID)
	defer model.Release()

	before := deltaIDs(model.Rows())
	model.ApplyDelta(Delta[deltaRow]{Move: []Move{{From: 2, To: 2}}})
	if got := deltaIDs(model.Rows()); !reflect.DeepEqual(got, before) {
		t.Fatalf("self-move changed order: got %v, want %v", got, before)
	}
}

// TestTableModelSetRowsDiffAppendEquivalence verifies SetRows(rows+appended)
// produces the same state as applying an Append-only delta.
func TestTableModelSetRowsDiffAppendEquivalence(t *testing.T) {
	base := deltaRows(16)
	extras := deltaRows(4)
	for i := range extras {
		extras[i].ID = "x-" + extras[i].ID
	}
	combined := append(append([]deltaRow(nil), base...), extras...)

	viaSetRows := NewTableModel(base, deltaRowID)
	defer viaSetRows.Release()
	viaSetRows.SetRows(combined)

	viaAppend := NewTableModel(base, deltaRowID)
	defer viaAppend.Release()
	viaAppend.Append(extras...)

	if !reflect.DeepEqual(viaSetRows.Rows(), viaAppend.Rows()) {
		t.Fatalf("SetRows-diff and Append diverged:\n  setRows=%v\n  append =%v", viaSetRows.Rows(), viaAppend.Rows())
	}
}

// TestTableModelSetRowsReplaceEquivalence verifies that for a non-append
// transition (reverse ordering), the model state matches what a full
// rebuild would produce.
func TestTableModelSetRowsReplaceEquivalence(t *testing.T) {
	base := deltaRows(8)
	reversed := append([]deltaRow(nil), base...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}

	model := NewTableModel(base, deltaRowID)
	defer model.Release()
	model.SetRows(reversed)

	if got := deltaIDs(model.Rows()); !reflect.DeepEqual(got, deltaIDs(reversed)) {
		t.Fatalf("SetRows(reversed) produced %v, want %v", got, deltaIDs(reversed))
	}
}

// TestTableModelRemoveSelectionPreserved asserts selection is dropped when
// a selected row is removed and preserved otherwise.
func TestTableModelRemoveSelectionPreserved(t *testing.T) {
	base := deltaRows(5)
	model := NewTableModel(base, deltaRowID)
	defer model.Release()

	model.SelectID("r-002")
	model.Remove(0, 1) // removes r-000, selection for r-002 survives
	if id, ok := model.SelectedID(); !ok || id != "r-002" {
		t.Fatalf("after remove[0] selection = %q,%v; want r-002,true", id, ok)
	}

	model.Remove(1, 1) // removes r-002, selection should drop
	if _, ok := model.SelectedID(); ok {
		t.Fatalf("selection should be cleared after removing selected row")
	}
}

// TestTableModelSortPreservedAcrossApplyDelta asserts that ApplyDelta
// respects an active sort descriptor by re-sorting after the mutation.
func TestTableModelSortPreservedAcrossApplyDelta(t *testing.T) {
	base := deltaRows(6)
	model := NewTableModel(base, deltaRowID)
	defer model.Release()

	// Sort descending by Value.
	model.SetSortColumn("value", false, func(a, b deltaRow) bool { return a.Value < b.Value })

	model.ApplyDelta(Delta[deltaRow]{
		Insert: []Indexed[deltaRow]{{Index: 0, Value: deltaRow{ID: "r-big", Value: 1000}}},
	})

	rows := model.Rows()
	if rows[0].ID != "r-big" || rows[0].Value != 1000 {
		t.Fatalf("after sorted ApplyDelta the largest row should be first; got %+v", rows[0])
	}
	// Confirm the remaining order is monotonically non-increasing.
	for i := 1; i < len(rows); i++ {
		if rows[i-1].Value < rows[i].Value {
			t.Fatalf("sort broken at index %d: %+v -> %+v", i, rows[i-1], rows[i])
		}
	}
}

// TestTableModelConcurrentMutations exercises concurrent Append/Remove under
// -race to confirm the lock discipline around the delta API.
func TestTableModelConcurrentMutations(t *testing.T) {
	model := NewTableModel(deltaRows(32), deltaRowID)
	defer model.Release()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 64; i++ {
			model.Append(deltaRow{ID: fmt.Sprintf("goA-%d", i), Value: i})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 64; i++ {
			model.Append(deltaRow{ID: fmt.Sprintf("goB-%d", i), Value: i})
		}
	}()
	wg.Wait()

	if want := 32 + 128; model.RowCount() != want {
		t.Fatalf("concurrent appends left %d rows, want %d", model.RowCount(), want)
	}
}

// TestOutlineModelAppendInsertRemove exercises the root-level delta API.
func TestOutlineModelAppendInsertRemove(t *testing.T) {
	roots := []benchmarkOutlineNode{{
		ID:    "a",
		Label: Text("A").AsView(),
	}, {
		ID:    "b",
		Label: Text("B").AsView(),
	}}
	model := NewOutlineModel(roots,
		func(n benchmarkOutlineNode) string { return n.ID },
		func(n benchmarkOutlineNode) []benchmarkOutlineNode { return n.Children },
	)
	defer model.Release()

	model.Append(benchmarkOutlineNode{ID: "c", Label: Text("C").AsView()})
	if got := len(model.Roots()); got != 3 {
		t.Fatalf("after Append got %d roots, want 3", got)
	}

	model.Insert(1, benchmarkOutlineNode{ID: "b1", Label: Text("B1").AsView()})
	if got := model.Roots()[1].ID; got != "b1" {
		t.Fatalf("after Insert(1) root[1] = %q, want b1", got)
	}

	model.Remove(0, 1) // removes "a"
	if got := model.Roots()[0].ID; got != "b1" {
		t.Fatalf("after Remove(0,1) root[0] = %q, want b1", got)
	}
}

// TestOutlineModelApplyDeltaPrunesExpanded confirms expanded state for
// removed branches is released.
func TestOutlineModelApplyDeltaPrunesExpanded(t *testing.T) {
	roots := []benchmarkOutlineNode{{
		ID:       "root",
		Label:    Text("Root").AsView(),
		Children: []benchmarkOutlineNode{{ID: "leaf-a"}, {ID: "leaf-b"}},
	}}
	model := NewOutlineModel(roots,
		func(n benchmarkOutlineNode) string { return n.ID },
		func(n benchmarkOutlineNode) []benchmarkOutlineNode { return n.Children },
	)
	defer model.Release()

	model.SetExpanded(roots[0], true)
	if got := model.ExpandedIDs(); len(got) != 1 || got[0] != "root" {
		t.Fatalf("expected expanded=[root], got %v", got)
	}

	model.ApplyDelta(Delta[benchmarkOutlineNode]{Remove: []int{0}})
	if got := model.ExpandedIDs(); len(got) != 0 {
		t.Fatalf("expanded IDs should prune on root removal; got %v", got)
	}
}
