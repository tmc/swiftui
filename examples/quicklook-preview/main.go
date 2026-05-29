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
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/quicklook"
)

func init() { runtime.LockOSThread() }

func main() {
	file := firstExisting(
		"/Volumes/tmc/go/src/github.com/tmc/swiftui/README.md",
		"/System/Library/Desktop Pictures/Big Sur.madesktop",
		"/System/Applications/Calculator.app",
	)
	if len(os.Args) > 1 {
		file = os.Args[1]
	}

	info, err := os.Stat(file)
	fileName := filepath.Base(file)
	fileSize := "Unavailable"
	modified := "Unavailable"
	if err == nil {
		fileSize = fmtBytes(info.Size())
		modified = info.ModTime().Format("2006-01-02 15:04")
	}

	previewCard := swiftui.VStackSpaced(12,
		swiftui.ZStack(
			swiftui.Circle().
				Fill(swiftui.RGBA(0.28, 0.55, 1.0, 0.14)).
				Frame(84, 84).
				AsView(),
			swiftui.Image("doc.richtext.fill").
				ForegroundStyle(swiftui.RGBA(0.35, 0.65, 1.0, 1.0)).
				ImageScale(swiftui.ImageScaleLarge),
		),
		swiftui.Text(fileName).
			Font(swiftui.FontTitle2).
			FontWeight(swiftui.WeightBold),
		swiftui.Text("Quick Look preview surface for the selected file.").
			Font(swiftui.FontCallout).
			ForegroundStyleNamed("secondary"),
		swiftui.Text(file).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("tertiary"),
	).Padding(32)

	previewPtr := quicklook.QuickLookPreview(previewCard.Pointer(), file)
	preview := swiftui.ViewFromPointer(previewPtr)
	if err :=

		// Display in a window.
		swiftui.Run(swiftui.AppConfig{
			Title:  "QuickLook Preview",
			Width:  800,
			Height: 600,
		}, swiftui.VStackSpaced(16,
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("File Preview").
						Font(swiftui.FontTitle2).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
					swiftui.Label("Quick Look Bridge", "eye.fill").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
				),
				swiftui.HStack(
					swiftui.Text("Wrap a document surface with Quick Look and keep useful metadata in view.").
						Font(swiftui.FontCallout).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			),
			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Selection",
					swiftui.VStackSpaced(10,
						infoRow("Name", fileName),
						infoRow("Type", filepath.Ext(file)),
						infoRow("Size", fileSize),
						infoRow("Modified", modified),
					).Padding(10),
				).Frame(240, 0),
				swiftui.GroupBox("Preview",
					preview.MaxFrame(-1, -1),
				).MaxFrame(-1, -1),
			).MaxFrame(-1, -1),
		).Padding(20)); err != nil {
		log.Fatal(err)
	}
}

func infoRow(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightMedium),
	)
}

func fmtBytes(n int64) string {
	if n < 1024 {
		return "1 KB"
	}
	kb := float64(n) / 1024
	if kb < 1024 {
		return sprintf1(kb, "KB")
	}
	mb := kb / 1024
	if mb < 1024 {
		return sprintf1(mb, "MB")
	}
	return sprintf1(mb/1024, "GB")
}

func sprintf1(v float64, unit string) string {
	return strconv.FormatFloat(v, 'f', 1, 64) + " " + unit
}

func firstExisting(paths ...string) string {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return paths[0]
}
