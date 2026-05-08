package swiftui

import (
	"fmt"
	"sort"
	"sync"
)

// TableModel owns row order, identity, and selection for table-style surfaces.
//
// Curated surface.
//
// The generic row type keeps the model practical for current examples while
// leaving the actual rendering to the existing curated bridge helpers.
type TableModel[T any] struct {
	mu                sync.RWMutex
	rows              []T
	id                func(T) string
	onActivate        func(T)
	selectedIDs       map[string]struct{}
	selectedID        string
	selectionAnchorID string
	hasSelection      bool
	sortColumn        string
	sortAscending     bool
	hasSort           bool
	sortLess          func(a, b T) bool
	revision          int

	revisionState *IntState
}

// NewTableModel creates a table model with the provided rows and identity function.
//
// If id is nil, the model falls back to fmt.Sprint(row) for identity.
func NewTableModel[T any](rows []T, id func(T) string) *TableModel[T] {
	m := &TableModel[T]{
		id:            normalizeTableRowID[T](id),
		revisionState: newIntStateIfReady(0),
	}
	m.SetRows(rows)
	return m
}

// Rows returns a copy of the current row slice.
func (m *TableModel[T]) Rows() []T {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]T(nil), m.rows...)
}

// RowCount reports the current number of rows.
func (m *TableModel[T]) RowCount() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rows)
}

// RowAt returns the row at index if it exists.
func (m *TableModel[T]) RowAt(index int) (T, bool) {
	var zero T
	if m == nil {
		return zero, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if index < 0 || index >= len(m.rows) {
		return zero, false
	}
	return m.rows[index], true
}

// IndexOfID reports the current row index for id.
func (m *TableModel[T]) IndexOfID(id string) (int, bool) {
	if m == nil || id == "" {
		return -1, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.indexOfIDLocked(id)
}

// RowByID reports the current row value for id.
func (m *TableModel[T]) RowByID(id string) (T, bool) {
	var zero T
	if m == nil || id == "" {
		return zero, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	index, ok := m.indexOfIDLocked(id)
	if !ok {
		return zero, false
	}
	return m.rows[index], true
}

// RowIDs returns the current row identifiers in order.
func (m *TableModel[T]) RowIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.rows))
	for _, row := range m.rows {
		ids = append(ids, m.id(row))
	}
	return ids
}

// RowID returns the stable identifier for row.
func (m *TableModel[T]) RowID(row T) string {
	if m == nil {
		return ""
	}
	return m.id(row)
}

// SetOnActivate sets the row activation callback used by table views.
func (m *TableModel[T]) SetOnActivate(fn func(T)) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.onActivate = fn
	m.mu.Unlock()
}

// SetRows replaces the current rows and preserves selection when possible.
//
// When the new rows are the current rows plus a trailing suffix and no sort
// is active, SetRows reuses the existing slice and grows in place — this
// makes paginated appends O(delta) instead of O(n). Otherwise it replaces
// the slice and re-sorts if a sort descriptor is active.
func (m *TableModel[T]) SetRows(rows []T) {
	if m == nil {
		return
	}
	m.mu.Lock()

	if !(m.hasSort && m.sortLess != nil) {
		if extras, ok := diffAppendOnly(m.rows, rows, m.id); ok && len(extras) > 0 {
			// Pure append: grow in place, preserving selection and anchor.
			m.rows = append(m.rows, extras...)
			m.syncSelectionLocked()
			m.revision++
			revisionState := m.revisionState
			revision := m.revision
			m.mu.Unlock()
			updateIntState(revisionState, revision)
			return
		}
	}

	m.rows = append([]T(nil), rows...)
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

// SelectIndex selects the row at index if it exists.
func (m *TableModel[T]) SelectIndex(index int) {
	row, ok := m.RowAt(index)
	if !ok {
		m.ClearSelection()
		return
	}
	m.SelectID(m.RowID(row))
}

// SelectRow selects the provided row if it exists in the model.
func (m *TableModel[T]) SelectRow(row T) {
	if m == nil {
		return
	}
	m.SelectID(m.RowID(row))
}

// RevealID selects the row with id and reports whether it exists.
func (m *TableModel[T]) RevealID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	if _, ok := m.IndexOfID(id); !ok {
		return false
	}
	m.SelectID(id)
	return true
}

// RevealRow selects row and reports whether it exists in the model.
func (m *TableModel[T]) RevealRow(row T) bool {
	if m == nil {
		return false
	}
	return m.RevealID(m.RowID(row))
}

// SelectedID reports the selected row identifier.
func (m *TableModel[T]) SelectedID() (string, bool) {
	if m == nil {
		return "", false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.hasSelection {
		return "", false
	}
	return m.selectedID, true
}

// SelectedIDs reports the selected row identifiers in current row order.
func (m *TableModel[T]) SelectedIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.selectedIDsLocked()...)
}

// HasSelectedID reports whether id is currently selected.
func (m *TableModel[T]) HasSelectedID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.selectedIDs[id]
	return ok
}

// SelectedIndex reports the selected row index.
func (m *TableModel[T]) SelectedIndex() (int, bool) {
	if m == nil {
		return -1, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.hasSelection {
		return -1, false
	}
	index, ok := m.indexOfIDLocked(m.selectedID)
	if !ok {
		return -1, false
	}
	return index, true
}

// SelectedRow reports the selected row value.
func (m *TableModel[T]) SelectedRow() (T, bool) {
	var zero T
	if m == nil {
		return zero, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.hasSelection {
		return zero, false
	}
	index, ok := m.indexOfIDLocked(m.selectedID)
	if !ok {
		return zero, false
	}
	return m.rows[index], true
}

// SelectedRows reports the selected row values in current row order.
func (m *TableModel[T]) SelectedRows() []T {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.selectedIDs) == 0 {
		return nil
	}
	rows := make([]T, 0, len(m.selectedIDs))
	for _, row := range m.rows {
		if _, ok := m.selectedIDs[m.id(row)]; ok {
			rows = append(rows, row)
		}
	}
	return rows
}

// SelectedCount reports the number of selected rows.
func (m *TableModel[T]) SelectedCount() int {
	return len(m.SelectedIDs())
}

// SelectionAnchorID reports the current range-selection anchor identifier.
func (m *TableModel[T]) SelectionAnchorID() (string, bool) {
	if m == nil {
		return "", false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.selectionAnchorID == "" {
		return "", false
	}
	return m.selectionAnchorID, true
}

// SelectID selects the row with the given identifier.
func (m *TableModel[T]) SelectID(id string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.selectIDsLocked([]string{id})
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// SelectIDs selects the rows with the given identifiers.
//
// The primary selection reported by SelectedID follows the current row order.
func (m *TableModel[T]) SelectIDs(ids ...string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.selectIDsLocked(ids)
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// AddSelectedID adds id to the current multi-selection and reports whether it was added.
func (m *TableModel[T]) AddSelectedID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	m.mu.Lock()
	if _, ok := m.indexOfIDLocked(id); !ok {
		m.mu.Unlock()
		return false
	}
	if m.selectedIDs == nil {
		m.selectedIDs = make(map[string]struct{})
	}
	if _, ok := m.selectedIDs[id]; ok {
		m.mu.Unlock()
		return false
	}
	m.selectedIDs[id] = struct{}{}
	if m.selectionAnchorID == "" {
		m.selectionAnchorID = id
	}
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// SelectRangeByIndex selects the inclusive range between start and end.
func (m *TableModel[T]) SelectRangeByIndex(start, end int) bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	if len(m.rows) == 0 || start < 0 || end < 0 || start >= len(m.rows) || end >= len(m.rows) {
		m.mu.Unlock()
		return false
	}
	if start > end {
		start, end = end, start
	}
	ids := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		ids = append(ids, m.id(m.rows[i]))
	}
	m.selectionAnchorID = m.id(m.rows[start])
	m.selectIDsLocked(ids)
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// SelectRangeToID selects the inclusive range from the current anchor to id.
func (m *TableModel[T]) SelectRangeToID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	m.mu.Lock()
	target, ok := m.indexOfIDLocked(id)
	if !ok {
		m.mu.Unlock()
		return false
	}
	anchorID := m.selectionAnchorID
	if anchorID == "" {
		if m.hasSelection {
			anchorID = m.selectedID
		} else {
			anchorID = id
		}
	}
	anchor, ok := m.indexOfIDLocked(anchorID)
	if !ok {
		anchor = target
		anchorID = id
	}
	start, end := anchor, target
	if start > end {
		start, end = end, start
	}
	ids := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		ids = append(ids, m.id(m.rows[i]))
	}
	m.selectionAnchorID = anchorID
	m.selectIDsLocked(ids)
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// ToggleSelectedID toggles membership for id in the multi-selection set.
func (m *TableModel[T]) ToggleSelectedID(id string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if _, ok := m.indexOfIDLocked(id); !ok {
		m.mu.Unlock()
		return
	}
	if m.selectedIDs == nil {
		m.selectedIDs = make(map[string]struct{})
	}
	if _, ok := m.selectedIDs[id]; ok {
		delete(m.selectedIDs, id)
	} else {
		m.selectedIDs[id] = struct{}{}
	}
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// ClearSelection removes the current selection.
func (m *TableModel[T]) ClearSelection() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if !m.hasSelection {
		m.mu.Unlock()
		return
	}
	m.selectedIDs = nil
	m.selectedID = ""
	m.selectionAnchorID = ""
	m.hasSelection = false
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// SelectAll selects every row in current row order.
func (m *TableModel[T]) SelectAll() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	if len(m.rows) == 0 {
		m.mu.Unlock()
		return false
	}
	ids := make([]string, 0, len(m.rows))
	for _, row := range m.rows {
		ids = append(ids, m.id(row))
	}
	m.selectionAnchorID = ""
	m.selectIDsLocked(ids)
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// ActivateRow invokes the activation callback for row when one is registered.
func (m *TableModel[T]) ActivateRow(row T) {
	m.activate(row)
}

// ActivateSelected invokes the activation callback for the primary selected row.
func (m *TableModel[T]) ActivateSelected() bool {
	row, ok := m.SelectedRow()
	if !ok {
		return false
	}
	m.activate(row)
	return true
}

// SelectNext advances the primary selection by one row in current order.
func (m *TableModel[T]) SelectNext() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	index := -1
	if m.hasSelection {
		if i, ok := m.indexOfIDLocked(m.selectedID); ok {
			index = i
		}
	}
	next := index + 1
	if next < 0 || next >= len(m.rows) {
		m.mu.Unlock()
		return false
	}
	m.selectIDsLocked([]string{m.id(m.rows[next])})
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// SelectPrevious moves the primary selection backward by one row in current order.
func (m *TableModel[T]) SelectPrevious() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	index := len(m.rows)
	if m.hasSelection {
		if i, ok := m.indexOfIDLocked(m.selectedID); ok {
			index = i
		}
	}
	prev := index - 1
	if prev < 0 || prev >= len(m.rows) {
		m.mu.Unlock()
		return false
	}
	m.selectIDsLocked([]string{m.id(m.rows[prev])})
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// ExtendSelectionNext extends the current multi-selection from the existing
// anchor to the next row in current order.
func (m *TableModel[T]) ExtendSelectionNext() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	if len(m.rows) == 0 {
		m.mu.Unlock()
		return false
	}
	current := -1
	if m.hasSelection {
		if i, ok := m.indexOfIDLocked(m.selectedID); ok {
			current = i
		}
	}
	next := current + 1
	if next < 0 || next >= len(m.rows) {
		m.mu.Unlock()
		return false
	}
	if !m.extendSelectionToIndexLocked(next) {
		m.mu.Unlock()
		return false
	}
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// ExtendSelectionPrevious extends the current multi-selection from the
// existing anchor to the previous row in current order.
func (m *TableModel[T]) ExtendSelectionPrevious() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	if len(m.rows) == 0 {
		m.mu.Unlock()
		return false
	}
	current := len(m.rows)
	if m.hasSelection {
		if i, ok := m.indexOfIDLocked(m.selectedID); ok {
			current = i
		}
	}
	prev := current - 1
	if prev < 0 || prev >= len(m.rows) {
		m.mu.Unlock()
		return false
	}
	if !m.extendSelectionToIndexLocked(prev) {
		m.mu.Unlock()
		return false
	}
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// Sort reorders the rows using a stable comparison.
func (m *TableModel[T]) Sort(less func(a, b T) bool) {
	if m == nil || less == nil {
		return
	}
	m.mu.Lock()
	m.sortRowsLocked(less, true)
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// SortColumn reports the active sort column identifier and direction.
func (m *TableModel[T]) SortColumn() (id string, ascending bool, ok bool) {
	if m == nil {
		return "", false, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.hasSort {
		return "", false, false
	}
	return m.sortColumn, m.sortAscending, true
}

// ToggleSortColumn sorts by column, toggling direction when column is already active.
func (m *TableModel[T]) ToggleSortColumn(column string, less func(a, b T) bool) {
	if m == nil || less == nil {
		return
	}
	m.mu.Lock()
	ascending := true
	if m.hasSort && m.sortColumn == column {
		ascending = !m.sortAscending
	}
	m.sortColumn = column
	m.sortAscending = ascending
	m.hasSort = true
	m.sortLess = less
	m.sortRowsLocked(less, ascending)
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// SetSortColumn sorts by column using the requested direction.
func (m *TableModel[T]) SetSortColumn(column string, ascending bool, less func(a, b T) bool) {
	if m == nil || less == nil {
		return
	}
	m.mu.Lock()
	m.sortColumn = column
	m.sortAscending = ascending
	m.hasSort = true
	m.sortLess = less
	m.sortRowsLocked(less, ascending)
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// ClearSort clears the active sort descriptor without reordering current rows.
func (m *TableModel[T]) ClearSort() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if !m.hasSort && m.sortLess == nil {
		m.mu.Unlock()
		return
	}
	m.sortColumn = ""
	m.sortAscending = false
	m.hasSort = false
	m.sortLess = nil
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Release clears the model contents and selection.
func (m *TableModel[T]) Release() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.rows = nil
	m.selectedIDs = nil
	m.selectedID = ""
	m.selectionAnchorID = ""
	m.hasSelection = false
	m.sortColumn = ""
	m.sortAscending = false
	m.hasSort = false
	m.sortLess = nil
	revisionState := m.revisionState
	m.revisionState = nil
	m.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

// Revision reports the current mutation counter.
func (m *TableModel[T]) Revision() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.revision
}

// RevisionState exposes the owned revision IntState when the bridge is available.
func (m *TableModel[T]) RevisionState() *IntState {
	if m == nil {
		return nil
	}
	return m.revisionState
}

func (m *TableModel[T]) indexOfIDLocked(id string) (int, bool) {
	for i, row := range m.rows {
		if m.id(row) == id {
			return i, true
		}
	}
	return -1, false
}

func (m *TableModel[T]) sortRowsLocked(less func(a, b T) bool, ascending bool) {
	sort.SliceStable(m.rows, func(i, j int) bool {
		if ascending {
			return less(m.rows[i], m.rows[j])
		}
		return less(m.rows[j], m.rows[i])
	})
}

func (m *TableModel[T]) extendSelectionToIndexLocked(target int) bool {
	if target < 0 || target >= len(m.rows) {
		return false
	}
	anchorID := m.selectionAnchorID
	if anchorID == "" {
		if m.hasSelection {
			anchorID = m.selectedID
		} else {
			anchorID = m.id(m.rows[target])
		}
	}
	anchor, ok := m.indexOfIDLocked(anchorID)
	if !ok {
		anchor = target
		anchorID = m.id(m.rows[target])
	}
	start, end := anchor, target
	if start > end {
		start, end = end, start
	}
	ids := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		ids = append(ids, m.id(m.rows[i]))
	}
	m.selectionAnchorID = anchorID
	m.selectIDsLocked(ids)
	return true
}

func (m *TableModel[T]) selectIDsLocked(ids []string) {
	if m.selectedIDs == nil {
		m.selectedIDs = make(map[string]struct{})
	} else {
		for id := range m.selectedIDs {
			delete(m.selectedIDs, id)
		}
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := m.indexOfIDLocked(id); !ok {
			continue
		}
		m.selectedIDs[id] = struct{}{}
	}
	if len(ids) > 0 {
		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, ok := m.selectedIDs[id]; ok {
				if m.selectionAnchorID == "" {
					m.selectionAnchorID = id
				}
				break
			}
		}
	}
	m.syncSelectionLocked()
}

func (m *TableModel[T]) selectedIDsLocked() []string {
	if len(m.selectedIDs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m.selectedIDs))
	for _, row := range m.rows {
		id := m.id(row)
		if _, ok := m.selectedIDs[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m *TableModel[T]) syncSelectionLocked() {
	if len(m.selectedIDs) == 0 {
		m.selectedIDs = nil
		m.selectedID = ""
		m.selectionAnchorID = ""
		m.hasSelection = false
		return
	}
	ids := m.selectedIDsLocked()
	if len(ids) == 0 {
		m.selectedIDs = nil
		m.selectedID = ""
		m.selectionAnchorID = ""
		m.hasSelection = false
		return
	}
	kept := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		kept[id] = struct{}{}
	}
	m.selectedIDs = kept
	m.selectedID = ids[0]
	if _, ok := kept[m.selectionAnchorID]; !ok {
		m.selectionAnchorID = ids[0]
	}
	m.hasSelection = true
}

func (m *TableModel[T]) activate(row T) {
	if m == nil {
		return
	}
	m.mu.RLock()
	fn := m.onActivate
	m.mu.RUnlock()
	if fn != nil {
		fn(row)
	}
}

func normalizeTableRowID[T any](id func(T) string) func(T) string {
	if id != nil {
		return id
	}
	return func(v T) string {
		return fmt.Sprint(v)
	}
}
