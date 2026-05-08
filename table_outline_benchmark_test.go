package swiftui

import (
	"fmt"
	"testing"
)

type benchmarkTableRow struct {
	ID   string
	Load int
}

type benchmarkOutlineNode struct {
	ID       string
	Label    View
	Children []benchmarkOutlineNode
}

func benchmarkTableRows(n int) []benchmarkTableRow {
	rows := make([]benchmarkTableRow, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, benchmarkTableRow{
			ID:   fmt.Sprintf("svc-%04d", i),
			Load: i,
		})
	}
	return rows
}

func benchmarkReverseTableRows(rows []benchmarkTableRow) []benchmarkTableRow {
	out := append([]benchmarkTableRow(nil), rows...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func benchmarkOutlineNodes(n int) []benchmarkOutlineNode {
	children := make([]benchmarkOutlineNode, 0, n)
	for i := 0; i < n; i++ {
		children = append(children, benchmarkOutlineNode{
			ID:    fmt.Sprintf("item-%04d", i),
			Label: Text(fmt.Sprintf("Item %04d", i)).AsView(),
		})
	}
	return []benchmarkOutlineNode{{
		ID:       "root",
		Label:    Text("Root").AsView(),
		Children: children,
	}}
}

func benchmarkReverseOutlineNodes(nodes []benchmarkOutlineNode) []benchmarkOutlineNode {
	out := append([]benchmarkOutlineNode(nil), nodes...)
	if len(out) == 0 {
		return out
	}
	children := append([]benchmarkOutlineNode(nil), out[0].Children...)
	for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
		children[i], children[j] = children[j], children[i]
	}
	out[0].Children = children
	return out
}

func BenchmarkTableModelSetRows(b *testing.B) {
	for _, size := range []int{64, 1024, 16384} {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			rows := benchmarkTableRows(size)
			reversed := benchmarkReverseTableRows(rows)
			model := NewTableModel(rows, func(r benchmarkTableRow) string { return r.ID })
			defer model.Release()

			model.SetSortColumn("load", true, func(a, b benchmarkTableRow) bool { return a.Load < b.Load })
			// Seed a representative selection span within the row range.
			anchor := fmt.Sprintf("svc-%04d", size/10)
			toward := fmt.Sprintf("svc-%04d", (size*9)/10-1)
			model.SelectID(anchor)
			if !model.SelectRangeToID(toward) {
				b.Fatalf("SelectRangeToID(%s) = false, want true", toward)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if i%2 == 0 {
					model.SetRows(reversed)
					continue
				}
				model.SetRows(rows)
			}
		})
	}
}

// BenchmarkTableModelAppend measures the charter fast-path: append 20 rows to
// a 1024-row table. Compare against BenchmarkTableModelSetRows/1024 to gauge
// delta-vs-replace speedup for paginated-append workloads.
func BenchmarkTableModelAppend(b *testing.B) {
	base := benchmarkTableRows(1024)
	extras := make([]benchmarkTableRow, 20)
	for i := range extras {
		extras[i] = benchmarkTableRow{ID: fmt.Sprintf("svc-extra-%04d", i), Load: 1024 + i}
	}
	combined := append(append([]benchmarkTableRow(nil), base...), extras...)
	_ = combined

	b.Run("1024+20-SetRows", func(b *testing.B) {
		model := NewTableModel(base, func(r benchmarkTableRow) string { return r.ID })
		defer model.Release()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate paginated append using SetRows: rebuild with the
			// appended rows, then reset to base for the next iteration.
			model.SetRows(combined)
			model.SetRows(base)
		}
	})

	b.Run("1024+20-Append", func(b *testing.B) {
		model := NewTableModel(base, func(r benchmarkTableRow) string { return r.ID })
		defer model.Release()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model.Append(extras...)
			model.Remove(len(base), len(extras))
		}
	})
}

// BenchmarkTableModelApplyDelta exercises the explicit delta operations on a
// 1024-row model. Every subtest round-trips to the original state so the
// per-iteration cost reflects two symmetric mutations.
func BenchmarkTableModelApplyDelta(b *testing.B) {
	base := benchmarkTableRows(1024)
	id := func(r benchmarkTableRow) string { return r.ID }

	b.Run("insert", func(b *testing.B) {
		model := NewTableModel(base, id)
		defer model.Release()
		ins := []Indexed[benchmarkTableRow]{{
			Index: 500,
			Value: benchmarkTableRow{ID: "svc-inserted", Load: -1},
		}}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model.ApplyDelta(Delta[benchmarkTableRow]{Insert: ins})
			model.ApplyDelta(Delta[benchmarkTableRow]{Remove: []int{500}})
		}
	})

	b.Run("remove", func(b *testing.B) {
		model := NewTableModel(base, id)
		defer model.Release()
		saved := base[500]
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model.ApplyDelta(Delta[benchmarkTableRow]{Remove: []int{500}})
			model.ApplyDelta(Delta[benchmarkTableRow]{Insert: []Indexed[benchmarkTableRow]{{Index: 500, Value: saved}}})
		}
	})

	b.Run("update", func(b *testing.B) {
		model := NewTableModel(base, id)
		defer model.Release()
		upd := []Indexed[benchmarkTableRow]{{
			Index: 500,
			Value: benchmarkTableRow{ID: base[500].ID, Load: 9999},
		}}
		restore := []Indexed[benchmarkTableRow]{{Index: 500, Value: base[500]}}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model.ApplyDelta(Delta[benchmarkTableRow]{Update: upd})
			model.ApplyDelta(Delta[benchmarkTableRow]{Update: restore})
		}
	})

	b.Run("move", func(b *testing.B) {
		model := NewTableModel(base, id)
		defer model.Release()
		fwd := []Move{{From: 100, To: 200}}
		rev := []Move{{From: 200, To: 100}}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			model.ApplyDelta(Delta[benchmarkTableRow]{Move: fwd})
			model.ApplyDelta(Delta[benchmarkTableRow]{Move: rev})
		}
	})
}

func BenchmarkOutlineModelSetRoots(b *testing.B) {
	// The outline benchmark varies the number of child rows under a single
	// "root" node because benchmarkOutlineNodes only wraps at that level.
	for _, size := range []int{64, 1024, 16384} {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			roots := benchmarkOutlineNodes(size)
			reversed := benchmarkReverseOutlineNodes(roots)
			model := NewOutlineModel(roots,
				func(n benchmarkOutlineNode) string { return n.ID },
				func(n benchmarkOutlineNode) []benchmarkOutlineNode { return n.Children },
			)
			defer model.Release()

			model.SetExpanded(roots[0], true)
			anchor := fmt.Sprintf("item-%04d", size/10)
			toward := fmt.Sprintf("item-%04d", (size*9)/10-1)
			model.SelectID(anchor)
			if !model.SelectVisibleRangeToID(toward) {
				b.Fatalf("SelectVisibleRangeToID(%s) = false, want true", toward)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if i%2 == 0 {
					model.SetRoots(reversed)
					continue
				}
				model.SetRoots(roots)
			}
		})
	}
}

func BenchmarkNativeTableModelSetRows(b *testing.B) {
	rows := make([]NativeTableRow, 0, 1024)
	for i := 0; i < 1024; i++ {
		rows = append(rows, NativeTableRow{ID: fmt.Sprintf("svc-%04d", i)})
	}
	reversed := append([]NativeTableRow(nil), rows...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	selection := NewNativeTableSelectionState("svc-0100", "svc-0500", "svc-0900")
	defer selection.Release()
	model := NewNativeTableModel(rows, selection)
	defer model.Release()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			model.SetRows(reversed)
			continue
		}
		model.SetRows(rows)
	}
}

func BenchmarkNativeOutlineModelSetRoots(b *testing.B) {
	children := make([]NativeOutlineNode, 0, 1024)
	for i := 0; i < 1024; i++ {
		children = append(children, NativeOutlineNode{
			ID:    fmt.Sprintf("item-%04d", i),
			Label: Text(fmt.Sprintf("Item %04d", i)).AsView(),
		})
	}
	roots := []NativeOutlineNode{{
		ID:       "root",
		Label:    Text("Root").AsView(),
		Children: children,
	}}
	reversed := append([]NativeOutlineNode(nil), roots...)
	if len(reversed) > 0 {
		reversedChildren := append([]NativeOutlineNode(nil), reversed[0].Children...)
		for i, j := 0, len(reversedChildren)-1; i < j; i, j = i+1, j-1 {
			reversedChildren[i], reversedChildren[j] = reversedChildren[j], reversedChildren[i]
		}
		reversed[0].Children = reversedChildren
	}
	selection := NewNativeOutlineSelectionState("item-0100", "item-0500", "item-0900")
	defer selection.Release()
	expansion := NewNativeOutlineExpansionState("root")
	defer expansion.Release()
	model := NewNativeOutlineModel(roots, selection, expansion)
	defer model.Release()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			model.SetRoots(reversed)
			continue
		}
		model.SetRoots(roots)
	}
}
