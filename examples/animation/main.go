//go:build darwin
// +build darwin

// Command animation demonstrates SwiftUI animations and transitions from Go.
//
// It showcases animated view transitions, curve-controlled state changes,
// transform-driven scenes, layered card arrangements, live banner updates,
// and animated progress changes.
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
	moveScene := swiftui.NewIntState(0)
	opacityScene := swiftui.NewIntState(0)
	scaleScene := swiftui.NewIntState(0)
	pushScene := swiftui.NewIntState(0)
	curveScene := swiftui.NewIntState(0)
	transformScene := swiftui.NewIntState(0)
	stackScene := swiftui.NewIntState(0)
	bannerScene := swiftui.NewIntState(0)
	sceneCycle := swiftui.NewIntState(0)
	progress := swiftui.NewFloatState(0.0)
	if err := swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
		Title:  "Animations",
		Width:  600,
		Height: 860,
		Root: swiftui.ScrollView(
			swiftui.VStackSpaced(20,
				// Header
				swiftui.Text("Animation Showcase").
					Font(swiftui.FontLargeTitle).
					FontWeight(swiftui.WeightBold),

				// Transition demos
				swiftui.GroupBox("Transitions",
					swiftui.VStackSpaced(12,
						transitionDemo("Move", swiftui.TransitionMove, moveScene),
						transitionDemo("Opacity", swiftui.TransitionOpacity, opacityScene),
						transitionDemo("Scale", swiftui.TransitionScale, scaleScene),
						transitionDemo("Push", swiftui.TransitionPush, pushScene),
					).Padding(8),
				).MaxFrame(-1, 0),

				// Progress bar with animated state
				swiftui.GroupBox("Animated Progress",
					swiftui.VStackSpaced(12,
						swiftui.FloatProgressView(progress, 1.0).
							Tint(swiftui.RGBA(0.35, 0.65, 1.0, 1.0)),
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

				// Curve-controlled state changes
				swiftui.GroupBox("Animation Curves",
					swiftui.VStackSpaced(12,
						swiftui.Text("Each button drives the same scene change with a different animation curve.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.AnimatedDynamicView(curveScene, swiftui.TransitionScale, curveView),
						swiftui.HStackSpaced(6,
							curveButton("Ease In-Out", curveScene, swiftui.AnimationEaseInOut),
							curveButton("Ease In", curveScene, swiftui.AnimationEaseIn),
							curveButton("Ease Out", curveScene, swiftui.AnimationEaseOut),
							curveButton("Spring", curveScene, swiftui.AnimationSpring),
							curveButton("Bouncy", curveScene, swiftui.AnimationBouncy),
						),
					).Padding(8),
				).MaxFrame(-1, 0),

				// Transform-driven scenes
				swiftui.GroupBox("Transform Presets",
					swiftui.VStackSpaced(12,
						swiftui.Text("Scale, rotation, offset, opacity, and shadow combined into scene presets.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.AnimatedDynamicView(transformScene, swiftui.TransitionScale, transformView),
						swiftui.HStackSpaced(6,
							sceneButton("Docked", transformScene, 0),
							sceneButton("Lifted", transformScene, 1),
							sceneButton("Tilted", transformScene, 2),
							sceneButton("Orbit", transformScene, 3),
							sceneButton("Burst", transformScene, 4),
						),
					).Padding(8),
				).MaxFrame(-1, 0),

				// Layered cards
				swiftui.GroupBox("Layered Cards",
					swiftui.VStackSpaced(12,
						swiftui.Text("Complex stacks stay readable when motion also carries hierarchy.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.AnimatedDynamicView(stackScene, swiftui.TransitionPush, stackView),
						swiftui.HStackSpaced(6,
							sceneButton("Stack", stackScene, 0),
							sceneButton("Fan", stackScene, 1),
							sceneButton("Spread", stackScene, 2),
							sceneButton("Spotlight", stackScene, 3),
						),
					).Padding(8),
				).MaxFrame(-1, 0),

				// Banners
				swiftui.GroupBox("Live Banners",
					swiftui.VStackSpaced(12,
						swiftui.Text("Push-style transitions work well for transient status and build feedback.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.AnimatedDynamicView(bannerScene, swiftui.TransitionPush, bannerView),
						swiftui.HStackSpaced(6,
							sceneButton("Preview", bannerScene, 0),
							sceneButton("Success", bannerScene, 1),
							sceneButton("Review", bannerScene, 2),
							sceneButton("Offline", bannerScene, 3),
						),
					).Padding(8),
				).MaxFrame(-1, 0),

				// Scene cycling demo
				swiftui.GroupBox("Scene Cycling",
					swiftui.VStackSpaced(12,
						swiftui.AnimatedDynamicView(sceneCycle, swiftui.TransitionMove, func(v int) swiftui.View {
							return sceneView(v)
						}),
						swiftui.HStack(
							swiftui.Button("Previous", func() {
								sceneCycle.SetAnimated(sceneCycle.Get() - 1)
							}).ButtonStyle(swiftui.ButtonStyleBordered),
							swiftui.Spacer(),
							swiftui.TextFrom(sceneCycle).
								Font(swiftui.FontBody).
								MonospacedDigit(),
							swiftui.Spacer(),
							swiftui.Button("Next", func() {
								sceneCycle.SetAnimated(sceneCycle.Get() + 1)
							}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
						),
					).Padding(8),
				).MaxFrame(-1, 0),
			).Padding(24),
		)}}}); err != nil {
		log.Fatal(err)
	}
}

func transitionDemo(label string, transition swiftui.Transition, state *swiftui.IntState) swiftui.View {
	preview := swiftui.ZStack(
		swiftui.RoundedRectangle(8).
			Fill(swiftui.RGBA(1.0, 1.0, 1.0, 0.08)).
			Frame(180, 36).
			AsView(),
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
				Fill(swiftui.RGBA(c[0], c[1], c[2], 0.85)).
				AsView().
				Frame(180, 36)
		}),
	).Frame(180, 36)

	return swiftui.HStack(
		swiftui.Text(label).
			Font(swiftui.FontBody).
			FontWeight(swiftui.WeightMedium).
			Frame(70, 0),
		preview,
		swiftui.Button("Cycle", func() {
			state.SetAnimated(state.Get() + 1)
		}).ButtonStyle(swiftui.ButtonStyleBordered),
	)
}

func sceneButton(label string, state *swiftui.IntState, idx int) swiftui.View {
	return swiftui.Button(label, func() {
		state.SetAnimated(idx)
	}).ButtonStyle(swiftui.ButtonStyleBordered)
}

func curveButton(label string, state *swiftui.IntState, kind swiftui.AnimationKind) swiftui.View {
	return swiftui.Button(label, func() {
		state.SetAnimatedWith(state.Get()+1, kind)
	}).ButtonStyle(swiftui.ButtonStyleBordered)
}

func curveView(v int) swiftui.View {
	presets := []struct {
		title   string
		icon    string
		r, g, b float64
	}{
		{"Glide", "wind", 0.35, 0.65, 1.0},
		{"Snap", "bolt.fill", 0.95, 0.62, 0.22},
		{"Bloom", "sparkles", 0.90, 0.42, 0.78},
		{"Launch", "paperplane.fill", 0.32, 0.78, 0.45},
		{"Land", "checkmark.circle.fill", 0.42, 0.72, 1.0},
	}
	p := presets[abs(v)%len(presets)]

	return swiftui.HStackSpaced(14,
		swiftui.ZStack(
			swiftui.Circle().
				Fill(swiftui.RGBA(p.r, p.g, p.b, 0.18)).
				Frame(52, 52).
				AsView(),
			swiftui.Image(p.icon).
				ForegroundStyle(swiftui.RGBA(p.r, p.g, p.b, 1.0)).
				ImageScale(swiftui.ImageScaleLarge),
		).Frame(52, 52),
		swiftui.VStackSpaced(4,
			swiftui.Text(p.title).
				Font(swiftui.FontTitle3).
				FontWeight(swiftui.WeightBold),
			swiftui.Text(fmt.Sprintf("Scene %d", abs(v)%len(presets)+1)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
		),
		swiftui.Spacer(),
	).
		Padding(16).
		BackgroundRoundedRect(swiftui.RGBA(0.12, 0.15, 0.20, 0.96), 18).
		Overlay(
			swiftui.RoundedRectangle(18).
				Stroke(swiftui.RGBA(p.r, p.g, p.b, 0.30), 1.2).
				AsView(),
		).
		MaxFrame(-1, 0)
}

func transformView(v int) swiftui.View {
	presets := []struct {
		title    string
		subtitle string
		icon     string
		r, g, b  float64
		scale    float64
		rotation float64
		x, y     float64
		opacity  float64
		shadow   float64
	}{
		{"Docked", "Balanced and ready for input", "bolt.fill", 0.35, 0.65, 1.0, 1.00, 0, 0, 0, 1.00, 8},
		{"Lifted", "Extra depth for emphasis", "paperplane.fill", 0.48, 0.74, 1.0, 1.05, -4, 0, -10, 1.00, 18},
		{"Tilted", "Directional motion and urgency", "triangle.fill", 0.98, 0.62, 0.22, 1.08, 12, 14, -6, 1.00, 20},
		{"Orbit", "Offset and rotation suggest travel", "sparkles", 0.90, 0.42, 0.78, 0.96, -18, 24, -16, 0.94, 18},
		{"Burst", "Scaled up with softer opacity", "wand.and.stars", 1.00, 0.74, 0.28, 1.14, 6, 0, -12, 0.90, 24},
	}
	p := presets[abs(v)%len(presets)]

	return swiftui.ZStack(
		swiftui.RoundedRectangle(24).
			Fill(swiftui.RGBA(0.12, 0.15, 0.20, 0.96)).
			Frame(360, 190).
			AsView().
			Overlay(
				swiftui.RoundedRectangle(24).
					Stroke(swiftui.RGBA(p.r, p.g, p.b, 0.35), 1.5).
					AsView(),
			),
		swiftui.VStackSpaced(10,
			swiftui.ZStack(
				swiftui.Circle().
					Fill(swiftui.RGBA(p.r, p.g, p.b, 0.18)).
					Frame(108, 108).
					AsView(),
				swiftui.Circle().
					Fill(swiftui.RGBA(p.r, p.g, p.b, 0.32)).
					Frame(78, 78).
					AsView(),
				swiftui.Image(p.icon).
					ForegroundStyle(swiftui.RGBA(p.r, p.g, p.b, 1.0)).
					ImageScale(swiftui.ImageScaleLarge).
					ScaleEffect(2.1),
			).Frame(120, 120),
			swiftui.Text(p.title).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold),
			swiftui.Text(p.subtitle).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
		),
	).
		Frame(360, 190).
		ScaleEffect(p.scale).
		RotationEffect(p.rotation).
		Offset(p.x, p.y).
		Opacity(p.opacity).
		Shadow(swiftui.RGBA(0, 0, 0, 0.24), p.shadow, 0, 10).
		MaxFrame(-1, 0)
}

func stackView(v int) swiftui.View {
	type pose struct {
		title    string
		subtitle string
		r, g, b  float64
		x, y     float64
		rotation float64
		scale    float64
		z        float64
	}

	layouts := [][]pose{
		{
			{"Discover", "Input", 0.35, 0.65, 1.0, 0, 0, 0, 1.00, 3},
			{"Compose", "Transform", 0.90, 0.55, 0.22, 0, 0, 0, 0.94, 2},
			{"Ship", "Output", 0.32, 0.78, 0.45, 0, 0, 0, 0.88, 1},
		},
		{
			{"Discover", "Input", 0.35, 0.65, 1.0, -62, 8, -12, 0.96, 1},
			{"Compose", "Transform", 0.90, 0.55, 0.22, 0, -8, 0, 1.00, 3},
			{"Ship", "Output", 0.32, 0.78, 0.45, 62, 8, 12, 0.96, 2},
		},
		{
			{"Discover", "Input", 0.35, 0.65, 1.0, -78, -4, -8, 0.92, 1},
			{"Compose", "Transform", 0.90, 0.55, 0.22, 0, 0, 0, 1.00, 3},
			{"Ship", "Output", 0.32, 0.78, 0.45, 78, 10, 8, 0.92, 2},
		},
		{
			{"Discover", "Input", 0.35, 0.65, 1.0, -36, 20, -6, 0.88, 1},
			{"Compose", "Transform", 0.90, 0.55, 0.22, 0, -12, 0, 1.05, 4},
			{"Ship", "Output", 0.32, 0.78, 0.45, 42, 24, 7, 0.86, 2},
		},
	}

	layout := layouts[abs(v)%len(layouts)]
	layers := make([]swiftui.Viewable, 0, len(layout))
	for _, p := range layout {
		card := swiftui.ZStack(
			swiftui.RoundedRectangle(20).
				Fill(swiftui.RGBA(p.r, p.g, p.b, 0.95)).
				Frame(240, 132).
				AsView(),
			swiftui.VStackSpaced(6,
				swiftui.Text(p.title).
					Font(swiftui.FontTitle3).
					FontWeight(swiftui.WeightBold).
					ForegroundStyle(swiftui.RGBA(1.0, 1.0, 1.0, 1.0)),
				swiftui.Text(p.subtitle).
					Font(swiftui.FontCaption).
					ForegroundStyle(swiftui.RGBA(1.0, 1.0, 1.0, 0.82)),
			),
		).
			Frame(240, 132).
			ScaleEffect(p.scale).
			RotationEffect(p.rotation).
			Offset(p.x, p.y).
			Shadow(swiftui.RGBA(0, 0, 0, 0.18), 18, 0, 10).
			ZIndex(p.z)
		layers = append(layers, card)
	}

	return swiftui.ZStack(layers...).Frame(360, 190).MaxFrame(-1, 0)
}

func bannerView(v int) swiftui.View {
	banners := []struct {
		title    string
		message  string
		tag      string
		icon     string
		r, g, b  float64
		tagAlpha float64
	}{
		{"Preview Ready", "Choose a status below to animate the banner between states.", "LIVE", "sparkles", 0.35, 0.65, 1.0, 0.22},
		{"Build Succeeded", "Universal dylibs finished and the release bundle is ready.", "DONE", "checkmark.circle.fill", 0.32, 0.78, 0.45, 0.28},
		{"Needs Review", "The example gallery still needs screenshots before shipping.", "TODO", "exclamationmark.triangle.fill", 0.98, 0.68, 0.24, 0.28},
		{"Offline Mode", "Queued network work will resume automatically when the link returns.", "PAUSED", "wifi.slash", 0.90, 0.38, 0.32, 0.26},
	}
	b := banners[abs(v)%len(banners)]

	return swiftui.HStackSpaced(12,
		swiftui.ZStack(
			swiftui.Circle().
				Fill(swiftui.RGBA(b.r, b.g, b.b, 0.18)).
				Frame(42, 42).
				AsView(),
			swiftui.Image(b.icon).
				ForegroundStyle(swiftui.RGBA(b.r, b.g, b.b, 1.0)).
				ImageScale(swiftui.ImageScaleLarge),
		).Frame(42, 42),
		swiftui.VStackSpaced(4,
			swiftui.Text(b.title).
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightBold),
			swiftui.Text(b.message).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
		),
		swiftui.Spacer(),
		swiftui.Text(b.tag).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold).
			Padding(8).
			BackgroundRoundedRect(swiftui.RGBA(b.r, b.g, b.b, b.tagAlpha), 10),
	).
		Padding(14).
		BackgroundRoundedRect(swiftui.RGBA(0.12, 0.15, 0.20, 0.96), 18).
		Overlay(
			swiftui.RoundedRectangle(18).
				Stroke(swiftui.RGBA(b.r, b.g, b.b, 0.30), 1.2).
				AsView(),
		).
		Shadow(swiftui.RGBA(0, 0, 0, 0.14), 12, 0, 6).
		MaxFrame(-1, 0)
}

func sceneView(v int) swiftui.View {
	scenes := []struct {
		title   string
		icon    string
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
			ForegroundStyle(swiftui.RGBA(s.r, s.g, s.b, 1.0)).
			ImageScale(swiftui.ImageScaleLarge),
		swiftui.Text(s.title).
			Font(swiftui.FontTitle).
			FontWeight(swiftui.WeightBold).
			ForegroundStyle(swiftui.RGBA(s.r, s.g, s.b, 1.0)),
		swiftui.Text(fmt.Sprintf("Scene %d", idx+1)).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
	).Padding(20).
		Frame(220, 120).
		MaxFrame(-1, 0)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
