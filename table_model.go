package swiftui

import (
	"fmt"
	"sort"
	"sync"
)

// TableModel owns row order, identity, and selection for table-style surfaces.
//
// The generic row type keeps the model practical for current examples while
// leaving the actual rendering to the existing curated bridge helpers.
type TableModel[T any] struct {
	mu           sync.RWMutex
	rows         []T
	id           func(T) string
	selectedID   string
	hasSelection bool
	revision     int

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

// SetRows replaces the current rows and preserves selection when possible.
func (m *TableModel[T]) SetRows(rows []T) {
	if m == nil {
		return
	}
	m.mu.Lock()

	m.rows = append([]T(nil), rows...)
	if !m.hasSelection {
		m.revision++
		revisionState := m.revisionState
		revision := m.revision
		m.mu.Unlock()
		updateIntState(revisionState, revision)
		return
	}
	if _, ok := m.indexOfIDLocked(m.selectedID); !ok {
		m.selectedID = ""
		m.hasSelection = false
	}
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
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

// SelectID selects the row with the given identifier.
func (m *TableModel[T]) SelectID(id string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if _, ok := m.indexOfIDLocked(id); !ok {
		m.selectedID = ""
		m.hasSelection = false
	} else {
		m.selectedID = id
		m.hasSelection = true
	}
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
	m.selectedID = ""
	m.hasSelection = false
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Sort reorders the rows using a stable comparison.
func (m *TableModel[T]) Sort(less func(a, b T) bool) {
	if m == nil || less == nil {
		return
	}
	m.mu.Lock()

	selectedID := m.selectedID
	hasSelection := m.hasSelection
	sort.SliceStable(m.rows, func(i, j int) bool {
		return less(m.rows[i], m.rows[j])
	})
	if hasSelection {
		if _, ok := m.indexOfIDLocked(selectedID); !ok {
			m.selectedID = ""
			m.hasSelection = false
		} else {
			m.selectedID = selectedID
			m.hasSelection = true
		}
	}
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
	m.selectedID = ""
	m.hasSelection = false
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

func normalizeTableRowID[T any](id func(T) string) func(T) string {
	if id != nil {
		return id
	}
	return func(v T) string {
		return fmt.Sprint(v)
	}
}
