package swiftui

import (
	"sort"
	"strconv"
	"sync"
)

// TableModelColumn describes one typed column in a TableModelView.
//
// Curated surface.
type TableModelColumn[T any] struct {
	Header   View
	Cell     func(T) View
	Less     func(a, b T) bool
	ID       string
	MaxWidth float64
	Hidden   bool
}

// TableModelColumnView constructs a typed table column.
func TableModelColumnView[T any](header View, cell func(T) View) TableModelColumn[T] {
	return TableModelColumn[T]{Header: header, Cell: cell}
}

// SortableTableModelColumn constructs a typed sortable table column.
func SortableTableModelColumn[T any](header View, cell func(T) View, less func(a, b T) bool) TableModelColumn[T] {
	return TableModelColumn[T]{Header: header, Cell: cell, Less: less}
}

// TextTableModelColumn constructs a sortable text column.
func TextTableModelColumn[T any](title string, text func(T) string, less func(a, b T) bool) TableModelColumn[T] {
	column := SortableTableModelColumn(
		Text(title).FontWeight(WeightSemibold).AsView(),
		func(row T) View {
			if text == nil {
				return EmptyView()
			}
			return Text(text(row)).AsView()
		},
		less,
	)
	column.ID = title
	return column
}

// WithID returns a copy of c with a stable sort identifier.
func (c TableModelColumn[T]) WithID(id string) TableModelColumn[T] {
	c.ID = id
	return c
}

// WithMaxWidth returns a copy of c with a maximum rendered width.
func (c TableModelColumn[T]) WithMaxWidth(width float64) TableModelColumn[T] {
	c.MaxWidth = width
	return c
}

// WithHidden returns a copy of c with hidden state set.
func (c TableModelColumn[T]) WithHidden(hidden bool) TableModelColumn[T] {
	c.Hidden = hidden
	return c
}

// WithVisible returns a copy of c with the inverse hidden state set.
func (c TableModelColumn[T]) WithVisible(visible bool) TableModelColumn[T] {
	c.Hidden = !visible
	return c
}

// IsVisible reports whether the column is visible.
func (c TableModelColumn[T]) IsVisible() bool {
	return !c.Hidden
}

// TableModelView renders a TableModel with typed columns, row identity,
// selection, and stable sorting.
func TableModelView[T any](model *TableModel[T], columns ...TableModelColumn[T]) View {
	if model == nil {
		return EmptyView()
	}
	build := func() View {
		return tableModelView(model, columns...)
	}
	if model.RevisionState() == nil {
		return build()
	}
	return DynamicView(model.RevisionState(), func(int) View {
		return build()
	})
}

// TableModelViewWithVisibility renders a TableModelView that rebuilds when
// column visibility changes.
func TableModelViewWithVisibility[T any](model *TableModel[T], visibility *TableColumnVisibilityState, columns ...TableModelColumn[T]) View {
	if model == nil {
		return EmptyView()
	}
	build := func() View {
		return TableModelView(model, ApplyTableColumnVisibility(visibility, columns)...)
	}
	if visibility == nil || visibility.RevisionState() == nil {
		return build()
	}
	return DynamicView(visibility.RevisionState(), func(int) View {
		return build()
	})
}

// TableModelViewWithLayout renders a TableModelView that rebuilds when either
// column visibility or explicit width state changes.
func TableModelViewWithLayout[T any](model *TableModel[T], visibility *TableColumnVisibilityState, widths *TableColumnWidthState, columns ...TableModelColumn[T]) View {
	if model == nil {
		return EmptyView()
	}
	build := func() View {
		return TableModelView(model, ApplyTableColumnLayout(visibility, widths, columns)...)
	}
	if visibility == nil && (widths == nil || widths.RevisionState() == nil) {
		return build()
	}
	if widths == nil || widths.RevisionState() == nil {
		return DynamicView(visibility.RevisionState(), func(int) View {
			return build()
		})
	}
	if visibility == nil || visibility.RevisionState() == nil {
		return DynamicView(widths.RevisionState(), func(int) View {
			return build()
		})
	}
	return DynamicView(visibility.RevisionState(), func(int) View {
		return DynamicView(widths.RevisionState(), func(int) View {
			return build()
		})
	})
}

func tableModelView[T any](model *TableModel[T], columns ...TableModelColumn[T]) View {
	columns = tableModelVisibleColumns(columns)
	rows := model.Rows()
	children := make([]Viewable, 0, len(rows)+2)
	if len(columns) > 0 {
		children = append(children, tableModelHeaderRow(model, columns), Divider())
	}
	for _, row := range rows {
		children = append(children, tableModelBodyRow(model, row, columns))
	}
	return VStackSpaced(0, children...)
}

func tableModelHeaderRow[T any](model *TableModel[T], columns []TableModelColumn[T]) View {
	cells := make([]Viewable, 0, len(columns))
	for index, column := range columns {
		header := column.Header
		if column.Less != nil {
			less := column.Less
			id := tableModelColumnID(index, column)
			header = ButtonView(header, func() {
				model.ToggleSortColumn(id, less)
			}).ButtonStyle(ButtonStylePlain)
			if sortID, ascending, ok := model.SortColumn(); ok && sortID == id {
				glyph := "chevron.up"
				if !ascending {
					glyph = "chevron.down"
				}
				header = HStackSpaced(6, header, Image(glyph).ImageScale(ImageScaleSmall)).MaxFrame(-1, 0)
			}
		}
		cells = append(cells, tableModelCell(header.FontWeight(WeightSemibold), column.MaxWidth))
	}
	return HStackSpaced(12, cells...).PaddingEdge(EdgeVertical, 6)
}

func tableModelBodyRow[T any](model *TableModel[T], row T, columns []TableModelColumn[T]) View {
	cells := make([]Viewable, 0, len(columns))
	for _, column := range columns {
		cell := EmptyView()
		if column.Cell != nil {
			cell = column.Cell(row)
		}
		cells = append(cells, tableModelCell(cell, column.MaxWidth))
	}
	id := model.RowID(row)
	summary := rowStateSummary(model.HasSelectedID(id), false, false)
	rowContent := HStackSpaced(12, cells...).PaddingEdge(EdgeVertical, 6)
	if summary != "" {
		rowContent = HStackSpaced(12, rowContent, rowStateBadge(summary)).MaxFrame(-1, 0)
	}
	rowView := ButtonView(
		rowContent,
		func() {
			model.SelectID(id)
			model.activate(row)
		},
	).ButtonStyle(ButtonStylePlain).AccessibilityIdentifier(tableRowAccessibilityID(id))
	if summary != "" {
		rowView = rowView.AccessibilityHint(summary).AccessibilityValue(summary)
		return rowView.BackgroundRoundedRect(0.23, 0.47, 0.95, 0.16, 8)
	}
	return rowView
}

func tableModelCell(cell View, maxWidth float64) View {
	if maxWidth > 0 {
		return cell.Frame(maxWidth, 0)
	}
	return cell.MaxFrame(-1, 0)
}

func tableModelColumnID[T any](index int, column TableModelColumn[T]) string {
	if column.ID != "" {
		return column.ID
	}
	return "column-" + strconv.Itoa(index)
}

func tableModelVisibleColumns[T any](columns []TableModelColumn[T]) []TableModelColumn[T] {
	visible := make([]TableModelColumn[T], 0, len(columns))
	for _, column := range columns {
		if column.Hidden {
			continue
		}
		visible = append(visible, column)
	}
	return visible
}

func tableRowAccessibilityID(id string) string {
	return "table-row-" + id
}

func outlineRowAccessibilityID(id string) string {
	return "outline-row-" + id
}

func rowStateSummary(selected bool, expanded bool, hasChildren bool) string {
	summary := ""
	if selected {
		summary = "selected"
	}
	if hasChildren {
		state := "collapsed"
		if expanded {
			state = "expanded"
		}
		if summary == "" {
			summary = state
		} else {
			summary += ", " + state
		}
	}
	return summary
}

func rowStateBadge(summary string) View {
	if summary == "" {
		return EmptyView()
	}
	return Text(summary).
		Font(FontCaption2).
		ForegroundStyle(0.76, 0.80, 0.78, 1).
		PaddingEdge(EdgeVertical, 2).
		PaddingEdge(EdgeHorizontal, 6).
		BackgroundRoundedRect(0.15, 0.17, 0.20, 0.82, 8)
}

// ApplyTableColumnLayout returns a copy of columns with hidden flags and
// explicit width overrides applied from the current state.
func ApplyTableColumnLayout[T any](visibility *TableColumnVisibilityState, widths *TableColumnWidthState, columns []TableModelColumn[T]) []TableModelColumn[T] {
	out := ApplyTableColumnVisibility(visibility, columns)
	if widths == nil {
		return out
	}
	widthMap := widths.Widths()
	if len(widthMap) == 0 {
		return out
	}
	for i := range out {
		if out[i].ID == "" {
			continue
		}
		if width, ok := widthMap[out[i].ID]; ok {
			out[i].MaxWidth = width
		}
	}
	return out
}

// OutlineModel owns roots and expansion state for a typed outline tree.
//
// Curated surface.
type OutlineModel[T any] struct {
	mu                sync.RWMutex
	roots             []T
	id                func(T) string
	children          func(T) []T
	onActivate        func(T)
	expanded          map[string]*BoolState
	selectedIDs       map[string]struct{}
	selectedID        string
	selectionAnchorID string
	hasSelection      bool
	revision          int
	revisionState     *IntState
}

// NewOutlineModel creates an outline model with stable row identity.
func NewOutlineModel[T any](roots []T, id func(T) string, children func(T) []T) *OutlineModel[T] {
	m := &OutlineModel[T]{
		id:            normalizeTableRowID(id),
		children:      children,
		expanded:      make(map[string]*BoolState),
		revisionState: newIntStateIfReady(0),
	}
	m.SetRoots(roots)
	return m
}

// Roots returns a copy of the current root rows.
func (m *OutlineModel[T]) Roots() []T {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]T(nil), m.roots...)
}

// SetOnActivate sets the row activation callback used by outline views.
func (m *OutlineModel[T]) SetOnActivate(fn func(T)) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.onActivate = fn
	m.mu.Unlock()
}

// SetRoots replaces the current root rows.
//
// When the new roots are the current roots plus a trailing suffix, SetRoots
// reuses the existing slice and appends in place so paginated-append
// workloads stay O(delta) rather than O(n).
func (m *OutlineModel[T]) SetRoots(roots []T) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if extras, ok := diffAppendOnly(m.roots, roots, m.id); ok && len(extras) > 0 {
		// Pure append at the top level. New reachable IDs can't orphan any
		// expanded state so we skip the prune.
		m.roots = append(m.roots, extras...)
		m.syncSelectionLocked()
		m.revision++
		revisionState := m.revisionState
		revision := m.revision
		m.mu.Unlock()
		updateIntState(revisionState, revision)
		return
	}
	m.roots = append([]T(nil), roots...)
	m.pruneExpandedLocked()
	m.syncSelectionLocked()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

// ExpandedState returns the BoolState used to disclose row children.
func (m *OutlineModel[T]) ExpandedState(row T) *BoolState {
	if m == nil {
		return nil
	}
	id := m.id(row)
	m.mu.Lock()
	defer m.mu.Unlock()
	if state := m.expanded[id]; state != nil {
		return state
	}
	state := newBoolStateIfReady(false)
	m.expanded[id] = state
	return state
}

// SetExpanded sets the expansion state for a row.
func (m *OutlineModel[T]) SetExpanded(row T, expanded bool) {
	state := m.ExpandedState(row)
	if state == nil || state.Get() == expanded {
		return
	}
	updateBoolState(state, expanded)
	m.bumpRevision()
}

// ToggleExpanded toggles expansion for a branch row.
func (m *OutlineModel[T]) ToggleExpanded(row T) {
	state := m.ExpandedState(row)
	if state == nil {
		return
	}
	m.SetExpanded(row, !state.Get())
}

// SetExpandedAll sets expansion for every branch row currently reachable.
func (m *OutlineModel[T]) SetExpandedAll(expanded bool) {
	if m == nil {
		return
	}
	m.setExpandedRows(m.Roots(), expanded)
}

func (m *OutlineModel[T]) setExpandedRows(rows []T, expanded bool) {
	for _, row := range rows {
		children := []T(nil)
		if m.children != nil {
			children = m.children(row)
		}
		if len(children) == 0 {
			continue
		}
		m.SetExpanded(row, expanded)
		m.setExpandedRows(children, expanded)
	}
}

// SelectID selects the outline row with the given identifier.
func (m *OutlineModel[T]) SelectID(id string) {
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

// SelectIDs selects the outline rows with the given identifiers.
//
// The primary selection reported by SelectedID follows current tree order.
func (m *OutlineModel[T]) SelectIDs(ids ...string) {
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
func (m *OutlineModel[T]) AddSelectedID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	m.mu.Lock()
	reachable := m.reachableIDsLocked()
	if _, ok := reachable[id]; !ok {
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

// ToggleSelectedID toggles membership for id in the multi-selection set.
func (m *OutlineModel[T]) ToggleSelectedID(id string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	reachable := m.reachableIDsLocked()
	if _, ok := reachable[id]; !ok {
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

// SelectVisibleRangeToID selects the inclusive visible range from the current
// anchor to id, expanding ancestors first when needed.
func (m *OutlineModel[T]) SelectVisibleRangeToID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	m.mu.Lock()
	path, ok := m.pathToIDLocked(id, m.roots, nil)
	if !ok {
		m.mu.Unlock()
		return false
	}
	for _, ancestor := range path[:len(path)-1] {
		state := m.expanded[ancestor]
		if state == nil {
			state = newBoolStateIfReady(false)
			m.expanded[ancestor] = state
		}
		updateBoolState(state, true)
	}
	all := m.visibleIDsLocked()
	target := -1
	for i, visibleID := range all {
		if visibleID == id {
			target = i
			break
		}
	}
	if target < 0 {
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
	m.selectionAnchorID = anchorID
	m.selectIDsLocked(ids)
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// SelectRow selects row.
func (m *OutlineModel[T]) SelectRow(row T) {
	if m == nil {
		return
	}
	m.SelectID(m.id(row))
}

// RowByID reports the outline row for id.
func (m *OutlineModel[T]) RowByID(id string) (T, bool) {
	return m.rowByID(id)
}

// RevealRow expands ancestors for row, selects it, and reports whether it exists.
func (m *OutlineModel[T]) RevealRow(row T) bool {
	if m == nil {
		return false
	}
	return m.RevealID(m.id(row))
}

// ClearSelection removes the current outline selection.
func (m *OutlineModel[T]) ClearSelection() {
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

// SelectAll selects every reachable row in current tree order.
func (m *OutlineModel[T]) SelectAll() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	ids := m.allIDsLocked()
	if len(ids) == 0 {
		m.mu.Unlock()
		return false
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
func (m *OutlineModel[T]) ActivateRow(row T) {
	m.activate(row)
}

// ActivateSelected invokes the activation callback for the primary selected row.
func (m *OutlineModel[T]) ActivateSelected() bool {
	id, ok := m.SelectedID()
	if !ok {
		return false
	}
	row, ok := m.rowByID(id)
	if !ok {
		return false
	}
	m.activate(row)
	return true
}

// SelectedID reports the selected outline row identifier.
func (m *OutlineModel[T]) SelectedID() (string, bool) {
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

// SelectedIDs reports the selected outline row identifiers in tree order.
func (m *OutlineModel[T]) SelectedIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.selectedIDsLocked()...)
}

// SelectedRows reports the selected outline rows in tree order.
func (m *OutlineModel[T]) SelectedRows() []T {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.selectedIDs) == 0 {
		return nil
	}
	rows := make([]T, 0, len(m.selectedIDs))
	var visit func([]T)
	visit = func(nodes []T) {
		for _, row := range nodes {
			id := m.id(row)
			if _, ok := m.selectedIDs[id]; ok {
				rows = append(rows, row)
			}
			if m.children == nil {
				continue
			}
			children := m.children(row)
			if len(children) > 0 {
				visit(children)
			}
		}
	}
	visit(m.roots)
	return rows
}

// SelectedRow reports the primary selected outline row.
func (m *OutlineModel[T]) SelectedRow() (T, bool) {
	id, ok := m.SelectedID()
	if !ok {
		var zero T
		return zero, false
	}
	return m.rowByID(id)
}

// SelectedCount reports the number of selected outline rows.
func (m *OutlineModel[T]) SelectedCount() int {
	return len(m.SelectedIDs())
}

// SelectionAnchorID reports the current visible-range selection anchor.
func (m *OutlineModel[T]) SelectionAnchorID() (string, bool) {
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

// HasSelectedID reports whether id is currently selected.
func (m *OutlineModel[T]) HasSelectedID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.selectedIDs[id]
	return ok
}

// IsExpanded reports whether the row is currently expanded.
func (m *OutlineModel[T]) IsExpanded(row T) bool {
	state := m.ExpandedState(row)
	return state != nil && state.Get()
}

// ExpandedIDs reports the currently expanded reachable row IDs.
func (m *OutlineModel[T]) ExpandedIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	reachable := m.reachableIDsLocked()
	ids := make([]string, 0, len(m.expanded))
	for id, state := range m.expanded {
		if state == nil || !state.Get() || !reachable[id] {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// RevealID expands ancestors for id, selects it, and reports whether it exists.
func (m *OutlineModel[T]) RevealID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	m.mu.Lock()
	path, ok := m.pathToIDLocked(id, m.roots, nil)
	if !ok {
		m.mu.Unlock()
		return false
	}
	for _, ancestor := range path[:len(path)-1] {
		state := m.expanded[ancestor]
		if state == nil {
			state = newBoolStateIfReady(false)
			m.expanded[ancestor] = state
		}
		updateBoolState(state, true)
	}
	m.selectIDsLocked([]string{id})
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// SelectNextVisible advances the primary selection in current tree order.
func (m *OutlineModel[T]) SelectNextVisible() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	ids := m.selectedIDsLocked()
	all := m.visibleIDsLocked()
	if len(all) == 0 {
		m.mu.Unlock()
		return false
	}
	current := -1
	if len(ids) > 0 {
		for i, id := range all {
			if id == ids[0] {
				current = i
				break
			}
		}
	}
	next := current + 1
	if next < 0 || next >= len(all) {
		m.mu.Unlock()
		return false
	}
	m.selectIDsLocked([]string{all[next]})
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// SelectPreviousVisible moves the primary selection backward in current tree order.
func (m *OutlineModel[T]) SelectPreviousVisible() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	ids := m.selectedIDsLocked()
	all := m.visibleIDsLocked()
	if len(all) == 0 {
		m.mu.Unlock()
		return false
	}
	current := len(all)
	if len(ids) > 0 {
		for i, id := range all {
			if id == ids[0] {
				current = i
				break
			}
		}
	}
	prev := current - 1
	if prev < 0 || prev >= len(all) {
		m.mu.Unlock()
		return false
	}
	m.selectIDsLocked([]string{all[prev]})
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// ExtendSelectionNextVisible extends the current visible selection from the
// existing anchor to the next visible row.
func (m *OutlineModel[T]) ExtendSelectionNextVisible() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	all := m.visibleIDsLocked()
	if len(all) == 0 {
		m.mu.Unlock()
		return false
	}
	current := -1
	if m.hasSelection {
		for i, id := range all {
			if id == m.selectedID {
				current = i
				break
			}
		}
	}
	next := current + 1
	if next < 0 || next >= len(all) {
		m.mu.Unlock()
		return false
	}
	if !m.extendSelectionToVisibleIndexLocked(next, all) {
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

// ExtendSelectionPreviousVisible extends the current visible selection from
// the existing anchor to the previous visible row.
func (m *OutlineModel[T]) ExtendSelectionPreviousVisible() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	all := m.visibleIDsLocked()
	if len(all) == 0 {
		m.mu.Unlock()
		return false
	}
	current := len(all)
	if m.hasSelection {
		for i, id := range all {
			if id == m.selectedID {
				current = i
				break
			}
		}
	}
	prev := current - 1
	if prev < 0 || prev >= len(all) {
		m.mu.Unlock()
		return false
	}
	if !m.extendSelectionToVisibleIndexLocked(prev, all) {
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

// Revision reports the current outline mutation counter.
func (m *OutlineModel[T]) Revision() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.revision
}

// RevisionState exposes the owned revision IntState when the bridge is available.
func (m *OutlineModel[T]) RevisionState() *IntState {
	if m == nil {
		return nil
	}
	return m.revisionState
}

// Release releases state owned by the outline model.
func (m *OutlineModel[T]) Release() {
	if m == nil {
		return
	}
	m.mu.Lock()
	for id, state := range m.expanded {
		if state != nil {
			state.Release()
		}
		delete(m.expanded, id)
	}
	revisionState := m.revisionState
	m.revisionState = nil
	m.roots = nil
	m.selectedIDs = nil
	m.selectedID = ""
	m.selectionAnchorID = ""
	m.hasSelection = false
	m.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
}

// OutlineGroupModel renders a typed outline model with BoolState-backed
// disclosure state for each branch row.
func OutlineGroupModel[T any](model *OutlineModel[T], label func(T) View) View {
	if model == nil || label == nil {
		return EmptyView()
	}
	build := func() View {
		return OutlineGroup(outlineModelNodes(model, model.Roots(), label)...)
	}
	if model.revisionState == nil {
		return build()
	}
	return DynamicView(model.revisionState, func(int) View {
		return build()
	})
}

// SelectableOutlineGroupModel renders an outline whose rows update model selection.
func SelectableOutlineGroupModel[T any](model *OutlineModel[T], label func(T) View) View {
	if model == nil || label == nil {
		return EmptyView()
	}
	return OutlineGroupModel(model, func(row T) View {
		id := model.id(row)
		rowView := ButtonView(label(row), func() {
			model.SelectID(id)
			model.activate(row)
		}).ButtonStyle(ButtonStylePlain)
		if model.HasSelectedID(id) {
			return rowView.BackgroundRoundedRect(0.23, 0.47, 0.95, 0.16, 8)
		}
		return rowView
	})
}

func outlineModelNodes[T any](model *OutlineModel[T], rows []T, label func(T) View) []OutlineNode {
	nodes := make([]OutlineNode, 0, len(rows))
	for _, row := range rows {
		id := model.id(row)
		children := []T(nil)
		if model.children != nil {
			children = model.children(row)
		}
		expanded := false
		if len(children) > 0 {
			expanded = model.IsExpanded(row)
		}
		summary := rowStateSummary(model.HasSelectedID(id), expanded, len(children) > 0)
		rowLabel := label(row)
		if summary != "" {
			rowLabel = HStackSpaced(8, rowLabel, rowStateBadge(summary)).MaxFrame(-1, 0)
		}
		rowLabel = rowLabel.AccessibilityIdentifier(outlineRowAccessibilityID(id))
		if summary != "" {
			rowLabel = rowLabel.AccessibilityHint(summary).AccessibilityValue(summary)
		}
		node := OutlineNode{
			Label:    rowLabel,
			Children: outlineModelNodes(model, children, label),
		}
		if len(children) > 0 {
			node.Expanded = model.ExpandedState(row)
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func (m *OutlineModel[T]) bumpRevision() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.revision++
	revisionState := m.revisionState
	revision := m.revision
	m.mu.Unlock()
	updateIntState(revisionState, revision)
}

func (m *OutlineModel[T]) reachableIDsLocked() map[string]bool {
	reachable := make(map[string]bool)
	var visit func([]T)
	visit = func(rows []T) {
		for _, row := range rows {
			id := m.id(row)
			reachable[id] = true
			if m.children == nil {
				continue
			}
			children := m.children(row)
			if len(children) > 0 {
				visit(children)
			}
		}
	}
	visit(m.roots)
	return reachable
}

func (m *OutlineModel[T]) pathToIDLocked(id string, rows []T, path []string) ([]string, bool) {
	for _, row := range rows {
		rowID := m.id(row)
		next := append(append([]string(nil), path...), rowID)
		if rowID == id {
			return next, true
		}
		if m.children == nil {
			continue
		}
		children := m.children(row)
		if len(children) == 0 {
			continue
		}
		if found, ok := m.pathToIDLocked(id, children, next); ok {
			return found, true
		}
	}
	return nil, false
}

func (m *OutlineModel[T]) rowByID(id string) (T, bool) {
	var zero T
	if m == nil || id == "" {
		return zero, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rowByIDLocked(id, m.roots)
}

func (m *OutlineModel[T]) rowByIDLocked(id string, rows []T) (T, bool) {
	var zero T
	for _, row := range rows {
		rowID := m.id(row)
		if rowID == id {
			return row, true
		}
		if m.children == nil {
			continue
		}
		children := m.children(row)
		if len(children) == 0 {
			continue
		}
		if found, ok := m.rowByIDLocked(id, children); ok {
			return found, true
		}
	}
	return zero, false
}

func (m *OutlineModel[T]) activate(row T) {
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

func (m *OutlineModel[T]) selectIDsLocked(ids []string) {
	if m.selectedIDs == nil {
		m.selectedIDs = make(map[string]struct{})
	} else {
		for id := range m.selectedIDs {
			delete(m.selectedIDs, id)
		}
	}
	reachable := m.reachableIDsLocked()
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := reachable[id]; !ok {
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

func (m *OutlineModel[T]) selectedIDsLocked() []string {
	if len(m.selectedIDs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m.selectedIDs))
	var visit func([]T)
	visit = func(rows []T) {
		for _, row := range rows {
			id := m.id(row)
			if _, ok := m.selectedIDs[id]; ok {
				ids = append(ids, id)
			}
			if m.children == nil {
				continue
			}
			children := m.children(row)
			if len(children) > 0 {
				visit(children)
			}
		}
	}
	visit(m.roots)
	return ids
}

func (m *OutlineModel[T]) allIDsLocked() []string {
	ids := make([]string, 0)
	var visit func([]T)
	visit = func(rows []T) {
		for _, row := range rows {
			ids = append(ids, m.id(row))
			if m.children == nil {
				continue
			}
			children := m.children(row)
			if len(children) > 0 {
				visit(children)
			}
		}
	}
	visit(m.roots)
	return ids
}

func (m *OutlineModel[T]) visibleIDsLocked() []string {
	ids := make([]string, 0)
	var visit func([]T)
	visit = func(rows []T) {
		for _, row := range rows {
			id := m.id(row)
			ids = append(ids, id)
			if m.children == nil {
				continue
			}
			children := m.children(row)
			if len(children) == 0 {
				continue
			}
			state := m.expanded[id]
			if state == nil || !state.Get() {
				continue
			}
			visit(children)
		}
	}
	visit(m.roots)
	return ids
}

func (m *OutlineModel[T]) extendSelectionToVisibleIndexLocked(target int, all []string) bool {
	if target < 0 || target >= len(all) {
		return false
	}
	anchorID := m.selectionAnchorID
	if anchorID == "" {
		if m.hasSelection {
			anchorID = m.selectedID
		} else {
			anchorID = all[target]
		}
	}
	anchor := -1
	for i, id := range all {
		if id == anchorID {
			anchor = i
			break
		}
	}
	if anchor < 0 {
		anchor = target
		anchorID = all[target]
	}
	start, end := anchor, target
	if start > end {
		start, end = end, start
	}
	ids := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		ids = append(ids, all[i])
	}
	m.selectionAnchorID = anchorID
	m.selectIDsLocked(ids)
	return true
}

func (m *OutlineModel[T]) syncSelectionLocked() {
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
