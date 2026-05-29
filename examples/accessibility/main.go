//go:build darwin
// +build darwin

// Command accessibility demonstrates SwiftUI accessibility modifiers from Go.
//
// It showcases AccessibilityLabel, AccessibilityHint, and AccessibilityHidden
// applied to interactive elements, images, and decorative views. Each section
// explains what the modifier does and shows it in use.
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
	name := swiftui.NewStringState("")
	volume := swiftui.NewIntState(50)
	if err := swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
		Title:  "Accessibility",
		Width:  520,
		Height: 680,
		Root: swiftui.ScrollView(
			swiftui.VStackSpaced(16,
				// Title
				swiftui.Text("Accessibility Showcase").
					Font(swiftui.FontTitle).
					FontWeight(swiftui.WeightBold),

				// --- AccessibilityLabel ---
				swiftui.GroupBox("AccessibilityLabel",
					swiftui.VStackSpaced(10,
						swiftui.Text("Overrides the default VoiceOver description. Use it to give meaningful names to icons and images.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.Divider().AccessibilityHidden(true),
						swiftui.Image("heart.fill").
							ImageScale(swiftui.ImageScaleLarge).
							ForegroundStyle(swiftui.RGBA(0.9, 0.3, 0.3, 1.0)).
							AccessibilityLabel("Favorite heart icon"),
						swiftui.Image("globe").
							ImageScale(swiftui.ImageScaleLarge).
							ForegroundStyle(swiftui.RGBA(0.3, 0.6, 1.0, 1.0)).
							AccessibilityLabel("Globe representing international content"),
					).Padding(8),
				).MaxFrame(-1, 0),

				// --- AccessibilityHint ---
				swiftui.GroupBox("AccessibilityHint",
					swiftui.VStackSpaced(10,
						swiftui.Text("Provides extra context about what happens when an element is activated. VoiceOver reads the hint after a brief pause.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.Divider().AccessibilityHidden(true),
						swiftui.Button("Save Document", func() {}).
							ButtonStyle(swiftui.ButtonStyleBorderedProminent).
							ControlSize(swiftui.ControlSizeLarge).
							AccessibilityLabel("Save").
							AccessibilityHint("Saves the current document to disk"),
						swiftui.Button("Delete All", func() {}).
							ButtonStyle(swiftui.ButtonStyleBordered).
							ForegroundStyle(swiftui.RGBA(0.9, 0.3, 0.3, 1.0)).
							AccessibilityLabel("Delete all items").
							AccessibilityHint("Permanently removes all items from the list"),
					).Padding(8),
				).MaxFrame(-1, 0),

				// --- AccessibilityHidden ---
				swiftui.GroupBox("AccessibilityHidden",
					swiftui.VStackSpaced(10,
						swiftui.Text("Removes purely decorative elements from the accessibility tree so VoiceOver skips them entirely.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.Divider().AccessibilityHidden(true),
						swiftui.HStack(
							swiftui.Text("Decorative dividers below are hidden from VoiceOver:").
								Font(swiftui.FontFootnote),
							swiftui.Spacer(),
						),
						swiftui.Divider().AccessibilityHidden(true),
						swiftui.Divider().AccessibilityHidden(true),
						swiftui.Divider().AccessibilityHidden(true),
						swiftui.Image("sparkles").
							ImageScale(swiftui.ImageScaleLarge).
							ForegroundStyle(swiftui.RGBA(0.9, 0.75, 0.2, 1.0)).
							AccessibilityHidden(true),
						swiftui.Text("The sparkle icon above is decorative and hidden from assistive technology.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
					).Padding(8),
				).MaxFrame(-1, 0),

				// --- Form with labeled controls ---
				swiftui.GroupBox("Accessible Form Controls",
					swiftui.VStackSpaced(10,
						swiftui.Text("Form controls with accessibility labels provide clear descriptions for screen readers.").
							Font(swiftui.FontCaption).
							ForegroundStyleNamed("secondary"),
						swiftui.Divider().AccessibilityHidden(true),
						swiftui.TextField("Enter your name", name, func() {}).
							AccessibilityLabel("Full name").
							AccessibilityHint("Type your first and last name"),
						swiftui.Slider("Volume", volume, 0, 100, func() {}).
							AccessibilityLabel("Volume control").
							AccessibilityHint("Adjust the playback volume from 0 to 100"),
						swiftui.Label("Settings", "gear").
							AccessibilityLabel("Application settings").
							AccessibilityHint("Opens the settings panel"),
					).Padding(8),
				).MaxFrame(-1, 0),
			).Padding(24),
		)}}}); err != nil {
		log.Fatal(err)
	}
}
