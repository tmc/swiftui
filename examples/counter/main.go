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
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	count := swiftui.NewIntState(0)
	haloScale := swiftui.NewFloatState(1.0)

	bumpHalo := func(kind swiftui.AnimationKind) {
		haloScale.SetAnimatedWith(1.04, kind)
		time.AfterFunc(140*time.Millisecond, func() {
			haloScale.SetAnimatedWith(1.0, swiftui.AnimationEaseOut)
		})
	}

	adjust := func(delta int, kind swiftui.AnimationKind) {
		count.SetAnimatedWith(count.Get()+delta, kind)
		bumpHalo(kind)
	}

	reset := func() {
		count.SetAnimatedWith(0, swiftui.AnimationEaseInOut)
		bumpHalo(swiftui.AnimationEaseInOut)
	}
	if err := swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
		Title:  "Counter",
		Width:  380,
		Height: 360,
		Root: swiftui.VStackSpaced(18,
			swiftui.HStack(
				swiftui.VStackSpaced(4,
					swiftui.HStack(
						swiftui.Text("Counter").
							Font(swiftui.FontTitle2).
							FontWeight(swiftui.WeightBold),
						swiftui.Spacer(),
					),
					swiftui.HStack(
						swiftui.Text("A tiny reactive view with clear state transitions.").
							Font(swiftui.FontCallout).
							ForegroundStyleNamed("secondary"),
						swiftui.Spacer(),
					),
				).MaxFrame(-1, 0),
			),
			swiftui.ZStack(
				swiftui.AnimatedDynamicFloatView(haloScale, swiftui.TransitionScale, func(scale float64) swiftui.View {
					return swiftui.ZStack(
						swiftui.Circle().
							Fill(swiftui.RGBA(0.2, 0.45, 0.9, 0.09)).
							Frame(164, 164).
							AsView().
							ScaleEffect(scale),
						swiftui.Circle().
							Fill(swiftui.RGBA(0.2, 0.45, 0.9, 0.16)).
							Frame(132, 132).
							AsView().
							ScaleEffect((scale+1.0)/2.0),
					)
				}),
				swiftui.AnimatedDynamicView(count, swiftui.TransitionScale, func(v int) swiftui.View {
					return swiftui.Text(fmt.Sprintf("%d", v)).
						Font(swiftui.FontSystem(72)).
						FontWeight(swiftui.WeightBold).
						FontDesign(swiftui.DesignRounded).
						MonospacedDigit().
						AsView()
				}),
			),
			swiftui.HStackSpaced(12,
				swiftui.Button("Subtract", func() {
					adjust(-1, swiftui.AnimationEaseInOut)
				}).
					ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBordered),
				swiftui.Button("Reset", func() {
					reset()
				}).
					ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBordered),
				swiftui.Button("Add", func() {
					adjust(1, swiftui.AnimationSpring)
				}).
					ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBorderedProminent),
			),
			swiftui.AnimatedDynamicView(count, swiftui.TransitionOpacity, func(v int) swiftui.View {
				return counterFootnote(v)
			}),
		).Padding(24),
	}}}); err != nil {
		log.Fatal(err)
	}
}

func counterFootnote(v int) swiftui.View {
	message := "Balanced state with nowhere to hide."
	switch {
	case v > 0:
		message = "Positive values spring in with a little extra energy."
	case v < 0:
		message = "Negative values settle with the same UI rhythm."
	}
	return swiftui.Text(message).
		Font(swiftui.FontCaption).
		ForegroundStyleNamed("secondary").
		MultilineTextAlignment(swiftui.TextAlignmentCenter).
		AsView()
}
