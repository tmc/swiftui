//go:build darwin
// +build darwin

// Command dark-light showcases semantic foreground and background styles.
//
// It displays a card layout with SwiftUI's named styles — primary through
// quaternary foreground text and material backgrounds — rendered in two
// palettes that can be toggled with a switch.
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
	mode := swiftui.NewIntState(0)
	if // 0 = light palette, 1 = dark palette
	err := swiftui.Run(swiftui.AppConfig{
		Title:  "Dark & Light Styles",
		Width:  520,
		Height: 620,
	}, swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			// Header
			swiftui.Text("Theme Showcase").
				Font(swiftui.FontLargeTitle).
				FontWeight(swiftui.WeightBold),

			// Toggle between palettes
			swiftui.Toggle("Dark palette", mode, func() {}),

			swiftui.Divider(),

			// Dynamic content based on toggle
			swiftui.DynamicView(mode, func(v int) swiftui.View {
				if v == 1 {
					return darkPalette()
				}
				return lightPalette()
			}),
		).Padding(24),
	)); err !=

		// lightPalette shows semantic styles on light-toned backgrounds.
		nil {
		log.Fatal(err)
	}
}

func lightPalette() swiftui.View {
	return swiftui.VStackSpaced(14,
		swiftui.Text("Light Palette").
			Font(swiftui.FontTitle2).
			FontWeight(swiftui.WeightSemibold),

		// Foreground style tiers
		styleCard("Foreground Styles",
			swiftui.VStackSpaced(8,
				styledRow("primary", "Primary — full contrast"),
				styledRow("secondary", "Secondary — medium emphasis"),
				styledRow("tertiary", "Tertiary — low emphasis"),
				styledRow("quaternary", "Quaternary — minimal"),
			),
		).BackgroundStyle("regularMaterial"),

		// Material backgrounds
		materialCard("Material Backgrounds",
			0.95, 0.95, 0.97,
		),

		// Semantic colors
		colorCard("Semantic Colors — Light",
			0.96, 0.96, 0.98,
		),
	)
}

// darkPalette shows semantic styles on dark-toned backgrounds.
func darkPalette() swiftui.View {
	return swiftui.VStackSpaced(14,
		swiftui.Text("Dark Palette").
			Font(swiftui.FontTitle2).
			FontWeight(swiftui.WeightSemibold),

		// Foreground style tiers
		styleCard("Foreground Styles",
			swiftui.VStackSpaced(8,
				styledRow("primary", "Primary — full contrast"),
				styledRow("secondary", "Secondary — medium emphasis"),
				styledRow("tertiary", "Tertiary — low emphasis"),
				styledRow("quaternary", "Quaternary — minimal"),
			),
		).BackgroundStyle("thinMaterial"),

		// Material backgrounds
		materialCard("Material Backgrounds",
			0.15, 0.15, 0.18,
		),

		// Semantic colors
		colorCard("Semantic Colors — Dark",
			0.12, 0.12, 0.14,
		),
	)
}

// styledRow renders a single line of text using a named foreground style.
func styledRow(style, description string) swiftui.Viewable {
	return swiftui.HStack(
		swiftui.Image("circle.fill").
			ForegroundStyleNamed(style).
			ImageScale(swiftui.ImageScaleSmall),
		swiftui.Text(description).
			Font(swiftui.FontBody).
			ForegroundStyleNamed(style),
		swiftui.Spacer(),
	)
}

// styleCard wraps content in a labeled group box.
func styleCard(title string, content swiftui.View) swiftui.View {
	return swiftui.GroupBox(title, content).
		MaxFrame(-1, 0)
}

// materialCard demonstrates material background styles.
func materialCard(title string, bgR, bgG, bgB float64) swiftui.View {
	materials := []struct {
		name  string
		label string
	}{
		{"regularMaterial", "Regular Material"},
		{"thinMaterial", "Thin Material"},
		{"ultraThinMaterial", "Ultra Thin Material"},
		{"thickMaterial", "Thick Material"},
		{"windowBackground", "Window Background"},
	}

	var rows []swiftui.Viewable
	for _, m := range materials {
		rows = append(rows,
			swiftui.Text(m.label).
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("primary").
				Padding(10).
				MaxFrame(-1, 0).
				BackgroundStyle(m.name).
				CornerRadius(8),
		)
	}

	return swiftui.GroupBox(title,
		swiftui.VStackSpaced(6, rows...).
			Padding(4),
	).MaxFrame(-1, 0).
		Background(swiftui.RGBA(bgR, bgG, bgB, 1.0)).
		CornerRadius(12)
}

// colorCard demonstrates named foreground style colors in labeled swatches.
func colorCard(title string, bgR, bgG, bgB float64) swiftui.View {
	styles := []struct {
		name  string
		label string
	}{
		{"primary", "Primary"},
		{"secondary", "Secondary"},
		{"tertiary", "Tertiary"},
		{"quaternary", "Quaternary"},
	}

	var rows []swiftui.Viewable
	for _, s := range styles {
		rows = append(rows,
			swiftui.HStack(
				swiftui.RoundedRectangle(4).
					Fill(swiftui.RGBA(0.5, 0.5, 0.5, 1.0)).
					Frame(28, 28).
					ForegroundStyleNamed(s.name).
					AsView(),
				swiftui.Text(s.label).
					Font(swiftui.FontCallout).
					ForegroundStyleNamed(s.name),
				swiftui.Spacer(),
			),
		)
	}

	return swiftui.GroupBox(title,
		swiftui.VStackSpaced(6, rows...).
			Padding(4),
	).MaxFrame(-1, 0).
		Background(swiftui.RGBA(bgR, bgG, bgB, 1.0)).
		CornerRadius(12)
}
