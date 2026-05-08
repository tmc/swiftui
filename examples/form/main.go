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
	"runtime"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	name := swiftui.NewStringState("Taylor")
	password := swiftui.NewStringState("")
	bio := swiftui.NewStringState("Building native macOS UIs from Go.")
	nameValid := swiftui.NewBoolState(true)
	passwordValid := swiftui.NewBoolState(true)
	bioValid := swiftui.NewBoolState(true)
	nameFocus := swiftui.NewBoolState(true)
	passwordFocus := swiftui.NewBoolState(false)
	notifications := swiftui.NewIntState(1)
	volume := swiftui.NewIntState(50)
	color := swiftui.NewColorState(0.2, 0.5, 1.0, 1.0)
	date := swiftui.NewDateState(float64(time.Now().Add(2 * time.Hour).Unix()))

	swiftui.Run(swiftui.AppConfig{
		Title:  "Profile Form",
		Width:  660,
		Height: 600,
	}, swiftui.ScrollView(
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
					ForegroundStyle(0.95, 0.7, 0.2, 1.0),
			),

			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Account",
					swiftui.VStackSpaced(12,
						swiftui.TextFieldPolicy("Name", name, swiftui.TextInputPolicy{
							AllowedPattern:    `[A-Za-z .'-]*`,
							ValidationPattern: `^[A-Za-z][A-Za-z .'-]{1,31}$`,
							ValidState:        nameValid,
						}, nil, func() {
							nameFocus.Set(false)
							passwordFocus.Set(true)
						}).
							Focused(nameFocus).
							SubmitLabel(swiftui.SubmitLabelNext).
							TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
						swiftui.SecureFieldPolicy("Password", password, swiftui.TextInputPolicy{
							ValidationPattern: `^.{8,}$`,
							ValidState:        passwordValid,
						}, nil, func() {
							passwordFocus.Set(false)
						}).
							Focused(passwordFocus).
							SubmitLabel(swiftui.SubmitLabelDone).
							TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
						swiftui.HStackSpaced(8,
							swiftui.Button("Focus name", func() {
								passwordFocus.Set(false)
								nameFocus.Set(true)
							}),
							swiftui.Button("Focus password", func() {
								nameFocus.Set(false)
								passwordFocus.Set(true)
							}),
							swiftui.Button("Clear focus", func() {
								nameFocus.Set(false)
								passwordFocus.Set(false)
							}),
						),
						infoLine("Focused field", focusedFieldLabel(nameFocus, passwordFocus)),
						swiftui.DynamicBoolView(nameValid, func(ok bool) swiftui.View {
							return infoLine("Name validation", onOffLabel(ok, "Accepted", "Use 2-32 letters"))
						}),
						swiftui.DynamicBoolView(passwordValid, func(ok bool) swiftui.View {
							return infoLine("Password validation", onOffLabel(ok, "Accepted", "Use at least 8 characters"))
						}),
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
				swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 8,
					swiftui.TextEditorPolicy(bio, swiftui.TextInputPolicy{
						ValidationPattern: `(?s)^.{0,240}$`,
						ValidState:        bioValid,
					}, nil).
						Padding(6).
						Frame(600, 110).
						ScrollContentBackgroundHidden().
						BackgroundRoundedRect(0.15, 0.16, 0.20, 0.98, 10).
						ClipRoundedRect(10),
					swiftui.DynamicBoolView(bioValid, func(ok bool) swiftui.View {
						return infoLine("Bio validation", onOffLabel(ok, "Within 240 characters", "Too long"))
					}),
				),
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
	))
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

func focusedFieldLabel(nameFocus, passwordFocus *swiftui.BoolState) string {
	switch {
	case nameFocus.Get():
		return "Name"
	case passwordFocus.Get():
		return "Password"
	default:
		return "None"
	}
}
