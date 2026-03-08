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
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

// sUIWebViewPage wraps a WebPage pointer into a SwiftUI View pointer.
var sUIWebViewPage func(uintptr) uintptr

func normalizeAddress(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if u, err := url.Parse(s); err == nil && u.Scheme != "" && u.Host != "" {
		return u.String()
	}
	if strings.ContainsAny(s, " \t\n") || !strings.Contains(s, ".") {
		return "https://duckduckgo.com/?q=" + url.QueryEscape(s)
	}
	return "https://" + s
}

func main() {
	homeURL := "https://go.dev"
	address := homeURL
	if len(os.Args) > 1 {
		address = os.Args[1]
	}
	address = normalizeAddress(address)
	if address == "" {
		address = homeURL
	}

	// Create a WebPage and load the URL.
	page := swiftui.NewWebPageURL(address)

	// Register the SUIWebViewPage bridge function (exported from the dylib
	// but not yet bound in the Go layer).
	sym, _ := purego.Dlsym(purego.RTLD_DEFAULT, "SUIWebViewPage")
	purego.RegisterFunc(&sUIWebViewPage, sym)

	webViewPtr := sUIWebViewPage(page.Pointer())
	webView := swiftui.ViewFromPointer(webViewPtr)

	urlState := swiftui.NewStringState(address)

	loadAddress := func(raw string) {
		resolved := normalizeAddress(raw)
		if resolved == "" {
			return
		}
		urlState.Set(resolved)
		page.LoadURL(resolved)
	}

	swiftui.Run(swiftui.AppConfig{
		Title:  "Web Browser",
		Width:  900,
		Height: 700,
	}, swiftui.VStack(
		// Toolbar
		swiftui.HStackSpaced(4,
			swiftui.ButtonWithImage("house", func() {
				loadAddress(homeURL)
			}).
				ButtonStyle(swiftui.ButtonStyleBorderless),
			swiftui.ButtonWithImage("arrow.clockwise", func() {
				page.Reload()
			}).ButtonStyle(swiftui.ButtonStyleBorderless),
			swiftui.TextField("Search or enter URL", urlState, func() {
				loadAddress(urlState.Get())
			}).TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
			swiftui.ButtonWithImage("safari", func() {
				resolved := normalizeAddress(urlState.Get())
				if resolved == "" {
					return
				}
				urlState.Set(resolved)
				_ = exec.Command("open", resolved).Start()
			}).
				ButtonStyle(swiftui.ButtonStyleBorderless),
		).Padding(6),
		swiftui.Divider(),
		// Web content
		webView.
			WebViewBackForwardNavigationGestures(swiftui.WebViewBehaviorEnabled).
			WebViewMagnificationGestures(swiftui.WebViewBehaviorEnabled),
	))
}
