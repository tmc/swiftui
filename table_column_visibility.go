package swiftui

import (
	"sort"
	"strings"
	"sync"
)

// TableColumnPreset describes one named hidden-column preset.
//
// Curated surface.
type TableColumnPreset struct {
	ID        string
	Label     string
	HiddenIDs []string
}

// TableColumnVisibilityState owns hidden-column membership for curated table views.
//
// Curated surface.
//
// The zero value is not usable. Use NewTableColumnVisibilityState.
type TableColumnVisibilityState struct {
	mu sync.RWMutex

	hidden map[string]struct{}

	revision      int
	revisionState *IntState
}

// TableColumnPresetState owns named visibility presets for curated table views.
//
// Curated surface.
//
// Presets are kept in memory and applied to a TableColumnVisibilityState. The
// zero value is not usable. Use NewTableColumnPresetState.
type TableColumnPresetState struct {
	mu sync.RWMutex

	presets   map[string]TableColumnPreset
	currentID string

	revision      int
	revisionState *IntState
}

// TableColumnPresetSnapshot captures one serializable preset registry snapshot.
//
// Curated surface.
type TableColumnPresetSnapshot struct {
	CurrentPresetID string
	Presets         []TableColumnPreset
}

// NewTableColumnVisibilityState creates a new visibility state with the given hidden IDs.
func NewTableColumnVisibilityState(hiddenIDs ...string) *TableColumnVisibilityState {
	s := &TableColumnVisibilityState{
		hidden:        make(map[string]struct{}),
		revisionState: newIntStateIfReady(0),
	}
	for _, id := range hiddenIDs {
		if id == "" {
			continue
		}
		s.hidden[id] = struct{}{}
	}
	return s
}

// NewTableColumnPresetState creates a new named preset registry.
func NewTableColumnPresetState(presets ...TableColumnPreset) *TableColumnPresetState {
	s := &TableColumnPresetState{
		presets:       make(map[string]TableColumnPreset),
		revisionState: newIntStateIfReady(0),
	}
	for _, preset := range presets {
		preset = normalizeTableColumnPreset(preset)
		if preset.ID == "" {
			continue
		}
		s.presets[preset.ID] = preset
	}
	return s
}

// Visible reports whether id is currently visible.
func (s *TableColumnVisibilityState) Visible(id string) bool {
	if s == nil || id == "" {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, hidden := s.hidden[id]
	return !hidden
}

// SetVisible updates the visibility of id.
func (s *TableColumnVisibilityState) SetVisible(id string, visible bool) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	_, hidden := s.hidden[id]
	if visible && !hidden {
		s.mu.Unlock()
		return
	}
	if !visible && hidden {
		s.mu.Unlock()
		return
	}
	if visible {
		delete(s.hidden, id)
	} else {
		s.hidden[id] = struct{}{}
	}
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Toggle flips the visibility of id.
func (s *TableColumnVisibilityState) Toggle(id string) {
	if s == nil || id == "" {
		return
	}
	s.SetVisible(id, !s.Visible(id))
}

// SetVisibleAll updates visibility for every provided column ID.
func (s *TableColumnVisibilityState) SetVisibleAll(visible bool, ids ...string) {
	if s == nil {
		return
	}
	for _, id := range ids {
		s.SetVisible(id, visible)
	}
}

// ReplaceHiddenIDs replaces the current hidden-column set with hiddenIDs.
func (s *TableColumnVisibilityState) ReplaceHiddenIDs(hiddenIDs ...string) {
	if s == nil {
		return
	}
	next := make(map[string]struct{}, len(hiddenIDs))
	for _, id := range hiddenIDs {
		if id == "" {
			continue
		}
		next[id] = struct{}{}
	}
	s.mu.Lock()
	if mapsEqualStrings(s.hidden, next) {
		s.mu.Unlock()
		return
	}
	s.hidden = next
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// ShowOnly hides every column in allIDs except those listed in visibleIDs.
func (s *TableColumnVisibilityState) ShowOnly(allIDs []string, visibleIDs ...string) {
	if s == nil {
		return
	}
	keep := make(map[string]struct{}, len(visibleIDs))
	for _, id := range visibleIDs {
		if id == "" {
			continue
		}
		keep[id] = struct{}{}
	}
	for _, id := range allIDs {
		if id == "" {
			continue
		}
		_, visible := keep[id]
		s.SetVisible(id, visible)
	}
}

// ApplyTableColumnVisibility returns a copy of columns with hidden flags derived
// from the current state.
func ApplyTableColumnVisibility[T any](s *TableColumnVisibilityState, columns []TableModelColumn[T]) []TableModelColumn[T] {
	out := make([]TableModelColumn[T], len(columns))
	copy(out, columns)
	if s == nil {
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range out {
		if out[i].ID == "" {
			continue
		}
		_, hidden := s.hidden[out[i].ID]
		out[i].Hidden = hidden
	}
	return out
}

// HiddenIDs reports hidden IDs in unspecified order.
func (s *TableColumnVisibilityState) HiddenIDs() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.hidden))
	for id := range s.hidden {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// VisibleIDs reports the visible IDs from allIDs in their original order.
func (s *TableColumnVisibilityState) VisibleIDs(allIDs []string) []string {
	if len(allIDs) == 0 {
		return nil
	}
	out := make([]string, 0, len(allIDs))
	for _, id := range allIDs {
		if id == "" || !s.Visible(id) {
			continue
		}
		out = append(out, id)
	}
	return out
}

// RevisionState reports the owned revision state for dynamic rebuilding.
func (s *TableColumnVisibilityState) RevisionState() *IntState {
	if s == nil {
		return nil
	}
	return s.revisionState
}

// Release releases owned state handles.
func (s *TableColumnVisibilityState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	revisionState := s.revisionState
	s.revisionState = nil
	s.hidden = nil
	s.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

// PresetIDs reports the known preset identifiers in sorted order.
func (s *TableColumnPresetState) PresetIDs() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.presets))
	for id := range s.presets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// CurrentPresetID reports the last applied or captured preset identifier.
func (s *TableColumnPresetState) CurrentPresetID() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentID
}

// Preset reports one normalized preset by identifier.
func (s *TableColumnPresetState) Preset(id string) (TableColumnPreset, bool) {
	if s == nil || id == "" {
		return TableColumnPreset{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	preset, ok := s.presets[id]
	return preset, ok
}

// SavePreset captures the current hidden-column state under id.
func (s *TableColumnPresetState) SavePreset(id, label string, visibility *TableColumnVisibilityState) bool {
	if s == nil || id == "" || visibility == nil {
		return false
	}
	preset := normalizeTableColumnPreset(TableColumnPreset{
		ID:        id,
		Label:     label,
		HiddenIDs: visibility.HiddenIDs(),
	})
	s.mu.Lock()
	current, ok := s.presets[preset.ID]
	if ok && current.Label == preset.Label && slicesEqualStrings(current.HiddenIDs, preset.HiddenIDs) && s.currentID == preset.ID {
		s.mu.Unlock()
		return false
	}
	s.presets[preset.ID] = preset
	s.currentID = preset.ID
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// ApplyPreset applies the named preset to visibility and makes it current.
func (s *TableColumnPresetState) ApplyPreset(id string, visibility *TableColumnVisibilityState) bool {
	if s == nil || id == "" || visibility == nil {
		return false
	}
	s.mu.Lock()
	preset, ok := s.presets[id]
	if !ok {
		s.mu.Unlock()
		return false
	}
	changedCurrent := s.currentID != id
	s.currentID = id
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	visibility.ReplaceHiddenIDs(preset.HiddenIDs...)
	if !changedCurrent && revisionState == nil {
		return true
	}
	updateIntState(revisionState, revision)
	return true
}

// DeletePreset removes the named preset.
func (s *TableColumnPresetState) DeletePreset(id string) bool {
	if s == nil || id == "" {
		return false
	}
	s.mu.Lock()
	if _, ok := s.presets[id]; !ok {
		s.mu.Unlock()
		return false
	}
	delete(s.presets, id)
	if s.currentID == id {
		s.currentID = ""
	}
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// Snapshot reports the current preset registry as a concrete value object.
func (s *TableColumnPresetState) Snapshot() TableColumnPresetSnapshot {
	if s == nil {
		return TableColumnPresetSnapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.presets))
	for id := range s.presets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	presets := make([]TableColumnPreset, 0, len(ids))
	for _, id := range ids {
		presets = append(presets, s.presets[id])
	}
	return TableColumnPresetSnapshot{
		CurrentPresetID: s.currentID,
		Presets:         presets,
	}
}

// ReplaceSnapshot replaces the current preset registry from snapshot.
func (s *TableColumnPresetState) ReplaceSnapshot(snapshot TableColumnPresetSnapshot) bool {
	if s == nil {
		return false
	}
	next := make(map[string]TableColumnPreset, len(snapshot.Presets))
	for _, preset := range snapshot.Presets {
		preset = normalizeTableColumnPreset(preset)
		if preset.ID == "" {
			continue
		}
		next[preset.ID] = preset
	}
	currentID := strings.TrimSpace(snapshot.CurrentPresetID)
	if _, ok := next[currentID]; !ok {
		currentID = ""
	}
	s.mu.Lock()
	if tableColumnPresetMapsEqual(s.presets, next) && s.currentID == currentID {
		s.mu.Unlock()
		return false
	}
	s.presets = next
	s.currentID = currentID
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// RevisionState reports the owned revision state for preset-driven rebuilding.
func (s *TableColumnPresetState) RevisionState() *IntState {
	if s == nil {
		return nil
	}
	return s.revisionState
}

// Release releases owned preset-state handles.
func (s *TableColumnPresetState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	revisionState := s.revisionState
	s.revisionState = nil
	s.presets = nil
	s.currentID = ""
	s.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

func mapsEqualStrings(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func normalizeTableColumnPreset(preset TableColumnPreset) TableColumnPreset {
	preset.HiddenIDs = uniqueSortedStrings(preset.HiddenIDs)
	if preset.Label == "" {
		preset.Label = preset.ID
	}
	return preset
}

func uniqueSortedStrings(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func slicesEqualStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tableColumnPresetMapsEqual(a, b map[string]TableColumnPreset) bool {
	if len(a) != len(b) {
		return false
	}
	for id, presetA := range a {
		presetB, ok := b[id]
		if !ok {
			return false
		}
		if presetA.Label != presetB.Label || !slicesEqualStrings(presetA.HiddenIDs, presetB.HiddenIDs) {
			return false
		}
	}
	return true
}
