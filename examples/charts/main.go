// Command charts demonstrates building data visualizations with SwiftUI from Go.
//
// It renders a bar chart, a gauge dashboard, metric cards, and progress
// indicators using shapes, layout composition, and SF Symbols.
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

type bar struct {
	label string
	value float64 // 0.0–1.0
	r, g, b float64
}

func main() {
	weekly := []bar{
		{"Mon", 0.60, 0.35, 0.65, 1.0},
		{"Tue", 0.82, 0.35, 0.65, 1.0},
		{"Wed", 0.45, 0.35, 0.65, 1.0},
		{"Thu", 0.93, 0.35, 0.65, 1.0},
		{"Fri", 0.71, 0.35, 0.65, 1.0},
		{"Sat", 0.30, 0.55, 0.55, 0.65},
		{"Sun", 0.18, 0.55, 0.55, 0.65},
	}

	swiftui.Run(swiftui.AppConfig{
		Title:  "Analytics Dashboard",
		Width:  700,
		Height: 650,
	}, swiftui.ScrollView(
		swiftui.VStackSpaced(20,
			// Header
			swiftui.HStack(
				swiftui.Text("Analytics").
					Font(swiftui.FontTitle).
					FontWeight(swiftui.WeightBold),
				swiftui.Spacer(),
				swiftui.Label("Live", "circle.fill").
					ForegroundStyle(0.3, 0.8, 0.4, 1.0).
					Font(swiftui.FontCaption),
			),

			// Metric cards
			swiftui.HStackSpaced(12,
				metricCard("person.2.fill", "Users", "2,847", "+12.5%", true),
				metricCard("chart.line.uptrend.xyaxis", "Revenue", "$48.2k", "+8.3%", true),
				metricCard("arrow.up.arrow.down", "Requests", "1.2M", "-2.1%", false),
				metricCard("clock.fill", "Latency", "42ms", "-15.0%", true),
			),

			// Bar chart
			swiftui.GroupBox("Weekly Activity",
				barChart(weekly, 160),
			).MaxFrame(-1, 0),

			// Bottom row: gauges + progress
			swiftui.HStackSpaced(12,
				swiftui.GroupBox("System Health",
					swiftui.VStackSpaced(16,
						swiftui.HStackSpaced(24,
							gauge("CPU", 0.72, "bolt.fill", 0.9, 0.5, 0.2),
							gauge("RAM", 0.45, "memorychip.fill", 0.3, 0.7, 1.0),
						),
						swiftui.HStackSpaced(24,
							gauge("Disk", 0.88, "internaldrive.fill", 0.9, 0.3, 0.3),
							gauge("Net", 0.31, "network", 0.3, 0.8, 0.4),
						),
					).Padding(12),
				).MaxFrame(-1, -1),

				swiftui.GroupBox("Pipeline",
					swiftui.VStackSpaced(14,
						progressRow("Build", "hammer.fill", 1.0),
						progressRow("Test", "checkmark.circle.fill", 0.85),
						progressRow("Lint", "text.magnifyingglass", 0.72),
						progressRow("Security", "lock.shield.fill", 0.58),
						progressRow("Deploy", "icloud.and.arrow.up.fill", 0.15),
					).Padding(12),
				).MaxFrame(-1, -1),
			),
		).Padding(24),
	))
}

func metricCard(icon, label, value, change string, positive bool) swiftui.View {
	cr, cg, cb := 0.9, 0.35, 0.35
	arrow := "arrow.down.right"
	if positive {
		cr, cg, cb = 0.3, 0.75, 0.4
		arrow = "arrow.up.right"
	}
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyleNamed("secondary").
				ImageScale(swiftui.ImageScaleSmall),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Label(change, arrow).
				Font(swiftui.FontCaption2).
				ForegroundStyle(cr, cg, cb, 1.0),
		),
	).Padding(12).
		Background(0.2, 0.2, 0.25, 0.5).
		CornerRadius(10)
}

func barChart(bars []bar, maxHeight float64) swiftui.View {
	var columns []swiftui.Viewable
	for _, b := range bars {
		columns = append(columns, swiftui.VStack(
			swiftui.Spacer(),
			swiftui.Text(fmt.Sprintf("%.0f", b.value*100)).
				Font(swiftui.FontCaption2).
				FontWeight(swiftui.WeightMedium).
				ForegroundStyleNamed("secondary"),
			swiftui.RoundedRectangle(4).
				Fill(b.r, b.g, b.b, 0.85).
				Frame(36, b.value*maxHeight).
				AsView(),
			swiftui.Text(b.label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("tertiary"),
		).MaxFrame(-1, 0))
	}
	return swiftui.HStackSpaced(6, columns...).
		MaxFrame(-1, 0).
		Padding(12)
}

func gauge(label string, value float64, icon string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.ZStack(
			swiftui.Circle().
				Stroke(r, g, b, 0.15, 8).
				Frame(64, 64).
				AsView(),
			swiftui.Circle().
				Fill(r, g, b, value*0.2).
				Frame(64, 64).
				AsView(),
			swiftui.VStack(
				swiftui.Image(icon).
					ForegroundStyle(r, g, b, 1.0).
					ImageScale(swiftui.ImageScaleSmall),
				swiftui.Text(fmt.Sprintf("%.0f%%", value*100)).
					Font(swiftui.FontCaption2).
					FontWeight(swiftui.WeightSemibold).
					ForegroundStyle(r, g, b, 1.0),
			),
		),
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary"),
	)
}

func progressRow(label, icon string, value float64) swiftui.View {
	r, g, b := 0.35, 0.65, 1.0
	if value >= 1.0 {
		r, g, b = 0.3, 0.8, 0.4
	} else if value < 0.3 {
		r, g, b = 0.9, 0.5, 0.2
	}
	return swiftui.HStack(
		swiftui.Image(icon).
			ForegroundStyle(r, g, b, 1.0).
			ImageScale(swiftui.ImageScaleSmall).
			Frame(16, 0),
		swiftui.Text(label).
			Font(swiftui.FontCallout).
			Frame(72, 0),
		swiftui.ProgressLinear(value, 1.0).
			Tint(r, g, b, 1.0),
		swiftui.Text(fmt.Sprintf("%.0f%%", value*100)).
			Font(swiftui.FontCaption).
			MonospacedDigit().
			ForegroundStyleNamed("secondary").
			Frame(36, 0),
	)
}
