//go:build darwin
// +build darwin

// Command bridge-coverage demonstrates recently bridged SwiftUI surfaces.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"runtime"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	refreshes := swiftui.NewIntState(0)
	defer refreshes.Release()

	pastedText := swiftui.NewStringState("Paste text into the app.")
	defer pastedText.Release()

	pastedPayload := swiftui.NewStringState("Paste a text, URL, or file path payload into the app.")
	defer pastedPayload.Release()

	dropped := swiftui.NewStringState("Drop text on the target.")
	defer dropped.Release()

	swiftui.Run(swiftui.AppConfig{
		Title:  "Bridge Coverage",
		Width:  920,
		Height: 720,
	}, swiftui.ZStack(
		swiftui.LinearGradient(
			swiftui.RGB(0.05, 0.10, 0.16),
			swiftui.RGB(0.20, 0.14, 0.08),
			swiftui.UnitPointTopLeading,
			swiftui.UnitPointBottomTrailing,
		),
		swiftui.ScrollView(
			swiftui.VStackSpaced(14,
				header(),
				swiftui.HStackSpaced(14,
					interactionPanel(refreshes, pastedText, pastedPayload, dropped).MaxFrame(-1, 0),
					gridPanel().MaxFrame(-1, 0),
				),
				gradientPanel(),
			).Padding(18),
		).SafeAreaInset(swiftui.EdgeBottom, 8, footer()),
	))
}

func header() swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text("Bridge coverage").
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold).
				ForegroundStyle(0.97, 0.98, 0.94, 1),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text("Refreshable, text paste, concrete payload paste, share, concrete drag/drop, grids, gradients, and safe-area chrome as current bridged surfaces.").
				Font(swiftui.FontCallout).
				ForegroundStyle(0.76, 0.81, 0.78, 1),
			swiftui.Spacer(),
		),
	)
}

func interactionPanel(refreshes *swiftui.IntState, pastedText, pastedPayload, dropped *swiftui.StringState) swiftui.View {
	refresh := func() {
		refreshes.Set(refreshes.Get() + 1)
	}
	return panel("Interaction APIs",
		swiftui.VStackSpaced(12,
			swiftui.List(
				swiftui.HStack(
					swiftui.Text("Refresh count").FontWeight(swiftui.WeightSemibold),
					swiftui.Spacer(),
					swiftui.TextFrom(refreshes).MonospacedDigit(),
				),
				swiftui.Text("iOS exposes pull to refresh here; on macOS use the button below to exercise the same callback path.").AsView(),
			).Refreshable(refresh).MaxFrame(-1, 120),
			swiftui.HStackSpaced(10,
				swiftui.Button("Refresh Now", refresh).
					ButtonStyle(swiftui.ButtonStyleBorderedProminent),
				swiftui.Text("Native refreshable is attached to the list; macOS does not surface the iOS pull gesture here.").
					Font(swiftui.FontCaption).
					ForegroundStyle(0.76, 0.81, 0.78, 1).
					LineLimit(2),
				swiftui.Spacer(),
			),
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
				swiftui.ShareLink("Share project", "https://github.com/tmc/swiftui"),
				swiftui.ShareLinkItem("Share note", swiftui.ShareItem{
					Title: "Bridge coverage note",
					Text:  "Shareable text from Go SwiftUI",
				}),
				swiftui.Spacer(),
			),
			swiftui.TextFromString(pastedText).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.72, 0.92, 1, 1),
			swiftui.TextFromString(pastedPayload).
				Font(swiftui.FontCaption).
				ForegroundStyle(0.72, 0.92, 1, 1),
			swiftui.Text("PasteButton stays text-only today. PasteButtonPayload, ShareItem, and DropPayload stay concrete: text, URL, and file-path payloads are normalized before the bridge handles them.").
				Font(swiftui.FontCaption).
				ForegroundStyle(0.76, 0.81, 0.78, 1),
			swiftui.HStackSpaced(12,
				swiftui.Text("Drag this payload").
					FontWeight(swiftui.WeightSemibold).
					AsView().
					Padding(10).
					BackgroundRoundedRect(0.10, 0.28, 0.42, 0.78, 12).
					DraggableText("Dropped from Go SwiftUI"),
				swiftui.TextFromString(dropped).
					AsView().
					Padding(10).
					BackgroundRoundedRect(0.18, 0.12, 0.08, 0.78, 12).
					DropDestination(func(payload swiftui.DropPayload) bool {
						switch {
						case payload.Text != "":
							dropped.Set("text: " + payload.Text)
						case payload.URL != "":
							dropped.Set("url: " + payload.URL)
						case payload.FilePath != "":
							dropped.Set("file: " + payload.FilePath)
						default:
							dropped.Set("empty payload")
						}
						return true
					}),
			),
		),
	)
}

func gridPanel() swiftui.View {
	var cells []swiftui.Viewable
	for i := 1; i <= 9; i++ {
		cells = append(cells, swiftui.Text(fmt.Sprintf("%02d", i)).
			Font(swiftui.FontHeadline).
			FontWeight(swiftui.WeightBold).
			MonospacedDigit().
			AsView().
			Padding(14).
			BackgroundRoundedRect(0.07, 0.10, 0.12, 0.72, 14))
	}
	return panel("Curated grids",
		swiftui.VStackSpaced(12,
			swiftui.LazyVGrid(
				[]swiftui.GridItem{
					swiftui.FlexibleGridItem(80, 160),
					swiftui.FlexibleGridItem(80, 160),
					swiftui.FlexibleGridItem(80, 160),
				},
				10,
				cells...,
			),
			swiftui.LazyHGrid(
				[]swiftui.GridItem{
					swiftui.FixedGridItem(44),
					swiftui.FixedGridItem(44),
				},
				8,
				swiftui.Text("A").AsView().Padding(10).BackgroundRoundedRect(0.16, 0.25, 0.16, 0.85, 10),
				swiftui.Text("B").AsView().Padding(10).BackgroundRoundedRect(0.18, 0.18, 0.28, 0.85, 10),
				swiftui.Text("C").AsView().Padding(10).BackgroundRoundedRect(0.28, 0.18, 0.18, 0.85, 10),
				swiftui.Text("D").AsView().Padding(10).BackgroundRoundedRect(0.24, 0.22, 0.12, 0.85, 10),
			),
		),
	)
}

func gradientPanel() swiftui.View {
	return panel("Gradient constructors",
		swiftui.HStackSpaced(12,
			swiftui.LinearGradientHorizontal(swiftui.RGB(0.16, 0.42, 0.62), swiftui.RGB(0.85, 0.45, 0.20)).
				Frame(220, 110).
				CornerRadius(18),
			swiftui.RadialGradient(swiftui.RGB(0.96, 0.78, 0.28), swiftui.RGB(0.09, 0.12, 0.18), swiftui.UnitPointCenter, 0, 160).
				Frame(220, 110).
				CornerRadius(18),
			swiftui.MeshGradient4(
				swiftui.RGB(0.18, 0.24, 0.62),
				swiftui.RGB(0.18, 0.58, 0.42),
				swiftui.RGB(0.68, 0.24, 0.18),
				swiftui.RGB(0.20, 0.12, 0.36),
			).Frame(220, 110).CornerRadius(18),
		),
	)
}

func footer() swiftui.View {
	return swiftui.HStack(
		swiftui.Text("SafeAreaInset footer").Font(swiftui.FontCaption).ForegroundStyle(0.90, 0.94, 0.92, 1),
		swiftui.Spacer(),
		swiftui.Text("generated + runtime-backed smoke surface").Font(swiftui.FontCaption).ForegroundStyle(0.72, 0.76, 0.75, 1),
	).Padding(10).Background(0.04, 0.05, 0.06, 0.88)
}

func panel(title string, content swiftui.View) swiftui.View {
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Text(title).
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightSemibold).
				ForegroundStyle(0.95, 0.97, 0.94, 1),
			swiftui.Spacer(),
		),
		content,
	).Padding(14).Background(0.05, 0.07, 0.09, 0.78).CornerRadius(18)
}
