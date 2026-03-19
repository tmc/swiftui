//go:build darwin
// +build darwin

// Command tabs demonstrates a polished tabbed workspace built with SwiftUI
// from Go.
//
// It combines overview cards, configurable preferences, and a concise about
// page to show how TabView can hold distinct but related product surfaces.
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
	darkMode := swiftui.NewIntState(1)
	fontSize := swiftui.NewIntState(15)
	compactMode := swiftui.NewIntState(0)
	notifications := swiftui.NewIntState(1)

	swiftui.Run(swiftui.AppConfig{
		Title:  "Tabs Demo",
		Width:  720,
		Height: 560,
	}, swiftui.TabView(
		homeTab(),
		settingsTab(darkMode, fontSize, compactMode, notifications),
		aboutTab(),
	))
}

func homeTab() swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(18,
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("Workspace").
						Font(swiftui.FontTitle).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
				),
				swiftui.HStack(
					swiftui.Text("A tabbed shell with overview cards, shared controls, and compact supporting detail.").
						Font(swiftui.FontCallout).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			),

			swiftui.HStackSpaced(12,
				summaryCard("bolt.fill", "Launch Time", "280 ms", "Cold start", 0.35, 0.65, 1.0),
				summaryCard("shippingbox.fill", "Bridges", "14", "Loaded overlays", 0.95, 0.55, 0.25),
				summaryCard("checkmark.seal.fill", "Checks", "26", "Example suite", 0.3, 0.8, 0.4),
			),

			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Flow",
					swiftui.VStackSpaced(10,
						flowRow("1", "Build in Go", "Compose native views with familiar structs and functions."),
						flowRow("2", "Bridge through purego", "State updates cross into Swift without cgo."),
						flowRow("3", "Render in SwiftUI", "macOS layout, animation, and controls stay native."),
					).Padding(10),
				).MaxFrame(-1, 0),
				swiftui.GroupBox("Active Surfaces",
					swiftui.VStackSpaced(10,
						infoRow("Overview", "High-level status and quick context"),
						infoRow("Preferences", "Controls that reshape the current workspace"),
						infoRow("About", "Purpose, constraints, and provenance"),
					).Padding(10),
				).MaxFrame(-1, 0),
			),

			swiftui.GroupBox("Architecture Snapshot",
				swiftui.HStackSpaced(18,
					architectureStep("Go", "State + layout"),
					architectureArrow(),
					architectureStep("Bridge", "Callbacks + views"),
					architectureArrow(),
					architectureStep("SwiftUI", "Native rendering"),
				).Padding(12),
			).MaxFrame(-1, 0),
		).Padding(24),
	).TabItem("Home", "square.grid.2x2.fill")
}

func settingsTab(darkMode, fontSize, compactMode, notifications *swiftui.IntState) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("Preferences").
						Font(swiftui.FontTitle2).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
				),
				swiftui.HStack(
					swiftui.Text("Form controls do not need to be bare. Group them around real decisions and visible outcomes.").
						Font(swiftui.FontCallout).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			),

			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Appearance",
					swiftui.VStackSpaced(12,
						swiftui.Toggle("Dark palette", darkMode, func() {}),
						swiftui.Slider("Font size", fontSize, 11, 24, func() {}),
						swiftui.Toggle("Compact density", compactMode, func() {}),
						infoRow("Preview", previewSummary(darkMode, fontSize, compactMode)),
					).Padding(10),
				).MaxFrame(-1, 0),

				swiftui.GroupBox("Behavior",
					swiftui.VStackSpaced(12,
						swiftui.Toggle("Notifications", notifications, func() {}),
						infoRow("Autosave", "Enabled every 30 seconds"),
						infoRow("Sync", "Local-first with remote replay"),
						infoRow("Keyboard", "Command palette + tab navigation"),
					).Padding(10),
				).MaxFrame(-1, 0),
			),

			swiftui.GroupBox("Current Selection",
				swiftui.VStackSpaced(10,
					infoRow("Theme", onOffLabel(darkMode.Get() != 0, "Dark", "Light")),
					infoRow("Type Scale", fmt.Sprintf("%d pt", fontSize.Get())),
					infoRow("Density", onOffLabel(compactMode.Get() != 0, "Compact", "Comfortable")),
					infoRow("Notifications", onOffLabel(notifications.Get() != 0, "Enabled", "Muted")),
				).Padding(10),
			).MaxFrame(-1, 0),
		).Padding(24),
	).TabItem("Settings", "slider.horizontal.3")
}

func aboutTab() swiftui.View {
	return swiftui.VStackSpaced(20,
		swiftui.Spacer(),
		swiftui.ZStack(
			swiftui.Circle().
				Fill(0.2, 0.45, 0.9, 0.16).
				Frame(132, 132).
				AsView(),
			swiftui.Circle().
				Stroke(0.2, 0.45, 0.9, 0.35, 1.5).
				Frame(132, 132).
				AsView(),
			swiftui.Image("swift").
				ForegroundStyle(0.3, 0.6, 1.0, 1.0).
				ImageScale(swiftui.ImageScaleLarge),
		),
		swiftui.Text("SwiftUI for Go").
			Font(swiftui.FontTitle).
			FontWeight(swiftui.WeightBold),
		swiftui.Text("Native macOS interfaces from plain Go.\nThe point is not novelty. The point is staying productive without giving up platform quality.").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary").
			MultilineTextAlignment(swiftui.TextAlignmentCenter),
		swiftui.HStackSpaced(12,
			aboutPillar("No cgo", "purego bridge"),
			aboutPillar("Native", "SwiftUI surfaces"),
			aboutPillar("Runnable", "go run examples"),
		),
		swiftui.Link("View Repository", "https://github.com/tmc/swiftui").
			ButtonStyle(swiftui.ButtonStyleBordered),
		swiftui.Spacer(),
	).Padding(40).TabItem("About", "info.circle.fill")
}

func summaryCard(icon, label, value, note string, r, g, b float64) swiftui.View {
	return swiftui.VStackSpaced(8,
		swiftui.HStack(
			swiftui.Image(icon).
				ForegroundStyle(r, g, b, 1.0).
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
		),
		swiftui.HStack(
			swiftui.Text(note).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(0.2, 0.2, 0.25, 0.45).
		CornerRadius(10)
}

func flowRow(step, title, body string) swiftui.View {
	return swiftui.HStackSpaced(10,
		swiftui.Text(step).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightBold).
			ForegroundStyle(0.3, 0.6, 1.0, 1.0).
			Frame(18, 18),
		swiftui.VStackSpaced(2,
			swiftui.HStack(
				swiftui.Text(title).
					Font(swiftui.FontCallout).
					FontWeight(swiftui.WeightSemibold),
				swiftui.Spacer(),
			),
			swiftui.HStack(
				swiftui.Text(body).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
				swiftui.Spacer(),
			),
		).MaxFrame(-1, 0),
	)
}

func architectureStep(title, body string) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.Text(title).
			Font(swiftui.FontHeadline).
			FontWeight(swiftui.WeightBold),
		swiftui.Text(body).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
	).MaxFrame(-1, 0)
}

func architectureArrow() swiftui.View {
	return swiftui.Image("arrow.right").
		ForegroundStyleNamed("tertiary").
		ImageScale(swiftui.ImageScaleSmall)
}

func aboutPillar(title, body string) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.Text(title).
			Font(swiftui.FontHeadline).
			FontWeight(swiftui.WeightSemibold),
		swiftui.Text(body).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
	).Padding(12).
		Background(0.2, 0.2, 0.25, 0.32).
		CornerRadius(10)
}

func previewSummary(darkMode, fontSize, compactMode *swiftui.IntState) string {
	theme := onOffLabel(darkMode.Get() != 0, "dark", "light")
	density := onOffLabel(compactMode.Get() != 0, "compact", "comfortable")
	return fmt.Sprintf("%s, %d pt, %s", theme, fontSize.Get(), density)
}

func onOffLabel(v bool, on, off string) string {
	if v {
		return on
	}
	return off
}

func infoRow(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			FontWeight(swiftui.WeightMedium),
	)
}
