//go:build darwin
// +build darwin

// Command glass demonstrates material and translucency effects in SwiftUI
// from Go, showcasing the liquid glass aesthetic on macOS 26+.
//
// It layers material backgrounds, frosted panels, and floating controls
// over a colorful gradient backdrop.
//
// Usage:
//
//	go run .
package main

import (
	"log"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	playing := swiftui.NewIntState(0)
	volume := swiftui.NewIntState(65)
	progress := swiftui.NewFloatState(0.35)
	if err := swiftui.Run(swiftui.AppConfig{
		Title:  "Glass",
		Width:  600,
		Height: 550,
	}, swiftui.ZStack(
		// Colorful gradient background
		background(),

		// Floating glass panels
		swiftui.VStackSpaced(16,
			// Top bar — ultra thin material
			swiftui.HStack(
				swiftui.Image("music.note").
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.9)).
					ImageScale(swiftui.ImageScaleLarge),
				swiftui.VStack(
					swiftui.Text("Now Playing").
						Font(swiftui.FontHeadline).
						ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.95)),
					swiftui.Text("Ambient Waves - Glass Sessions").
						Font(swiftui.FontSubheadline).
						ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.6)),
				),
				swiftui.Spacer(),
				swiftui.ButtonWithImage("ellipsis.circle", func() {}).
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.7)),
			).Padding(16).
				BackgroundStyle("ultraThinMaterial").
				CornerRadius(16),

			swiftui.Spacer(),

			// Center card — regular material
			swiftui.VStackSpaced(16,
				// Album art placeholder
				swiftui.ZStack(
					swiftui.RoundedRectangle(16).
						Fill(swiftui.RGBA(1, 1, 1, 0.08)).
						Frame(200, 200).
						AsView(),
					swiftui.VStack(
						swiftui.Image("waveform").
							ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.4)).
							ImageScale(swiftui.ImageScaleLarge),
						swiftui.Text("Waveform").
							Font(swiftui.FontCaption).
							ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.3)),
					),
				),

				// Track info
				swiftui.Text("Liquid Glass").
					Font(swiftui.FontTitle2).
					FontWeight(swiftui.WeightBold).
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.95)),
				swiftui.Text("Ambient Waves").
					Font(swiftui.FontBody).
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.5)),

				// Progress
				swiftui.FloatProgressView(progress, 1.0).
					Tint(swiftui.RGBA(1, 1, 1, 0.6)),

				// Playback controls
				swiftui.HStackSpaced(24,
					swiftui.ButtonWithImage("shuffle", func() {}).
						ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.5)),
					swiftui.ButtonWithImage("backward.fill", func() {}).
						ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.8)).
						ImageScale(swiftui.ImageScaleLarge),
					swiftui.ButtonWithImage(playIcon(playing), func() {
						if playing.Get() == 0 {
							playing.Set(1)
						} else {
							playing.Set(0)
						}
					}).ForegroundStyle(swiftui.RGBA(1, 1, 1, 1.0)).
						ImageScale(swiftui.ImageScaleLarge),
					swiftui.ButtonWithImage("forward.fill", func() {}).
						ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.8)).
						ImageScale(swiftui.ImageScaleLarge),
					swiftui.ButtonWithImage("repeat", func() {}).
						ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.5)),
				),
			).Padding(24).
				BackgroundStyle("regularMaterial").
				CornerRadius(20),

			swiftui.Spacer(),

			// Bottom bar — thin material with volume
			swiftui.HStackSpaced(12,
				swiftui.Image("speaker.fill").
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.5)).
					ImageScale(swiftui.ImageScaleSmall),
				swiftui.Slider("", volume, 0, 100, func() {}),
				swiftui.Image("speaker.wave.3.fill").
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.5)).
					ImageScale(swiftui.ImageScaleSmall),
				swiftui.Divider().Frame(1, 20),
				swiftui.ButtonWithImage("airplayaudio", func() {}).
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.5)),
				swiftui.ButtonWithImage("list.bullet", func() {}).
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.5)),
			).Padding(12).
				BackgroundStyle("thinMaterial").
				CornerRadius(14),
		).Padding(20),
	)); err != nil {
		log.Fatal(err)
	}
}

func background() swiftui.View {
	// Layered colored circles for a gradient-like backdrop
	return swiftui.ZStack(
		swiftui.ColorView(swiftui.RGBA(0.05, 0.02, 0.15, 1.0)),
		swiftui.Circle().
			Fill(swiftui.RGBA(0.4, 0.1, 0.7, 0.6)).
			Frame(300, 300).
			AsView().
			Offset(-100, -120).
			Blur(60),
		swiftui.Circle().
			Fill(swiftui.RGBA(0.1, 0.3, 0.8, 0.5)).
			Frame(250, 250).
			AsView().
			Offset(120, 80).
			Blur(50),
		swiftui.Circle().
			Fill(swiftui.RGBA(0.8, 0.2, 0.5, 0.4)).
			Frame(200, 200).
			AsView().
			Offset(-50, 150).
			Blur(40),
	)
}

func playIcon(state *swiftui.IntState) string {
	if state.Get() == 0 {
		return "play.fill"
	}
	return "pause.fill"
}
