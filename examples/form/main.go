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
	"runtime"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	name := swiftui.NewStringState("")
	password := swiftui.NewStringState("")
	bio := swiftui.NewStringState("")
	notifications := swiftui.NewIntState(1) // toggle on
	volume := swiftui.NewIntState(50)
	color := swiftui.NewColorState(0.2, 0.5, 1.0, 1.0)
	date := swiftui.NewDateState(float64(time.Now().Unix()))

	swiftui.Run(swiftui.AppConfig{
		Title:  "Settings",
		Width:  500,
		Height: 500,
	}, swiftui.Form(
		swiftui.Section("Account",
			swiftui.VStack(
				swiftui.TextField("Name", name, func() {}),
				swiftui.SecureField("Password", password, func() {}),
			),
		),
		swiftui.Section("Profile",
			swiftui.TextEditor(bio).Frame(0, 80),
		),
		swiftui.Section("Preferences",
			swiftui.VStack(
				swiftui.Toggle("Notifications", notifications, func() {}),
				swiftui.Slider("Volume", volume, 0, 100, func() {}),
				swiftui.ColorPicker("Accent Color", color, func() {}),
				swiftui.DatePicker("Reminder", date, func() {}),
			),
		),
	).Padding(20))
}
