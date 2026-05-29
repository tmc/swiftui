//go:build darwin
// +build darwin

// Command menu-bar demonstrates modal presentations in SwiftUI from Go.
//
// It shows buttons that trigger six different modal types: Sheet, Alert,
// ConfirmationDialog, Popover, FullScreenCover, and ContextMenu.
// Each presentation uses IntState for visibility binding (0=hidden, 1=shown).
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
	showSheet := swiftui.NewIntState(0)
	showAlert := swiftui.NewIntState(0)
	showConfirm := swiftui.NewIntState(0)
	showPopover := swiftui.NewIntState(0)
	showFullScreen := swiftui.NewIntState(0)
	if err := swiftui.Run(swiftui.WithWindow(swiftui.AppConfig{
		Title:  "Modal Presentations",
		Width:  500,
		Height: 520,
	}, swiftui.VStackSpaced(16,
		swiftui.Text("Modal Presentations").
			Font(swiftui.FontTitle).
			FontWeight(swiftui.WeightBold).
			AsView(),

		swiftui.Text("Tap each button to trigger a different modal type.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			AsView(),

		swiftui.Divider(),

		// Sheet
		swiftui.Button("Sheet", func() {
			showSheet.Set(1)
		}).ButtonStyle(swiftui.ButtonStyleBordered).
			ControlSize(swiftui.ControlSizeLarge).
			Help("Present a modal sheet").
			Sheet(showSheet, sheetContent()),

		// Alert
		swiftui.Button("Alert", func() {
			showAlert.Set(1)
		}).ButtonStyle(swiftui.ButtonStyleBordered).
			ControlSize(swiftui.ControlSizeLarge).
			Help("Show an alert dialog").
			Alert("File Saved", "Your document has been saved successfully.", showAlert),

		// Confirmation Dialog
		swiftui.Button("Confirmation Dialog", func() {
			showConfirm.Set(1)
		}).ButtonStyle(swiftui.ButtonStyleBordered).
			ControlSize(swiftui.ControlSizeLarge).
			Help("Show a confirmation dialog with actions").
			ConfirmationDialog("Delete Item?", showConfirm, swiftui.VStack(
				swiftui.Button("Delete", func() {}),
				swiftui.Button("Archive Instead", func() {}),
			)),

		// Popover
		swiftui.Button("Popover", func() {
			showPopover.Set(1)
		}).ButtonStyle(swiftui.ButtonStyleBordered).
			ControlSize(swiftui.ControlSizeLarge).
			Help("Show a popover").
			Popover(showPopover, popoverContent()),

		// Full Screen Cover
		swiftui.Button("Full Screen Cover", func() {
			showFullScreen.Set(1)
		}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
			ControlSize(swiftui.ControlSizeLarge).
			Help("Present a full-screen modal").
			FullScreenCover(showFullScreen, fullScreenContent(showFullScreen)),

		swiftui.Divider(),

		// Context Menu (right-click)
		swiftui.GroupBox("Context Menu", swiftui.VStack(
			swiftui.Label("Right-click here", "cursorarrow.click.2").
				Font(swiftui.FontBody).
				AsView().
				Padding(20).
				ContextMenu(swiftui.VStack(
					swiftui.Button("Cut", func() {}),
					swiftui.Button("Copy", func() {}),
					swiftui.Button("Paste", func() {}),
					swiftui.Divider(),
					swiftui.Button("Select All", func() {}),
				)),
		)),
	).Padding(30))); err != nil {
		log.Fatal(err)
	}
}

func sheetContent() swiftui.View {
	return swiftui.VStackSpaced(16,
		swiftui.Image("doc.text.fill").
			ImageScale(swiftui.ImageScaleLarge).
			ForegroundStyleNamed("blue"),
		swiftui.Text("Sheet Content").
			Font(swiftui.FontTitle2).
			FontWeight(swiftui.WeightBold).
			AsView(),
		swiftui.Text("This is a modal sheet. Drag down or press Escape to dismiss.").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary").
			AsView(),
	).Padding(40).Frame(400, 250)
}

func popoverContent() swiftui.View {
	return swiftui.VStackSpaced(8,
		swiftui.Text("Quick Info").
			Font(swiftui.FontHeadline).
			AsView(),
		swiftui.Divider(),
		swiftui.Label("3 items selected", "checkmark.circle").
			Font(swiftui.FontBody).
			AsView(),
		swiftui.Label("Last modified today", "clock").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			AsView(),
	).Padding(16)
}

func fullScreenContent(state *swiftui.IntState) swiftui.View {
	return swiftui.VStackSpaced(20,
		swiftui.Spacer(),
		swiftui.Image("rectangle.expand.vertical").
			ImageScale(swiftui.ImageScaleLarge).
			ForegroundStyleNamed("blue"),
		swiftui.Text("Full Screen Cover").
			Font(swiftui.FontTitle).
			FontWeight(swiftui.WeightBold).
			AsView(),
		swiftui.Text("This modal covers the entire screen.").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary").
			AsView(),
		swiftui.Button("Dismiss", func() {
			state.Set(0)
		}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
			ControlSize(swiftui.ControlSizeLarge),
		swiftui.Spacer(),
	).Padding(40)
}
