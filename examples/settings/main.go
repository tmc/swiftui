// Command settings demonstrates a comprehensive macOS settings panel built
// with SwiftUI from Go. It uses TabView with three tabs (General, Appearance,
// Advanced), each containing Form sections with various input controls.
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
	// Change tracker
	changes := swiftui.NewIntState(0)
	inc := func() { changes.Set(changes.Get() + 1) }

	// General tab state
	username := swiftui.NewStringState("")
	apiKey := swiftui.NewStringState("")
	bio := swiftui.NewStringState("")
	emailNotif := swiftui.NewIntState(1)
	pushNotif := swiftui.NewIntState(1)
	soundNotif := swiftui.NewIntState(0)
	startTime := swiftui.NewDateState(float64(time.Now().Unix()))
	endTime := swiftui.NewDateState(float64(time.Now().Add(8 * time.Hour).Unix()))
	reminderInterval := swiftui.NewIntState(15)

	// Appearance tab state
	theme := swiftui.NewIntState(0)
	accentColor := swiftui.NewColorState(0.2, 0.5, 1.0, 1.0)
	opacity := swiftui.NewFloatState(1.0)
	fontFamily := swiftui.NewIntState(0)
	fontSize := swiftui.NewIntState(14)
	sidebarWidth := swiftui.NewIntState(250)
	compactMode := swiftui.NewIntState(0)
	gridColumns := swiftui.NewIntState(3)

	// Advanced tab state
	hwAccel := swiftui.NewIntState(1)
	cacheSize := swiftui.NewIntState(512)
	analytics := swiftui.NewIntState(0)
	proxyURL := swiftui.NewStringState("")
	timeout := swiftui.NewIntState(30)
	protocol := swiftui.NewIntState(1)
	verboseLog := swiftui.NewIntState(0)
	devTools := swiftui.NewIntState(0)
	showResetConfirm := swiftui.NewIntState(0)
	showClearAlert := swiftui.NewIntState(0)
	showDeleteConfirm := swiftui.NewIntState(0)

	generalTab := swiftui.Form(
		swiftui.Section("Profile",
			swiftui.VStack(
				swiftui.TextField("Username", username, func() { inc() }),
				swiftui.SecureField("API Key", apiKey, func() { inc() }),
				swiftui.TextEditor(bio).Frame(0, 80),
			),
		),
		swiftui.Section("Notifications",
			swiftui.VStack(
				swiftui.Toggle("Email notifications", emailNotif, inc),
				swiftui.Toggle("Push notifications", pushNotif, inc),
				swiftui.Toggle("Sound alerts", soundNotif, inc),
			),
		),
		swiftui.Section("Schedule",
			swiftui.VStack(
				swiftui.DatePicker("Start time", startTime, inc),
				swiftui.DatePicker("End time", endTime, inc),
				swiftui.Stepper("Reminder interval (min)", reminderInterval, 1, 60, inc),
			),
		),
	).TabItem("General", "gearshape")

	appearanceTab := swiftui.Form(
		swiftui.Section("Theme",
			swiftui.VStack(
				swiftui.PickerSegmented("Appearance", theme,
					swiftui.VStack(
						swiftui.Text("Light").AsView().Tag(int32(0)),
						swiftui.Text("Dark").AsView().Tag(int32(1)),
						swiftui.Text("Auto").AsView().Tag(int32(2)),
					), inc,
				),
				swiftui.ColorPicker("Accent color", accentColor, inc),
				swiftui.FloatSlider("Opacity", opacity, 0.0, 1.0, inc),
			),
		),
		swiftui.Section("Typography",
			swiftui.VStack(
				swiftui.PickerMenu("Font family", fontFamily,
					swiftui.VStack(
						swiftui.Text("System").AsView().Tag(int32(0)),
						swiftui.Text("Monospaced").AsView().Tag(int32(1)),
						swiftui.Text("Serif").AsView().Tag(int32(2)),
						swiftui.Text("Rounded").AsView().Tag(int32(3)),
					), inc,
				),
				swiftui.Slider("Font size", fontSize, 10, 32, inc),
				swiftui.DynamicView(fontFamily, func(fam int) swiftui.View {
					designs := []swiftui.Design{
						swiftui.DesignDefault,
						swiftui.DesignMonospaced,
						swiftui.DesignSerif,
						swiftui.DesignRounded,
					}
					size := float64(fontSize.Get())
					return swiftui.Text("The quick brown fox jumps over the lazy dog").
						Font(swiftui.FontSystemDesign(size, swiftui.WeightRegular, designs[fam])).
						AsView()
				}),
			),
		),
		swiftui.Section("Layout",
			swiftui.VStack(
				swiftui.Slider("Sidebar width", sidebarWidth, 150, 400, inc),
				swiftui.Toggle("Compact mode", compactMode, inc),
				swiftui.Stepper("Grid columns", gridColumns, 1, 6, inc),
			),
		),
	).TabItem("Appearance", "paintbrush")

	advancedTab := swiftui.Form(
		swiftui.Section("Performance",
			swiftui.VStack(
				swiftui.Toggle("Hardware acceleration", hwAccel, inc),
				swiftui.Slider("Cache size (MB)", cacheSize, 64, 2048, inc),
				swiftui.Toggle("Analytics", analytics, inc),
			),
		),
		swiftui.Section("Network",
			swiftui.VStack(
				swiftui.TextField("Proxy URL", proxyURL, func() { inc() }),
				swiftui.Stepper("Timeout (seconds)", timeout, 5, 120, inc),
				swiftui.PickerSegmented("Protocol", protocol,
					swiftui.VStack(
						swiftui.Text("HTTP").AsView().Tag(int32(0)),
						swiftui.Text("HTTPS").AsView().Tag(int32(1)),
						swiftui.Text("SOCKS5").AsView().Tag(int32(2)),
					), inc,
				),
			),
		),
		swiftui.Section("Debug",
			swiftui.VStack(
				swiftui.Toggle("Verbose logging", verboseLog, inc),
				swiftui.Toggle("Developer tools", devTools, inc),
				swiftui.Button("Reset All Settings", func() {
					showResetConfirm.Set(1)
				}).ConfirmationDialog("Are you sure you want to reset all settings?", showResetConfirm,
					swiftui.VStack(
						swiftui.Button("Reset", func() {
							changes.Set(0)
						}),
					),
				),
			),
		),
		swiftui.Section("Danger Zone",
			swiftui.VStack(
				swiftui.Button("Clear Cache", func() {
					showClearAlert.Set(1)
				}).Alert("Clear Cache", "This will remove all cached data.", showClearAlert),
				swiftui.Button("Delete Account", func() {
					showDeleteConfirm.Set(1)
				}).ForegroundStyle(1.0, 0.3, 0.3, 1.0).
					ConfirmationDialog("This action cannot be undone. Delete your account?", showDeleteConfirm,
						swiftui.VStack(
							swiftui.Button("Delete Account", func() {}),
						),
					),
			),
		),
	).TabItem("Advanced", "wrench.and.screwdriver")

	statusBar := swiftui.HStack(
		swiftui.DynamicView(changes, func(n int) swiftui.View {
			return swiftui.Text(fmt.Sprintf("Settings modified: %d changes", n)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView()
		}),
		swiftui.Spacer(),
		swiftui.Button("Revert", func() {
			changes.Set(0)
		}).ButtonStyle(swiftui.ButtonStyleBordered),
		swiftui.Button("Save", func() {
			changes.Set(0)
		}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
	).Padding(12)

	swiftui.Run(swiftui.AppConfig{
		Title:  "Settings",
		Width:  600,
		Height: 700,
	}, swiftui.VStack(
		swiftui.TabView(
			generalTab,
			appearanceTab,
			advancedTab,
		).MaxFrame(-1, -1),
		swiftui.Divider(),
		statusBar,
	))
}
