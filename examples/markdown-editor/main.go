//go:build darwin
// +build darwin

// Command markdown-editor demonstrates a split-pane text editor with live preview.
//
// The left pane is a TextEditor bound to a StringState. The right pane is a
// DynamicView that rebuilds whenever the refresh counter changes, rendering
// the curated Markdown surface with native text selection enabled.
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
	content := swiftui.NewStringState("# Welcome\n\nThis is a **markdown** editor.\n\n- Item one\n- Item two\n\n# Another Section\n\nWrite here and press Refresh to preview.")
	version := swiftui.NewIntState(0)

	swiftui.Run(swiftui.AppConfig{
		Title:  "Markdown Editor",
		Width:  900,
		Height: 600,
	}, swiftui.VStack(
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
				Padding(8).
				MaxFrame(-1, -1).
				ScrollContentBackgroundHidden().
				BackgroundRoundedRect(0.14, 0.15, 0.18, 0.98, 14).
				ClipRoundedRect(14).
				Font(swiftui.FontBody),
			swiftui.Divider(),
			// Right: preview
			swiftui.DynamicView(version, func(_ int) swiftui.View {
				return renderPreview(content.Get())
			}).MaxFrame(-1, -1),
		).MaxFrame(-1, -1),
	).Padding(0))
}

func renderPreview(text string) swiftui.View {
	return swiftui.ScrollView(
		swiftui.VStack(
			swiftui.Markdown(text),
		).
			MaxFrame(-1, 0).
			Padding(12),
	)
}
