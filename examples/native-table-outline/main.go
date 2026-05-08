//go:build darwin
// +build darwin

// Command native-table-outline demonstrates the native-backed table and outline layer.
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
	Region string
	Owner  string
	Status string
	Load   int
}

func main() {
	rows := []service{
		{ID: "edge", Name: "Edge Router", Region: "us-west", Owner: "Traffic", Status: "Healthy", Load: 38},
		{ID: "search", Name: "Search Cluster", Region: "us-east", Owner: "Discovery", Status: "Busy", Load: 82},
		{ID: "billing", Name: "Billing API", Region: "eu-west", Owner: "Commerce", Status: "Degraded", Load: 64},
		{ID: "events", Name: "Event Ingest", Region: "us-central", Owner: "Pipeline", Status: "Healthy", Load: 57},
	}
	services := make(map[string]service, len(rows))
	tableRows := make([]swiftui.NativeTableRow, 0, len(rows))
	for _, row := range rows {
		services[row.ID] = row
		tableRows = append(tableRows, swiftui.NativeTableRow{ID: row.ID})
	}

	tableSelection := swiftui.NewNativeTableSelectionState("edge")
	defer tableSelection.Release()
	table := swiftui.NewNativeTableModel(tableRows, tableSelection)
	defer table.Release()

	columnVisibility := swiftui.NewTableColumnVisibilityState()
	defer columnVisibility.Release()
	columnWidths := swiftui.NewTableColumnWidthState(map[string]float64{
		"name":   220,
		"region": 120,
		"owner":  150,
		"status": 120,
		"load":   90,
	})
	defer columnWidths.Release()
	columnPresets := swiftui.NewTableColumnPresetState(
		swiftui.TableColumnPreset{ID: "all", Label: "All"},
		swiftui.TableColumnPreset{ID: "ops", Label: "Ops", HiddenIDs: []string{"owner"}},
		swiftui.TableColumnPreset{ID: "compact", Label: "Compact", HiddenIDs: []string{"owner", "region"}},
	)
	defer columnPresets.Release()
	var savedLayout swiftui.TableColumnLayoutSnapshot
	savedLayoutReady := false
	layoutStatus := swiftui.NewStringState("No saved native column snapshot yet.")
	defer layoutStatus.Release()

	activatedTable := swiftui.NewStringState("No native table row activated yet.")
	defer activatedTable.Release()
	table.SetOnActivate(func(row swiftui.NativeTableRow) {
		svc := services[row.ID]
		activatedTable.Set("Activated native table row: " + svc.Name)
	})

	outlineDetails := map[string]string{
		"workspace":                 "workspace group",
		"workspace/services":        "services folder",
		"workspace/services/api":    "api.go",
		"workspace/services/search": "search.go",
		"workspace/runtime":         "runtime folder",
		"workspace/runtime/scene":   "scene_model.go",
		"workspace/runtime/table":   "native_table_outline.go",
	}
	outlineSelection := swiftui.NewNativeOutlineSelectionState("workspace/runtime/table")
	defer outlineSelection.Release()
	outlineExpansion := swiftui.NewNativeOutlineExpansionState("workspace", "workspace/runtime")
	defer outlineExpansion.Release()
	outline := swiftui.NewNativeOutlineModel([]swiftui.NativeOutlineNode{
		{
			ID:    "workspace",
			Label: outlineLabel("Workspace", "group"),
			Children: []swiftui.NativeOutlineNode{
				{
					ID:    "workspace/services",
					Label: outlineLabel("Services", "folder"),
					Children: []swiftui.NativeOutlineNode{
						{ID: "workspace/services/api", Label: outlineLabel("api.go", "go")},
						{ID: "workspace/services/search", Label: outlineLabel("search.go", "go")},
					},
				},
				{
					ID:    "workspace/runtime",
					Label: outlineLabel("Runtime", "folder"),
					Children: []swiftui.NativeOutlineNode{
						{ID: "workspace/runtime/scene", Label: outlineLabel("scene_model.go", "go")},
						{ID: "workspace/runtime/table", Label: outlineLabel("native_table_outline.go", "go")},
					},
				},
			},
		},
	}, outlineSelection, outlineExpansion)
	defer outline.Release()

	activatedOutline := swiftui.NewStringState("No native outline row activated yet.")
	defer activatedOutline.Release()
	outline.SetOnActivate(func(row swiftui.NativeOutlineNode) {
		activatedOutline.Set("Activated native outline row: " + row.ID)
	})

	swiftui.Run(swiftui.AppConfig{
		Title:  "Native Table + Outline",
		Width:  1080,
		Height: 700,
	}, swiftui.ZStack(
		swiftui.MeshGradient4(
			swiftui.RGB(0.06, 0.10, 0.16),
			swiftui.RGB(0.08, 0.18, 0.28),
			swiftui.RGB(0.05, 0.06, 0.10),
			swiftui.RGB(0.15, 0.10, 0.07),
		),
		swiftui.VStackSpaced(14,
			header(),
			swiftui.HStackSpaced(14,
				tablePanel(table, services, columnVisibility, columnWidths, columnPresets, activatedTable, layoutStatus, &savedLayout, &savedLayoutReady).MaxFrame(-1, 0),
				outlinePanel(outline, outlineDetails, activatedOutline).Frame(340, 0),
			),
		).Padding(18),
	))
}

func header() swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text("Native-backed table and outline").
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold).
				ForegroundStyle(0.96, 0.97, 0.94, 1),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text("This example uses NativeTableModel and NativeOutlineModel directly. The curated TableModel and OutlineModel remain the default Go-first API; this layer is lower-level and closer to dense desktop behavior without claiming raw SwiftUI Table or OutlineGroup parity.").
				Font(swiftui.FontCallout).
				ForegroundStyle(0.76, 0.80, 0.78, 1),
			swiftui.Spacer(),
		),
	)
}

func tablePanel(model *swiftui.NativeTableModel, services map[string]service, visibility *swiftui.TableColumnVisibilityState, widths *swiftui.TableColumnWidthState, presets *swiftui.TableColumnPresetState, activated, layoutStatus *swiftui.StringState, savedLayout *swiftui.TableColumnLayoutSnapshot, savedLayoutReady *bool) swiftui.View {
	return panel("Native-backed Table", "native-table-panel",
		swiftui.DynamicView(model.RevisionState(), func(int) swiftui.View {
			return swiftui.DynamicView(model.SelectionState().RevisionState(), func(int) swiftui.View {
				return swiftui.DynamicView(visibility.RevisionState(), func(int) swiftui.View {
					return swiftui.DynamicView(widths.RevisionState(), func(int) swiftui.View {
						return swiftui.DynamicView(presets.RevisionState(), func(int) swiftui.View {
							selectedRows := model.SelectedRows()
							selectedIDs := model.SelectionState().SelectedIDs()
							anchorID, anchorOK := model.SelectionAnchorID()
							anchor := "none"
							if anchorOK {
								anchor = anchorID
							}
							currentPreset := presets.CurrentPresetID()
							if currentPreset == "" {
								currentPreset = "custom"
							}
							return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 10,
								swiftui.Text("The native-backed path keeps explicit string-ID selection state and lower-level column definitions. It reuses width and preset state, adds select-all, toggle, clear ergonomics, and layout snapshot persistence, and does not pretend to be raw SwiftUI Table parity.").
									Font(swiftui.FontCaption).
									ForegroundStyle(0.76, 0.80, 0.78, 1).
									AsView(),
								swiftui.HStackSpaced(8,
									swiftui.Button("Prev", func() { model.SelectPrevious() }).AccessibilityIdentifier("native-table-prev").ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Button("Next", func() { model.SelectNext() }).AccessibilityIdentifier("native-table-next").ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Button("Range to billing", func() { model.SelectRangeToID("billing") }).ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Button("Reveal billing", func() { model.RevealID("billing") }).ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Button("Toggle billing", func() { model.ToggleSelectedID("billing") }).ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Button("Activate selected", func() { model.ActivateSelected() }).AccessibilityIdentifier("native-table-activate").ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Button("Select all", func() { model.SelectAll() }).ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Button("Clear", func() { model.ClearSelection() }).ButtonStyle(swiftui.ButtonStyleBorderless),
									swiftui.Spacer(),
								),
								swiftui.HStackSpaced(8,
									swiftui.Button("Preset: All", func() { presets.ApplyPreset("all", visibility) }).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Button("Preset: Ops", func() { presets.ApplyPreset("ops", visibility) }).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Button("Preset: Compact", func() { presets.ApplyPreset("compact", visibility) }).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Button("Capture Current", func() { presets.SavePreset("custom", "Custom", visibility) }).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Button("Save Layout Snapshot", func() {
										*savedLayout = swiftui.CaptureTableColumnLayoutSnapshot(visibility, widths, presets)
										*savedLayoutReady = true
										layoutStatus.Set(layoutSnapshotSummary(*savedLayout))
									}).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Button("Restore Snapshot", func() {
										if !*savedLayoutReady {
											layoutStatus.Set("No saved native column snapshot yet.")
											return
										}
										swiftui.ApplyTableColumnLayoutSnapshot(*savedLayout, visibility, widths, presets)
										layoutStatus.Set("Restored " + layoutSnapshotSummary(*savedLayout))
									}).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Spacer(),
								),
								swiftui.HStackSpaced(8,
									swiftui.Button("Readable widths", func() {
										widths.ReplaceWidths(map[string]float64{"name": 220, "region": 120, "owner": 150, "status": 120, "load": 90})
									}).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Button("Dense widths", func() {
										widths.ReplaceWidths(map[string]float64{"name": 180, "region": 96, "owner": 120, "status": 100, "load": 72})
									}).ButtonStyle(swiftui.ButtonStyleBordered),
									swiftui.Spacer(),
								),
								swiftui.HStackSpaced(8,
									columnChip(visibility, "name", "Name"),
									columnChip(visibility, "region", "Region"),
									columnChip(visibility, "owner", "Owner"),
									columnChip(visibility, "status", "Status"),
									columnChip(visibility, "load", "Load"),
									swiftui.Spacer(),
								),
								swiftui.ScrollView(
									swiftui.NativeTableView(model, nativeColumns(services, visibility, widths)...).AccessibilityIdentifier("native-table-surface"),
								).MaxFrame(-1, -1),
								swiftui.Text(fmt.Sprintf("Primary anchor: %s · selected ids: %v · count: %d · preset: %s", anchor, selectedIDs, model.SelectedCount(), currentPreset)).
									Font(swiftui.FontCaption).
									ForegroundStyle(0.72, 0.92, 1.0, 1).
									AsView().
									AccessibilityIdentifier("native-table-summary"),
								swiftui.Text(fmt.Sprintf("AX row state: %s", model.SelectedRowStateSummary())).
									Font(swiftui.FontCaption).
									ForegroundStyle(0.80, 0.84, 0.82, 1).
									AsView().
									AccessibilityIdentifier("native-table-ax-state"),
								swiftui.Text(fmt.Sprintf("Visible columns: %v · widths: %v", visibility.VisibleIDs(nativeColumnIDs()), widths.Widths())).
									Font(swiftui.FontCaption).
									ForegroundStyle(0.76, 0.80, 0.78, 1).
									AsView(),
								swiftui.TextFromString(layoutStatus).
									Font(swiftui.FontCaption).
									ForegroundStyle(0.80, 0.84, 0.82, 1).
									AccessibilityIdentifier("native-table-layout-status"),
								tableSelectionDetail(services, selectedRows),
								swiftui.TextFromString(activated).
									Font(swiftui.FontCaption).
									ForegroundStyle(0.80, 0.84, 0.82, 1).
									AsView().
									AccessibilityIdentifier("native-table-activation"),
							)
						})
					})
				})
			})
		}),
	)
}

func outlinePanel(model *swiftui.NativeOutlineModel, details map[string]string, activated *swiftui.StringState) swiftui.View {
	return panel("Native-backed Outline", "native-outline-panel",
		swiftui.DynamicView(model.RevisionState(), func(int) swiftui.View {
			return swiftui.DynamicView(model.SelectionState().RevisionState(), func(int) swiftui.View {
				return swiftui.DynamicView(model.ExpansionState().RevisionState(), func(int) swiftui.View {
					selectedRows := model.SelectedRows()
					selectedIDs := model.SelectionState().SelectedIDs()
					anchorID, anchorOK := model.SelectionAnchorID()
					anchor := "none"
					if anchorOK {
						anchor = anchorID
					}
					return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 10,
						swiftui.Text("The native-backed outline uses explicit expansion state and lower-level nodes. It adds select-all, toggle, and clear ergonomics without replacing the curated OutlineModel path.").
							Font(swiftui.FontCaption).
							ForegroundStyle(0.76, 0.80, 0.78, 1).
							AsView(),
						swiftui.HStackSpaced(8,
							swiftui.Button("Prev visible", func() { model.SelectPreviousVisible() }).AccessibilityIdentifier("native-outline-prev").ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Button("Next visible", func() { model.SelectNextVisible() }).AccessibilityIdentifier("native-outline-next").ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Button("Range to runtime", func() { model.SelectVisibleRangeToID("workspace/runtime/table") }).ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Button("Reveal runtime", func() { model.RevealID("workspace/runtime/table") }).ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Button("Toggle runtime", func() { model.ToggleSelectedID("workspace/runtime/table") }).ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Button("Select all", func() { model.SelectAll() }).ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Button("Clear", func() { model.ClearSelection() }).ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Spacer(),
						),
						swiftui.HStackSpaced(8,
							swiftui.Button("Expand services", func() { model.SetExpandedID("workspace/services", true) }).AccessibilityIdentifier("native-outline-expand-services").ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Button("Activate selected", func() { model.ActivateSelected() }).AccessibilityIdentifier("native-outline-activate").ButtonStyle(swiftui.ButtonStyleBorderless),
							swiftui.Spacer(),
						),
						swiftui.NativeOutlineView(model).MaxFrame(-1, -1).AccessibilityIdentifier("native-outline-surface"),
						swiftui.Text(fmt.Sprintf("Expanded ids: %v · anchor: %s · selected ids: %v · count: %d", model.ExpandedIDs(), anchor, selectedIDs, model.SelectedCount())).
							Font(swiftui.FontCaption).
							ForegroundStyle(0.72, 0.92, 1.0, 1).
							AsView().
							AccessibilityIdentifier("native-outline-summary"),
						swiftui.Text(fmt.Sprintf("AX row state: %s", model.SelectedRowStateSummary())).
							Font(swiftui.FontCaption).
							ForegroundStyle(0.80, 0.84, 0.82, 1).
							AsView().
							AccessibilityIdentifier("native-outline-ax-state"),
						outlineSelectionDetail(details, selectedRows),
						swiftui.TextFromString(activated).
							Font(swiftui.FontCaption).
							ForegroundStyle(0.80, 0.84, 0.82, 1).
							AsView().
							AccessibilityIdentifier("native-outline-activation"),
					)
				})
			})
		}),
	)
}

func nativeColumns(services map[string]service, visibility *swiftui.TableColumnVisibilityState, widths *swiftui.TableColumnWidthState) []swiftui.NativeTableColumn {
	columns := []swiftui.NativeTableColumn{
		nativeTextColumn("name", "Service", services, widths, func(s service) string { return s.Name }),
		nativeTextColumn("region", "Region", services, widths, func(s service) string { return s.Region }),
		nativeTextColumn("owner", "Owner", services, widths, func(s service) string { return s.Owner }),
		nativeTextColumn("status", "Status", services, widths, func(s service) string { return s.Status }),
		nativeTextColumn("load", "Load", services, widths, func(s service) string { return fmt.Sprintf("%d%%", s.Load) }),
	}
	if visibility == nil {
		return columns
	}
	out := make([]swiftui.NativeTableColumn, 0, len(columns))
	for _, column := range columns {
		if visibility.Visible(column.ID) {
			out = append(out, column)
		}
	}
	return out
}

func nativeTextColumn(id, label string, services map[string]service, widths *swiftui.TableColumnWidthState, value func(service) string) swiftui.NativeTableColumn {
	width, _ := widths.Width(id)
	return swiftui.NativeTableColumn{
		ID:     id,
		Header: swiftui.Text(label).Font(swiftui.FontCaption).FontWeight(swiftui.WeightSemibold).ForegroundStyle(0.88, 0.90, 0.88, 1).AsView(),
		Cell: func(row swiftui.NativeTableRow) swiftui.View {
			svc := services[row.ID]
			return swiftui.Text(value(svc)).
				Font(swiftui.FontCallout).
				ForegroundStyle(0.94, 0.96, 0.94, 1).
				AsView()
		},
		Width: width,
	}
}

func nativeColumnIDs() []string {
	return []string{"name", "region", "owner", "status", "load"}
}

func layoutSnapshotSummary(snapshot swiftui.TableColumnLayoutSnapshot) string {
	currentPreset := snapshot.CurrentPresetID
	if currentPreset == "" {
		currentPreset = "custom"
	}
	return fmt.Sprintf("Saved column snapshot: preset %s · hidden %v · widths %v", currentPreset, snapshot.HiddenIDs, snapshot.Widths)
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

func tableSelectionDetail(services map[string]service, rows []swiftui.NativeTableRow) swiftui.View {
	if len(rows) == 0 {
		return swiftui.Text("Select one or more rows to inspect the native-backed selection set.").
			Font(swiftui.FontCaption).
			ForegroundStyle(0.72, 0.76, 0.75, 1).
			AsView().
			AccessibilityIdentifier("native-table-selection-detail")
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
		svc := services[row.ID]
		parts = append(parts,
			swiftui.Text(fmt.Sprintf("%s · %s · %s · %d%%", svc.Name, svc.Region, svc.Status, svc.Load)).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.76, 0.80, 0.78, 1).
				AsView(),
		)
	}
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4, parts...).
		AccessibilityIdentifier("native-table-selection-detail")
}

func outlineSelectionDetail(details map[string]string, rows []swiftui.NativeOutlineNode) swiftui.View {
	if len(rows) == 0 {
		return swiftui.Text("Reveal and select nodes to inspect the current native-backed disclosure state.").
			Font(swiftui.FontCaption).
			ForegroundStyle(0.72, 0.76, 0.75, 1).
			AsView().
			AccessibilityIdentifier("native-outline-selection-detail")
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
			swiftui.Text(fmt.Sprintf("%s · %s", row.ID, details[row.ID])).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.76, 0.80, 0.78, 1).
				AsView(),
		)
	}
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4, parts...).
		AccessibilityIdentifier("native-outline-selection-detail")
}

func outlineLabel(name, kind string) swiftui.View {
	return swiftui.HStackSpaced(8,
		swiftui.Image(iconFor(kind)).
			ForegroundStyle(0.72, 0.86, 1.0, 1),
		swiftui.Text(name).
			Font(swiftui.FontCallout).
			ForegroundStyle(0.94, 0.96, 0.94, 1),
		swiftui.Spacer(),
		swiftui.Text(kind).
			Font(swiftui.FontCaption).
			ForegroundStyle(0.68, 0.74, 0.72, 1),
	).PaddingEdge(swiftui.EdgeVertical, 4)
}

func iconFor(kind string) string {
	switch kind {
	case "group", "folder":
		return "folder.fill"
	default:
		return "doc.text"
	}
}

func panel(title, id string, content swiftui.View) swiftui.View {
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyle(0.95, 0.96, 0.92, 1),
			swiftui.Spacer(),
		),
		content.MaxFrame(-1, -1),
	).Padding(14).
		AccessibilityIdentifier(id).
		Background(0.06, 0.08, 0.10, 0.76).
		CornerRadius(18).
		Overlay(swiftui.RoundedRectangle(18).Stroke(1, 1, 1, 0.12, 1).AsView())
}
