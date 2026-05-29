//go:build darwin
// +build darwin

// Command form demonstrates SwiftUI form controls from Go.
//
// It shows TextField, SecureField, Toggle, Slider, ColorPicker, and
// DatePicker all bound to reactive state.
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
	name := swiftui.NewStringState("Taylor")
	password := swiftui.NewStringState("")
	bio := swiftui.NewStringState("Building native macOS UIs from Go.")
	notifications := swiftui.NewIntState(1)
	volume := swiftui.NewIntState(50)
	color := swiftui.NewColorState(0.2, 0.5, 1.0, 1.0)
	date := swiftui.NewDateState(float64(time.Now().Add(2 * time.Hour).Unix()))
	if err := swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
		Title:  "Profile Form",
		Width:  660,
		Height: 600,
		Root: swiftui.ScrollView(
			swiftui.VStackSpaced(16,
				swiftui.HStack(
					swiftui.VStackSpaced(4,
						swiftui.HStack(
							swiftui.Text("Profile Form").
								Font(swiftui.FontTitle2).
								FontWeight(swiftui.WeightBold),
							swiftui.Spacer(),
						),
						swiftui.HStack(
							swiftui.Text("A compact settings-style surface that shows common SwiftUI form controls working together.").
								Font(swiftui.FontCallout).
								ForegroundStyleNamed("secondary"),
							swiftui.Spacer(),
						),
					).MaxFrame(-1, 0),
					swiftui.Label("Draft", "square.and.pencil").
						Font(swiftui.FontCaption).
						ForegroundStyle(swiftui.RGBA(0.95, 0.7, 0.2, 1.0)),
				),

				swiftui.HStackSpaced(12,
					swiftui.GroupBox("Account",
						swiftui.VStackSpaced(12,
							swiftui.TextField("Name", name, func() {}),
							swiftui.SecureField("Password", password, func() {}),
							infoLine("Access", "Member workspace"),
						).Padding(10),
					).MaxFrame(-1, 0),
					swiftui.GroupBox("Summary",
						swiftui.VStackSpaced(10,
							infoLine("Notifications", onOffLabel(notifications.Get() != 0, "Enabled", "Muted")),
							infoLine("Volume", fmt.Sprintf("%d%%", volume.Get())),
							infoLine("Reminder", "Today"),
							infoLine("Accent", "Linked to picker below"),
						).Padding(10),
					).MaxFrame(-1, 0),
				),

				swiftui.GroupBox("Profile",
					swiftui.TextEditor(bio).Frame(600, 110),
				).MaxFrame(-1, 0),

				swiftui.GroupBox("Preferences",
					swiftui.VStackSpaced(12,
						swiftui.Toggle("Notifications", notifications, func() {}),
						swiftui.Slider("Volume", volume, 0, 100, func() {}),
						swiftui.ColorPicker("Accent Color", color, func() {}),
						swiftui.DatePicker("Reminder", date, func() {}),
					).Padding(10),
				).MaxFrame(-1, 0),
			).Padding(24),
		),
	}}}); err != nil {
		log.Fatal(err)
	}
}

func infoLine(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightMedium),
	)
}

func onOffLabel(v bool, on, off string) string {
	if v {
		return on
	}
	return off
}
