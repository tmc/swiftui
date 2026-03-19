//go:build darwin
// +build darwin

// Command hello-world is a minimal but polished SwiftUI app from Go.
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
		Title:  "Hello World",
		Width:  500,
		Height: 320,
	}, swiftui.VStackSpaced(18,
		swiftui.Spacer(),
		swiftui.ZStack(
			swiftui.Circle().
				Fill(0.28, 0.55, 1.0, 0.14).
				Frame(112, 112).
				AsView(),
			swiftui.Image("hand.wave.fill").
				ForegroundStyle(0.35, 0.65, 1.0, 1.0).
				ImageScale(swiftui.ImageScaleLarge),
		),
		swiftui.Text("Hello from Go!").
			Font(swiftui.FontLargeTitle).
			FontWeight(swiftui.WeightBold),
		swiftui.Text("A native macOS window rendered through SwiftUI and driven from Go.").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary").
			MultilineTextAlignment(swiftui.TextAlignmentCenter),
		swiftui.Label("purego bridge", "sparkles").
			Font(swiftui.FontCaption).
			ForegroundStyle(0.3, 0.8, 0.4, 1.0),
		swiftui.Spacer(),
	).Padding(28))
}
