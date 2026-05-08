//go:build darwin

// Command accessibility demonstrates SwiftUI accessibility modifiers from Go.
//
// It showcases AccessibilityLabel, AccessibilityHint, AccessibilityValue,
// AccessibilityHidden, AccessibilityIdentifier, and AccessibilityRotor applied to interactive
// elements, images, and decorative views. Each section explains what the
// modifier does and shows it in use. App commands and lifecycle hooks are
// included to exercise the full T1 surface.
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
	name := swiftui.NewStringState("")
	searchText := swiftui.NewStringState("")
	searchFocused := swiftui.NewBoolState(false)
	routingText := swiftui.NewStringState("")
	routingFocused := swiftui.NewBoolState(false)
	focusTarget := swiftui.NewIntState(0)
	defer focusTarget.Release()
	focusRoutingNS := swiftui.NewFocusNamespace()
	defer focusRoutingNS.Release()
	focusStatus := swiftui.NewStringState("Use Focus > Search Field or Cmd+L to jump to the search box. The routed field is configured with preferred default focus inside its section.")
	volume := swiftui.NewIntState(50)
	rotor := swiftui.NewAccessibilityRotorModel("Transcript navigation",
		swiftui.NewAccessibilityRotorEntry("Overview", "overview"),
		swiftui.NewAccessibilityRotorEntry("Troubleshooting", "troubleshooting"),
		swiftui.NewAccessibilityRotorEntry("Summary", "summary"),
	)

	lifecycle := swiftui.AppLifecycle{
		OnActivate:     func() { log.Println("accessibility: activated") },
		OnResignActive: func() { log.Println("accessibility: resigned active") },
		OnTerminate:    func() { log.Println("accessibility: terminating") },
	}

	body := swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			// Title
			swiftui.Text("Accessibility Showcase").
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold).
				AsView().
				AccessibilityIdentifier("accessibility-title"),
			swiftui.DynamicView(focusTarget, func(value int) swiftui.View {
				return swiftui.TextFromString(focusStatus).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AccessibilityValue(focusTargetAXValue(value)).
					AccessibilityIdentifier("focus-status")
			}),

			// --- AccessibilityLabel ---
			swiftui.GroupBox("AccessibilityLabel",
				swiftui.VStackSpaced(10,
					swiftui.Text("Overrides the default VoiceOver description. Use it to give meaningful names to icons and images.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Divider().AccessibilityHidden(true),
					swiftui.Image("heart.fill").
						ImageScale(swiftui.ImageScaleLarge).
						ForegroundStyle(0.9, 0.3, 0.3, 1.0).
						AccessibilityLabel("Favorite heart icon").
						AccessibilityIdentifier("heart-icon"),
					swiftui.Image("globe").
						ImageScale(swiftui.ImageScaleLarge).
						ForegroundStyle(0.3, 0.6, 1.0, 1.0).
						AccessibilityLabel("Globe representing international content"),
				).Padding(8),
			).MaxFrame(-1, 0).AccessibilityIdentifier("label-section"),

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
						AccessibilityHint("Saves the current document to disk").
						AccessibilityIdentifier("save-button"),
					swiftui.Button("Delete All", func() {}).
						ButtonStyle(swiftui.ButtonStyleBordered).
						ForegroundStyle(0.9, 0.3, 0.3, 1.0).
						AccessibilityLabel("Delete all items").
						AccessibilityHint("Permanently removes all items from the list").
						AccessibilityIdentifier("delete-all-button"),
				).Padding(8),
			).MaxFrame(-1, 0).AccessibilityIdentifier("hint-section"),

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
						ForegroundStyle(0.9, 0.75, 0.2, 1.0).
						AccessibilityHidden(true),
					swiftui.Text("The sparkle icon above is decorative and hidden from assistive technology.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
				).Padding(8),
			).MaxFrame(-1, 0).AccessibilityIdentifier("hidden-section"),

			// --- Form with labeled controls ---
			swiftui.GroupBox("Accessible Form Controls",
				swiftui.VStackSpaced(10,
					swiftui.Text("Form controls with accessibility labels provide clear descriptions for screen readers.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Divider().AccessibilityHidden(true),
					swiftui.TextField("Enter your name", name, func() {}).
						TextFieldStyle(swiftui.TextFieldStyleRoundedBorder).
						AccessibilityLabel("Full name").
						AccessibilityHint("Type your first and last name").
						AccessibilityIdentifier("name-field"),
					swiftui.DynamicView(volume, func(v int) swiftui.View {
						return swiftui.Slider("Volume", volume, 0, 100, func() {}).
							AccessibilityLabel("Volume control").
							AccessibilityHint("Adjust the playback volume from 0 to 100").
							AccessibilityValue(fmt.Sprintf("Volume %d", v)).
							AccessibilityIdentifier("volume-slider")
					}),
					swiftui.Label("Settings", "gear").
						AccessibilityLabel("Application settings").
						AccessibilityHint("Opens the settings panel"),
					swiftui.Button("Focus search field", func() {
						searchFocused.Set(true)
						focusTarget.Set(1)
						focusStatus.Set("Keyboard focus moved to the search field.")
					}).
						ButtonStyle(swiftui.ButtonStyleBordered).
						AccessibilityIdentifier("focus-search-button").
						AccessibilityHint("Moves keyboard focus to the search field below"),
				).Padding(8),
			).MaxFrame(-1, 0).AccessibilityIdentifier("form-section"),

			// --- Focus routing ---
			swiftui.GroupBox("Focus Routing",
				swiftui.VStackSpaced(10,
					swiftui.Text("FocusSection keeps related controls in a single keyboard-routing cluster. The routed field is configured as the preferred default focus target inside the section.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Divider().AccessibilityHidden(true),
					swiftui.VStackSpaced(8,
						swiftui.TextField("Route search", routingText, func() {}).
							TextFieldStyle(swiftui.TextFieldStyleRoundedBorder).
							Focused(routingFocused).
							PrefersDefaultFocus(true, focusRoutingNS).
							AccessibilityIdentifier("routing-search-field").
							AccessibilityHint("Receives the preferred default focus inside the focus section"),
						swiftui.Button("Focus routed search field", func() {
							routingFocused.Set(true)
							focusTarget.Set(2)
							focusStatus.Set("Keyboard focus moved to the routed search field.")
						}).
							ButtonStyle(swiftui.ButtonStyleBordered).
							AccessibilityIdentifier("routing-focus-button").
							AccessibilityHint("Moves keyboard focus to the routed search field"),
					).FocusSection().FocusScopeID(focusRoutingNS),
				).Padding(8),
			).MaxFrame(-1, 0).AccessibilityIdentifier("focus-routing-section"),

			// --- AccessibilityIdentifier ---
			swiftui.GroupBox("AccessibilityIdentifier",
				swiftui.VStackSpaced(10,
					swiftui.Text("Assigns a stable identifier for automated testing. Unlike AccessibilityLabel, identifiers are not read by VoiceOver — they exist for UI test frameworks and AX inspection tools.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Divider().AccessibilityHidden(true),
					swiftui.Button("Submit Form", func() {}).
						ButtonStyle(swiftui.ButtonStyleBorderedProminent).
						AccessibilityIdentifier("submit-form-button").
						AccessibilityLabel("Submit").
						AccessibilityHint("Submits the form data"),
					swiftui.TextField("Search", searchText, func() {}).
						TextFieldStyle(swiftui.TextFieldStyleRoundedBorder).
						Focused(searchFocused).
						AccessibilityIdentifier("search-field").
						AccessibilityHint("Searches the current accessibility example"),
					swiftui.Text("Use Accessibility Inspector or axmcp to verify these identifiers are queryable.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
				).Padding(8),
			).MaxFrame(-1, 0).AccessibilityIdentifier("identifier-section"),

			// --- AccessibilityRotor ---
			swiftui.GroupBox("AccessibilityRotor",
				swiftui.VStackSpaced(10,
					swiftui.Text("VoiceOver users can jump between named transcript sections with a custom rotor.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Divider().AccessibilityHidden(true),
					swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 6,
						swiftui.Text("Overview").Font(swiftui.FontHeadline),
						swiftui.Text("The session starts with a concise overview of the current surface."),
						swiftui.Text("Troubleshooting").Font(swiftui.FontHeadline),
						swiftui.Text("Use this section when a bridge call or example render fails."),
						swiftui.Text("Summary").Font(swiftui.FontHeadline),
						swiftui.Text("The rotor is a static bridge-backed entry list for assistive navigation."),
					),
					swiftui.HStackSpaced(8,
						rotorChip("overview"),
						rotorChip("troubleshooting"),
						rotorChip("summary"),
						swiftui.Spacer(),
					),
					swiftui.Text("Each rotor entry has a stable label and ID so transcript-style surfaces can expose predictable navigation points without custom accessibility glue.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
				).Padding(8).
					AccessibilityRotor(rotor),
			).MaxFrame(-1, 0).AccessibilityIdentifier("rotor-section"),
		).Padding(24),
	)

	err := swiftui.RunScenes(
		swiftui.Window("accessibility", swiftui.AppConfig{
			Title:  "Accessibility",
			Width:  620,
			Height: 760,
		}, body),
		swiftui.WithCommands(swiftui.Commands(
			swiftui.CommandMenu{
				Title: "Focus",
				Items: []swiftui.CommandItem{
					{
						Title:    "Focus Search Field",
						Shortcut: swiftui.KeyboardShortcut{Key: "l", Modifiers: swiftui.ModCommand},
						Action: func(swiftui.CommandContext) {
							searchFocused.Set(true)
							focusTarget.Set(1)
							focusStatus.Set("Keyboard focus moved to the search field.")
						},
					},
					{
						Title:    "Focus Routed Search Field",
						Shortcut: swiftui.KeyboardShortcut{Key: "l", Modifiers: swiftui.ModCommand | swiftui.ModShift},
						Action: func(swiftui.CommandContext) {
							routingFocused.Set(true)
							focusTarget.Set(2)
							focusStatus.Set("Keyboard focus moved to the routed search field.")
						},
					},
				},
			},
			swiftui.StandardEditMenu(),
			swiftui.StandardWindowMenu(),
			swiftui.StandardHelpMenu(),
		)),
		swiftui.WithLifecycle(lifecycle),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func rotorChip(label string) swiftui.View {
	return swiftui.Text(label).
		Font(swiftui.FontCaption).
		AsView().
		Padding(8).
		BackgroundRoundedRect(0.18, 0.24, 0.36, 0.85, 10).
		AccessibilityIdentifier("rotor-chip-" + label)
}

func focusTargetAXValue(target int) string {
	switch target {
	case 1:
		return "Current focus target: search field"
	case 2:
		return "Current focus target: routed search field"
	default:
		return "Current focus target: none"
	}
}
