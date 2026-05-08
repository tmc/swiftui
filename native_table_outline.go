package swiftui

import (
	"sort"
	"sync"
)

// NativeSelectionState owns a string-ID selection set for native-backed table
// and outline surfaces.
//
// Curated surface.
//
// The zero value is not usable. Use NewNativeSelectionState.
type NativeSelectionState struct {
	mu sync.RWMutex

	selectedIDs   map[string]struct{}
	selectedOrder []string
	selectedID    string
	anchorID      string

	revision      int
	revisionState *IntState
}

// NewNativeSelectionState creates a new string-ID selection state.
func NewNativeSelectionState(ids ...string) *NativeSelectionState {
	s := &NativeSelectionState{
		selectedIDs:   make(map[string]struct{}),
		revisionState: newIntStateIfReady(0),
	}
	s.Replace(ids...)
	return s
}

// NativeTableSelectionState is the explicit selection state for NativeTableModel.
//
// Curated surface.
type NativeTableSelectionState = NativeSelectionState

// NewNativeTableSelectionState creates a new table selection state.
func NewNativeTableSelectionState(ids ...string) *NativeTableSelectionState {
	return NewNativeSelectionState(ids...)
}

// NativeOutlineSelectionState is the explicit selection state for NativeOutlineModel.
//
// Curated surface.
type NativeOutlineSelectionState = NativeSelectionState

// NewNativeOutlineSelectionState creates a new outline selection state.
func NewNativeOutlineSelectionState(ids ...string) *NativeOutlineSelectionState {
	return NewNativeSelectionState(ids...)
}

// SelectedID reports the primary selected ID.
func (s *NativeSelectionState) SelectedID() (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.selectedID == "" {
		return "", false
	}
	return s.selectedID, true
}

// SelectedIDs reports the selected IDs in selection order.
func (s *NativeSelectionState) SelectedIDs() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.selectedOrder...)
}

// SelectedCount reports the number of selected IDs.
func (s *NativeSelectionState) SelectedCount() int {
	return len(s.SelectedIDs())
}

// AnchorID reports the current selection anchor.
func (s *NativeSelectionState) AnchorID() (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.anchorID == "" {
		return "", false
	}
	return s.anchorID, true
}

// Has reports whether id is selected.
func (s *NativeSelectionState) Has(id string) bool {
	if s == nil || id == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.selectedIDs[id]
	return ok
}

// Replace replaces the current selection with ids.
func (s *NativeSelectionState) Replace(ids ...string) {
	if s == nil {
		return
	}
	s.replaceOrderedWithPrimary(ids, "", "")
}

// Add adds id to the current selection.
func (s *NativeSelectionState) Add(id string) bool {
	if s == nil || id == "" {
		return false
	}
	s.mu.Lock()
	if _, ok := s.selectedIDs[id]; ok {
		s.mu.Unlock()
		return false
	}
	s.selectedIDs[id] = struct{}{}
	s.selectedOrder = append(s.selectedOrder, id)
	if s.selectedID == "" {
		s.selectedID = id
	}
	if s.anchorID == "" {
		s.anchorID = id
	}
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

func (s *NativeSelectionState) replaceOrdered(ids []string, anchorID string) {
	s.replaceOrderedWithPrimary(ids, "", anchorID)
}

func (s *NativeSelectionState) replaceOrderedWithPrimary(ids []string, primaryID, anchorID string) {
	if s == nil {
		return
	}
	next := make(map[string]struct{}, len(ids))
	ordered := make([]string, 0, len(ids))
	var primary string
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := next[id]; ok {
			continue
		}
		if primary == "" {
			primary = id
		}
		next[id] = struct{}{}
		ordered = append(ordered, id)
	}
	if primaryID != "" {
		if _, ok := next[primaryID]; ok {
			primary = primaryID
		}
	}
	s.mu.Lock()
	if primary == "" {
		s.selectedIDs = make(map[string]struct{})
		s.selectedOrder = nil
		s.selectedID = ""
		s.anchorID = ""
	} else {
		s.selectedIDs = next
		s.selectedOrder = ordered
		s.selectedID = primary
		if _, ok := next[anchorID]; ok {
			s.anchorID = anchorID
		} else {
			s.anchorID = primary
		}
	}
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Clear removes the current selection.
func (s *NativeSelectionState) Clear() {
	if s == nil {
		return
	}
	s.Replace()
}

// RevisionState reports the owned revision state.
func (s *NativeSelectionState) RevisionState() *IntState {
	if s == nil {
		return nil
	}
	return s.revisionState
}

// Release releases owned state handles.
func (s *NativeSelectionState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	revisionState := s.revisionState
	s.revisionState = nil
	s.selectedIDs = nil
	s.selectedOrder = nil
	s.selectedID = ""
	s.anchorID = ""
	s.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

// NativeOutlineExpansionState owns disclosure state keyed by string ID.
//
// Curated surface.
//
// The zero value is not usable. Use NewNativeOutlineExpansionState.
type NativeOutlineExpansionState struct {
	mu sync.RWMutex

	expanded map[string]*BoolState

	revision      int
	revisionState *IntState
}

// NewNativeOutlineExpansionState creates a new outline expansion state.
func NewNativeOutlineExpansionState(expandedIDs ...string) *NativeOutlineExpansionState {
	s := &NativeOutlineExpansionState{
		expanded:      make(map[string]*BoolState),
		revisionState: newIntStateIfReady(0),
	}
	for _, id := range expandedIDs {
		if id == "" {
			continue
		}
		state := newBoolStateIfReady(true)
		s.expanded[id] = state
	}
	return s
}

// State returns the disclosure BoolState for id, creating it when needed.
func (s *NativeOutlineExpansionState) State(id string) *BoolState {
	if s == nil || id == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if state := s.expanded[id]; state != nil {
		return state
	}
	state := newBoolStateIfReady(false)
	s.expanded[id] = state
	return state
}

// IsExpanded reports whether id is expanded.
func (s *NativeOutlineExpansionState) IsExpanded(id string) bool {
	state := s.State(id)
	return state != nil && state.Get()
}

// SetExpanded sets the expanded state for id.
func (s *NativeOutlineExpansionState) SetExpanded(id string, expanded bool) {
	state := s.State(id)
	if state == nil || state.Get() == expanded {
		return
	}
	updateBoolState(state, expanded)
	s.bump()
}

// ExpandedIDs reports expanded IDs in sorted order.
func (s *NativeOutlineExpansionState) ExpandedIDs() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.expanded))
	for id, state := range s.expanded {
		if state != nil && state.Get() {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// RevisionState reports the owned revision state.
func (s *NativeOutlineExpansionState) RevisionState() *IntState {
	if s == nil {
		return nil
	}
	return s.revisionState
}

// Release releases owned state handles.
func (s *NativeOutlineExpansionState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	for id, state := range s.expanded {
		if state != nil {
			state.Release()
		}
		delete(s.expanded, id)
	}
	revisionState := s.revisionState
	s.revisionState = nil
	s.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

func (s *NativeOutlineExpansionState) bump() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// NativeTableRow describes one string-keyed row for NativeTableModel.
//
// Curated surface.
type NativeTableRow struct {
	ID string
}

// NativeTableColumn describes one rendered column in NativeTableView.
//
// Curated surface.
type NativeTableColumn struct {
	ID     string
	Header View
	Cell   func(NativeTableRow) View
	Width  float64
}

// NativeTableModel owns string-keyed rows plus explicit selection state for a
// first native-backed table layer.
//
// Curated surface.
type NativeTableModel struct {
	mu sync.RWMutex

	rows       []NativeTableRow
	selection  *NativeSelectionState
	onActivate func(NativeTableRow)

	revision      int
	revisionState *IntState
}

// NewNativeTableModel creates a new native-backed table model.
func NewNativeTableModel(rows []NativeTableRow, selection *NativeTableSelectionState) *NativeTableModel {
	if selection == nil {
		selection = NewNativeSelectionState()
	}
	m := &NativeTableModel{
		selection:     selection,
		revisionState: newIntStateIfReady(0),
	}
	m.SetRows(rows)
	return m
}

// Rows returns a copy of the current rows.
func (m *NativeTableModel) Rows() []NativeTableRow {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]NativeTableRow(nil), m.rows...)
}

// SetRows replaces the current rows and prunes selection to reachable IDs.
func (m *NativeTableModel) SetRows(rows []NativeTableRow) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.rows = append([]NativeTableRow(nil), rows...)
	revisionState := m.revisionState
	m.revision++
	revision := m.revision
	m.mu.Unlock()
	m.syncSelection()
	updateIntState(revisionState, revision)
}

// RowByID reports the current row for id.
func (m *NativeTableModel) RowByID(id string) (NativeTableRow, bool) {
	var zero NativeTableRow
	if m == nil || id == "" {
		return zero, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, row := range m.rows {
		if row.ID == id {
			return row, true
		}
	}
	return zero, false
}

// SelectID selects id when it exists.
func (m *NativeTableModel) SelectID(id string) bool {
	if m == nil {
		return false
	}
	if _, ok := m.RowByID(id); !ok {
		return false
	}
	m.selection.Replace(id)
	return true
}

// ClearSelection removes the current table selection.
func (m *NativeTableModel) ClearSelection() {
	if m == nil || m.selection == nil {
		return
	}
	m.selection.Clear()
}

// SelectAll selects every row in current row order.
func (m *NativeTableModel) SelectAll() bool {
	if m == nil {
		return false
	}
	rows := m.Rows()
	if len(rows) == 0 {
		return false
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	m.selection.replaceOrdered(ids, "")
	return true
}

// ToggleSelectedID toggles membership for id in the current table selection.
func (m *NativeTableModel) ToggleSelectedID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	if _, ok := m.RowByID(id); !ok {
		return false
	}
	if m.selection.Has(id) {
		selected := m.selection.SelectedIDs()
		kept := make([]string, 0, len(selected))
		for _, selectedID := range selected {
			if selectedID == id {
				continue
			}
			kept = append(kept, selectedID)
		}
		if len(kept) == 0 {
			m.selection.Clear()
			return true
		}
		m.selection.Replace(kept...)
		return true
	}
	return m.selection.Add(id)
}

// SelectRangeToID selects the inclusive row range from the current anchor to id.
func (m *NativeTableModel) SelectRangeToID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	rows := m.Rows()
	if len(rows) == 0 {
		return false
	}
	target := -1
	for i, row := range rows {
		if row.ID == id {
			target = i
			break
		}
	}
	if target < 0 {
		return false
	}
	anchorID, ok := m.SelectionAnchorID()
	if !ok {
		if currentID, currentOK := m.SelectionState().SelectedID(); currentOK {
			anchorID = currentID
		} else {
			anchorID = id
		}
	}
	anchor := -1
	for i, row := range rows {
		if row.ID == anchorID {
			anchor = i
			break
		}
	}
	if anchor < 0 {
		anchor = target
		anchorID = id
	}
	start, end := anchor, target
	if start > end {
		start, end = end, start
	}
	ids := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		ids = append(ids, rows[i].ID)
	}
	m.selection.replaceOrdered(ids, anchorID)
	return true
}

// RevealID selects id when it exists.
func (m *NativeTableModel) RevealID(id string) bool {
	return m.SelectID(id)
}

// SelectedRow reports the primary selected row.
func (m *NativeTableModel) SelectedRow() (NativeTableRow, bool) {
	var zero NativeTableRow
	if m == nil {
		return zero, false
	}
	id, ok := m.selection.SelectedID()
	if !ok {
		return zero, false
	}
	return m.RowByID(id)
}

// SelectedRows reports the selected rows in current row order.
func (m *NativeTableModel) SelectedRows() []NativeTableRow {
	if m == nil {
		return nil
	}
	selected := m.SelectionState()
	if selected == nil {
		return nil
	}
	ids := make(map[string]struct{}, len(selected.SelectedIDs()))
	for _, id := range selected.SelectedIDs() {
		ids[id] = struct{}{}
	}
	rows := m.Rows()
	selectedRows := make([]NativeTableRow, 0, len(ids))
	for _, row := range rows {
		if _, ok := ids[row.ID]; ok {
			selectedRows = append(selectedRows, row)
		}
	}
	return selectedRows
}

// SelectedCount reports the number of selected rows.
func (m *NativeTableModel) SelectedCount() int {
	if m == nil || m.selection == nil {
		return 0
	}
	return m.selection.SelectedCount()
}

// SelectedRowStateSummary reports the current primary row state in AX-friendly text.
func (m *NativeTableModel) SelectedRowStateSummary() string {
	if m == nil {
		return "none"
	}
	row, ok := m.SelectedRow()
	if !ok {
		return "none"
	}
	return rowStateSummary(m.selection.Has(row.ID), false, false)
}

// SelectionAnchorID reports the current selection anchor identifier.
func (m *NativeTableModel) SelectionAnchorID() (string, bool) {
	if m == nil || m.selection == nil {
		return "", false
	}
	return m.selection.AnchorID()
}

// SetOnActivate sets the row activation callback.
func (m *NativeTableModel) SetOnActivate(fn func(NativeTableRow)) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.onActivate = fn
	m.mu.Unlock()
}

// ActivateSelected invokes the activation callback for the selected row.
func (m *NativeTableModel) ActivateSelected() bool {
	row, ok := m.SelectedRow()
	if !ok {
		return false
	}
	m.activate(row)
	return true
}

// SelectNext advances the primary selection by one row in current order.
func (m *NativeTableModel) SelectNext() bool {
	if m == nil {
		return false
	}
	rows := m.Rows()
	if len(rows) == 0 {
		return false
	}
	current := -1
	if id, ok := m.SelectionState().SelectedID(); ok {
		for i, row := range rows {
			if row.ID == id {
				current = i
				break
			}
		}
	}
	next := current + 1
	if next < 0 || next >= len(rows) {
		return false
	}
	return m.SelectID(rows[next].ID)
}

// ExtendSelectionNext extends the current row-order selection to the next row.
func (m *NativeTableModel) ExtendSelectionNext() bool {
	if m == nil {
		return false
	}
	rows := m.Rows()
	if len(rows) == 0 {
		return false
	}
	start := -1
	end := -1
	for i, row := range rows {
		if !m.selection.Has(row.ID) {
			continue
		}
		if start < 0 {
			start = i
		}
		end = i
	}
	hadSelection := start >= 0
	if start < 0 {
		start = 0
		end = -1
	}
	next := end + 1
	if next < 0 || next >= len(rows) {
		return false
	}
	anchorID, ok := m.SelectionAnchorID()
	if !ok {
		if hadSelection {
			anchorID = rows[start].ID
		} else {
			anchorID = rows[next].ID
		}
	}
	ids := make([]string, 0, next-start+1)
	for i := start; i <= next; i++ {
		ids = append(ids, rows[i].ID)
	}
	m.selection.replaceOrdered(ids, anchorID)
	return true
}

// SelectPrevious moves the primary selection backward by one row in current order.
func (m *NativeTableModel) SelectPrevious() bool {
	if m == nil {
		return false
	}
	rows := m.Rows()
	if len(rows) == 0 {
		return false
	}
	current := len(rows)
	if id, ok := m.SelectionState().SelectedID(); ok {
		for i, row := range rows {
			if row.ID == id {
				current = i
				break
			}
		}
	}
	prev := current - 1
	if prev < 0 || prev >= len(rows) {
		return false
	}
	return m.SelectID(rows[prev].ID)
}

// ExtendSelectionPrevious extends the current row-order selection to the previous row.
func (m *NativeTableModel) ExtendSelectionPrevious() bool {
	if m == nil {
		return false
	}
	rows := m.Rows()
	if len(rows) == 0 {
		return false
	}
	start := -1
	end := -1
	for i, row := range rows {
		if !m.selection.Has(row.ID) {
			continue
		}
		if start < 0 {
			start = i
		}
		end = i
	}
	hadSelection := start >= 0
	if end < 0 {
		start = len(rows)
		end = len(rows)
	}
	prev := start - 1
	if prev < 0 || prev >= len(rows) {
		return false
	}
	anchorID, ok := m.SelectionAnchorID()
	if !ok {
		if hadSelection {
			anchorID = rows[end].ID
		} else {
			anchorID = rows[prev].ID
		}
	}
	ids := make([]string, 0, end-prev+1)
	for i := prev; i <= end; i++ {
		ids = append(ids, rows[i].ID)
	}
	m.selection.replaceOrdered(ids, anchorID)
	return true
}

// SelectionState reports the owned selection state.
func (m *NativeTableModel) SelectionState() *NativeTableSelectionState {
	if m == nil {
		return nil
	}
	return m.selection
}

// RevisionState reports the owned revision state.
func (m *NativeTableModel) RevisionState() *IntState {
	if m == nil {
		return nil
	}
	return m.revisionState
}

// Release releases owned state handles.
func (m *NativeTableModel) Release() {
	if m == nil {
		return
	}
	m.mu.Lock()
	revisionState := m.revisionState
	m.revisionState = nil
	m.rows = nil
	m.onActivate = nil
	m.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

func (m *NativeTableModel) syncSelection() {
	if m == nil || m.selection == nil {
		return
	}
	reachable := make(map[string]struct{}, len(m.Rows()))
	for _, row := range m.Rows() {
		if row.ID == "" {
			continue
		}
		reachable[row.ID] = struct{}{}
	}
	kept := make([]string, 0, len(reachable))
	for _, id := range m.selection.SelectedIDs() {
		if _, ok := reachable[id]; ok {
			kept = append(kept, id)
		}
	}
	if len(kept) == 0 {
		m.selection.Clear()
		return
	}
	primary := ""
	for _, row := range m.Rows() {
		if _, ok := reachable[row.ID]; !ok {
			continue
		}
		if m.selection.Has(row.ID) {
			primary = row.ID
			break
		}
	}
	if primary == "" {
		primary = kept[0]
	}
	m.selection.replaceOrderedWithPrimary(kept, primary, primary)
}

func (m *NativeTableModel) activate(row NativeTableRow) {
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

// NativeOutlineNode describes one string-keyed node for NativeOutlineModel.
//
// Curated surface.
type NativeOutlineNode struct {
	ID       string
	Label    View
	Children []NativeOutlineNode
}

// NativeOutlineModel owns explicit selection and expansion state for a first
// native-backed outline layer.
//
// Curated surface.
type NativeOutlineModel struct {
	mu sync.RWMutex

	roots      []NativeOutlineNode
	selection  *NativeSelectionState
	expansion  *NativeOutlineExpansionState
	onActivate func(NativeOutlineNode)

	revision      int
	revisionState *IntState
}

// NewNativeOutlineModel creates a new native-backed outline model.
func NewNativeOutlineModel(roots []NativeOutlineNode, selection *NativeOutlineSelectionState, expansion *NativeOutlineExpansionState) *NativeOutlineModel {
	if selection == nil {
		selection = NewNativeSelectionState()
	}
	if expansion == nil {
		expansion = NewNativeOutlineExpansionState()
	}
	m := &NativeOutlineModel{
		selection:     selection,
		expansion:     expansion,
		revisionState: newIntStateIfReady(0),
	}
	m.SetRoots(roots)
	return m
}

// Roots returns a copy of the current roots.
func (m *NativeOutlineModel) Roots() []NativeOutlineNode {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]NativeOutlineNode(nil), m.roots...)
}

// SetRoots replaces the current roots and prunes selection/expansion to
// reachable IDs.
func (m *NativeOutlineModel) SetRoots(roots []NativeOutlineNode) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.roots = append([]NativeOutlineNode(nil), roots...)
	revisionState := m.revisionState
	m.revision++
	revision := m.revision
	m.mu.Unlock()
	m.syncState()
	updateIntState(revisionState, revision)
}

// RowByID reports the current node for id.
func (m *NativeOutlineModel) RowByID(id string) (NativeOutlineNode, bool) {
	var zero NativeOutlineNode
	if m == nil || id == "" {
		return zero, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return nativeOutlineRowByID(id, m.roots)
}

// RevealID expands ancestors for id and selects it.
func (m *NativeOutlineModel) RevealID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	path, ok := m.pathToID(id)
	if !ok {
		return false
	}
	for _, ancestor := range path[:len(path)-1] {
		m.expansion.SetExpanded(ancestor, true)
	}
	m.selection.Replace(id)
	return true
}

// SelectedRow reports the primary selected node.
func (m *NativeOutlineModel) SelectedRow() (NativeOutlineNode, bool) {
	var zero NativeOutlineNode
	if m == nil {
		return zero, false
	}
	id, ok := m.selection.SelectedID()
	if !ok {
		return zero, false
	}
	return m.RowByID(id)
}

// SelectedRows reports the selected nodes in current tree order.
func (m *NativeOutlineModel) SelectedRows() []NativeOutlineNode {
	if m == nil {
		return nil
	}
	selected := m.SelectionState()
	if selected == nil {
		return nil
	}
	ids := make(map[string]struct{}, len(selected.SelectedIDs()))
	for _, id := range selected.SelectedIDs() {
		ids[id] = struct{}{}
	}
	rows := make([]NativeOutlineNode, 0, len(ids))
	var visit func([]NativeOutlineNode)
	visit = func(nodes []NativeOutlineNode) {
		for _, node := range nodes {
			if _, ok := ids[node.ID]; ok {
				rows = append(rows, node)
			}
			if len(node.Children) > 0 {
				visit(node.Children)
			}
		}
	}
	visit(m.Roots())
	return rows
}

// SelectedCount reports the number of selected nodes.
func (m *NativeOutlineModel) SelectedCount() int {
	if m == nil || m.selection == nil {
		return 0
	}
	return m.selection.SelectedCount()
}

// SelectedRowStateSummary reports the current primary row state in AX-friendly text.
func (m *NativeOutlineModel) SelectedRowStateSummary() string {
	if m == nil {
		return "none"
	}
	row, ok := m.SelectedRow()
	if !ok {
		return "none"
	}
	expanded := false
	if len(row.Children) > 0 {
		expanded = m.expansion.IsExpanded(row.ID)
	}
	return rowStateSummary(m.selection.Has(row.ID), expanded, len(row.Children) > 0)
}

// SelectionAnchorID reports the current selection anchor identifier.
func (m *NativeOutlineModel) SelectionAnchorID() (string, bool) {
	if m == nil || m.selection == nil {
		return "", false
	}
	return m.selection.AnchorID()
}

// SetExpandedID sets explicit expansion for id.
func (m *NativeOutlineModel) SetExpandedID(id string, expanded bool) {
	if m == nil || id == "" {
		return
	}
	if _, ok := m.RowByID(id); !ok {
		return
	}
	m.expansion.SetExpanded(id, expanded)
}

// SelectID selects id when it exists, revealing ancestors first.
func (m *NativeOutlineModel) SelectID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	path, ok := m.pathToID(id)
	if !ok {
		return false
	}
	for _, ancestor := range path[:len(path)-1] {
		m.expansion.SetExpanded(ancestor, true)
	}
	m.selection.Replace(id)
	return true
}

// ClearSelection removes the current outline selection.
func (m *NativeOutlineModel) ClearSelection() {
	if m == nil || m.selection == nil {
		return
	}
	m.selection.Clear()
}

// SelectAll selects every reachable node in tree order.
func (m *NativeOutlineModel) SelectAll() bool {
	if m == nil {
		return false
	}
	ids := m.allIDs()
	if len(ids) == 0 {
		return false
	}
	m.selection.replaceOrdered(ids, "")
	return true
}

// ToggleSelectedID toggles membership for id in the current outline selection.
func (m *NativeOutlineModel) ToggleSelectedID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	path, ok := m.pathToID(id)
	if !ok {
		return false
	}
	if m.selection.Has(id) {
		selected := m.selection.SelectedIDs()
		kept := make([]string, 0, len(selected))
		for _, selectedID := range selected {
			if selectedID == id {
				continue
			}
			kept = append(kept, selectedID)
		}
		if len(kept) == 0 {
			m.selection.Clear()
			return true
		}
		m.selection.Replace(kept...)
		return true
	}
	for _, ancestor := range path[:len(path)-1] {
		m.expansion.SetExpanded(ancestor, true)
	}
	return m.selection.Add(id)
}

// SelectNextVisible advances the primary selection in current visible tree order.
func (m *NativeOutlineModel) SelectNextVisible() bool {
	if m == nil {
		return false
	}
	all := m.visibleIDs()
	if len(all) == 0 {
		return false
	}
	current := -1
	if id, ok := m.SelectionState().SelectedID(); ok {
		for i, visibleID := range all {
			if visibleID == id {
				current = i
				break
			}
		}
	}
	next := current + 1
	if next < 0 || next >= len(all) {
		return false
	}
	m.selection.Replace(all[next])
	return true
}

// SelectVisibleRangeToID selects the inclusive visible range from the current
// anchor to id, expanding ancestors first when needed.
func (m *NativeOutlineModel) SelectVisibleRangeToID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	path, ok := m.pathToID(id)
	if !ok {
		return false
	}
	for _, ancestor := range path[:len(path)-1] {
		m.expansion.SetExpanded(ancestor, true)
	}
	all := m.visibleIDs()
	target := -1
	for i, visibleID := range all {
		if visibleID == id {
			target = i
			break
		}
	}
	if target < 0 {
		return false
	}
	anchorID, ok := m.SelectionAnchorID()
	if !ok {
		if currentID, currentOK := m.SelectionState().SelectedID(); currentOK {
			anchorID = currentID
		} else {
			anchorID = id
		}
	}
	anchor := -1
	for i, visibleID := range all {
		if visibleID == anchorID {
			anchor = i
			break
		}
	}
	if anchor < 0 {
		anchor = target
		anchorID = id
	}
	start, end := anchor, target
	if start > end {
		start, end = end, start
	}
	ids := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		ids = append(ids, all[i])
	}
	m.selection.replaceOrdered(ids, anchorID)
	return true
}

// SelectPreviousVisible moves the primary selection backward in current visible tree order.
func (m *NativeOutlineModel) SelectPreviousVisible() bool {
	if m == nil {
		return false
	}
	all := m.visibleIDs()
	if len(all) == 0 {
		return false
	}
	current := len(all)
	if id, ok := m.SelectionState().SelectedID(); ok {
		for i, visibleID := range all {
			if visibleID == id {
				current = i
				break
			}
		}
	}
	prev := current - 1
	if prev < 0 || prev >= len(all) {
		return false
	}
	m.selection.Replace(all[prev])
	return true
}

// ExtendSelectionNextVisible extends the current visible selection to the next visible row.
func (m *NativeOutlineModel) ExtendSelectionNextVisible() bool {
	if m == nil {
		return false
	}
	all := m.visibleIDs()
	if len(all) == 0 {
		return false
	}
	start := -1
	end := -1
	for i, visibleID := range all {
		if !m.selection.Has(visibleID) {
			continue
		}
		if start < 0 {
			start = i
		}
		end = i
	}
	hadSelection := start >= 0
	if start < 0 {
		start = 0
		end = -1
	}
	next := end + 1
	if next < 0 || next >= len(all) {
		return false
	}
	anchorID, ok := m.SelectionAnchorID()
	if !ok {
		if hadSelection {
			anchorID = all[start]
		} else {
			anchorID = all[next]
		}
	}
	ids := make([]string, 0, next-start+1)
	for i := start; i <= next; i++ {
		ids = append(ids, all[i])
	}
	m.selection.replaceOrdered(ids, anchorID)
	return true
}

// ExtendSelectionPreviousVisible extends the current visible selection to the previous visible row.
func (m *NativeOutlineModel) ExtendSelectionPreviousVisible() bool {
	if m == nil {
		return false
	}
	all := m.visibleIDs()
	if len(all) == 0 {
		return false
	}
	start := -1
	end := -1
	for i, visibleID := range all {
		if !m.selection.Has(visibleID) {
			continue
		}
		if start < 0 {
			start = i
		}
		end = i
	}
	hadSelection := start >= 0
	if end < 0 {
		start = len(all)
		end = len(all)
	}
	prev := start - 1
	if prev < 0 || prev >= len(all) {
		return false
	}
	anchorID, ok := m.SelectionAnchorID()
	if !ok {
		if hadSelection {
			anchorID = all[end]
		} else {
			anchorID = all[prev]
		}
	}
	ids := make([]string, 0, end-prev+1)
	for i := prev; i <= end; i++ {
		ids = append(ids, all[i])
	}
	m.selection.replaceOrdered(ids, anchorID)
	return true
}

// ExpandedIDs reports expanded IDs in sorted order.
func (m *NativeOutlineModel) ExpandedIDs() []string {
	if m == nil {
		return nil
	}
	return m.expansion.ExpandedIDs()
}

// SelectionState reports the owned selection state.
func (m *NativeOutlineModel) SelectionState() *NativeOutlineSelectionState {
	if m == nil {
		return nil
	}
	return m.selection
}

// ExpansionState reports the owned expansion state.
func (m *NativeOutlineModel) ExpansionState() *NativeOutlineExpansionState {
	if m == nil {
		return nil
	}
	return m.expansion
}

// SetOnActivate sets the row activation callback.
func (m *NativeOutlineModel) SetOnActivate(fn func(NativeOutlineNode)) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.onActivate = fn
	m.mu.Unlock()
}

// ActivateSelected invokes the activation callback for the selected node.
func (m *NativeOutlineModel) ActivateSelected() bool {
	row, ok := m.SelectedRow()
	if !ok {
		return false
	}
	m.activate(row)
	return true
}

// RevisionState reports the owned revision state.
func (m *NativeOutlineModel) RevisionState() *IntState {
	if m == nil {
		return nil
	}
	return m.revisionState
}

// Release releases owned state handles.
func (m *NativeOutlineModel) Release() {
	if m == nil {
		return
	}
	m.mu.Lock()
	revisionState := m.revisionState
	m.revisionState = nil
	m.roots = nil
	m.onActivate = nil
	m.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

func (m *NativeOutlineModel) syncState() {
	if m == nil {
		return
	}
	reachable := make(map[string]struct{})
	var visit func([]NativeOutlineNode)
	visit = func(nodes []NativeOutlineNode) {
		for _, node := range nodes {
			if node.ID != "" {
				reachable[node.ID] = struct{}{}
			}
			if len(node.Children) > 0 {
				visit(node.Children)
			}
		}
	}
	visit(m.Roots())

	keptSelection := make([]string, 0, len(reachable))
	for _, id := range m.selection.SelectedIDs() {
		if _, ok := reachable[id]; ok {
			keptSelection = append(keptSelection, id)
		}
	}
	if len(keptSelection) == 0 {
		m.selection.Clear()
	} else {
		primary := ""
		var visitPrimary func([]NativeOutlineNode)
		visitPrimary = func(nodes []NativeOutlineNode) {
			for _, node := range nodes {
				if _, ok := reachable[node.ID]; !ok {
					continue
				}
				if m.selection.Has(node.ID) {
					primary = node.ID
					return
				}
				if len(node.Children) > 0 {
					visitPrimary(node.Children)
					if primary != "" {
						return
					}
				}
			}
		}
		visitPrimary(m.Roots())
		if primary == "" {
			primary = keptSelection[0]
		}
		m.selection.replaceOrderedWithPrimary(keptSelection, primary, primary)
	}

	for _, id := range m.expansion.ExpandedIDs() {
		if _, ok := reachable[id]; ok {
			continue
		}
		m.expansion.SetExpanded(id, false)
	}
}

func (m *NativeOutlineModel) pathToID(id string) ([]string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return nativeOutlinePathToID(id, m.roots, nil)
}

func (m *NativeOutlineModel) visibleIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0)
	var visit func([]NativeOutlineNode)
	visit = func(nodes []NativeOutlineNode) {
		for _, node := range nodes {
			ids = append(ids, node.ID)
			if len(node.Children) == 0 {
				continue
			}
			if m.expansion.IsExpanded(node.ID) {
				visit(node.Children)
			}
		}
	}
	visit(m.roots)
	return ids
}

func (m *NativeOutlineModel) allIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0)
	var visit func([]NativeOutlineNode)
	visit = func(nodes []NativeOutlineNode) {
		for _, node := range nodes {
			ids = append(ids, node.ID)
			if len(node.Children) > 0 {
				visit(node.Children)
			}
		}
	}
	visit(m.roots)
	return ids
}

func (m *NativeOutlineModel) activate(row NativeOutlineNode) {
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

// NativeTableView renders a first native-backed table surface using explicit
// row identity and selection state.
func NativeTableView(model *NativeTableModel, columns ...NativeTableColumn) View {
	if model == nil {
		return EmptyView()
	}
	build := func() View {
		rows := model.Rows()
		children := make([]Viewable, 0, len(rows)+2)
		if len(columns) > 0 {
			children = append(children, nativeTableHeaderRow(columns), Divider())
		}
		for _, row := range rows {
			children = append(children, nativeTableBodyRow(model, row, columns))
		}
		return VStackSpaced(0, children...)
	}
	if model.RevisionState() == nil || model.SelectionState() == nil || model.SelectionState().RevisionState() == nil {
		return build()
	}
	return DynamicView(model.RevisionState(), func(int) View {
		return DynamicView(model.SelectionState().RevisionState(), func(int) View {
			return build()
		})
	})
}

func nativeTableHeaderRow(columns []NativeTableColumn) View {
	cells := make([]Viewable, 0, len(columns))
	for _, column := range columns {
		cells = append(cells, nativeTableCell(column.Header, column.Width))
	}
	return HStackSpaced(12, cells...).PaddingEdge(EdgeVertical, 6)
}

func nativeTableBodyRow(model *NativeTableModel, row NativeTableRow, columns []NativeTableColumn) View {
	cells := make([]Viewable, 0, len(columns))
	for _, column := range columns {
		cell := EmptyView()
		if column.Cell != nil {
			cell = column.Cell(row)
		}
		cells = append(cells, nativeTableCell(cell, column.Width))
	}
	summary := rowStateSummary(model.SelectionState().Has(row.ID), false, false)
	rowContent := HStackSpaced(12, cells...).PaddingEdge(EdgeVertical, 6)
	if summary != "" {
		rowContent = HStackSpaced(12, rowContent, rowStateBadge(summary)).MaxFrame(-1, 0)
	}
	rowView := ButtonView(
		rowContent,
		func() {
			model.SelectID(row.ID)
		},
	).ButtonStyle(ButtonStylePlain).AccessibilityIdentifier(tableRowAccessibilityID(row.ID))
	if summary != "" {
		rowView = rowView.AccessibilityHint(summary).AccessibilityValue(summary)
		return rowView.BackgroundRoundedRect(0.23, 0.47, 0.95, 0.16, 8)
	}
	return rowView
}

func nativeTableCell(cell View, width float64) View {
	if width > 0 {
		return cell.Frame(width, 0)
	}
	return cell.MaxFrame(-1, 0)
}

// NativeOutlineView renders a first native-backed outline surface using the
// low-level OutlineGroup helper plus explicit string-ID state.
func NativeOutlineView(model *NativeOutlineModel) View {
	if model == nil {
		return EmptyView()
	}
	build := func() View {
		return OutlineGroup(nativeOutlineNodes(model, model.Roots())...)
	}
	if model.RevisionState() == nil || model.SelectionState() == nil || model.SelectionState().RevisionState() == nil || model.ExpansionState() == nil || model.ExpansionState().RevisionState() == nil {
		return build()
	}
	return DynamicView(model.RevisionState(), func(int) View {
		return DynamicView(model.SelectionState().RevisionState(), func(int) View {
			return DynamicView(model.ExpansionState().RevisionState(), func(int) View {
				return build()
			})
		})
	})
}

func nativeOutlineNodes(model *NativeOutlineModel, rows []NativeOutlineNode) []OutlineNode {
	nodes := make([]OutlineNode, 0, len(rows))
	for _, row := range rows {
		rowCopy := row
		expanded := false
		if len(rowCopy.Children) > 0 {
			expanded = model.expansion.IsExpanded(rowCopy.ID)
		}
		summary := rowStateSummary(model.selection.Has(rowCopy.ID), expanded, len(rowCopy.Children) > 0)
		label := rowCopy.Label
		if summary != "" {
			label = HStackSpaced(8, label, rowStateBadge(summary)).MaxFrame(-1, 0)
		}
		label = label.AccessibilityIdentifier(outlineRowAccessibilityID(rowCopy.ID))
		if summary != "" {
			label = label.AccessibilityHint(summary).AccessibilityValue(summary)
		}
		label = ButtonView(label, func() {
			model.selection.Replace(rowCopy.ID)
			model.activate(rowCopy)
		}).ButtonStyle(ButtonStylePlain)
		if model.selection.Has(rowCopy.ID) {
			label = label.BackgroundRoundedRect(0.23, 0.47, 0.95, 0.16, 8)
		}
		node := OutlineNode{
			Label: label,
		}
		if len(rowCopy.Children) > 0 {
			node.Expanded = model.expansion.State(rowCopy.ID)
			node.Children = nativeOutlineNodes(model, rowCopy.Children)
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func nativeOutlineRowByID(id string, rows []NativeOutlineNode) (NativeOutlineNode, bool) {
	var zero NativeOutlineNode
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
		if len(row.Children) == 0 {
			continue
		}
		if found, ok := nativeOutlineRowByID(id, row.Children); ok {
			return found, true
		}
	}
	return zero, false
}

func nativeOutlinePathToID(id string, rows []NativeOutlineNode, path []string) ([]string, bool) {
	for _, row := range rows {
		next := append(append([]string(nil), path...), row.ID)
		if row.ID == id {
			return next, true
		}
		if len(row.Children) == 0 {
			continue
		}
		if found, ok := nativeOutlinePathToID(id, row.Children, next); ok {
			return found, true
		}
	}
	return nil, false
}
