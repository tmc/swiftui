//go:build darwin
// +build darwin

// Command markdown-editor demonstrates a split-pane text editor with live preview.
//
// The left pane is a TextEditor bound to a StringState. The right pane is a
// DynamicView that rebuilds whenever the refresh counter changes, rendering
// lines starting with # as titles, lines starting with - as labeled items,
// and everything else as body text.
//
// Usage:
//
//	go run .
package main

import (
	"log"
	"runtime"
	"strings"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	content := swiftui.NewStringState("# Welcome\n\nThis is a **markdown** editor.\n\n- Item one\n- Item two\n\n# Another Section\n\nWrite here and press Refresh to preview.")
	version := swiftui.NewIntState(0)
	if err := swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
		Title:  "Markdown Editor",
		Width:  900,
		Height: 600,
		Root: swiftui.VStack(
			// Toolbar
			swiftui.HStack(
				swiftui.Text("Markdown Editor").
					Font(swiftui.FontTitle2).
					FontWeight(swiftui.WeightBold),
				swiftui.Spacer(),
				swiftui.Button("Refresh Preview", func() {
					version.Set(version.Get() + 1)
				}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
			).Padding(12),
			swiftui.Divider(),
			// Split pane
			swiftui.HStack(
				// Left: editor
				swiftui.TextEditor(content).
					MaxFrame(-1, -1).
					Font(swiftui.FontBody),
				swiftui.Divider(),
				// Right: preview
				swiftui.DynamicView(version, func(_ int) swiftui.View {
					return renderPreview(content.Get())
				}).MaxFrame(-1, -1),
			).MaxFrame(-1, -1),
		).Padding(0),
	}}}); err != nil {
		log.Fatal(err)
	}
}

func renderPreview(text string) swiftui.View {
	lines := strings.Split(text, "\n")
	var views []swiftui.Viewable
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# "):
			views = append(views, swiftui.Text(strings.TrimPrefix(trimmed, "# ")).
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold))
		case strings.HasPrefix(trimmed, "- "):
			views = append(views, swiftui.Label(strings.TrimPrefix(trimmed, "- "), "circle.fill"))
		case trimmed == "":
			views = append(views, swiftui.Spacer().Frame(0, 8))
		default:
			views = append(views, swiftui.Text(trimmed).Font(swiftui.FontBody))
		}
	}
	return swiftui.ScrollView(
		swiftui.VStackSpaced(4, views...).
			MaxFrame(-1, 0).
			Padding(12),
	)
}
