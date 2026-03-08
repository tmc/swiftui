// Command animation demonstrates SwiftUI animations and transitions from Go.
//
// It showcases animated view transitions (slide, opacity, scale, push),
// animated state changes on a progress bar, and different animation curves
// (ease-in-out, spring, bouncy).
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
	scene := swiftui.NewIntState(0)
	progress := swiftui.NewFloatState(0.0)
	animKind := swiftui.NewIntState(0)

	swiftui.Run(swiftui.AppConfig{
		Title:  "Animations",
		Width:  600,
		Height: 700,
	}, swiftui.ScrollView(
		swiftui.VStackSpaced(20,
			// Header
			swiftui.Text("Animation Showcase").
				Font(swiftui.FontLargeTitle).
				FontWeight(swiftui.WeightBold),

			// Transition demos
			swiftui.GroupBox("Transitions",
				swiftui.VStackSpaced(12,
					transitionDemo("Slide", swiftui.TransitionSlide, scene, 0),
					transitionDemo("Opacity", swiftui.TransitionOpacity, scene, 1),
					transitionDemo("Scale", swiftui.TransitionScale, scene, 2),
					transitionDemo("Push", swiftui.TransitionPush, scene, 3),
				).Padding(8),
			).MaxFrame(-1, 0),

			// Progress bar with animated state
			swiftui.GroupBox("Animated Progress",
				swiftui.VStackSpaced(12,
					swiftui.FloatProgressView(progress, 1.0).
						Tint(0.35, 0.65, 1.0, 1.0),
					swiftui.HStackSpaced(8,
						swiftui.Button("0%", func() {
							progress.SetAnimated(0.0)
						}).ButtonStyle(swiftui.ButtonStyleBordered),
						swiftui.Button("25%", func() {
							progress.SetAnimated(0.25)
						}).ButtonStyle(swiftui.ButtonStyleBordered),
						swiftui.Button("50%", func() {
							progress.SetAnimated(0.5)
						}).ButtonStyle(swiftui.ButtonStyleBordered),
						swiftui.Button("75%", func() {
							progress.SetAnimated(0.75)
						}).ButtonStyle(swiftui.ButtonStyleBordered),
						swiftui.Button("100%", func() {
							progress.SetAnimated(1.0)
						}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
					),
				).Padding(8),
			).MaxFrame(-1, 0),

			// Animation curves
			swiftui.GroupBox("Animation Curves",
				swiftui.VStackSpaced(12,
					swiftui.AnimatedDynamicView(animKind, swiftui.TransitionScale, func(v int) swiftui.View {
						kinds := []string{"Ease In-Out", "Ease In", "Ease Out", "Spring", "Bouncy"}
						colors := [][3]float64{
							{0.35, 0.65, 1.0},
							{0.9, 0.5, 0.2},
							{0.3, 0.8, 0.4},
							{0.8, 0.3, 0.7},
							{1.0, 0.6, 0.2},
						}
						idx := v % len(kinds)
						c := colors[idx]
						return swiftui.VStack(
							swiftui.RoundedRectangle(12).
								Fill(c[0], c[1], c[2], 0.85).
								Frame(0, 60).
								AsView().
								MaxFrame(-1, 0),
							swiftui.Text(kinds[idx]).
								Font(swiftui.FontTitle2).
								FontWeight(swiftui.WeightSemibold).
								ForegroundStyle(c[0], c[1], c[2], 1.0),
						)
					}),
					swiftui.HStackSpaced(6,
						curveButton("Ease In-Out", swiftui.AnimationEaseInOut, animKind, 0),
						curveButton("Ease In", swiftui.AnimationEaseIn, animKind, 1),
						curveButton("Ease Out", swiftui.AnimationEaseOut, animKind, 2),
						curveButton("Spring", swiftui.AnimationSpring, animKind, 3),
						curveButton("Bouncy", swiftui.AnimationBouncy, animKind, 4),
					),
				).Padding(8),
			).MaxFrame(-1, 0),

			// Scene cycling demo
			swiftui.GroupBox("Scene Cycling",
				swiftui.VStackSpaced(12,
					swiftui.AnimatedDynamicView(scene, swiftui.TransitionSlide, func(v int) swiftui.View {
						return sceneView(v)
					}),
					swiftui.HStack(
						swiftui.Button("Previous", func() {
							scene.SetAnimated(scene.Get() - 1)
						}).ButtonStyle(swiftui.ButtonStyleBordered),
						swiftui.Spacer(),
						swiftui.TextFrom(scene).
							Font(swiftui.FontBody).
							MonospacedDigit(),
						swiftui.Spacer(),
						swiftui.Button("Next", func() {
							scene.SetAnimated(scene.Get() + 1)
						}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
					),
				).Padding(8),
			).MaxFrame(-1, 0),
		).Padding(24),
	))
}

func transitionDemo(label string, transition swiftui.Transition, state *swiftui.IntState, idx int) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			Font(swiftui.FontBody).
			FontWeight(swiftui.WeightMedium).
			Frame(70, 0),
		swiftui.AnimatedDynamicView(state, transition, func(v int) swiftui.View {
			val := v % 4
			colors := [][3]float64{
				{0.35, 0.65, 1.0},
				{0.9, 0.5, 0.2},
				{0.3, 0.8, 0.4},
				{0.8, 0.3, 0.7},
			}
			c := colors[abs(val)%len(colors)]
			return swiftui.RoundedRectangle(8).
				Fill(c[0], c[1], c[2], 0.85).
				AsView().
				MaxFrame(-1, 0).
				Frame(0, 36)
		}).MaxFrame(-1, 0),
		swiftui.Button("Cycle", func() {
			state.SetAnimated(state.Get() + 1)
		}).ButtonStyle(swiftui.ButtonStyleBordered),
	)
}

func curveButton(label string, kind swiftui.AnimationKind, state *swiftui.IntState, idx int) swiftui.View {
	return swiftui.Button(label, func() {
		state.SetAnimated(idx)
	}).ButtonStyle(swiftui.ButtonStyleBordered).
		Animation(kind)
}

func sceneView(v int) swiftui.View {
	scenes := []struct {
		title string
		icon  string
		r, g, b float64
	}{
		{"Welcome", "hand.wave.fill", 0.35, 0.65, 1.0},
		{"Settings", "gearshape.fill", 0.9, 0.5, 0.2},
		{"Profile", "person.fill", 0.3, 0.8, 0.4},
		{"Complete", "checkmark.circle.fill", 0.8, 0.3, 0.7},
	}
	idx := abs(v) % len(scenes)
	s := scenes[idx]
	return swiftui.VStack(
		swiftui.Image(s.icon).
			ForegroundStyle(s.r, s.g, s.b, 1.0).
			ImageScale(swiftui.ImageScaleLarge),
		swiftui.Text(s.title).
			Font(swiftui.FontTitle).
			FontWeight(swiftui.WeightBold).
			ForegroundStyle(s.r, s.g, s.b, 1.0),
		swiftui.Text(fmt.Sprintf("Scene %d", idx+1)).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
	).Padding(20).
		Frame(0, 120).
		MaxFrame(-1, 0)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
