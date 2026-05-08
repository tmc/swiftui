//go:build darwin
// +build darwin

// Command layout demonstrates SwiftUI layout composition and visual styling
// from Go.
//
// It builds a dashboard-style layout with nested stacks, shapes, colors,
// padding, shadows, and other visual modifiers.
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

func main() {
	layoutMode := swiftui.NewIntState(0)
	defer layoutMode.Release()

	cardViews := []swiftui.Viewable{
		statCard("Active Users", "42", "This week", "person.2.fill", 0.27, 0.62, 1.0),
		statCard("Open Events", "128", "Current total", "bolt.fill", 0.96, 0.47, 0.29),
		statCard("Monthly Revenue", "$9.8k", "Month to date", "banknote.fill", 0.28, 0.78, 0.44),
	}
	swiftui.Run(swiftui.AppConfig{
		Title:  "Layout Demo",
		Width:  720,
		Height: 600,
	}, swiftui.ZStack(
		background(),
		swiftui.ScrollView(
			swiftui.VStackSpaced(12,
				header(),
				layoutSwitcher(layoutMode),
				swiftui.DynamicView(layoutMode, func(mode int) swiftui.View {
					return cardStrip(mode, cardViews...)
				}),
				swiftui.HStackSpaced(12,
					trafficPanel().MaxFrame(-1, 0),
					checklistPanel().MaxFrame(-1, 0),
				),
				swiftui.HStackSpaced(12,
					activityPanel().MaxFrame(-1, 0),
					actionsPanel().MaxFrame(-1, 0),
				),
			).Padding(16),
		),
	))
}

func background() swiftui.View {
	return swiftui.ZStack(
		swiftui.Circle().
			Fill(0.20, 0.34, 0.52, 0.08).
			Frame(260, 260).
			AsView().
			Offset(-240, -220).
			Blur(96),
	)
}

func header() swiftui.View {
	return swiftui.HStackSpaced(12,
		swiftui.VStackSpaced(1,
			swiftui.HStack(
				swiftui.Text("Operations").
					Font(swiftui.FontLargeTitle).
					FontWeight(swiftui.WeightBold),
				swiftui.Spacer(),
			),
			swiftui.HStack(
				swiftui.Text("A compact example built from stacks, cards, and panels.").
					Font(swiftui.FontSubheadline).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
			),
		),
	)
}

func layoutSwitcher(mode *swiftui.IntState) swiftui.View {
	return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 8,
		swiftui.HStackSpaced(8,
			swiftui.Button("Row", func() { mode.Set(0) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Grid", func() { mode.Set(1) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Flow", func() { mode.Set(2) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Responsive", func() { mode.Set(3) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Adaptive Grid", func() { mode.Set(4) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Spacer(),
		),
		swiftui.HStackSpaced(8,
			swiftui.Button("Viewport Rules", func() { mode.Set(5) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Tagged", func() { mode.Set(6) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Tag Counts", func() { mode.Set(7) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Tag Order", func() { mode.Set(8) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Preset", func() { mode.Set(9) }).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
			swiftui.Button("Shell", func() { mode.Set(10) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("Board", func() { mode.Set(11) }).ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Spacer(),
		),
		swiftui.Text("LayoutProposal names the constrained width, height, and child-count inputs. Tags plus concrete placement hints (span and priority) remain the only metadata path, and the layouts stay narrower than SwiftUI's raw Layout protocol.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			LineLimit(0).
			AsView(),
		swiftui.Text(swiftui.NewLayoutProposal(760, 260, 3).String()).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			AsView(),
	)
}

func cardStrip(mode int, cards ...swiftui.Viewable) swiftui.View {
	if mode == 3 {
		model := swiftui.NewResponsiveLayoutModel(
			swiftui.VStackLayout(swiftui.HorizontalAlignmentLeading, 12),
			swiftui.AtLeastWidth(460, swiftui.HStackLayout(swiftui.VerticalAlignmentTop, 12)),
			swiftui.AtLeastWidth(760, swiftui.VGridLayout([]swiftui.GridItem{
				swiftui.FlexibleGridItem(180, 260),
				swiftui.FlexibleGridItem(180, 260),
			}, 12)),
		)
		return swiftui.CustomLayout(model, cards...).Frame(640, 220)
	}
	if mode == 4 {
		model := swiftui.NewAdaptiveGridModel(
			swiftui.VStackLayout(swiftui.HorizontalAlignmentLeading, 12),
			180,
			12,
		)
		model.MaxColumns = 3
		return swiftui.CustomLayout(model, cards...).Frame(640, 220)
	}
	if mode == 5 {
		model := swiftui.NewRuleBasedLayoutModel(
			swiftui.VStackLayout(swiftui.HorizontalAlignmentLeading, 12),
			swiftui.LayoutRule{
				MaxChildren: 2,
				Layout:      swiftui.HStackLayout(swiftui.VerticalAlignmentTop, 12),
			},
			swiftui.LayoutRule{
				MaxWidth:    420,
				MinChildren: 3,
				Layout:      swiftui.FlowLayout(180, 12),
			},
			swiftui.LayoutRule{
				MinWidth:    760,
				MinHeight:   220,
				MinChildren: 3,
				Layout: swiftui.VGridLayout([]swiftui.GridItem{
					swiftui.FlexibleGridItem(180, 260),
					swiftui.FlexibleGridItem(180, 260),
				}, 12),
			},
		)
		return swiftui.CustomLayout(model, cards...).Frame(640, 200)
	}
	if mode == 6 {
		model := swiftui.NewTaggedRuleBasedLayoutModel(
			swiftui.VStackLayout(swiftui.HorizontalAlignmentLeading, 12),
			swiftui.TaggedLayoutRule{
				MinWidth:    900,
				MinChildren: 4,
				RequireTags: []swiftui.LayoutTag{"hero"},
				Layout: swiftui.VGridLayout([]swiftui.GridItem{
					swiftui.FlexibleGridItem(180, 260),
					swiftui.FlexibleGridItem(180, 260),
				}, 12),
			},
			swiftui.TaggedLayoutRule{
				MinWidth:    640,
				RequireTags: []swiftui.LayoutTag{"hero", "detail"},
				Layout:      swiftui.HStackLayout(swiftui.VerticalAlignmentTop, 12),
			},
			swiftui.TaggedLayoutRule{
				MaxWidth:    420,
				MinChildren: 3,
				RequireTags: []swiftui.LayoutTag{"hero"},
				Layout:      swiftui.FlowLayout(180, 12),
			},
		)
		return swiftui.CustomLayoutTagged(model, taggedOverviewCards()...).Frame(640, 220)
	}
	if mode == 7 {
		model := swiftui.NewTaggedRuleBasedLayoutModel(
			swiftui.VStackLayout(swiftui.HorizontalAlignmentLeading, 12),
			swiftui.TaggedLayoutRule{
				MinWidth:    900,
				MinChildren: 4,
				RequireTags: []swiftui.LayoutTag{"hero"},
				TagCounts: []swiftui.TagCountConstraint{
					swiftui.AtLeastTagCount("meta", 2),
				},
				Layout: swiftui.VGridLayout([]swiftui.GridItem{
					swiftui.FlexibleGridItem(180, 260),
					swiftui.FlexibleGridItem(180, 260),
				}, 12),
			},
			swiftui.TaggedLayoutRule{
				MinWidth:    640,
				RequireTags: []swiftui.LayoutTag{"hero", "detail"},
				TagCounts: []swiftui.TagCountConstraint{
					swiftui.TagCountBetween("meta", 1, 2),
				},
				Layout: swiftui.HStackLayout(swiftui.VerticalAlignmentTop, 12),
			},
			swiftui.TaggedLayoutRule{
				MaxWidth: 420,
				TagCounts: []swiftui.TagCountConstraint{
					swiftui.AtLeastTagCount("meta", 2),
				},
				Layout: swiftui.FlowLayout(180, 12),
			},
		)
		return swiftui.CustomLayoutTagged(model, taggedBoardCards()...).Frame(640, 240)
	}
	if mode == 8 {
		model := swiftui.NewTaggedRuleBasedLayoutModel(
			swiftui.VStackLayout(swiftui.HorizontalAlignmentLeading, 12),
			swiftui.TaggedLayoutRule{
				MinWidth:    900,
				MinChildren: 4,
				RequireTags: []swiftui.LayoutTag{"hero"},
				TagCounts: []swiftui.TagCountConstraint{
					swiftui.AtLeastTagCount("meta", 2),
				},
				LeadingTags: []swiftui.LayoutTag{"hero"},
				Layout: swiftui.VGridLayout([]swiftui.GridItem{
					swiftui.FlexibleGridItem(180, 260),
					swiftui.FlexibleGridItem(180, 260),
				}, 12),
			},
			swiftui.TaggedLayoutRule{
				MinWidth:    640,
				RequireTags: []swiftui.LayoutTag{"hero", "detail"},
				TagCounts: []swiftui.TagCountConstraint{
					swiftui.TagCountBetween("meta", 1, 2),
				},
				LeadingTags:  []swiftui.LayoutTag{"hero", "detail"},
				TrailingTags: []swiftui.LayoutTag{"meta"},
				Layout:       swiftui.HStackLayout(swiftui.VerticalAlignmentTop, 12),
			},
			swiftui.TaggedLayoutRule{
				MaxWidth: 420,
				TagCounts: []swiftui.TagCountConstraint{
					swiftui.AtLeastTagCount("meta", 2),
				},
				Layout: swiftui.FlowLayout(180, 12),
			},
		)
		return swiftui.CustomLayoutTagged(model, orderedBoardCards()...).Frame(640, 240)
	}
	if mode == 9 {
		model := swiftui.NewPlacementLayoutModel(swiftui.PlacementPresetFeaturedGrid, 12)
		model.Breakpoint = 680
		model.MinItemWidth = 180
		return swiftui.CustomLayoutTagged(model, taggedOverviewCards()...).Frame(640, 220)
	}
	if mode == 10 {
		model := swiftui.NewPlacementLayoutModel(swiftui.PlacementPresetPrimarySecondary, 14)
		model.Breakpoint = 700
		return swiftui.CustomLayoutTagged(model, shellCards()...).Frame(640, 220)
	}
	if mode == 11 {
		model := swiftui.NewPlacementLayoutModel(swiftui.PlacementPresetDashboardBoard, 14)
		model.Breakpoint = 700
		model.MinItemWidth = 200
		return swiftui.CustomLayoutTagged(model, boardCards()...).Frame(700, 240)
	}
	spec := swiftui.HStackLayout(swiftui.VerticalAlignmentTop, 12)
	switch mode {
	case 1:
		spec = swiftui.VGridLayout([]swiftui.GridItem{
			swiftui.FlexibleGridItem(180, 260),
			swiftui.FlexibleGridItem(180, 260),
			swiftui.FlexibleGridItem(180, 260),
		}, 12)
	case 2:
		spec = swiftui.FlowLayout(210, 12)
	}
	return swiftui.AnyLayout(spec, cards...)
}

func taggedOverviewCards() []swiftui.TaggedView {
	return []swiftui.TaggedView{
		swiftui.Tagged("hero", statCard("North Star", "94%", "Primary KPI", "star.fill", 0.27, 0.62, 1.0)),
		swiftui.Tagged("detail", statCard("Launches", "12", "This week", "paperplane.fill", 0.96, 0.47, 0.29)),
		swiftui.Tagged("meta", statCard("Risk Review", "3", "Open follow-ups", "flag.fill", 0.89, 0.32, 0.45)),
		swiftui.Tagged("meta", statCard("Capacity", "71%", "Available headroom", "gauge.with.dots.needle.67percent", 0.28, 0.78, 0.44)),
	}
}

func taggedBoardCards() []swiftui.TaggedView {
	return []swiftui.TaggedView{
		swiftui.Tagged("hero", statCard("Launch Readiness", "88%", "Primary board signal", "star.square.fill", 0.27, 0.62, 1.0)),
		swiftui.Tagged("detail", statCard("Owner Notes", "7", "Review items", "text.bubble.fill", 0.96, 0.47, 0.29)),
		swiftui.Tagged("meta", statCard("Legal", "2", "Open approvals", "doc.text.fill", 0.89, 0.32, 0.45)),
		swiftui.Tagged("meta", statCard("Assets", "5", "Pending exports", "photo.stack.fill", 0.28, 0.78, 0.44)),
		swiftui.Tagged("meta", statCard("Ops", "1", "Deploy block", "shippingbox.fill", 0.95, 0.73, 0.24)),
	}
}

func orderedBoardCards() []swiftui.TaggedView {
	return []swiftui.TaggedView{
		swiftui.Tagged("hero", statCard("Launch Narrative", "3", "Lead decision", "doc.richtext.fill", 0.27, 0.62, 1.0)),
		swiftui.Tagged("detail", statCard("Review Deck", "9", "Slides ready", "rectangle.on.rectangle.fill", 0.96, 0.47, 0.29)),
		swiftui.Tagged("meta", statCard("Approvals", "2", "Remaining sign-offs", "checkmark.seal.fill", 0.89, 0.32, 0.45)),
		swiftui.Tagged("meta", statCard("Risks", "4", "Open items", "exclamationmark.triangle.fill", 0.95, 0.73, 0.24)),
	}
}

func shellCards() []swiftui.TaggedView {
	return []swiftui.TaggedView{
		swiftui.Tagged("primary", statCard("Primary Signal", "76%", "Decision-ready summary", "sidebar.left", 0.27, 0.62, 1.0)),
		swiftui.Tagged("secondary", statCard("Secondary Detail", "14", "Supporting evidence", "doc.text.magnifyingglass", 0.96, 0.47, 0.29)),
		swiftui.Tagged("meta", statCard("Context", "3", "Open follow-ups", "flag.fill", 0.89, 0.32, 0.45)),
	}
}

func boardCards() []swiftui.TaggedView {
	return []swiftui.TaggedView{
		swiftui.TaggedWithPlacement("primary", swiftui.PlacementHint(2, 2), statCard("Primary Signal", "91%", "Board-ready summary", "rectangle.grid.3x2.fill", 0.27, 0.62, 1.0)),
		swiftui.TaggedWithPlacement("detail", swiftui.PlacementHint(1, 1), statCard("Detail", "16", "Supporting metrics", "text.magnifyingglass", 0.96, 0.47, 0.29)),
		swiftui.TaggedWithPlacement("meta", swiftui.PlacementHint(1, 0), statCard("Meta A", "4", "Open items", "flag.fill", 0.89, 0.32, 0.45)),
		swiftui.TaggedWithPlacement("meta", swiftui.PlacementHint(1, 0), statCard("Meta B", "2", "Next follow-up", "checkmark.seal.fill", 0.28, 0.78, 0.44)),
	}
}

func statCard(title, value, note, icon string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyle(r, g, b, 1.0).
				ImageScale(swiftui.ImageScaleMedium),
			swiftui.Spacer(),
			swiftui.Text(note).
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyleNamed("secondary"),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontSystem(26)).
				FontWeight(swiftui.WeightBold).
				MonospacedDigit(),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
	).Padding(12).
		MaxFrame(-1, 0).
		BackgroundStyle("regularMaterial").
		CornerRadius(16).
		Shadow(0, 0, 0, 0.10, 10, 0, 4)
}

func trafficPanel() swiftui.View {
	return panel("Weekly Snapshot",
		swiftui.VStackSpaced(10,
			swiftui.HStackSpaced(10,
				panelMetric("Visits", "18.4k"),
				panelMetric("Conversion", "4.8%"),
			),
			progressLine("Monday", 0.82, 0.27, 0.62, 1.0),
			progressLine("Tuesday", 0.63, 0.96, 0.47, 0.29),
		),
	)
}

func panelMetric(label, value string) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontTitle3).
				FontWeight(swiftui.WeightBold).
				MonospacedDigit(),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
	).Padding(8).
		MaxFrame(-1, 0).
		BackgroundStyle("thinMaterial").
		CornerRadius(12)
}

func progressLine(label string, value, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Text(percentString(value)).
				Font(swiftui.FontCaption).
				FontWeight(swiftui.WeightSemibold).
				MonospacedDigit(),
		),
		swiftui.FloatProgressView(swiftui.NewFloatState(value), 1.0).
			Tint(r, g, b, 1.0),
	)
}

func checklistPanel() swiftui.View {
	return panel("Release Checklist",
		swiftui.VStackSpaced(6,
			checklistRow("Copy approved", "checkmark.circle.fill", 0.28, 0.78, 0.44),
			swiftui.Divider(),
			checklistRow("Demo build signed", "checkmark.circle.fill", 0.28, 0.78, 0.44),
			swiftui.Divider(),
			checklistRow("Assets exported", "clock.fill", 0.96, 0.68, 0.24),
		),
	)
}

func checklistRow(label, icon string, r, g, b float64) swiftui.View {
	return swiftui.HStackSpaced(10,
		swiftui.Image(icon).
			ForegroundStyle(r, g, b, 1.0).
			ImageScale(swiftui.ImageScaleSmall),
		swiftui.Text(label).
			Font(swiftui.FontBody),
		swiftui.Spacer(),
	)
}

func activityPanel() swiftui.View {
	return panel("Recent Activity",
		swiftui.VStackSpaced(6,
			activityRow("New enterprise signup", "person.crop.circle.badge.plus"),
			swiftui.Divider(),
			activityRow("Invoice batch cleared", "creditcard.fill"),
		),
	)
}

func activityRow(label, icon string) swiftui.View {
	return swiftui.HStackSpaced(10,
		swiftui.Image(icon).
			ForegroundStyleNamed("secondary").
			ImageScale(swiftui.ImageScaleSmall),
		swiftui.Text(label).
			Font(swiftui.FontBody),
		swiftui.Spacer(),
	)
}

func actionsPanel() swiftui.View {
	return panel("Quick Actions",
		swiftui.HStackSpaced(8,
			actionButton("Export"),
			actionButton("Status"),
		),
	)
}

func actionButton(title string) swiftui.View {
	return swiftui.Button(title, func() {}).
		ButtonStyle(swiftui.ButtonStyleBordered).
		ControlSize(swiftui.ControlSizeRegular).
		MaxFrame(-1, 0)
}

func panel(title string, content swiftui.View) swiftui.View {
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontHeadline),
			swiftui.Spacer(),
		),
		content,
	).Padding(12).
		MaxFrame(-1, 0).
		BackgroundStyle("regularMaterial").
		CornerRadius(18).
		Shadow(0, 0, 0, 0.08, 10, 0, 4)
}

func percentString(v float64) string {
	return fmt.Sprintf("%.0f%%", v*100)
}
