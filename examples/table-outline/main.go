//go:build darwin
// +build darwin

// Command table-outline demonstrates model-backed table and outline helpers.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

type service struct {
	ID     string
	Name   string
	Owner  string
	Status string
	Load   int
}

type fileNode struct {
	ID       string
	Name     string
	Kind     string
	Children []fileNode
}

var outlineRoots = projectTree()

func main() {
	services := []service{
		{ID: "api", Name: "API Gateway", Owner: "Platform", Status: "Healthy", Load: 42},
		{ID: "search", Name: "Search Index", Owner: "Discovery", Status: "Busy", Load: 81},
		{ID: "billing", Name: "Billing", Owner: "Commerce", Status: "Degraded", Load: 67},
		{ID: "worker", Name: "Batch Worker", Owner: "Data", Status: "Healthy", Load: 29},
		{ID: "cache", Name: "Cache Layer", Owner: "Platform", Status: "Healthy", Load: 18},
		{ID: "ingest", Name: "Event Ingest", Owner: "Pipeline", Status: "Busy", Load: 74},
		{ID: "notify", Name: "Notification Fanout", Owner: "Messaging", Status: "Healthy", Load: 33},
	}
	table := swiftui.NewTableModel(services, func(s service) string { return s.ID })
	defer table.Release()
	columns := swiftui.NewTableColumnVisibilityState()
	defer columns.Release()
	widths := swiftui.NewTableColumnWidthState(map[string]float64{
		"service": 220,
		"owner":   160,
		"status":  120,
		"load":    96,
	})
	defer widths.Release()
	presets := swiftui.NewTableColumnPresetState(
		swiftui.TableColumnPreset{ID: "all", Label: "All"},
		swiftui.TableColumnPreset{ID: "compact", Label: "Compact", HiddenIDs: []string{"owner", "load"}},
		swiftui.TableColumnPreset{ID: "ops", Label: "Ops", HiddenIDs: []string{"owner"}},
	)
	defer presets.Release()
	var savedLayout swiftui.TableColumnLayoutSnapshot
	savedLayoutReady := false
	layoutStatus := swiftui.NewStringState("No saved column snapshot yet.")
	defer layoutStatus.Release()
	activatedService := swiftui.NewStringState("No row activated yet.")
	defer activatedService.Release()
	table.SetOnActivate(func(row service) {
		activatedService.Set("Activated service: " + row.Name)
	})

	outline := swiftui.NewOutlineModel(outlineRoots,
		func(n fileNode) string { return n.ID },
		func(n fileNode) []fileNode { return n.Children },
	)
	defer outline.Release()
	activatedNode := swiftui.NewStringState("No outline row activated yet.")
	defer activatedNode.Release()
	outline.SetOnActivate(func(row fileNode) {
		activatedNode.Set("Activated node: " + row.Name)
	})
	for _, root := range outline.Roots() {
		outline.SetExpanded(root, true)
	}

	swiftui.Run(swiftui.AppConfig{
		Title:  "Table + Outline",
		Width:  980,
		Height: 640,
	}, swiftui.ZStack(
		swiftui.MeshGradient4(
			swiftui.RGB(0.08, 0.12, 0.18),
			swiftui.RGB(0.10, 0.22, 0.30),
			swiftui.RGB(0.03, 0.08, 0.12),
			swiftui.RGB(0.18, 0.12, 0.08),
		),
		swiftui.VStackSpaced(14,
			header(),
			swiftui.HStackSpaced(14,
				tablePanel(table, columns, widths, presets, activatedService, layoutStatus, &savedLayout, &savedLayoutReady).MaxFrame(-1, 0),
				outlinePanel(outline, activatedNode).MaxFrame(-1, 0),
			),
		).Padding(18),
	))
}

func header() swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text("Model-backed table and outline").
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold).
				ForegroundStyle(0.96, 0.97, 0.94, 1),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text("Curated TableModelView and OutlineModel now cover sorting, column presets, explicit width state, additive selection, anchor-aware range extension, activation, reveal, and visible-range selection. Native Table/OutlineGroup parity is still a separate backlog.").
				Font(swiftui.FontCallout).
				ForegroundStyle(0.76, 0.80, 0.78, 1),
			swiftui.Spacer(),
		),
	)
}

func tablePanel(model *swiftui.TableModel[service], columns *swiftui.TableColumnVisibilityState, widths *swiftui.TableColumnWidthState, presets *swiftui.TableColumnPresetState, activated, layoutStatus *swiftui.StringState, savedLayout *swiftui.TableColumnLayoutSnapshot, savedLayoutReady *bool) swiftui.View {
	return panel("Services", "curated-table-panel",
		swiftui.DynamicView(model.RevisionState(), func(_ int) swiftui.View {
			return swiftui.DynamicView(columns.RevisionState(), func(_ int) swiftui.View {
				return swiftui.DynamicView(widths.RevisionState(), func(_ int) swiftui.View {
					return swiftui.DynamicView(presets.RevisionState(), func(_ int) swiftui.View {
						selectedIDs := model.SelectedIDs()
						selectedRows := model.SelectedRows()
						selectedSummary := "none"
						if len(selectedIDs) > 0 {
							selectedSummary = fmt.Sprintf("%v", selectedIDs)
						}
						sortID, ascending, sorted := model.SortColumn()
						sortText := "unsorted"
						if sorted {
							dir := "ascending"
							if !ascending {
								dir = "descending"
							}
							sortText = fmt.Sprintf("%s %s", sortID, dir)
						}
						currentPreset := presets.CurrentPresetID()
						if currentPreset == "" {
							currentPreset = "custom"
						}
						widthSummary := widths.Widths()
						anchorID, anchorOK := model.SelectionAnchorID()
						anchorSummary := "none"
						if anchorOK {
							anchorSummary = anchorID
						}
						return swiftui.VStackSpaced(10,
							swiftui.Text(fmt.Sprintf("%d rows, stable row IDs, model-backed activation, additive selection, anchor-aware range extension, select-all/clear ergonomics, persistent column presets, and explicit width state.", model.RowCount())).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.76, 0.80, 0.78, 1).
								AccessibilityIdentifier("curated-table-summary"),
							swiftui.HStackSpaced(8,
								swiftui.ButtonWithLabel("Prev", "arrow.up", func() {
									model.SelectPrevious()
								}).AccessibilityIdentifier("curated-table-prev").ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Next", "arrow.down", func() {
									model.SelectNext()
								}).AccessibilityIdentifier("curated-table-next").ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Extend prev", "arrow.up.to.line", func() {
									model.ExtendSelectionPrevious()
								}).AccessibilityIdentifier("curated-table-extend-prev").ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Extend next", "arrow.down.to.line", func() {
									model.ExtendSelectionNext()
								}).AccessibilityIdentifier("curated-table-extend-next").ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Add cache", "plus.circle", func() {
									model.AddSelectedID("cache")
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Range search..notify", "rectangle.expand.vertical", func() {
									model.SelectID("search")
									model.SelectRangeToID("notify")
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Reveal billing", "scope", func() {
									model.RevealID("billing")
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Sort load desc", "arrow.down", func() {
									model.SetSortColumn("load", false, func(a, b service) bool { return a.Load < b.Load })
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Reset sort", "line.3.horizontal.decrease.circle", func() {
									model.ClearSort()
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Activate primary", "checkmark.circle", func() {
									model.ActivateSelected()
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Select all", "checkmark.circle.fill", func() {
									model.SelectAll()
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.ButtonWithLabel("Clear", "xmark.circle", func() {
									model.ClearSelection()
								}).ButtonStyle(swiftui.ButtonStyleBorderless),
								swiftui.Spacer(),
							),
							swiftui.HStackSpaced(8,
								swiftui.Button("Preset: All", func() {
									presets.ApplyPreset("all", columns)
								}).ButtonStyle(swiftui.ButtonStyleBordered),
								swiftui.Button("Preset: Compact", func() {
									presets.ApplyPreset("compact", columns)
								}).ButtonStyle(swiftui.ButtonStyleBordered),
								swiftui.Button("Preset: Ops", func() {
									presets.ApplyPreset("ops", columns)
								}).ButtonStyle(swiftui.ButtonStyleBordered),
								swiftui.Button("Capture Current", func() {
									presets.SavePreset("custom", "Custom", columns)
								}).ButtonStyle(swiftui.ButtonStyleBordered),
								swiftui.Button("Save Layout Snapshot", func() {
									*savedLayout = swiftui.CaptureTableColumnLayoutSnapshot(columns, widths, presets)
									*savedLayoutReady = true
									layoutStatus.Set(layoutSnapshotSummary(*savedLayout))
								}).ButtonStyle(swiftui.ButtonStyleBordered),
								swiftui.Button("Restore Snapshot", func() {
									if !*savedLayoutReady {
										layoutStatus.Set("No saved column snapshot yet.")
										return
									}
									swiftui.ApplyTableColumnLayoutSnapshot(*savedLayout, columns, widths, presets)
									layoutStatus.Set("Restored " + layoutSnapshotSummary(*savedLayout))
								}).ButtonStyle(swiftui.ButtonStyleBordered),
								swiftui.Spacer(),
							),
							tableWidthChooser(widths),
							tableColumnChooser(columns),
							swiftui.TableModelViewWithLayout(model, columns, widths, tableColumns()...).AccessibilityIdentifier("curated-table-surface"),
							tableSelectionDetail(selectedRows),
							swiftui.Text(fmt.Sprintf("Hidden columns: %v", columns.HiddenIDs())).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.76, 0.80, 0.78, 1).
								AsView(),
							swiftui.Text(fmt.Sprintf("Visible columns: %v", columns.VisibleIDs(tableColumnIDs()))).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.76, 0.80, 0.78, 1).
								AsView(),
							swiftui.Text(fmt.Sprintf("Current preset: %s · known presets: %v", currentPreset, presets.PresetIDs())).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.76, 0.80, 0.78, 1).
								AsView(),
							swiftui.TextFromString(layoutStatus).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.80, 0.84, 0.82, 1).
								AccessibilityIdentifier("curated-table-layout-status"),
							swiftui.Text(fmt.Sprintf("Column widths: %v", widthSummary)).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.76, 0.80, 0.78, 1).
								AsView(),
							swiftui.Text(fmt.Sprintf("Primary selection follows row order. Anchor: %s · selected IDs: %s · count: %d · sort: %s", anchorSummary, selectedSummary, model.SelectedCount(), sortText)).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.72, 0.92, 1.0, 1).
								AccessibilityIdentifier("curated-table-state"),
							swiftui.TextFromString(activated).
								Font(swiftui.FontCaption).
								ForegroundStyle(0.80, 0.84, 0.82, 1).
								AccessibilityIdentifier("curated-table-activation"),
						)
					})
				})
			})
		}),
	)
}

func layoutSnapshotSummary(snapshot swiftui.TableColumnLayoutSnapshot) string {
	currentPreset := snapshot.CurrentPresetID
	if currentPreset == "" {
		currentPreset = "custom"
	}
	return fmt.Sprintf(
		"Saved column snapshot: preset %s · hidden %v · widths %v",
		currentPreset,
		snapshot.HiddenIDs,
		snapshot.Widths,
	)
}

func outlinePanel(model *swiftui.OutlineModel[fileNode], activated *swiftui.StringState) swiftui.View {
	return panel("Project Outline", "curated-outline-panel",
		swiftui.VStackSpaced(10,
			swiftui.Text(fmt.Sprintf("Curated outline disclosure with stable IDs, ancestor reveal, activation, and select-all/clear ergonomics. %d roots are currently modeled.", len(model.Roots()))).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.76, 0.80, 0.78, 1),
			swiftui.HStackSpaced(8,
				swiftui.ButtonWithLabel("Prev visible", "arrow.up", func() {
					model.SelectPreviousVisible()
				}).AccessibilityIdentifier("curated-outline-prev").ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Next visible", "arrow.down", func() {
					model.SelectNextVisible()
				}).AccessibilityIdentifier("curated-outline-next").ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Extend prev", "arrow.up.to.line", func() {
					model.ExtendSelectionPreviousVisible()
				}).AccessibilityIdentifier("curated-outline-extend-prev").ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Extend next", "arrow.down.to.line", func() {
					model.ExtendSelectionNextVisible()
				}).AccessibilityIdentifier("curated-outline-extend-next").ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Expand all", "arrow.down.right.and.arrow.up.left", func() {
					model.SetExpandedAll(true)
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Collapse all", "arrow.up.left.and.arrow.down.right", func() {
					model.SetExpandedAll(false)
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Reveal table.go", "scope", func() {
					model.RevealID("app/views/table")
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Reveal bridge runtime", "scope", func() {
					model.RevealID("bridge/runtime")
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Add bridge runtime", "plus.circle", func() {
					model.AddSelectedID("bridge/runtime")
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Range app..table.go", "rectangle.expand.vertical", func() {
					model.SelectID("app")
					model.SelectVisibleRangeToID("app/views/table")
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Activate selected", "checkmark.circle", func() {
					model.ActivateSelected()
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Select all", "checkmark.circle.fill", func() {
					model.SelectAll()
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithLabel("Clear", "xmark.circle", func() {
					model.ClearSelection()
				}).ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.Spacer(),
			),
			swiftui.ScrollView(
				swiftui.SelectableOutlineGroupModel(model, func(n fileNode) swiftui.View {
					return swiftui.HStackSpaced(8,
						swiftui.Image(iconFor(n)).
							ForegroundStyle(0.72, 0.86, 1.0, 1),
						swiftui.Text(n.Name).
							Font(swiftui.FontCallout).
							ForegroundStyle(0.94, 0.96, 0.94, 1),
						swiftui.Spacer(),
						swiftui.Text(n.Kind).
							Font(swiftui.FontCaption).
							ForegroundStyle(0.68, 0.74, 0.72, 1),
					).PaddingEdge(swiftui.EdgeVertical, 4)
				}).AccessibilityIdentifier("curated-outline-surface"),
			),
			swiftui.DynamicView(model.RevisionState(), func(int) swiftui.View {
				expanded := model.ExpandedIDs()
				selectedIDs := model.SelectedIDs()
				selectedRows := model.SelectedRows()
				anchorID, anchorOK := model.SelectionAnchorID()
				anchorSummary := "none"
				if anchorOK {
					anchorSummary = anchorID
				}
				row, ok := model.SelectedRow()
				summary := swiftui.Text(fmt.Sprintf("Expanded groups: %d · anchor: %s · selected ids: %v · count: %d · reveal expands ancestors and keeps disclosure state explicit.", len(expanded), anchorSummary, selectedIDs, model.SelectedCount())).
					Font(swiftui.FontCaption).
					ForegroundStyle(0.72, 0.76, 0.75, 1).
					AccessibilityIdentifier("curated-outline-summary")
				if !ok {
					return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 6,
						summary,
						outlineSelectionDetail(selectedRows),
					)
				}
				if ok {
					summary = swiftui.Text(fmt.Sprintf("Primary: %s (%s) · anchor: %s · selected ids: %v · %d expanded branches", row.Name, row.Kind, anchorSummary, selectedIDs, len(expanded))).
						Font(swiftui.FontCaption).
						ForegroundStyle(0.72, 0.92, 1.0, 1).
						AccessibilityIdentifier("curated-outline-summary")
				} else {
					id, _ := model.SelectedID()
					summary = swiftui.Text(fmt.Sprintf("Primary: %s · anchor: %s · selected ids: %v · %d expanded branches", id, anchorSummary, selectedIDs, len(expanded))).
						Font(swiftui.FontCaption).
						ForegroundStyle(0.72, 0.92, 1.0, 1).
						AccessibilityIdentifier("curated-outline-summary")
				}
				return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 6,
					summary,
					outlineSelectionDetail(selectedRows),
				)
			}),
			swiftui.TextFromString(activated).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.80, 0.84, 0.82, 1).
				AccessibilityIdentifier("curated-outline-activation"),
		),
	)
}

func tableColumns() []swiftui.TableModelColumn[service] {
	return []swiftui.TableModelColumn[service]{
		swiftui.TextTableModelColumn("Service",
			func(s service) string { return s.Name },
			func(a, b service) bool { return a.Name < b.Name },
		).WithID("service"),
		swiftui.TextTableModelColumn("Owner",
			func(s service) string { return s.Owner },
			func(a, b service) bool { return a.Owner < b.Owner },
		).WithID("owner"),
		swiftui.TextTableModelColumn("Status",
			func(s service) string { return s.Status },
			func(a, b service) bool { return a.Status < b.Status },
		).WithID("status"),
		swiftui.TextTableModelColumn("Load",
			func(s service) string { return fmt.Sprintf("%d%%", s.Load) },
			func(a, b service) bool { return a.Load < b.Load },
		).WithID("load").WithMaxWidth(96),
	}
}

func tableColumnIDs() []string {
	return []string{"service", "owner", "status", "load"}
}

func tableColumnChooser(columns *swiftui.TableColumnVisibilityState) swiftui.View {
	return swiftui.HStackSpaced(8,
		columnChip(columns, "service", "Service"),
		columnChip(columns, "owner", "Owner"),
		columnChip(columns, "status", "Status"),
		columnChip(columns, "load", "Load"),
		swiftui.Spacer(),
	)
}

func tableWidthChooser(widths *swiftui.TableColumnWidthState) swiftui.View {
	return swiftui.HStackSpaced(8,
		swiftui.Button("Readable widths", func() {
			widths.ReplaceWidths(map[string]float64{
				"service": 240,
				"owner":   176,
				"status":  128,
				"load":    96,
			})
		}).ButtonStyle(swiftui.ButtonStyleBordered),
		swiftui.Button("Compact widths", func() {
			widths.ReplaceWidths(map[string]float64{
				"service": 188,
				"owner":   132,
				"status":  108,
				"load":    84,
			})
		}).ButtonStyle(swiftui.ButtonStyleBordered),
		swiftui.Button("Ops widths", func() {
			widths.ReplaceWidths(map[string]float64{
				"service": 204,
				"owner":   144,
				"status":  148,
				"load":    88,
			})
		}).ButtonStyle(swiftui.ButtonStyleBordered),
		swiftui.Button("Reset widths", func() {
			widths.ReplaceWidths(map[string]float64{
				"service": 220,
				"owner":   160,
				"status":  120,
				"load":    96,
			})
		}).ButtonStyle(swiftui.ButtonStyleBordered),
		swiftui.Spacer(),
	)
}

func columnChip(columns *swiftui.TableColumnVisibilityState, id, label string) swiftui.View {
	text := label
	if !columns.Visible(id) {
		text += " hidden"
	}
	return swiftui.Button(text, func() {
		columns.Toggle(id)
	}).ButtonStyle(swiftui.ButtonStyleBordered)
}

func tableSelectionDetail(rows []service) swiftui.View {
	if len(rows) == 0 {
		return swiftui.Text("Select one or more services to inspect owner, status, and load in a model-driven detail row.").
			Font(swiftui.FontCaption).
			ForegroundStyle(0.72, 0.76, 0.75, 1).
			AccessibilityIdentifier("curated-table-selection-detail")
	}
	parts := make([]swiftui.Viewable, 0, len(rows)+1)
	parts = append(parts,
		swiftui.Text("Selection detail").
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			ForegroundStyle(0.94, 0.96, 0.94, 1).
			AsView(),
	)
	for _, row := range rows {
		parts = append(parts,
			swiftui.Text(fmt.Sprintf("%s · %s · %s · %d%%", row.Name, row.Owner, row.Status, row.Load)).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.76, 0.80, 0.78, 1).
				AsView(),
		)
	}
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4, parts...).
		AccessibilityIdentifier("curated-table-selection-detail")
}

func outlineSelectionDetail(rows []fileNode) swiftui.View {
	if len(rows) == 0 {
		return swiftui.Text("Reveal and select nodes to inspect the current file-focused selection set.").
			Font(swiftui.FontCaption).
			ForegroundStyle(0.72, 0.76, 0.75, 1).
			AccessibilityIdentifier("curated-outline-selection-detail")
	}
	parts := make([]swiftui.Viewable, 0, len(rows)+1)
	parts = append(parts,
		swiftui.Text("Selection detail").
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			ForegroundStyle(0.94, 0.96, 0.94, 1).
			AsView(),
	)
	for _, row := range rows {
		parts = append(parts,
			swiftui.Text(fmt.Sprintf("%s · %s", row.Name, row.Kind)).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.76, 0.80, 0.78, 1).
				AsView(),
		)
	}
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4, parts...).
		AccessibilityIdentifier("curated-outline-selection-detail")
}

func panel(title, id string, content swiftui.View) swiftui.View {
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyle(0.95, 0.96, 0.92, 1).
				AccessibilityIdentifier(id+"-title"),
			swiftui.Spacer(),
		),
		content.MaxFrame(-1, -1),
	).Padding(14).
		AccessibilityIdentifier(id).
		Background(0.06, 0.08, 0.10, 0.76).
		CornerRadius(18).
		Overlay(swiftui.RoundedRectangle(18).Stroke(1, 1, 1, 0.12, 1).AsView())
}

func iconFor(n fileNode) string {
	if len(n.Children) > 0 {
		return "folder.fill"
	}
	return "doc.text"
}

func projectTree() []fileNode {
	return []fileNode{
		{
			ID:   "app",
			Name: "App",
			Kind: "group",
			Children: []fileNode{
				{ID: "app/main", Name: "main.go", Kind: "go"},
				{ID: "app/theme", Name: "theme.go", Kind: "go"},
				{
					ID:   "app/views",
					Name: "Views",
					Kind: "group",
					Children: []fileNode{
						{ID: "app/views/table", Name: "table.go", Kind: "go"},
						{ID: "app/views/outline", Name: "outline.go", Kind: "go"},
					},
				},
			},
		},
		{
			ID:   "bridge",
			Name: "Bridge",
			Kind: "group",
			Children: []fileNode{
				{ID: "bridge/swift", Name: "bridge_views.gen.swift", Kind: "swift"},
				{ID: "bridge/runtime", Name: "table_outline_views.go", Kind: "go"},
			},
		},
	}
}
