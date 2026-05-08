package swiftui

import (
	"maps"
	"sort"
	"sync"
)

// TableColumnWidthState owns explicit per-column widths for curated table views.
//
// Curated surface.
//
// The zero value is not usable. Use NewTableColumnWidthState.
type TableColumnWidthState struct {
	mu sync.RWMutex

	widths map[string]float64

	revision      int
	revisionState *IntState
}

// NewTableColumnWidthState creates a new width state from id-width pairs.
func NewTableColumnWidthState(widths map[string]float64) *TableColumnWidthState {
	s := &TableColumnWidthState{
		widths:        normalizeTableColumnWidths(widths),
		revisionState: newIntStateIfReady(0),
	}
	return s
}

// Width reports the explicit width for id.
func (s *TableColumnWidthState) Width(id string) (float64, bool) {
	if s == nil || id == "" {
		return 0, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	width, ok := s.widths[id]
	return width, ok
}

// SetWidth sets the explicit width for id.
func (s *TableColumnWidthState) SetWidth(id string, width float64) {
	if s == nil || id == "" || width <= 0 {
		return
	}
	s.mu.Lock()
	if current, ok := s.widths[id]; ok && current == width {
		s.mu.Unlock()
		return
	}
	s.widths[id] = width
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// ClearWidth removes the explicit width for id.
func (s *TableColumnWidthState) ClearWidth(id string) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	if _, ok := s.widths[id]; !ok {
		s.mu.Unlock()
		return
	}
	delete(s.widths, id)
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// ReplaceWidths replaces the current explicit widths with widths.
func (s *TableColumnWidthState) ReplaceWidths(widths map[string]float64) {
	if s == nil {
		return
	}
	next := normalizeTableColumnWidths(widths)
	s.mu.Lock()
	if maps.Equal(s.widths, next) {
		s.mu.Unlock()
		return
	}
	s.widths = next
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// WidthIDs reports IDs with explicit widths in sorted order.
func (s *TableColumnWidthState) WidthIDs() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.widths))
	for id := range s.widths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Widths returns a copy of the explicit width map.
func (s *TableColumnWidthState) Widths() map[string]float64 {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.widths)
}

// RevisionState reports the owned revision state for dynamic rebuilding.
func (s *TableColumnWidthState) RevisionState() *IntState {
	if s == nil {
		return nil
	}
	return s.revisionState
}

// Release releases owned state handles.
func (s *TableColumnWidthState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	revisionState := s.revisionState
	s.revisionState = nil
	s.widths = nil
	s.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

func normalizeTableColumnWidths(widths map[string]float64) map[string]float64 {
	if len(widths) == 0 {
		return make(map[string]float64)
	}
	out := make(map[string]float64, len(widths))
	for id, width := range widths {
		if id == "" || width <= 0 {
			continue
		}
		out[id] = width
	}
	return out
}
