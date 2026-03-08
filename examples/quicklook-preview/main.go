//go:build darwin
// +build darwin

// Command quicklook-preview demonstrates the QuickLook SwiftUI overlay bridge.
//
// It creates a Text view with a quickLookPreview modifier applied,
// then displays it in a window using swiftui.Run.
//
// Usage:
//
//	go run . [file]
package main

import (
	"os"
	"runtime"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/quicklook"
)

func init() { runtime.LockOSThread() }

func main() {
	file := "/System/Library/Desktop Pictures/Sequoia.heic"
	if len(os.Args) > 1 {
		file = os.Args[1]
	}

	// Create a Text view and apply the quickLookPreview modifier.
	text := swiftui.Text("Preview: " + file)
	previewPtr := quicklook.QuickLookPreview(text.Pointer(), file)

	// Display in a window.
	swiftui.Run(swiftui.AppConfig{
		Title:  "QuickLook Preview",
		Width:  800,
		Height: 600,
	}, swiftui.ViewFromPointer(previewPtr))
}
