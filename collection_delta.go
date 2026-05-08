// Package swiftui: collection mutation deltas (P5).
//
// Delta[T], Indexed[T], and Move describe incremental mutations that can be
// applied to TableModel/OutlineModel instead of replacing the full row slice.
// ApplyDelta mutates the underlying slice in place so the cost scales with
// the delta size rather than the row count. Append, Insert, and Remove skip
// the diff entirely and are direct fast paths.
//
// SetRows still accepts any slice and preserves its external behavior, but
// it diffs against the current rows by row ID so the paginated-append case
// (existing prefix + a few new rows) avoids copying and re-sorting the
// whole slice. Sorted tables fall back to the replace-and-resort path.
//
// TODO(swift-diff): once the Swift renderer gains an int-index diff API we
// should plumb the Delta[T] slots directly to SwiftUI's animated diff
// helper instead of bumping the revision counter. The current
// DynamicView-based bridge rebuilds the whole VStack from Go on every
// revision tick, so a Swift-side mutation path would require a new
// @_cdecl accepting parallel int-index arrays.

package swiftui

// Indexed pairs a value with an index so deltas can describe exact placement.
//
// Curated surface.
type Indexed[T any] struct {
	Index int
	Value T
}

// Move describes a single positional move within a collection.
//
// From is the index of the row in the current ordering; To is the destination
// index evaluated after the row has been removed from its original position.
//
// Curated surface.
type Move struct {
	From int
	To   int
}

// Delta captures the minimum set of mutations that, applied in order
// (Remove first with indices interpreted against the current slice, then
// Insert with indices interpreted against the post-remove slice, then
// Update, then Move) produce the target state.
//
// Consumers populate only the fields they need. An empty Delta is a no-op.
//
// Curated surface.
type Delta[T any] struct {
	Insert []Indexed[T]
	Remove []int
	Update []Indexed[T]
	Move   []Move
}

// IsEmpty reports whether d describes no mutations.
func (d Delta[T]) IsEmpty() bool {
	return len(d.Insert) == 0 && len(d.Remove) == 0 && len(d.Update) == 0 && len(d.Move) == 0
}

// ApplyDelta applies d to the table model in place.
func (m *TableModel[T]) ApplyDelta(d Delta[T]) {
	if m == nil || d.IsEmpty() {
		return
	}
	m.mu.Lock()
	m.rows = applyDeltaToSlice(m.rows, d)
	if m.hasSort && m.sortLess != nil {
		m.sortRowsLocked(m.sortLess, m.sortAscending)
	}
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Append adds rows to the end of the table.
//
// This is a fast, non-diffing path; selection and sort state are preserved.
func (m *TableModel[T]) Append(rows ...T) {
	if m == nil || len(rows) == 0 {
		return
	}
	m.mu.Lock()
	m.rows = append(m.rows, rows...)
	if m.hasSort && m.sortLess != nil {
		m.sortRowsLocked(m.sortLess, m.sortAscending)
	}
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Insert places rows at index. If index is out of bounds the rows are
// appended.
func (m *TableModel[T]) Insert(index int, rows ...T) {
	if m == nil || len(rows) == 0 {
		return
	}
	m.mu.Lock()
	m.rows = insertSliceAt(m.rows, index, rows)
	if m.hasSort && m.sortLess != nil {
		m.sortRowsLocked(m.sortLess, m.sortAscending)
	}
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Remove deletes count rows starting at index.
func (m *TableModel[T]) Remove(index, count int) {
	if m == nil || count <= 0 {
		return
	}
	m.mu.Lock()
	m.rows = removeSliceAt(m.rows, index, count)
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// ApplyDelta applies d to the root rows.
func (m *OutlineModel[T]) ApplyDelta(d Delta[T]) {
	if m == nil || d.IsEmpty() {
		return
	}
	m.mu.Lock()
	m.roots = applyDeltaToSlice(m.roots, d)
	m.pruneExpandedLocked()
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Append adds rows to the end of the root list.
func (m *OutlineModel[T]) Append(rows ...T) {
	if m == nil || len(rows) == 0 {
		return
	}
	m.mu.Lock()
	m.roots = append(m.roots, rows...)
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Insert places rows at index in the root list.
func (m *OutlineModel[T]) Insert(index int, rows ...T) {
	if m == nil || len(rows) == 0 {
		return
	}
	m.mu.Lock()
	m.roots = insertSliceAt(m.roots, index, rows)
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Remove deletes count roots starting at index.
func (m *OutlineModel[T]) Remove(index, count int) {
	if m == nil || count <= 0 {
		return
	}
	m.mu.Lock()
	m.roots = removeSliceAt(m.roots, index, count)
	m.pruneExpandedLocked()
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// pruneExpandedLocked drops any expanded BoolState whose row ID is no longer
// reachable in the outline tree.
func (m *OutlineModel[T]) pruneExpandedLocked() {
	if len(m.expanded) == 0 {
		return
	}
	reachable := m.reachableIDsLocked()
	for id, state := range m.expanded {
		if reachable[id] {
			continue
		}
		if state != nil {
			state.Release()
		}
		delete(m.expanded, id)
	}
}

// applyDeltaToSlice mutates rows in place where possible and returns the
// resulting slice. Order of operations: Remove (descending indices), Insert
// (ascending indices), Update, Move.
func applyDeltaToSlice[T any](rows []T, d Delta[T]) []T {
	if len(d.Remove) > 0 {
		idx := append([]int(nil), d.Remove...)
		sortInts(idx)
		// Walk ascending indices from the end so earlier deletions keep
		// their meaning.
		for i := len(idx) - 1; i >= 0; i-- {
			k := idx[i]
			if k < 0 || k >= len(rows) {
				continue
			}
			rows = append(rows[:k], rows[k+1:]...)
		}
	}
	if len(d.Insert) > 0 {
		ins := append([]Indexed[T](nil), d.Insert...)
		sortInserts(ins)
		for _, it := range ins {
			rows = insertSliceAt(rows, it.Index, []T{it.Value})
		}
	}
	if len(d.Update) > 0 {
		for _, it := range d.Update {
			if it.Index < 0 || it.Index >= len(rows) {
				continue
			}
			rows[it.Index] = it.Value
		}
	}
	if len(d.Move) > 0 {
		for _, mv := range d.Move {
			if mv.From == mv.To {
				continue
			}
			if mv.From < 0 || mv.From >= len(rows) {
				continue
			}
			row := rows[mv.From]
			rows = append(rows[:mv.From], rows[mv.From+1:]...)
			to := mv.To
			if to < 0 {
				to = 0
			}
			if to > len(rows) {
				to = len(rows)
			}
			rows = insertSliceAt(rows, to, []T{row})
		}
	}
	return rows
}

func insertSliceAt[T any](rows []T, index int, values []T) []T {
	if len(values) == 0 {
		return rows
	}
	if index < 0 {
		index = 0
	}
	if index > len(rows) {
		index = len(rows)
	}
	out := make([]T, len(rows)+len(values))
	copy(out, rows[:index])
	copy(out[index:], values)
	copy(out[index+len(values):], rows[index:])
	return out
}

func removeSliceAt[T any](rows []T, index, count int) []T {
	if count <= 0 || index < 0 || index >= len(rows) {
		return rows
	}
	if index+count > len(rows) {
		count = len(rows) - index
	}
	return append(rows[:index], rows[index+count:]...)
}

func sortInts(a []int) {
	// Insertion sort — deltas almost always carry short index slices.
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

func sortInserts[T any](a []Indexed[T]) {
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j].Index > v.Index {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

// diffAppendOnly reports whether next is the result of appending zero or more
// rows to prev, matching only on row IDs. When it returns true the caller
// can apply a cheap append instead of replacing the whole slice.
func diffAppendOnly[T any](prev, next []T, id func(T) string) (extras []T, ok bool) {
	if len(next) < len(prev) {
		return nil, false
	}
	for i := range prev {
		if id(prev[i]) != id(next[i]) {
			return nil, false
		}
	}
	if len(next) == len(prev) {
		return nil, true
	}
	return next[len(prev):], true
}
