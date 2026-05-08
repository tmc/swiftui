//go:build darwin
// +build darwin

// Command workbench demonstrates the current SwiftUI-for-Go path for
// split-view apps, selection-driven lists, disclosure sections, staged state
// transitions, timer/status surfaces, planner-style scheduling, data grids,
// document previews, and the remaining runtime-model gaps.
//
// It uses NavigationSplitViewTripleVisibility as the shell, SelectableList for
// routing and row selection, and SectionExpanded for inspector disclosure.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/quicklook"
)

func init() { runtime.LockOSThread() }

const (
	routeShell = iota + 1
	routeTree
	routeMotion
	routeTimer
	routeRouter
	routePlanner
	routeGrid
	routeDocs
	routeGaps
)

type surface struct {
	ID       int
	Title    string
	Subtitle string
	Icon     string
}

type module struct {
	ID      int
	Title   string
	Summary string
	Icon    string
	Color   [3]float64
}

type treeFile struct {
	ID      int
	Group   string
	Name    string
	Summary string
	Body    string
}

type gridRow struct {
	ID        int
	Service   string
	SLO       string
	Latency   string
	Capacity  string
	Owner     string
	Readiness string
}

type demoDocument struct {
	ID      int
	Title   string
	Kind    string
	Path    string
	Summary string
}

type workbench struct {
	route      *swiftui.IntState
	visibility *swiftui.IntState

	inspectorSelection *swiftui.BoolState
	inspectorPath      *swiftui.BoolState
	inspectorBacklog   *swiftui.BoolState

	workspaceSelection *swiftui.IntState

	treeApp       *swiftui.BoolState
	treeBridge    *swiftui.BoolState
	treeDocs      *swiftui.BoolState
	fileSelection *swiftui.IntState

	phase *swiftui.IntState

	timer     *swiftui.TimerState
	remaining *swiftui.IntState

	routerState    *swiftui.IntState
	routerOverview *swiftui.BoolState
	routerChecks   *swiftui.BoolState
	routerPath     *swiftui.NavigationPathState

	plannerWindow    *swiftui.DateRangeState
	plannerSelection *swiftui.DateSelectionState
	plannerAnchor    *swiftui.DateState
	plannerMask      *swiftui.IntState
	plannerTick      *swiftui.IntState

	grid          *swiftui.TableModel[gridRow]
	gridSelection *swiftui.IntState

	documentSelection *swiftui.IntState
	refreshState      *swiftui.IntState
	shareDialog       *swiftui.IntState
	lastRefresh       *swiftui.StringState

	docs []demoDocument
}

var surfaces = []surface{
	{routeShell, "Workbench Shell", "NavigationSplitView*, SelectableList, and SectionExpanded in one shell.", "sidebar.left"},
	{routeTree, "File Browser", "Curated OutlineModel path: expandable sections plus explicit selection and reveal.", "folder.fill"},
	{routeMotion, "Motion States", "Current PhaseAnimator path: staged transitions with AnimatedDynamicView.", "sparkles"},
	{routeTimer, "Timer Status", "Current timerInterval path: Go-driven countdown, status, and progress.", "timer"},
	{routeRouter, "Navigation", "Legacy links, disclosure, and the current state-router pattern.", "point.topleft.down.curvedto.point.bottomright.up"},
	{routePlanner, "Planner", "Calendar-like planning and multi-date selection with today’s state model.", "calendar"},
	{routeGrid, "Data Grid", "Spreadsheet-style rows, selection, and operator detail.", "tablecells"},
	{routeDocs, "Documents", "DocumentGroupWithHandle with stable identity, Quick Look preview, and scene-plan-owned auxiliary windows.", "doc.richtext.fill"},
	{routeGaps, "Runtime Gaps", "Immersive space, full share-sheet parity, keyframes, and true SwiftUI environment scenes still need deeper runtime work.", "bolt.trianglebadge.exclamationmark"},
}

var workspaceModules = []module{
	{1, "Overview", "Root route selection and column visibility stay data-driven.", "square.grid.2x2.fill", [3]float64{0.28, 0.58, 0.96}},
	{2, "Deploys", "Selection-driven rows feed a focused detail pane without bespoke glue.", "shippingbox.fill", [3]float64{0.93, 0.56, 0.22}},
	{3, "Alerts", "Disclosure sections keep secondary status nearby but out of the way.", "bell.badge.fill", [3]float64{0.94, 0.35, 0.38}},
	{4, "Inspector", "The right column stays explicit instead of magic state hidden in closures.", "sidebar.right", [3]float64{0.33, 0.78, 0.44}},
}

var treeFiles = []treeFile{
	{101, "App", "main.go", "Shell assembly for the split-view workbench.", "Build the shell with NavigationSplitViewTripleVisibility, route selection, and an inspector column."},
	{102, "App", "router.go", "State-router helpers used before NavigationPath lands.", "Keep route IDs and detail builders explicit so the Go side stays simple and testable."},
	{103, "App", "planner.go", "Planner-oriented state and summary formatting.", "DatePicker plus a bitmask-backed day selection model covers planner flows today."},
	{201, "Bridge", "views.go", "Generated view constructors for curated SwiftUI surface area.", "The generated layer exposes SelectableList, SectionExpanded, NavigationSplitView*, DocumentGroupWithHandle, and the rest of the current catalog."},
	{202, "Bridge", "callback.go", "Callback, builder, and gesture plumbing.", "Dynamic builders and callbacks are the critical runtime boundary for today’s bridge."},
	{203, "Bridge", "state.go", "Reactive state handles shared between Go and SwiftUI.", "IntState, BoolState, FloatState, DateState, and StringState drive everything in this demo."},
	{301, "Docs", "swiftui-report.json", "Analysis backlog for promotable and runtime-model features.", "Use the report to separate catalog coverage from runtime work instead of pretending every symbol is codegen-ready."},
	{302, "Docs", "notes/runtime-gaps.md", "Share sheet, scene-host parity, refreshable, and immersive-space notes.", "These remain real runtime-model gaps, not missing demo polish."},
}

var rows = []gridRow{
	{1, "gateway", "99.95%", "184 ms", "72%", "platform", "ready"},
	{2, "router", "99.90%", "236 ms", "68%", "infra", "watch"},
	{3, "planner", "99.99%", "92 ms", "51%", "product", "ready"},
	{4, "docs", "99.80%", "411 ms", "83%", "desktop", "drill"},
	{5, "search", "99.95%", "167 ms", "64%", "discovery", "ready"},
}

func main() {
	wb := newWorkbench()
	swiftui.Run(swiftui.AppConfig{
		Title:  "Surface Workbench",
		Width:  1460,
		Height: 920,
	}, wb.root())
}

func newWorkbench() *workbench {
	now := time.Now()
	plannerMask := 0b0010111
	anchor := normalizeDay(now)
	windowStart, windowEnd := plannerWindow(anchor)
	wb := &workbench{
		route:              swiftui.NewIntState(routeShell),
		visibility:         swiftui.NewIntState(int(swiftui.NavigationSplitViewVisibilityAll)),
		inspectorSelection: swiftui.NewBoolState(true),
		inspectorPath:      swiftui.NewBoolState(true),
		inspectorBacklog:   swiftui.NewBoolState(true),
		workspaceSelection: swiftui.NewIntState(1),
		treeApp:            swiftui.NewBoolState(true),
		treeBridge:         swiftui.NewBoolState(true),
		treeDocs:           swiftui.NewBoolState(false),
		fileSelection:      swiftui.NewIntState(101),
		phase:              swiftui.NewIntState(0),
		timer:              swiftui.NewTimerState(18*time.Minute, 18*time.Minute, false),
		routerState:        swiftui.NewIntState(1),
		routerOverview:     swiftui.NewBoolState(true),
		routerChecks:       swiftui.NewBoolState(false),
		routerPath:         swiftui.NewNavigationPathStateWith("dashboard"),
		plannerWindow:      swiftui.NewDateRangeState(windowStart, windowEnd, true),
		plannerSelection:   swiftui.NewDateSelectionState(plannerSelection(anchor, plannerMask)...),
		plannerAnchor:      swiftui.NewDateState(float64(now.Unix())),
		plannerMask:        swiftui.NewIntState(plannerMask),
		plannerTick:        swiftui.NewIntState(1),
		grid:               swiftui.NewTableModel(rows, func(row gridRow) string { return strconv.Itoa(row.ID) }),
		gridSelection:      swiftui.NewIntState(1),
		documentSelection:  swiftui.NewIntState(1),
		refreshState:       swiftui.NewIntState(0),
		shareDialog:        swiftui.NewIntState(0),
		lastRefresh:        swiftui.NewStringState(now.Format("15:04:05")),
		docs:               writeDemoDocuments(),
	}
	wb.grid.SelectID("1")
	wb.remaining = wb.timer.RemainingState()

	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for range tick.C {
			wb.timer.Tick()
		}
	}()

	return wb
}

func (w *workbench) root() swiftui.View {
	return swiftui.NavigationSplitViewTripleVisibility(
		w.visibility,
		w.sidebar(),
		w.content().MaxFrame(-1, -1).LayoutPriority(1),
		w.inspector().FixedSizeAxis(true, false),
	).NavigationSplitViewStyle(swiftui.NavigationSplitViewStyleAutomatic)
}

func (w *workbench) sidebar() swiftui.View {
	rows := make([]swiftui.Viewable, 0, len(surfaces))
	for _, s := range surfaces {
		rows = append(rows, sidebarRow(s).Tag(int32(s.ID)))
	}
	return swiftui.VStackSpaced(14,
		sidebarWidthShim(),
		sidebarHeader(),
		swiftui.SelectableList(w.route, rows...).
			ListStyle(swiftui.ListStyleSidebar).
			MaxFrame(-1, -1),
		swiftui.DynamicView(w.route, func(id int) swiftui.View {
			color := routeStatusColor(id)
			return swiftui.HStackSpaced(8,
				swiftui.Circle().
					Fill(color[0], color[1], color[2], 1.0).
					Frame(8, 8).
					AsView(),
				swiftui.Text(routeStatus(id)).
					Font(swiftui.FontCaption).
					ForegroundStyle(color[0], color[1], color[2], 1.0).
					AsView(),
				swiftui.Spacer(),
			).Padding(10).
				BackgroundStyle("regularMaterial").
				CornerRadius(12)
		}),
	).Padding(14)
}

func (w *workbench) content() swiftui.View {
	return swiftui.DynamicView(w.route, func(id int) swiftui.View {
		switch id {
		case routeTree:
			return w.treePanel()
		case routeMotion:
			return w.motionPanel()
		case routeTimer:
			return w.timerPanel()
		case routeRouter:
			return w.routerPanel()
		case routePlanner:
			return w.plannerPanel()
		case routeGrid:
			return w.gridPanel()
		case routeDocs:
			return w.documentsPanel()
		case routeGaps:
			return w.gapsPanel()
		default:
			return w.shellPanel()
		}
	})
}

func (w *workbench) inspector() swiftui.View {
	return swiftui.DynamicView(w.route, func(id int) swiftui.View {
		return swiftui.VStackSpaced(0,
			inspectorWidthShim(),
			swiftui.Form(
				swiftui.SectionExpanded("Current Selection", w.inspectorSelection, w.selectionInspector(id)),
				swiftui.SectionExpanded("Current Path", w.inspectorPath, w.pathInspector(id)),
				swiftui.SectionExpanded("Next Runtime Step", w.inspectorBacklog, w.backlogInspector(id)),
			).Frame(200, 0),
		).
			Padding(14)
	})
}

func (w *workbench) shellPanel() swiftui.View {
	moduleRows := make([]swiftui.Viewable, 0, len(workspaceModules))
	for _, m := range workspaceModules {
		moduleRows = append(moduleRows, workspaceModuleRow(m).Tag(int32(m.ID)))
	}

	return scrollPanel(
		swiftui.VStackSpaced(18,
			panelHeader(
				"Workbench Shell",
				"The shell itself is the demo: NavigationSplitViewTripleVisibility at the top, SelectableList for routing, and SectionExpanded in the inspector.",
				"Exact Match",
				"sidebar.left",
			),
			swiftui.HStackSpaced(16,
				swiftui.GroupBox("Column Visibility",
					swiftui.VStackSpaced(12,
						swiftui.Text("The root shell drives column visibility with a plain IntState bound into NavigationSplitViewTripleVisibility.").
							Font(swiftui.FontCallout).
							ForegroundStyleNamed("secondary"),
						swiftui.PickerSegmented("Columns", w.visibility,
							swiftui.VStack(
								swiftui.Text("Auto").AsView().Tag(int32(swiftui.NavigationSplitViewVisibilityAutomatic)),
								swiftui.Text("All").AsView().Tag(int32(swiftui.NavigationSplitViewVisibilityAll)),
								swiftui.Text("Two").AsView().Tag(int32(swiftui.NavigationSplitViewVisibilityDoubleColumn)),
								swiftui.Text("Detail").AsView().Tag(int32(swiftui.NavigationSplitViewVisibilityDetailOnly)),
							),
							func() {},
						),
						swiftui.DynamicView(w.visibility, func(v int) swiftui.View {
							return infoCard("Current Mode", visibilityLabel(v), "sidebar.squares.right")
						}),
					).Padding(12),
				).MaxFrame(-1, 0),
				swiftui.GroupBox("Shell Guarantees",
					swiftui.VStackSpaced(10,
						featureRow("Split shell", "Three explicit columns keep navigation, content, and inspection separate."),
						featureRow("Selection", "List selection is data, not view-local hidden state."),
						featureRow("Disclosure", "Inspector sections collapse cleanly without custom plumbing."),
					).Padding(12),
				).MaxFrame(-1, 0),
			),
			swiftui.GroupBox("Selection-Driven Workspace",
				swiftui.HStackSpaced(18,
					swiftui.SelectableList(w.workspaceSelection, moduleRows...).
						ListStyle(swiftui.ListStyleInset).
						Frame(280, 280),
					swiftui.DynamicView(w.workspaceSelection, func(id int) swiftui.View {
						module := moduleByID(id)
						return swiftui.VStackSpaced(12,
							moduleHero(module),
							swiftui.HStackSpaced(12,
								infoCard("Selection", module.Title, module.Icon),
								infoCard("Path", "SelectableList", "list.bullet.rectangle.portrait"),
								infoCard("Inspector", "SectionExpanded", "rectangle.righthalf.inset.filled.arrow.right"),
							),
							swiftui.Text(module.Summary).
								Font(swiftui.FontBody).
								ForegroundStyleNamed("secondary"),
						).MaxFrame(-1, 0).Padding(12)
					}),
				).Padding(12),
			).MaxFrame(-1, 0),
			swiftui.GroupBox("Mapping To Your Review Buckets",
				swiftui.VStackSpaced(10,
					featureRow("Sidebar/detail apps", "This shell is the direct NavigationSplitView* path."),
					featureRow("Collapsible settings", "The right column and file browser use SectionExpanded."),
					featureRow("Selection-driven lists", "Both the route rail and the workspace module list are SelectableList."),
				).Padding(12),
			).MaxFrame(-1, 0),
		).Padding(24),
	)
}

func (w *workbench) treePanel() swiftui.View {
	return scrollPanel(
		swiftui.VStackSpaced(18,
			panelHeader(
				"File Browser",
				"OutlineModel now has curated typed helpers; this workbench route keeps expansion, reveal, and selection explicit and testable.",
				"Current Best Path",
				"folder.fill.badge.plus",
			),
			swiftui.HStackSpaced(18,
				swiftui.DynamicView(w.fileSelection, func(_ int) swiftui.View {
					return swiftui.Form(
						swiftui.SectionExpanded("App", w.treeApp, swiftui.VStackSpaced(6,
							w.treeFileButton(101),
							w.treeFileButton(102),
							w.treeFileButton(103),
						)),
						swiftui.SectionExpanded("Bridge", w.treeBridge, swiftui.VStackSpaced(6,
							w.treeFileButton(201),
							w.treeFileButton(202),
							w.treeFileButton(203),
						)),
						swiftui.SectionExpanded("Docs", w.treeDocs, swiftui.VStackSpaced(6,
							w.treeFileButton(301),
							w.treeFileButton(302),
						)),
					).Frame(360, 420)
				}),
				swiftui.DynamicView(w.fileSelection, func(id int) swiftui.View {
					f := fileByID(id)
					return swiftui.GroupBox("Preview",
						swiftui.VStackSpaced(14,
							infoCard("File", f.Name, iconForGroup(f.Group)),
							swiftui.HStackSpaced(12,
								infoCard("Group", f.Group, "folder"),
								infoCard("Selection", strconv.Itoa(f.ID), "number.square"),
							),
							swiftui.Text(f.Summary).
								Font(swiftui.FontCallout).
								ForegroundStyleNamed("secondary"),
							swiftui.Text(f.Body).
								Font(swiftui.FontBody).
								FontDesign(swiftui.DesignMonospaced).
								ForegroundStyleNamed("primary").
								LineLimit(0),
						).Padding(14),
					).MaxFrame(-1, 0)
				}),
			).MaxFrame(-1, 0),
			swiftui.GroupBox("Why This Exists",
				swiftui.VStackSpaced(10,
					featureRow("Tree shape", "Nested groups stay readable without inventing a generic tree ABI."),
					featureRow("Selection model", "A single IntState keeps preview and sidebar in lockstep."),
					featureRow("Reveal path", "RevealID expands ancestors and keeps deep nodes visible in the detail pane."),
				).Padding(12),
			).MaxFrame(-1, 0),
		).Padding(24),
	)
}

func (w *workbench) motionPanel() swiftui.View {
	return scrollPanel(
		swiftui.VStackSpaced(18,
			panelHeader(
				"Motion States",
				"PhaseAnimator is bridged; this route keeps the Go-side phase explicit so the current stage and transition policy stay inspectable.",
				"Current Best Path",
				"sparkles",
			),
			swiftui.GroupBox("State Transition",
				swiftui.VStackSpaced(16,
					swiftui.AnimatedDynamicView(w.phase, swiftui.TransitionPush, func(v int) swiftui.View {
						return phaseCard(v)
					}),
					swiftui.HStackSpaced(8,
						phaseButton("Discover", w.phase, 0, swiftui.AnimationEaseInOut),
						phaseButton("Plan", w.phase, 1, swiftui.AnimationEaseIn),
						phaseButton("Build", w.phase, 2, swiftui.AnimationSpring),
						phaseButton("Review", w.phase, 3, swiftui.AnimationBouncy),
						phaseButton("Ship", w.phase, 4, swiftui.AnimationEaseOut),
					),
				).Padding(14),
			).MaxFrame(-1, 0),
			swiftui.DynamicView(w.phase, func(v int) swiftui.View {
				stage := phaseSpec(v)
				return swiftui.HStackSpaced(14,
					infoCard("Current Stage", stage.Title, stage.Icon),
					infoCard("Primary Motion", stage.Motion, "waveform.path"),
					infoCard("Intent", stage.Intent, "target"),
				)
			}),
			swiftui.GroupBox("Mapping To PhaseAnimator",
				swiftui.VStackSpaced(10,
					featureRow("State machine", "The phase is explicit and easy to inspect from Go."),
					featureRow("Animation policy", "Each transition can pick its own animation curve."),
					featureRow("Bridge path", "PhaseAnimator is available; AnimatedDynamicView remains useful when the phase model needs Go-side control."),
				).Padding(12),
			).MaxFrame(-1, 0),
		).Padding(24).Frame(620, 0),
	)
}

func (w *workbench) timerPanel() swiftui.View {
	return scrollPanel(
		swiftui.DynamicView(w.timer.RevisionState(), func(_ int) swiftui.View {
			secs := int(w.timer.Remaining() / time.Second)
			return swiftui.VStackSpaced(18,
				panelHeader(
					"Timer Status",
					"TimerState owns remaining, running, and progress as one unit so the workbench stops juggling three primitives.",
					"Current Best Path",
					"timer",
				),
				swiftui.HStackSpaced(18,
					swiftui.GroupBox("Countdown",
						swiftui.VStackSpaced(14,
							swiftui.ZStack(
								swiftui.Circle().
									Fill(0.2, 0.45, 0.9, 0.12).
									Frame(220, 220).
									AsView(),
								swiftui.VStackSpaced(6,
									swiftui.Text(formatDuration(secs)).
										Font(swiftui.FontSystem(56)).
										FontWeight(swiftui.WeightBold).
										FontDesign(swiftui.DesignRounded).
										MonospacedDigit().
										AsView(),
									swiftui.Text(timerStatus(secs)).
										Font(swiftui.FontCallout).
										ForegroundStyleNamed("secondary").
										AsView(),
								),
							),
							swiftui.FloatProgressView(w.timer.ProgressState(), 1.0).
								Tint(0.24, 0.62, 0.98, 1.0),
						).Padding(14),
					).Frame(360, 0),
					swiftui.GroupBox("Controls",
						swiftui.VStackSpaced(14,
							swiftui.HStackSpaced(12,
								infoCard("Remaining", formatDuration(secs), "timer"),
								infoCard("Stage", timerStage(secs), "flag.checkered"),
							),
							swiftui.HStackSpaced(10,
								swiftui.Button("Start", func() {
									w.timer.SetRunning(true)
								}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
								swiftui.Button("Pause", func() {
									w.timer.SetRunning(false)
								}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
									Tint(0.94, 0.42, 0.26, 1),
								swiftui.Button("Reset", func() {
									w.timer.Reset()
								}).ButtonStyle(swiftui.ButtonStyleBordered),
							),
							swiftui.Text("This is the direct replacement pattern for timerInterval: the workbench now treats the timer as one owned runtime model instead of three independent values.").
								Font(swiftui.FontCallout).
								ForegroundStyleNamed("secondary"),
						).Padding(14),
					).MaxFrame(-1, 0),
				),
				swiftui.GroupBox("Status Cards",
					swiftui.HStackSpaced(12,
						infoCard("Attention", timerAttention(secs), "eye.fill"),
						infoCard("Next Break", nextBreak(secs), "cup.and.saucer.fill"),
						infoCard("Finish", finishLabel(secs), "clock.badge.checkmark"),
					).Padding(12),
				).MaxFrame(-1, 0),
			).Padding(24)
		}),
	)
}

func (w *workbench) routerPanel() swiftui.View {
	return scrollPanel(
		swiftui.VStackSpaced(18,
			panelHeader(
				"Navigation And Disclosure",
				"NavigationPathState is available for string-token routing; this route pairs it with simple NavigationLink leaves and explicit app state.",
				"Hybrid Path",
				"map",
			),
			swiftui.HStackSpaced(18,
				swiftui.GroupBox("Legacy Navigation",
					swiftui.NavigationStack(
						swiftui.List(
							swiftui.NavigationLink("Incident Review", navDestination("Incident Review", "Old-school push navigation still works well for bounded detail flows.")),
							swiftui.NavigationLink("Release Notes", navDestination("Release Notes", "Keep this for short, one-hop detail destinations.")),
							swiftui.NavigationLink("Operator Handoff", navDestination("Operator Handoff", "Use this when a simple link is enough and a full router would be overkill.")),
						).ListStyle(swiftui.ListStyleInset).
							NavigationTitle("Legacy Links"),
					).Frame(360, 360),
				).Frame(380, 0),
				swiftui.GroupBox("State Router",
					swiftui.VStackSpaced(12,
						swiftui.HStackSpaced(8,
							w.routerButton("Dashboard", 1),
							w.routerButton("Deploy", 2),
							w.routerButton("Review", 3),
							w.routerButton("Ship", 4),
						),
						swiftui.AnimatedDynamicView(w.routerState, swiftui.TransitionMove, func(v int) swiftui.View {
							return routerCard(v)
						}),
						swiftui.Form(
							swiftui.SectionExpanded("Overview", w.routerOverview, swiftui.VStackSpaced(8,
								featureRow("Why this exists", "Keep a single route enum on the Go side and let the detail surface rebuild from that."),
								featureRow("Where it fits", "Multi-step review and deploy flows that still benefit from explicit Go-side state."),
							)),
							swiftui.SectionExpanded("Checks", w.routerChecks, swiftui.VStackSpaced(8,
								featureRow("Static links", "Still use NavigationLink for shallow leaf screens."),
								featureRow("Deep flows", "Use the explicit router for multi-stage operator state."),
							)),
						).Frame(0, 260),
					).Padding(12),
				).MaxFrame(-1, 0),
			),
		).Padding(24),
	)
}

func (w *workbench) plannerPanel() swiftui.View {
	return scrollPanel(
		swiftui.VStackSpaced(18,
			panelHeader(
				"Planner",
				"MultiDatePicker is not bridged yet, so the current path is a DatePicker anchor plus a bitmask-backed multi-date selection model.",
				"Current Best Path",
				"calendar.badge.clock",
			),
			swiftui.HStackAlignedSpaced(swiftui.VerticalAlignmentTop, 18,
				swiftui.GroupBox("Planning Window",
					swiftui.VStackSpaced(14,
						swiftui.DatePicker("Week of", w.plannerAnchor, func() {
							w.bumpPlanner()
						}),
						swiftui.DynamicView(w.plannerTick, func(_ int) swiftui.View {
							mask := w.plannerMask.Get()
							anchor := anchorTime(w.plannerAnchor.Get())
							chips := make([]swiftui.Viewable, 0, 7)
							for i := 0; i < 7; i++ {
								chips = append(chips, plannerChip(anchor, i, mask&(1<<i) != 0, func(day int) func() {
									return func() {
										w.plannerMask.Set(toggleBit(w.plannerMask.Get(), day))
										w.bumpPlanner()
									}
								}(i)))
							}
							return swiftui.VStackSpaced(12,
								swiftui.HStackSpaced(8, chips...),
								swiftui.HStackSpaced(12,
									infoCard("Selected Days", strconv.Itoa(selectedDays(mask)), "checkmark.circle.fill"),
									infoCard("Pattern", plannerPattern(mask), "calendar"),
									infoCard("Current Path", "DatePicker + mask", "dial.medium.fill"),
								),
							)
						}),
					).Padding(14),
				).Frame(620, 0),
				swiftui.DynamicView(w.plannerTick, func(_ int) swiftui.View {
					anchor := anchorTime(w.plannerAnchor.Get())
					mask := w.plannerMask.Get()
					return swiftui.GroupBox("Agenda",
						swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 12,
							selectedAgenda(anchor, mask),
							swiftui.Text("This is the right current compromise: keep the planner state owned in Go and generate the visible agenda from that state.").
								Font(swiftui.FontCallout).
								ForegroundStyleNamed("secondary").
								LineLimit(4),
						).Padding(14),
					).Frame(340, 0)
				}),
			).MaxFrame(-1, 0),
		).Padding(24),
	)
}

func (w *workbench) gridPanel() swiftui.View {
	return scrollPanel(
		swiftui.VStackSpaced(18,
			panelHeader(
				"Data Grid",
				"TableModelView gives the bridge a typed table path today: stable row IDs, sortable headers, multi-selection state, optional columns, and a selected detail pane.",
				"Current Best Path",
				"tablecells.badge.ellipsis",
			),
			swiftui.VStackSpaced(18,
				swiftui.GroupBox("Grid",
					swiftui.VStackSpaced(10,
						swiftui.TableModelView(w.grid, gridColumns()...).
							Padding(8).
							BackgroundStyle("thinMaterial").
							CornerRadius(12),
						swiftui.Text("Click a header to toggle sorting; click a row to update the primary selection. The curated model also supports multi-selection and hidden columns.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary").
							LineLimit(2),
					).Padding(12),
				).MaxFrame(-1, 0),
				swiftui.DynamicView(w.grid.RevisionState(), func(_ int) swiftui.View {
					row := w.selectedGridRow()
					return swiftui.GroupBox("Selected Service",
						swiftui.VStackSpaced(12,
							infoCard("Service", row.Service, "shippingbox.fill"),
							swiftui.HStackSpaced(12,
								infoCard("Latency", row.Latency, "speedometer"),
								infoCard("Capacity", row.Capacity, "gauge.with.dots.needle.50percent"),
							),
							swiftui.HStackSpaced(12,
								infoCard("Owner", strings.Title(row.Owner), "person.crop.square"),
								infoCard("Readiness", strings.Title(row.Readiness), "checkmark.seal"),
							),
							swiftui.Text("This gives operators the key table behavior now: row comparison, multi-selection state, optional column visibility, and detail focus, without pretending we already have a real Table bridge.").
								Font(swiftui.FontCallout).
								ForegroundStyleNamed("secondary"),
						).Padding(14),
					).MaxFrame(-1, 0)
				}),
			),
		).Padding(24),
	)
}

func (w *workbench) documentsPanel() swiftui.View {
	docRows := make([]swiftui.Viewable, 0, len(w.docs))
	for _, doc := range w.docs {
		docRows = append(docRows, documentRow(doc).Tag(int32(doc.ID)))
	}

	return swiftui.HStackSpaced(18,
		swiftui.VStackSpaced(18,
			panelHeader(
				"Documents",
				"This route shows the document path that exists today: stable document identity, Quick Look preview, and scene-plan-owned auxiliary windows driven through RunScenes.",
				"Current Best Path",
				"doc.text.magnifyingglass",
			),
			swiftui.GroupBox("Documents",
				swiftui.SelectableList(w.documentSelection, docRows...).
					ListStyle(swiftui.ListStyleSidebar).
					Frame(320, 0),
			).MaxFrame(-1, 0),
			swiftui.GroupBox("Scene Status",
				swiftui.VStackSpaced(8,
					featureRow("What works now", "DocumentGroupWithHandle gives the document scene a stable ID, runtime metadata, and runner-backed scene actions."),
					featureRow("Multi-window", "RunScenes can now own more than one window scene and focus a sibling by scene ID."),
					featureRow("What still needs runtime", "Native SwiftUI environment scene ownership and restoration beyond the explicit runner-owned model still need a deeper App host."),
				).Padding(12),
			).MaxFrame(-1, 0),
		).Frame(380, 0),
		swiftui.DynamicView(w.documentSelection, func(id int) swiftui.View {
			doc := w.documentByID(id)
			preview := documentPreview(doc)
			return swiftui.VStackSpaced(18,
				swiftui.GroupBox("Preview",
					preview.MaxFrame(-1, -1),
				).MaxFrame(-1, -1),
				swiftui.HStackSpaced(12,
					infoCard("Document", doc.Title, "doc.richtext.fill"),
					infoCard("Kind", doc.Kind, "tag"),
					infoCard("Path", filepath.Base(doc.Path), "folder"),
				),
			).MaxFrame(-1, -1)
		}),
	).Padding(24)
}

func (w *workbench) gapsPanel() swiftui.View {
	shareButton := swiftui.Button("Share Fallback", func() {
		w.shareDialog.Set(1)
	}).ButtonStyle(swiftui.ButtonStyleBordered).
		ConfirmationDialog("Full share-sheet parity is not bridged yet", w.shareDialog, swiftui.VStack(
			swiftui.Button("Open document workspace", func() {
				w.route.Set(routeDocs)
				w.documentSelection.Set(1)
			}),
			swiftui.Button("Stay here", func() {}),
		))

	return scrollPanel(
		swiftui.VStackSpaced(18,
			panelHeader(
				"Runtime Gaps",
				"These are not demo gaps. They are real runtime-model gaps. The panel shows today’s fallback path and the exact missing capability.",
				"Needs Runtime Work",
				"exclamationmark.triangle.fill",
			),
			swiftui.HStackSpaced(18,
				swiftui.GroupBox("Refresh And Share",
					swiftui.VStackSpaced(12,
						swiftui.DynamicView(w.refreshState, func(v int) swiftui.View {
							switch v {
							case 1:
								return swiftui.HStackSpaced(10,
									swiftui.ProgressSpinning(),
									swiftui.Text("Refreshing preview data...").
										Font(swiftui.FontCallout).
										AsView(),
								)
							case 2:
								return statusStrip("Synced", "Manual refresh succeeded. See examples/bridge-coverage for the shared refresh callback path.", [3]float64{0.31, 0.78, 0.42}).AsView()
							default:
								return statusStrip("Idle", "This screen uses a button-triggered refresh. On macOS, native Refreshable should be demonstrated with an explicit refresh trigger.", [3]float64{0.55, 0.58, 0.62}).AsView()
							}
						}),
						swiftui.HStackSpaced(10,
							swiftui.Button("Refresh Snapshot", func() {
								w.runRefresh()
							}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
							shareButton,
						),
						swiftui.HStack(
							swiftui.Text("Last refresh").
								Font(swiftui.FontCaption).
								ForegroundStyleNamed("secondary"),
							swiftui.Spacer(),
							swiftui.TextFromString(w.lastRefresh).
								Font(swiftui.FontCaption).
								MonospacedDigit(),
						),
					).Padding(14),
				).MaxFrame(-1, 0),
				swiftui.GroupBox("Scenes And Space",
					swiftui.VStackSpaced(10,
						featureRow("Multi-window", "RunScenes now lowers multiple window/document scenes through the generated scene-plan runner."),
						featureRow("Document scenes", "DocumentGroup now carries typed config and document identity instead of acting like a plain window alias."),
						featureRow("Share sheet", "ShareLink is bridged for concrete sharing flows, but this shell does not claim full platform share-sheet parity."),
						featureRow("Refreshable", "Refreshable is bridged; the bridge-coverage example uses an explicit macOS refresh trigger for the same callback."),
						featureRow("Immersive space", "Pending a separate scene/runtime model. This macOS shell uses an explicit preview path instead of faking native spatial presentation."),
					).Padding(14),
				).MaxFrame(-1, 0),
			),
			swiftui.GroupBox("What To Build Next",
				swiftui.VStackSpaced(10,
					featureRow("Action environments", "OpenWindowAction, OpenDocumentAction, and RefreshAction are runner-backed for the current scene model."),
					featureRow("Scene model", "True SwiftUI environment scenes still need a deeper App/Scene runner than the current AppKit-owned plan."),
					featureRow("Specialized surfaces", "After the runtime model exists, promote the curated APIs that sit on top of it."),
				).Padding(14),
			).MaxFrame(-1, 0),
		).Padding(24),
	)
}

func (w *workbench) selectionInspector(route int) swiftui.View {
	switch route {
	case routeTree:
		return swiftui.DynamicView(w.fileSelection, func(id int) swiftui.View {
			f := fileByID(id)
			return inspectorStack(
				infoRow("Selected file", f.Name),
				infoRow("Group", f.Group),
				infoRow("Use", f.Summary),
			)
		})
	case routeMotion:
		return swiftui.DynamicView(w.phase, func(v int) swiftui.View {
			p := phaseSpec(v)
			return inspectorStack(
				infoRow("Stage", p.Title),
				infoRow("Intent", p.Intent),
				infoRow("Motion", p.Motion),
			)
		})
	case routeTimer:
		return swiftui.DynamicView(w.remaining, func(secs int) swiftui.View {
			return inspectorStack(
				infoRow("Remaining", formatDuration(secs)),
				infoRow("Status", timerStatus(secs)),
				infoRow("Attention", timerAttention(secs)),
			)
		})
	case routeRouter:
		return swiftui.DynamicView(w.routerState, func(v int) swiftui.View {
			path, ok := w.routerPath.Current()
			if !ok {
				path = "dashboard"
			}
			return inspectorStack(
				infoRow("Active route", routerSummary(v)),
				infoRow("Path", path),
				infoRow("Leaf nav", "NavigationLink"),
			)
		})
	case routePlanner:
		return swiftui.DynamicView(w.plannerTick, func(_ int) swiftui.View {
			start, end, ok := w.plannerWindow.Get()
			window := "unset"
			if ok {
				window = start.Format("Jan 2") + " to " + end.Format("Jan 2")
			}
			return inspectorStack(
				infoRow("Selected days", strconv.Itoa(w.plannerSelection.Count())),
				infoRow("Window", window),
				infoRow("Anchor", anchorTime(w.plannerAnchor.Get()).Format("Jan 2")),
			)
		})
	case routeGrid:
		return swiftui.DynamicView(w.grid.RevisionState(), func(_ int) swiftui.View {
			row := w.selectedGridRow()
			return inspectorStack(
				infoRow("Service", row.Service),
				infoRow("Owner", row.Owner),
				infoRow("Readiness", row.Readiness),
			)
		})
	case routeDocs:
		return swiftui.DynamicView(w.documentSelection, func(id int) swiftui.View {
			doc := w.documentByID(id)
			return inspectorStack(
				infoRow("Document", doc.Title),
				infoRow("Kind", doc.Kind),
				infoRow("Path", filepath.Base(doc.Path)),
			)
		})
	case routeGaps:
		return inspectorStack(
			infoRow("Refresh", "Manual button"),
			infoRow("Share", "Fallback dialog"),
			infoRow("Immersive", "Not modeled"),
		)
	default:
		return swiftui.DynamicView(w.workspaceSelection, func(id int) swiftui.View {
			module := moduleByID(id)
			return inspectorStack(
				infoRow("Module", module.Title),
				infoRow("Selection", "SelectableList"),
				infoRow("Shell", "NavigationSplitViewTripleVisibility"),
			)
		})
	}
}

func (w *workbench) pathInspector(route int) swiftui.View {
	if route == routeRouter {
		path := w.routerPath.String()
		if path == "" {
			path = "dashboard"
		}
		return inspectorStack(
			infoRow("Current path", path),
			infoRow("Exact symbol", "NavigationPathState"),
			infoRow("Why", routeStatusNote(route)),
		)
	}
	if route == routePlanner {
		start, end, ok := w.plannerWindow.Get()
		window := "unset"
		if ok {
			window = start.Format("Jan 2") + " to " + end.Format("Jan 2")
		}
		return inspectorStack(
			infoRow("Current window", window),
			infoRow("Exact symbol", "DateRangeState + DateSelectionState"),
			infoRow("Why", routeStatusNote(route)),
		)
	}
	return inspectorStack(
		infoRow("Current path", routeStatus(route)),
		infoRow("Exact symbol", routePath(route)),
		infoRow("Why", routeStatusNote(route)),
	)
}

func (w *workbench) backlogInspector(route int) swiftui.View {
	return inspectorStack(
		infoRow("Next step", routeNextStep(route)),
		infoRow("Likely work", routeWorkType(route)),
		infoRow("Risk", routeRisk(route)),
	)
}

func (w *workbench) treeFileButton(id int) swiftui.View {
	f := fileByID(id)
	selected := w.fileSelection.Get() == id
	bg := [4]float64{0.18, 0.2, 0.24, 0.04}
	if selected {
		bg = [4]float64{0.26, 0.54, 0.96, 0.16}
	}
	label := swiftui.HStackSpaced(10,
		swiftui.Image(iconForGroup(f.Group)).
			ForegroundStyle(0.28, 0.58, 0.96, 1).
			ImageScale(swiftui.ImageScaleSmall),
		swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 2,
			swiftui.Text(f.Name).
				Font(swiftui.FontBody).
				FontWeight(swiftui.WeightSemibold).
				AsView(),
			swiftui.Text(f.Summary).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView(),
		).MaxFrame(-1, 0),
	).Padding(10).
		BackgroundRoundedRect(bg[0], bg[1], bg[2], bg[3], 10)
	return swiftui.ButtonView(label, func() {
		w.fileSelection.Set(id)
	}).ButtonStyle(swiftui.ButtonStylePlain)
}

func (w *workbench) documentByID(id int) demoDocument {
	for _, doc := range w.docs {
		if doc.ID == id {
			return doc
		}
	}
	return w.docs[0]
}

func (w *workbench) bumpPlanner() {
	anchor := anchorTime(w.plannerAnchor.Get())
	start, end := plannerWindow(anchor)
	w.plannerWindow.Set(start, end)
	w.plannerSelection.Set(plannerSelection(anchor, w.plannerMask.Get()))
	w.plannerTick.Set(w.plannerTick.Get() + 1)
}

func (w *workbench) selectGridRow(id int) gridRow {
	w.grid.SelectID(strconv.Itoa(id))
	row, ok := w.grid.SelectedRow()
	if ok {
		return row
	}
	return gridRowByID(id)
}

func (w *workbench) selectedGridRow() gridRow {
	row, ok := w.grid.SelectedRow()
	if ok {
		return row
	}
	return rows[0]
}

func (w *workbench) runRefresh() {
	if w.refreshState.Get() == 1 {
		return
	}
	w.refreshState.SetAnimated(1)
	go func() {
		time.Sleep(900 * time.Millisecond)
		w.lastRefresh.Set(time.Now().Format("15:04:05"))
		w.refreshState.SetAnimated(2)
		time.Sleep(1200 * time.Millisecond)
		w.refreshState.Set(0)
	}()
}

func documentPreview(doc demoDocument) swiftui.View {
	card := swiftui.VStackSpaced(12,
		swiftui.ZStack(
			swiftui.Circle().
				Fill(0.28, 0.55, 1.0, 0.14).
				Frame(84, 84).
				AsView(),
			swiftui.Image("doc.richtext.fill").
				ForegroundStyle(0.35, 0.65, 1.0, 1.0).
				ImageScale(swiftui.ImageScaleLarge),
		),
		swiftui.Text(doc.Title).
			Font(swiftui.FontTitle2).
			FontWeight(swiftui.WeightBold),
		swiftui.Text(doc.Summary).
			Font(swiftui.FontCallout).
			ForegroundStyleNamed("secondary"),
		swiftui.Text(doc.Path).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("tertiary"),
	).Padding(24)
	previewPtr := quicklook.QuickLookPreview(card.Pointer(), doc.Path)
	return swiftui.ViewFromPointer(previewPtr)
}

func writeDemoDocuments() []demoDocument {
	dir := filepath.Join(os.TempDir(), "swiftui-workbench-docs")
	_ = os.MkdirAll(dir, 0o755)
	files := []struct {
		ID      int
		Name    string
		Kind    string
		Body    string
		Summary string
	}{
		{
			ID:      1,
			Name:    "handoff.md",
			Kind:    "Markdown",
			Summary: "Operator handoff brief with route, planner, and timer notes.",
			Body:    "# Surface Workbench\n\n- Split view shell is live.\n- Planner uses DatePicker plus a bitmask.\n- Documents stay single-window for now.\n",
		},
		{
			ID:      2,
			Name:    "deploy.json",
			Kind:    "JSON",
			Summary: "Structured deploy payload for a document-style detail screen.",
			Body:    "{\n  \"service\": \"planner\",\n  \"owner\": \"desktop\",\n  \"status\": \"ready\",\n  \"windowModel\": \"pending\"\n}\n",
		},
		{
			ID:      3,
			Name:    "runbook.log",
			Kind:    "Log",
			Summary: "A plain-text runbook surface suited to Quick Look preview.",
			Body:    "09:15 route=deploy state=review\n09:16 refresh=manual path=current-best\n09:18 window-scenes=scene-plan runner active\n",
		},
	}

	docs := make([]demoDocument, 0, len(files))
	for _, file := range files {
		path := filepath.Join(dir, file.Name)
		_ = os.WriteFile(path, []byte(file.Body), 0o644)
		docs = append(docs, demoDocument{
			ID:      file.ID,
			Title:   file.Name,
			Kind:    file.Kind,
			Path:    path,
			Summary: file.Summary,
		})
	}
	return docs
}

func sidebarRow(s surface) swiftui.View {
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4,
		swiftui.HStackSpaced(10,
			swiftui.Image(s.Icon).
				ForegroundStyle(0.3, 0.6, 1.0, 1.0).
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Text(s.Title).
				Font(swiftui.FontBody).
				FontWeight(swiftui.WeightSemibold).
				LineLimit(1).
				AsView(),
		),
		swiftui.Text(s.Subtitle).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			LineLimit(2).
			AsView(),
	).Padding(8)
}

func workspaceModuleRow(m module) swiftui.View {
	return swiftui.HStackSpaced(10,
		swiftui.ZStack(
			swiftui.RoundedRectangle(12).
				Fill(m.Color[0], m.Color[1], m.Color[2], 0.16).
				Frame(40, 40).
				AsView(),
			swiftui.Image(m.Icon).
				ForegroundStyle(m.Color[0], m.Color[1], m.Color[2], 1.0).
				ImageScale(swiftui.ImageScaleSmall),
		),
		swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 2,
			swiftui.Text(m.Title).
				Font(swiftui.FontBody).
				FontWeight(swiftui.WeightSemibold).
				AsView(),
			swiftui.Text(m.Summary).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				LineLimit(2).
				AsView(),
		).MaxFrame(-1, 0),
	).Padding(10)
}

func moduleHero(m module) swiftui.View {
	return swiftui.HStackSpaced(14,
		swiftui.ZStack(
			swiftui.RoundedRectangle(18).
				Fill(m.Color[0], m.Color[1], m.Color[2], 0.14).
				Frame(74, 74).
				AsView(),
			swiftui.Image(m.Icon).
				ForegroundStyle(m.Color[0], m.Color[1], m.Color[2], 1.0).
				ImageScale(swiftui.ImageScaleLarge),
		),
		swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4,
			swiftui.Text(m.Title).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold).
				AsView(),
			swiftui.Text(m.Summary).
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary").
				LineLimit(3).
				AsView(),
		).MaxFrame(-1, 0),
	)
}

func infoCard(label, value, icon string) swiftui.View {
	return swiftui.VStackSpaced(8,
		swiftui.HStack(
			swiftui.Label(label, icon).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightSemibold).
				LineLimit(2),
			swiftui.Spacer(),
		),
	).Padding(12).
		BackgroundStyle("regularMaterial").
		CornerRadius(12)
}

func featureRow(title, body string) swiftui.View {
	return swiftui.HStackAlignedSpaced(swiftui.VerticalAlignmentTop, 10,
		swiftui.Circle().
			Fill(0.3, 0.6, 1.0, 0.22).
			Frame(10, 10).
			AsView().
			PaddingEdge(swiftui.EdgeTop, 5),
		swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 2,
			swiftui.Text(title).
				Font(swiftui.FontBody).
				FontWeight(swiftui.WeightSemibold).
				AsView(),
			swiftui.Text(body).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				LineLimit(3).
				AsView(),
		).MaxFrame(-1, 0),
	)
}

func panelHeader(title, body, badge, icon string) swiftui.View {
	return swiftui.HStackSpaced(16,
		swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4,
			swiftui.Text(title).
				Font(swiftui.FontLargeTitle).
				FontWeight(swiftui.WeightBold).
				AsView(),
			swiftui.Text(body).
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary").
				LineLimit(3).
				AsView(),
		).MaxFrame(-1, 0),
		swiftui.Label(badge, icon).
			Font(swiftui.FontCaption).
			ForegroundStyle(0.3, 0.6, 1.0, 1.0).
			AsView(),
	)
}

func sidebarHeader() swiftui.View {
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 10,
		swiftui.HStackSpaced(10,
			swiftui.ZStack(
				swiftui.RoundedRectangle(14).
					Fill(0.28, 0.58, 0.96, 0.16).
					Frame(44, 44).
					AsView(),
				swiftui.Image("square.stack.3d.up.fill").
					ForegroundStyle(0.28, 0.58, 0.96, 1.0).
					ImageScale(swiftui.ImageScaleSmall),
			),
			swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 2,
				swiftui.Text("Surface Workbench").
					Font(swiftui.FontTitle3).
					FontWeight(swiftui.WeightBold).
					LineLimit(2).
					AsView(),
				swiftui.Text("Catalog-first showcase").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView(),
			).MaxFrame(-1, 0),
		),
		swiftui.Text("The left rail should read clearly, not collapse into a vertical stack of broken words.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			LineLimit(3),
	).Padding(12).
		BackgroundStyle("regularMaterial").
		CornerRadius(16)
}

func sidebarWidthShim() swiftui.View {
	return swiftui.Rectangle().
		Fill(0.2, 0.2, 0.2, 0.001).
		Frame(228, 1).
		AsView().
		AllowsHitTesting(false).
		AccessibilityHidden(true)
}

func inspectorWidthShim() swiftui.View {
	return swiftui.Rectangle().
		Fill(0.2, 0.2, 0.2, 0.001).
		Frame(200, 1).
		AsView().
		AllowsHitTesting(false).
		AccessibilityHidden(true)
}

func scrollPanel(content swiftui.View) swiftui.View {
	return swiftui.ScrollView(
		content.ScrollTargetLayout(),
	).
		DefaultScrollAnchor(swiftui.ScrollAnchorTop).
		ScrollTargetBehavior(swiftui.ScrollTargetBehaviorViewAligned).
		ScrollBounceBehavior(swiftui.ScrollBounceBasedOnSize, swiftui.AxisVertical)
}

func statusStrip(title, body string, rgb [3]float64) swiftui.TextView {
	return swiftui.Label(title, "circle.fill").
		Font(swiftui.FontCaption).
		ForegroundStyle(rgb[0], rgb[1], rgb[2], 1.0)
}

func inspectorStack(children ...swiftui.Viewable) swiftui.View {
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 10, children...).Padding(4)
}

func infoRow(label, value string) swiftui.View {
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 3,
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("tertiary").
			AsView(),
		swiftui.Text(value).
			Font(swiftui.FontBody).
			FontWeight(swiftui.WeightSemibold).
			LineLimit(3).
			AsView(),
	)
}

func phaseButton(label string, state *swiftui.IntState, v int, animation swiftui.AnimationKind) swiftui.View {
	return swiftui.Button(label, func() {
		state.SetAnimatedWith(v, animation)
	}).ButtonStyle(swiftui.ButtonStyleBordered)
}

func phaseCard(v int) swiftui.View {
	p := phaseSpec(v)
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Label(p.Title, p.Icon).
				Font(swiftui.FontTitle3).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(p.Body).
				Font(swiftui.FontBody).
				ForegroundStyleNamed("secondary").
				LineLimit(3),
			swiftui.Spacer(),
		),
		swiftui.HStackSpaced(12,
			infoCard("Intent", p.Intent, "target"),
			infoCard("Motion", p.Motion, "sparkle"),
		),
	).Padding(18).
		BackgroundRoundedRect(p.Color[0], p.Color[1], p.Color[2], 0.12, 18)
}

type phaseState struct {
	Title  string
	Body   string
	Intent string
	Motion string
	Icon   string
	Color  [3]float64
}

func phaseSpec(v int) phaseState {
	phases := []phaseState{
		{"Discover", "Start broad. Gather signal and keep motion gentle while the model is still forming.", "Framing", "Ease In-Out", "sparkles", [3]float64{0.29, 0.58, 0.96}},
		{"Plan", "Tighten the path, reduce branching, and make the next step explicit.", "Constraint", "Ease In", "point.forward.to.point.capsulepath", [3]float64{0.95, 0.61, 0.23}},
		{"Build", "Move decisively with a spring curve that makes state changes feel intentional.", "Execution", "Spring", "hammer.fill", [3]float64{0.31, 0.79, 0.42}},
		{"Review", "Slow down just enough to inspect consequences before shipping.", "Verification", "Bouncy", "checkmark.magnifyingglass", [3]float64{0.83, 0.36, 0.8}},
		{"Ship", "Exit cleanly and leave the next operator with a legible final state.", "Handoff", "Ease Out", "paperplane.fill", [3]float64{0.93, 0.36, 0.36}},
	}
	return phases[abs(v)%len(phases)]
}

func navDestination(title, body string) swiftui.View {
	return swiftui.VStackSpaced(14,
		swiftui.Text(title).
			Font(swiftui.FontTitle2).
			FontWeight(swiftui.WeightBold),
		swiftui.Text(body).
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
	).Padding(24)
}

func (w *workbench) routerButton(label string, v int) swiftui.View {
	return swiftui.Button(label, func() {
		w.routerState.SetAnimated(v)
		w.routerPath.Set([]string{routerPathLabel(v)})
	}).ButtonStyle(swiftui.ButtonStyleBordered)
}

func routerCard(v int) swiftui.View {
	title := routerSummary(v)
	body := routerBody(v)
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontTitle3).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(body).
				Font(swiftui.FontBody).
				ForegroundStyleNamed("secondary").
				LineLimit(4),
			swiftui.Spacer(),
		),
		swiftui.HStackSpaced(12,
			infoCard("Route", routerPathLabel(v), "point.topleft.down.curvedto.point.bottomright.up"),
			infoCard("State", "IntState", "dial.medium"),
		),
	).Padding(18).
		BackgroundStyle("regularMaterial").
		CornerRadius(18)
}

func plannerChip(anchor time.Time, offset int, selected bool, action func()) swiftui.View {
	day := anchor.AddDate(0, 0, offset)
	r, g, b, a := 0.18, 0.2, 0.24, 0.04
	if selected {
		r, g, b, a = 0.28, 0.58, 0.96, 0.16
	}
	label := swiftui.VStackSpaced(4,
		swiftui.Text(day.Format("Mon")).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			AsView(),
		swiftui.Text(day.Format("2")).
			Font(swiftui.FontHeadline).
			FontWeight(swiftui.WeightBold).
			AsView(),
	).Padding(10).
		Frame(58, 64).
		BackgroundRoundedRect(r, g, b, a, 12)
	return swiftui.ButtonView(label, action).ButtonStyle(swiftui.ButtonStylePlain)
}

func selectedAgenda(anchor time.Time, mask int) swiftui.View {
	if selectedDays(mask) == 0 {
		return swiftui.HStack(
			swiftui.Text("No focused days selected. Pick dates above to populate the agenda.").
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		)
	}
	items := make([]swiftui.Viewable, 0, 7)
	for i := 0; i < 7; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		day := anchor.AddDate(0, 0, i)
		items = append(items, swiftui.HStackSpaced(10,
			swiftui.Label(day.Format("Mon 2"), "calendar").
				Font(swiftui.FontBody),
			swiftui.Spacer(),
			swiftui.Text(plannerTask(i)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
		))
	}
	return swiftui.VStackSpaced(8, items...)
}

func gridHeader() swiftui.View {
	return swiftui.HStackSpaced(16,
		gridHeaderCell("SERVICE", 120),
		gridHeaderCell("SLO", 70),
		gridHeaderCell("LATENCY", 80),
		gridHeaderCell("CAPACITY", 80),
		gridHeaderCell("OWNER", 90),
		gridHeaderCell("STATE", 70),
	).Padding(6)
}

func gridColumns() []swiftui.TableModelColumn[gridRow] {
	return []swiftui.TableModelColumn[gridRow]{
		swiftui.TextTableModelColumn("SERVICE", func(row gridRow) string { return row.Service }, func(a, b gridRow) bool { return a.Service < b.Service }).
			WithID("service").
			WithMaxWidth(120),
		swiftui.TextTableModelColumn("SLO", func(row gridRow) string { return row.SLO }, func(a, b gridRow) bool { return a.SLO < b.SLO }).
			WithID("slo").
			WithMaxWidth(70),
		swiftui.TextTableModelColumn("LATENCY", func(row gridRow) string { return row.Latency }, func(a, b gridRow) bool { return a.Latency < b.Latency }).
			WithID("latency").
			WithMaxWidth(80),
		swiftui.TextTableModelColumn("CAPACITY", func(row gridRow) string { return row.Capacity }, func(a, b gridRow) bool { return a.Capacity < b.Capacity }).
			WithID("capacity").
			WithMaxWidth(80),
		swiftui.TextTableModelColumn("OWNER", func(row gridRow) string { return strings.Title(row.Owner) }, func(a, b gridRow) bool { return a.Owner < b.Owner }).
			WithID("owner").
			WithMaxWidth(90),
		swiftui.TextTableModelColumn("STATE", func(row gridRow) string { return strings.Title(row.Readiness) }, func(a, b gridRow) bool { return a.Readiness < b.Readiness }).
			WithID("state").
			WithMaxWidth(70),
	}
}

func gridHeaderCell(label string, width float64) swiftui.View {
	return swiftui.Text(label).
		Font(swiftui.FontCaption2).
		FontWeight(swiftui.WeightBold).
		ForegroundStyleNamed("tertiary").
		Frame(width, 0).
		AsView()
}

func gridDataRow(row gridRow) swiftui.View {
	return swiftui.HStackSpaced(16,
		gridCell(row.Service, 120),
		gridCell(row.SLO, 70),
		gridCell(row.Latency, 80),
		gridCell(row.Capacity, 80),
		gridCell(strings.Title(row.Owner), 90),
		gridCell(strings.Title(row.Readiness), 70),
	).Padding(8)
}

func gridCell(text string, width float64) swiftui.View {
	return swiftui.Text(text).
		Font(swiftui.FontCaption).
		FontDesign(swiftui.DesignMonospaced).
		Frame(width, 0).
		AsView()
}

func documentRow(doc demoDocument) swiftui.View {
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 4,
		swiftui.HStackSpaced(10,
			swiftui.Image("doc.text.fill").
				ForegroundStyle(0.29, 0.58, 0.96, 1).
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Text(doc.Title).
				Font(swiftui.FontBody).
				FontWeight(swiftui.WeightSemibold).
				AsView(),
		),
		swiftui.Text(doc.Summary).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			LineLimit(2).
			AsView(),
	).Padding(10)
}

func routeStatus(id int) string {
	switch id {
	case routeShell:
		return "Exact generated path"
	case routeGaps:
		return "Runtime-model gap"
	default:
		return "Current best path"
	}
}

func routeStatusNote(id int) string {
	switch id {
	case routeShell:
		return "This surface already exists exactly as generated API."
	case routeTree:
		return "Use OutlineModel with OutlineGroupModel for typed disclosure state."
	case routeMotion:
		return "PhaseAnimator is bridged; staged animation still works with explicit phase state."
	case routeTimer:
		return "timerInterval text and TimerState are both available."
	case routeRouter:
		return "Use NavigationLink for leaves and an explicit state router for deeper flows."
	case routePlanner:
		return "MultiDatePicker is pending; DatePicker plus a selection mask is the clean fallback."
	case routeGrid:
		return "Use TableModelView for curated table state, selection, and sorting."
	case routeDocs:
		return "DocumentGroupWithHandle now rides the scene-plan runner; sibling windows can be focused by scene ID while document identity stays explicit."
	default:
		return "These are runtime gaps, not demo polish gaps."
	}
}

func routePath(id int) string {
	switch id {
	case routeShell:
		return "NavigationSplitView*"
	case routeTree:
		return "SectionExpanded"
	case routeMotion:
		return "AnimatedDynamicView"
	case routeTimer:
		return "Ticker + Text"
	case routeRouter:
		return "NavigationStack + IntState"
	case routePlanner:
		return "DatePicker + mask"
	case routeGrid:
		return "TableModelView"
	case routeDocs:
		return "DocumentGroupWithHandle"
	default:
		return "Await runtime model"
	}
}

func routeNextStep(id int) string {
	switch id {
	case routeShell:
		return "Keep generation catalog-first."
	case routeTree:
		return "Promote a curated tree view if it improves call sites."
	case routeMotion:
		return "Add a phase-aware wrapper on top of the existing state model."
	case routeTimer:
		return "Specialize timer text if it reduces boilerplate."
	case routeRouter:
		return "Introduce a path model only when the runtime can own it cleanly."
	case routePlanner:
		return "Add typed multi-date state before promoting MultiDatePicker."
	case routeGrid:
		return "Decide whether a curated Table wrapper is worth the runtime cost."
	case routeDocs:
		return "Keep the scene-plan runner honest unless a concrete product need justifies a deeper App/Scene host."
	default:
		return "Only pursue native App/Scene host work when the current runner blocks a real product shape."
	}
}

func routeWorkType(id int) string {
	switch id {
	case routeShell:
		return "No new runtime work"
	case routeGaps, routeDocs:
		return "Runtime model"
	default:
		return "Curated API plus emitter work"
	}
}

func routeRisk(id int) string {
	switch id {
	case routeShell:
		return "Low"
	case routeGaps, routeDocs:
		return "High"
	case routePlanner, routeRouter:
		return "Medium"
	default:
		return "Low to medium"
	}
}

func routeStatusColor(id int) [3]float64 {
	switch id {
	case routeShell:
		return [3]float64{0.31, 0.78, 0.42}
	case routeGaps:
		return [3]float64{0.92, 0.43, 0.27}
	default:
		return [3]float64{0.29, 0.58, 0.96}
	}
}

func visibilityLabel(v int) string {
	switch swiftui.NavigationSplitViewVisibilityKind(v) {
	case swiftui.NavigationSplitViewVisibilityAll:
		return "All Columns"
	case swiftui.NavigationSplitViewVisibilityDoubleColumn:
		return "Two Columns"
	case swiftui.NavigationSplitViewVisibilityDetailOnly:
		return "Detail Only"
	default:
		return "Automatic"
	}
}

func moduleByID(id int) module {
	for _, m := range workspaceModules {
		if m.ID == id {
			return m
		}
	}
	return workspaceModules[0]
}

func fileByID(id int) treeFile {
	for _, f := range treeFiles {
		if f.ID == id {
			return f
		}
	}
	return treeFiles[0]
}

func iconForGroup(group string) string {
	switch group {
	case "Bridge":
		return "link"
	case "Docs":
		return "doc.text"
	default:
		return "swift"
	}
}

func timerStatus(secs int) string {
	switch {
	case secs == 0:
		return "Session complete"
	case secs < 5*60:
		return "Wrap up and hand off"
	case secs < 10*60:
		return "Focused finish"
	default:
		return "Deep work"
	}
}

func timerStage(secs int) string {
	switch {
	case secs == 0:
		return "Done"
	case secs < 5*60:
		return "Cooldown"
	case secs < 10*60:
		return "Review"
	default:
		return "Focus"
	}
}

func timerAttention(secs int) string {
	switch {
	case secs < 2*60:
		return "Immediate"
	case secs < 6*60:
		return "Soon"
	default:
		return "Stable"
	}
}

func nextBreak(secs int) string {
	if secs == 0 {
		return "Now"
	}
	if secs < 5*60 {
		return "In " + strconv.Itoa(secs/60+1) + "m"
	}
	return "At 05:00"
}

func finishLabel(secs int) string {
	return time.Now().Add(time.Duration(secs) * time.Second).Format("15:04")
}

func routerSummary(v int) string {
	switch v {
	case 2:
		return "Deploy"
	case 3:
		return "Review"
	case 4:
		return "Ship"
	default:
		return "Dashboard"
	}
}

func routerBody(v int) string {
	switch v {
	case 2:
		return "Promote the deploy route only after checks pass. This is the current good use of explicit route state."
	case 3:
		return "Review keeps a bounded checklist close to the route without requiring a general path model."
	case 4:
		return "Ship is the handoff state: the route is still inspectable from Go and simple to test."
	default:
		return "Dashboard is the stable entry point. Use it as the default route when the app rehydrates."
	}
}

func routerPathLabel(v int) string {
	return strings.ToLower(routerSummary(v))
}

func anchorTime(epoch float64) time.Time {
	return time.Unix(int64(epoch), 0)
}

func toggleBit(mask, bit int) int {
	return mask ^ (1 << bit)
}

func selectedDays(mask int) int {
	n := 0
	for mask != 0 {
		n += mask & 1
		mask >>= 1
	}
	return n
}

func plannerPattern(mask int) string {
	switch selectedDays(mask) {
	case 0:
		return "None"
	case 1:
		return "Single-day"
	case 2:
		return "Paired"
	default:
		return "Distributed"
	}
}

func plannerTask(day int) string {
	tasks := []string{
		"Design review",
		"Planner sync",
		"Backlog prune",
		"Ship window",
		"Ops review",
		"Customer notes",
		"Quiet day",
	}
	return tasks[day%len(tasks)]
}

func gridRowByID(id int) gridRow {
	for _, row := range rows {
		if row.ID == id {
			return row
		}
	}
	return rows[0]
}

func formatDuration(secs int) string {
	if secs < 0 {
		secs = 0
	}
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
