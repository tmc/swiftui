//go:build darwin
// +build darwin

// Command multi-window demonstrates a multi-window SwiftUI app driven from Go.
//
// It configures two windows with stable IDs and a Settings scene through a
// single [swiftui.App], then opens or focuses the inspector window at runtime
// with [swiftui.OpenWindow] from a button callback. State is shared across the
// windows by closing over the same [swiftui.IntState].
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

// Window IDs are stable identifiers, independent of the (mutable) Title. Run
// requires them to be non-empty and unique once an App has more than one window,
// and OpenWindow addresses windows by exactly these values.
const (
	mainWindowID      = "main"
	inspectorWindowID = "inspector"
)

func main() {
	count := swiftui.NewIntState(0)

	mainRoot := swiftui.VStackSpaced(18,
		swiftui.Spacer(),
		swiftui.Text("Multi-Window").
			Font(swiftui.FontLargeTitle).
			FontWeight(swiftui.WeightBold),
		swiftui.Text("Two windows and a Settings scene from one swiftui.App.").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary").
			MultilineTextAlignment(swiftui.TextAlignmentCenter),
		swiftui.HStackSpaced(6,
			swiftui.Text("Shared count:").Font(swiftui.FontTitle2),
			swiftui.TextFrom(count).Font(swiftui.FontTitle2).FontWeight(swiftui.WeightBold),
		),
		swiftui.HStackSpaced(12,
			swiftui.Button("Increment", func() {
				count.Set(count.Get() + 1)
			}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
			swiftui.Button("Open Inspector", func() {
				// OpenWindow opens the inspector if it is closed, or focuses it
				// if it is already on screen. A non-nil error means the id is
				// unknown or the app has not started yet.
				if err := swiftui.OpenWindow(inspectorWindowID); err != nil {
					log.Printf("open inspector: %v", err)
				}
			}).ButtonStyle(swiftui.ButtonStyleBordered),
		),
		swiftui.Spacer(),
	).Padding(28)

	inspectorRoot := swiftui.VStackSpaced(16,
		swiftui.Spacer(),
		swiftui.Image("sidebar.right").
			ForegroundStyle(swiftui.RGBA(0.35, 0.65, 1.0, 1.0)).
			ImageScale(swiftui.ImageScaleLarge),
		swiftui.Text("Inspector").
			Font(swiftui.FontTitle).
			FontWeight(swiftui.WeightSemibold),
		swiftui.HStackSpaced(6,
			swiftui.Text("Observing count:").
				Font(swiftui.FontBody).
				ForegroundStyleNamed("secondary"),
			swiftui.TextFrom(count).
				Font(swiftui.FontBody).
				ForegroundStyleNamed("secondary"),
		),
		swiftui.Text("This window shares state with the main window and is\nopened or focused by id at runtime.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			MultilineTextAlignment(swiftui.TextAlignmentCenter),
		swiftui.Spacer(),
	).Padding(24)

	settingsRoot := swiftui.VStackSpaced(12,
		swiftui.Text("Settings").
			Font(swiftui.FontTitle2).
			FontWeight(swiftui.WeightSemibold),
		swiftui.Text("Reachable from the app menu (Cmd-,).").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
	).Padding(24)

	app := swiftui.App{
		Windows: []swiftui.WindowConfig{
			{ID: mainWindowID, Title: "Multi-Window", Width: 520, Height: 360, Root: mainRoot},
			{ID: inspectorWindowID, Title: "Inspector", Width: 320, Height: 320, Root: inspectorRoot},
		},
		Settings: &swiftui.SettingsConfig{Title: "Settings", Width: 360, Height: 240, Root: settingsRoot},
	}

	if err := swiftui.Run(app); err != nil {
		log.Fatal(err)
	}
}
