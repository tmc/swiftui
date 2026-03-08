//go:build darwin
// +build darwin

// Command tabs demonstrates TabView and NavigationStack from Go.
//
// It creates a tabbed interface with three tabs: a home view with navigation,
// a settings form, and an about page with shapes and styling.
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
	darkMode := swiftui.NewIntState(0)
	fontSize := swiftui.NewIntState(14)

	swiftui.Run(swiftui.AppConfig{
		Title:  "Tabs Demo",
		Width:  600,
		Height: 500,
	}, swiftui.TabView(
		homeTab(),
		settingsTab(darkMode, fontSize),
		aboutTab(),
	))
}

func homeTab() swiftui.View {
	return swiftui.NavigationStack(
		swiftui.List(
			swiftui.NavigationLink("Getting Started",
				swiftui.Text("SwiftUI from Go lets you build native macOS UIs\nwithout writing any Swift or Objective-C.").
					Font(swiftui.FontBody).
					Padding(20).
					AsView(),
			),
			swiftui.NavigationLink("Architecture",
				swiftui.VStackSpaced(12,
					swiftui.Text("Architecture").Font(swiftui.FontTitle),
					swiftui.Text("Go program").Font(swiftui.FontBody),
					swiftui.Image("arrow.down"),
					swiftui.Text("purego (cgo-free FFI)").Font(swiftui.FontBody),
					swiftui.Image("arrow.down"),
					swiftui.Text("Swift bridge dylib").Font(swiftui.FontBody),
					swiftui.Image("arrow.down"),
					swiftui.Text("SwiftUI framework").Font(swiftui.FontBody),
				).Padding(20),
			),
			swiftui.NavigationLink("State Management",
				swiftui.VStackSpaced(8,
					swiftui.Text("Reactive State").Font(swiftui.FontTitle),
					swiftui.Text("IntState, StringState, FloatState, ColorState, DateState, BoolState").
						Font(swiftui.FontBody).
						ForegroundStyleNamed("secondary"),
					swiftui.Text("Call .Set() from any goroutine.\nThe Swift bridge dispatches to MainActor.").
						Font(swiftui.FontCallout),
				).Padding(20),
			),
		).NavigationTitle("Home"),
	).TabItem("Home", "house.fill")
}

func settingsTab(darkMode *swiftui.IntState, fontSize *swiftui.IntState) swiftui.View {
	return swiftui.Form(
		swiftui.Section("Appearance",
			swiftui.VStack(
				swiftui.Toggle("Dark Mode", darkMode, func() {}),
				swiftui.Slider("Font Size", fontSize, 10, 24, func() {}),
			),
		),
		swiftui.Section("About",
			swiftui.VStack(
				swiftui.HStack(
					swiftui.Text("Version").ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
					swiftui.Text("1.0.0"),
				),
				swiftui.HStack(
					swiftui.Text("Framework").ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
					swiftui.Text("SwiftUI"),
				),
				swiftui.HStack(
					swiftui.Text("Runtime").ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
					swiftui.Text("purego"),
				),
			),
		),
	).TabItem("Settings", "gearshape.fill")
}

func aboutTab() swiftui.View {
	return swiftui.VStackSpaced(20,
		swiftui.Spacer(),
		swiftui.ZStack(
			swiftui.Circle().
				Fill(0.3, 0.6, 1.0, 0.2).
				Frame(120, 120).
				AsView(),
			swiftui.Image("swift").
				ForegroundStyle(0.3, 0.6, 1.0, 1.0).
				ImageScale(swiftui.ImageScaleLarge),
		),
		swiftui.Text("SwiftUI for Go").
			Font(swiftui.FontTitle).
			FontWeight(swiftui.WeightBold),
		swiftui.Text("Native macOS UIs from pure Go.\nNo cgo. No Xcode. Just go run.").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary").
			MultilineTextAlignment(swiftui.TextAlignmentCenter),
		swiftui.Link("View on GitHub", "https://github.com/tmc/swiftui").
			ButtonStyle(swiftui.ButtonStyleBordered),
		swiftui.Spacer(),
	).Padding(40).TabItem("About", "info.circle.fill")
}
