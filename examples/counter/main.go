//go:build darwin
// +build darwin

// Command counter demonstrates reactive state management with SwiftUI from Go.
//
// It displays a count that increments and decrements via buttons, with the
// display updating automatically through IntState binding.
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
	count := swiftui.NewIntState(0)

	swiftui.Run(swiftui.AppConfig{
		Title:  "Counter",
		Width:  300,
		Height: 200,
	}, swiftui.VStack(
		swiftui.TextFrom(count).
			Font(swiftui.FontSystem(72)).
			FontWeight(swiftui.WeightBold).
			FontDesign(swiftui.DesignRounded).
			MonospacedDigit(),
		swiftui.Spacer(),
		swiftui.HStackSpaced(20,
			swiftui.Button("-", func() {
				count.Set(count.Get() - 1)
			}).ControlSize(swiftui.ControlSizeLarge).
				ButtonStyle(swiftui.ButtonStyleBordered),
			swiftui.Button("+", func() {
				count.Set(count.Get() + 1)
			}).ControlSize(swiftui.ControlSizeLarge).
				ButtonStyle(swiftui.ButtonStyleBorderedProminent),
		),
	).Padding(30))
}
