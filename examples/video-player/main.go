//go:build darwin
// +build darwin

// Command video-player demonstrates the AVKit SwiftUI overlay bridge.
//
// It creates an AVPlayer via the ObjC runtime, wraps it in a VideoPlayer
// SwiftUI view via the overlay bridge, and displays it in a window using
// swiftui.Run.
//
// Usage:
//
//	go run . [url]
package main

import (
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/avkit"
)

func init() { runtime.LockOSThread() }

func main() {
	// Ensure AVKit framework is loaded before SwiftUI bridge creates the window.
	purego.Dlopen("/System/Library/Frameworks/AVKit.framework/AVKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)

	urlStr := "https://devstreaming-cdn.apple.com/videos/streaming/examples/img_bipbop_adv_example_ts/master.m3u8"
	if len(os.Args) > 1 {
		urlStr = os.Args[1]
	}

	// Create NSURL and AVPlayer via ObjC runtime.
	nsurl := objc.ID(objc.GetClass("NSURL")).Send(
		objc.RegisterName("URLWithString:"),
		newNSString(urlStr),
	)
	player := objc.ID(objc.GetClass("AVPlayer")).Send(
		objc.RegisterName("playerWithURL:"),
		nsurl,
	)
	player.Send(objc.RegisterName("play"))

	// Wrap in VideoPlayer SwiftUI view via the overlay bridge.
	viewPtr := avkit.NewVideoPlayer(uintptr(player))
	videoView := swiftui.ViewFromPointer(viewPtr)
	sourceName := urlStr
	if i := strings.LastIndex(sourceName, "/"); i >= 0 && i < len(sourceName)-1 {
		sourceName = sourceName[i+1:]
	}
	if err :=

		// Display the video player in a window.
		swiftui.Run(swiftui.WithWindow(swiftui.AppConfig{
			Title:  "Video Player",
			Width:  800,
			Height: 600,
		}, swiftui.VStackSpaced(16,
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("Playback Demo").
						Font(swiftui.FontTitle2).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
					swiftui.Label("Streaming", "play.circle.fill").
						Font(swiftui.FontCaption).
						ForegroundStyle(swiftui.RGBA(0.3, 0.8, 0.4, 1.0)),
				),
				swiftui.HStack(
					swiftui.Text("AVPlayer bridged into SwiftUI with native controls and live playback.").
						Font(swiftui.FontCallout).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			),
			swiftui.GroupBox("Player",
				videoView.Frame(720, 420),
			).MaxFrame(-1, 0),
			swiftui.HStackSpaced(12,
				videoInfoCard("Source", sourceName),
				videoInfoCard("Transport", "Native AVKit controls"),
				videoInfoCard("Mode", "Autoplay on launch"),
			),
		).Padding(20))); err != nil {
		log.Fatal(err)
	}
}

func newNSString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(
		objc.RegisterName("stringWithUTF8String:"),
		s,
	)
}

func videoInfoCard(label, value string) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontCallout).
				FontWeight(swiftui.WeightMedium),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(swiftui.RGBA(0.18, 0.19, 0.22, 0.6)).
		CornerRadius(10)
}
