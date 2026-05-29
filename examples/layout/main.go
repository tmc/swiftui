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
	"log"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	if err := swiftui.Run(swiftui.AppConfig{
		Title:  "Layout Demo",
		Width:  720,
		Height: 600,
	}, swiftui.ZStack(
		background(),
		swiftui.ScrollView(
			swiftui.VStackSpaced(12,
				header(),
				swiftui.HStackSpaced(12,
					statCard("Active Users", "42", "This week", "person.2.fill", 0.27, 0.62, 1.0),
					statCard("Open Events", "128", "Current total", "bolt.fill", 0.96, 0.47, 0.29),
					statCard("Monthly Revenue", "$9.8k", "Month to date", "banknote.fill", 0.28, 0.78, 0.44),
				),
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
	)); err != nil {
		log.Fatal(err)
	}
}

func background() swiftui.View {
	return swiftui.ZStack(
		swiftui.ColorView(swiftui.RGBA(0.07, 0.08, 0.10, 1.0)),
		swiftui.Circle().
			Fill(swiftui.RGBA(0.20, 0.34, 0.52, 0.08)).
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

func statCard(title, value, note, icon string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyle(swiftui.RGBA(r, g, b, 1.0)).
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
				MonospacedDigit().
				ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.96)),
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
		Background(swiftui.RGBA(0.11, 0.13, 0.16, 0.92)).
		CornerRadius(16).
		Shadow(swiftui.RGBA(0, 0, 0, 0.10), 10, 0, 4)
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
		Background(swiftui.RGBA(1, 1, 1, 0.04)).
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
			Tint(swiftui.RGBA(r, g, b, 1.0)),
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
			ForegroundStyle(swiftui.RGBA(r, g, b, 1.0)).
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
				Font(swiftui.FontHeadline).
				ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.94)),
			swiftui.Spacer(),
		),
		content,
	).Padding(12).
		MaxFrame(-1, 0).
		Background(swiftui.RGBA(0.10, 0.12, 0.15, 0.92)).
		CornerRadius(18).
		Shadow(swiftui.RGBA(0, 0, 0, 0.08), 10, 0, 4)
}

func percentString(v float64) string {
	return fmt.Sprintf("%.0f%%", v*100)
}
