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
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	swiftui.Run(swiftui.AppConfig{
		Title:  "Layout Demo",
		Width:  600,
		Height: 500,
	}, swiftui.VStackSpaced(16,
		// Header
		swiftui.Text("Dashboard").
			Font(swiftui.FontLargeTitle).
			FontWeight(swiftui.WeightBold),

		// Stats row — each card expands to fill
		swiftui.HStackSpaced(12,
			statCard("Users", "42", 0.3, 0.7, 1.0),
			statCard("Events", "128", 0.9, 0.4, 0.3),
			statCard("Revenue", "$9.8k", 0.3, 0.8, 0.4),
		),

		// Chart placeholder — MaxFrame(-1, 0) expands width to fill
		swiftui.ZStack(
			swiftui.RoundedRectangle(12).
				Fill(0.15, 0.15, 0.2, 1.0).
				AsView(),
			swiftui.VStack(
				swiftui.Image("chart.bar.fill").
					ForegroundStyle(0.5, 0.5, 0.6, 1.0).
					ImageScale(swiftui.ImageScaleLarge),
				swiftui.Text("Chart Area").
					ForegroundStyleNamed("secondary"),
			),
		).MaxFrame(-1, 0).Frame(0, 150),

		// Bottom row — each GroupBox expands equally
		swiftui.HStackSpaced(12,
			swiftui.GroupBox("Recent",
				swiftui.VStack(
					swiftui.Label("New signup", "person.fill"),
					swiftui.Divider(),
					swiftui.Label("Payment received", "creditcard.fill"),
					swiftui.Divider(),
					swiftui.Label("Report generated", "doc.fill"),
				),
			).MaxFrame(-1, 0),
			swiftui.GroupBox("Quick Actions",
				swiftui.VStackSpaced(8,
					swiftui.Button("Export Data", func() {}).
						ButtonStyle(swiftui.ButtonStyleBordered).
						ControlSize(swiftui.ControlSizeSmall).
						MaxFrame(-1, 0),
					swiftui.Button("Send Report", func() {}).
						ButtonStyle(swiftui.ButtonStyleBordered).
						ControlSize(swiftui.ControlSizeSmall).
						MaxFrame(-1, 0),
					swiftui.Button("Settings", func() {}).
						ButtonStyle(swiftui.ButtonStyleBordered).
						ControlSize(swiftui.ControlSizeSmall).
						MaxFrame(-1, 0),
				),
			).MaxFrame(-1, 0),
		),
		swiftui.Spacer(),
	).Padding(24))
}

func statCard(title, value string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.Text(value).
			Font(swiftui.FontSystem(28)).
			FontWeight(swiftui.WeightBold).
			ForegroundStyle(r, g, b, 1.0),
		swiftui.Text(title).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
	).Padding(16).
		MaxFrame(-1, 0).
		Background(r, g, b, 0.1).
		CornerRadius(12)
}
