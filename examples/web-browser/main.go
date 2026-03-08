//go:build darwin
// +build darwin

// Command web-browser demonstrates WebView embedding in SwiftUI from Go.
//
// It creates a WebPage, loads a URL, and displays it in a SwiftUI window
// with navigation controls using SF Symbols.
//
// Usage:
//
//	go run . [url]
package main

import (
	"os"
	"runtime"

	"github.com/ebitengine/purego"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

// sUIWebViewPage wraps a WebPage pointer into a SwiftUI View pointer.
var sUIWebViewPage func(uintptr) uintptr

func main() {
	url := "https://go.dev"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	// Create a WebPage and load the URL.
	page := swiftui.NewWebPageURL(url)

	// Register the SUIWebViewPage bridge function (exported from the dylib
	// but not yet bound in the Go layer).
	sym, _ := purego.Dlsym(purego.RTLD_DEFAULT, "SUIWebViewPage")
	purego.RegisterFunc(&sUIWebViewPage, sym)

	webViewPtr := sUIWebViewPage(page.Pointer())
	webView := swiftui.ViewFromPointer(webViewPtr)

	urlState := swiftui.NewStringState(url)

	swiftui.Run(swiftui.AppConfig{
		Title:  "Web Browser",
		Width:  900,
		Height: 700,
	}, swiftui.VStack(
		// Toolbar
		swiftui.HStackSpaced(4,
			swiftui.ButtonWithImage("chevron.left", func() {}).
				ButtonStyle(swiftui.ButtonStyleBorderless),
			swiftui.ButtonWithImage("chevron.right", func() {}).
				ButtonStyle(swiftui.ButtonStyleBorderless),
			swiftui.ButtonWithImage("arrow.clockwise", func() {
				page.Reload()
			}).ButtonStyle(swiftui.ButtonStyleBorderless),
			swiftui.TextField("Search or enter URL", urlState, func() {
				page.LoadURL(urlState.Get())
			}).TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
			swiftui.ButtonWithImage("square.and.arrow.up", func() {}).
				ButtonStyle(swiftui.ButtonStyleBorderless),
		).Padding(6),
		swiftui.Divider(),
		// Web content
		webView.
			WebViewBackForwardNavigationGestures(swiftui.WebViewBehaviorEnabled).
			WebViewMagnificationGestures(swiftui.WebViewBehaviorEnabled),
	))
}
