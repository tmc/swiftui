//go:build darwin

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"runtime"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/a2uiruntime"
)

func init() { runtime.LockOSThread() }

var (
	serverFlag     = flag.String("server", "http://localhost:8090/sse", "A2UI SSE server URL")
	componentsFlag = flag.String("components", "", "path to components JSON file (offline mode)")
	dataFlag       = flag.String("data", "", "path to data model JSON file (offline mode)")
)

func main() {
	flag.Parse()

	rt := a2uiruntime.New(
		a2uiruntime.WithStrict(true),
	)
	client := a2uiruntime.NewClient(rt)
	revision := swiftui.NewIntState(0)
	statusRevision := swiftui.NewIntState(0)
	urlState := swiftui.NewStringState(*serverFlag)

	refresh := func(animated bool) {
		snap := rt.Snapshot()
		if animated {
			revision.SetAnimated(snap.Revision)
			statusRevision.SetAnimated(snap.StatusRevision)
			return
		}
		revision.Set(snap.Revision)
		statusRevision.Set(snap.StatusRevision)
	}

	connect := func() {
		url := urlState.Get()
		if url == "" {
			return
		}
		rt.SetTransport(a2uiruntime.HTTPTransport{ActionURL: a2uiruntime.ActionURLFromSSE(url)})
		go func() {
			err := client.ConnectSSE(context.Background(), url, func(snap a2uiruntime.Snapshot) {
				revision.SetAnimated(snap.Revision)
				statusRevision.SetAnimated(snap.StatusRevision)
			})
			if err != nil && err != context.Canceled {
				log.Printf("sse connect: %v", err)
			}
		}()
	}

	if *componentsFlag != "" {
		if err := rt.LoadFiles(*componentsFlag, *dataFlag); err != nil {
			log.Fatalf("load files: %v", err)
		}
		refresh(false)
	} else {
		go connect()
	}

	header := swiftui.AnimatedDynamicView(statusRevision, swiftui.TransitionOpacity, func(_ int) swiftui.View {
		snap := rt.Snapshot()
		presentation := snap.Presentation()
		title := "A2UI Runtime"
		if presentation.AgentName != "" {
			title = presentation.AgentName
		}
		badge := swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 2,
			swiftui.Text(title).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
			swiftui.Text(urlState.Get()).Font(swiftui.FontCaption).ForegroundStyleNamed("primary"),
		)
		if presentation.IconURL != "" {
			badge = swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 2,
				swiftui.HStackSpaced(8,
					swiftui.AsyncImageFit(presentation.IconURL, swiftui.ImageFitContain).Frame(18, 18),
					swiftui.Text(title).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
				),
				swiftui.Text(urlState.Get()).Font(swiftui.FontCaption).ForegroundStyleNamed("primary"),
			)
		}
		if snap.Status == a2uiruntime.StatusConnected || snap.Status == a2uiruntime.StatusConnecting || snap.Status == a2uiruntime.StatusFile {
			return swiftui.HStackSpaced(8,
				badge.
					Padding(10).
					BackgroundStyle("thinMaterial").
					CornerRadius(10).
					MaxFrameAligned(-1, 0, swiftui.HorizontalAlignmentLeading, swiftui.VerticalAlignmentCenter),
				swiftui.Button("Reconnect", connect).ButtonStyle(swiftui.ButtonStyleBordered),
			)
		}
		return swiftui.HStackSpaced(8,
			swiftui.TextField("Server URL", urlState, connect).TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
			swiftui.Button("Connect", connect).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
		)
	})

	content := swiftui.AnimatedDynamicView(revision, swiftui.TransitionOpacity, func(_ int) swiftui.View {
		return rt.RenderActiveSurface()
	})

	footer := swiftui.AnimatedDynamicView(statusRevision, swiftui.TransitionOpacity, func(_ int) swiftui.View {
		snap := rt.Snapshot()
		presentation := snap.Presentation()
		iconName := "circle.fill"
		var iconR, iconG, iconB float64
		statusText := "Disconnected"
		switch snap.Status {
		case a2uiruntime.StatusConnected:
			iconR, iconG, iconB = 0.3, 0.8, 0.4
			statusText = "Connected"
			if snap.SurfaceID != "" {
				statusText = fmt.Sprintf("Connected to %s", snap.SurfaceID)
			}
		case a2uiruntime.StatusConnecting:
			iconR, iconG, iconB = 0.9, 0.7, 0.2
			statusText = "Connecting..."
		case a2uiruntime.StatusFile:
			iconR, iconG, iconB = 0.4, 0.6, 0.9
			statusText = "Loaded from file"
		default:
			iconR, iconG, iconB = 0.8, 0.3, 0.3
		}
		right := fmt.Sprintf("Rev: %d", snap.Revision)
		if snap.LastError != "" {
			right = snap.LastError
		}
		return swiftui.HStackSpaced(6,
			swiftui.Image(iconName).ForegroundStyle(iconR, iconG, iconB, 1.0).ImageScale(swiftui.ImageScaleSmall),
			swiftui.Text(statusText).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Text(presentation.AgentName).Font(swiftui.FontCaption).ForegroundStyleNamed("tertiary"),
			swiftui.Text(right).Font(swiftui.FontCaption).ForegroundStyleNamed("tertiary").MonospacedDigit(),
		)
	})

	rootView := swiftui.VStackSpaced(0,
		header.Padding(12),
		swiftui.Divider(),
		swiftui.ScrollView(content.Padding(16)).MaxFrame(-1, -1),
		swiftui.Divider(),
		footer.PaddingEdge(swiftui.EdgeHorizontal, 12).PaddingEdge(swiftui.EdgeVertical, 6),
	)

	swiftui.Run(swiftui.AppConfig{
		Title:  "A2UI Renderer",
		Width:  700,
		Height: 600,
	}, rootView)
}
