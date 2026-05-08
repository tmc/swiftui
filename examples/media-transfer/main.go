//go:build darwin
// +build darwin

// Command media-transfer demonstrates concrete paste/share/drop payload flows.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	pastedText := swiftui.NewStringState("Paste text into the app.")
	defer pastedText.Release()
	pastedPayload := swiftui.NewStringState("Paste a text, URL, or file path payload into the app.")
	defer pastedPayload.Release()

	dropped := swiftui.NewStringState("Drop text, a URL, or a file onto the target.")
	defer dropped.Release()
	lazyPreview := swiftui.NewStringState("Load a curated sample item to resolve its lazy file handle.")
	defer lazyPreview.Release()
	posterSample := lazyPosterSampleItem()

	photos := swiftui.NewPhotosPickerSelectionState(
		posterSample,
	)
	defer photos.Release()

	swiftui.Run(swiftui.AppConfig{
		Title:  "Media Transfer",
		Width:  840,
		Height: 620,
	}, swiftui.ZStack(
		swiftui.LinearGradientHorizontal(swiftui.RGB(0.06, 0.12, 0.18), swiftui.RGB(0.18, 0.12, 0.08)),
		swiftui.ScrollView(
			swiftui.VStackSpaced(14,
				header(),
				payloadPanel(pastedText, pastedPayload, dropped).MaxFrame(-1, 0),
				photoStatePanel(photos, lazyPreview, posterSample).MaxFrame(-1, 0),
			).Padding(18),
		),
	))
}

func header() swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text("Media and transfer").
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text("Concrete share, drop, and paste payloads are bridged for text, URL, and file paths today. PasteButton stays text-only, while PasteButtonPayload handles the concrete clipboard payload. PhotosPicker now has a native bridge-backed path with ordering and media-kind metadata plus a deterministic curated sample mode and an optional lazy file handle.").
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
	)
}

func payloadPanel(pastedText, pastedPayload, dropped *swiftui.StringState) swiftui.View {
	return panel("Payload flows",
		swiftui.VStackSpaced(12,
			swiftui.HStackSpaced(10,
				swiftui.PasteButton("Paste text", pastedText),
				swiftui.PasteButtonPayload("Paste payload", func(payload swiftui.PastePayload) bool {
					if payload.Kind() == "" {
						pastedPayload.Set("empty payload")
						return true
					}
					pastedPayload.Set(payload.Kind() + ": " + payload.Value())
					return true
				}),
				swiftui.ShareLink("Share project URL", "https://github.com/tmc/swiftui"),
				swiftui.ShareLinkItem("Share note", swiftui.ShareItem{
					Title: "Media transfer note",
					Text:  "Concrete text payload from Go SwiftUI",
				}),
				swiftui.ShareLinkItem("Share file", swiftui.ShareItem{
					Title:    "Release artifact",
					FilePath: "/tmp/release-notes.txt",
				}),
				swiftui.Spacer(),
			),
			swiftui.TextFromString(pastedText).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.72, 0.92, 1.0, 1),
			swiftui.TextFromString(pastedPayload).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.72, 0.92, 1.0, 1),
			swiftui.Text("PasteButton remains plain text, while PasteButtonPayload covers text, URL, and file-path payloads. Share and drop stay concrete across the same payload kinds.").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.HStackSpaced(12,
				payloadChip("Drag text", swiftui.ShareItem{Text: "Dropped from Go SwiftUI"}),
				payloadChip("Drag URL", swiftui.ShareItem{URL: "https://github.com/tmc/swiftui"}),
				payloadChip("Drag file", swiftui.ShareItem{FilePath: "/tmp/release-notes.txt"}),
				swiftui.TextFromString(dropped).
					AsView().
					Padding(10).
					BackgroundRoundedRect(0.18, 0.12, 0.08, 0.78, 12).
					DropDestination(func(payload swiftui.DropPayload) bool {
						if payload.Kind() == "" {
							dropped.Set("empty payload")
							return true
						}
						dropped.Set(payload.Kind() + ": " + payload.Value())
						return true
					}),
			),
		),
	)
}

func photoStatePanel(photos *swiftui.PhotosPickerSelectionState, lazyPreview *swiftui.StringState, posterSample swiftui.PhotosPickerItem) swiftui.View {
	return panel("PhotosPicker state",
		swiftui.DynamicView(photos.RevisionState(), func(_ int) swiftui.View {
			items := photos.Items()
			rows := make([]swiftui.Viewable, 0, len(items)+3)
			rows = append(rows,
				swiftui.Text("The native picker below writes normalized item metadata, inferred media kind, and selection ordering into the owned state. The sample picker remains useful for deterministic demo flows.").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView(),
				swiftui.HStackSpaced(8,
					swiftui.Text("Selected:"),
					swiftui.TextFrom(photos.CountState()).MonospacedDigit(),
					swiftui.Spacer(),
				),
				swiftui.HStackSpaced(8,
					swiftui.PhotosPickerNative("Choose from library", photos, swiftui.PhotosPickerConfig{
						Matching:          swiftui.PhotosPickerMatchingImages,
						MaxSelectionCount: 5,
					}),
					swiftui.PhotosPickerNative("Choose videos", photos, swiftui.PhotosPickerConfig{
						Matching:          swiftui.PhotosPickerMatchingVideos,
						MaxSelectionCount: 3,
					}),
					swiftui.PhotosPickerMenu("Choose sample photos", photos,
						posterSample,
						swiftui.PhotosPickerItem{ID: "receipt", Filename: "receipt.heic", UTType: "public.heic", Order: 1},
						swiftui.PhotosPickerItem{ID: "diagram", Filename: "system-diagram.jpg", UTType: "public.jpeg", Order: 2},
					),
					swiftui.Spacer(),
				),
				swiftui.HStackSpaced(8,
					swiftui.Button("Load first selected asset", func() {
						selected := photos.Items()
						if len(selected) == 0 || selected[0].LazyFile == nil {
							lazyPreview.Set("No lazy file handle selected.")
							return
						}
						data, err := selected[0].LazyFile.Load()
						if err != nil {
							lazyPreview.Set("Lazy load error: " + err.Error())
							return
						}
						lazyPreview.Set(fmt.Sprintf("Loaded %d bytes from %s", len(data), selected[0].LazyFile.Path))
					}).ButtonStyle(swiftui.ButtonStyleBordered),
					swiftui.TextFromString(lazyPreview).
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
				),
			)
			for _, item := range items {
				item := item
				rows = append(rows, swiftui.HStackSpaced(8,
					swiftui.Text(fmt.Sprintf("#%d", item.Order+1)).
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary").
						AsView(),
					swiftui.Text(item.Filename),
					swiftui.Spacer(),
					swiftui.Text(item.MediaKind).Font(swiftui.FontCaption).AsView(),
					swiftui.Text(item.UTType).Font(swiftui.FontCaption).AsView(),
					swiftui.ButtonWithLabel("", "xmark.circle.fill", func() {
						photos.Remove(item.ID)
					}),
				))
			}
			rows = append(rows, swiftui.HStackSpaced(8,
				swiftui.Button("Clear", func() {
					photos.Clear()
				}).ButtonStyle(swiftui.ButtonStyleBordered),
				swiftui.Spacer(),
			))
			return swiftui.VStackSpaced(10, rows...)
		}),
	)
}

func lazyPosterSampleItem() swiftui.PhotosPickerItem {
	path := filepath.Join(os.TempDir(), "swiftui-media-transfer-lazy-poster.txt")
	_ = os.WriteFile(path, []byte("lazy photo payload for the media-transfer example"), 0o600)
	return swiftui.PhotosPickerItem{
		ID:       "poster",
		Filename: "launch-poster.png",
		UTType:   "public.png",
		Order:    0,
		LazyFile: swiftui.NewPhotosPickerLazyFileHandle(path),
	}
}

func panel(title string, content swiftui.View) swiftui.View {
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightSemibold),
			swiftui.Spacer(),
		),
		content,
	).Padding(14).
		Background(0.05, 0.07, 0.09, 0.78).
		CornerRadius(18)
}

func payloadChip(title string, item swiftui.ShareItem) swiftui.View {
	return swiftui.Text(title).
		AsView().
		Padding(10).
		BackgroundRoundedRect(0.10, 0.28, 0.42, 0.78, 12).
		Draggable(item)
}
