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
	"os"
	"runtime"

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

	// Wrap in VideoPlayer SwiftUI view via the overlay bridge.
	viewPtr := avkit.NewVideoPlayer(uintptr(player))

	// Display the video player in a window.
	swiftui.Run(swiftui.AppConfig{
		Title:  "Video Player",
		Width:  800,
		Height: 600,
	}, swiftui.ViewFromPointer(viewPtr))
}

func newNSString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(
		objc.RegisterName("stringWithUTF8String:"),
		s,
	)
}
