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
		Width:  1000,
		Height: 720,
	}, swiftui.VStackSpaced(0,
		swiftui.VStackSpaced(10,
			swiftui.HStack(
				swiftui.Text("Browser").
					Font(swiftui.FontTitle2).
					FontWeight(swiftui.WeightBold),
				swiftui.Spacer(),
				swiftui.Label("Embedded WKWebView", "safari.fill").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
			),
			swiftui.HStackSpaced(8,
				swiftui.ButtonWithImage("house", func() {
					loadAddress(homeURL)
				}).
					ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.ButtonWithImage("arrow.clockwise", func() {
					page.Reload()
				}).
					ButtonStyle(swiftui.ButtonStyleBorderless),
				swiftui.TextField("Search or enter URL", urlState, func() {
					loadAddress(urlState.Get())
				}).
					SubmitLabel(swiftui.SubmitLabelGo).
					TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
				swiftui.Button("Open in Safari", func() {
					resolved := normalizeAddress(urlState.Get())
					if resolved == "" {
						return
					}
					urlState.Set(resolved)
					_ = exec.Command("open", resolved).Start()
				}).ButtonStyle(swiftui.ButtonStyleBordered),
			),
		).Padding(16).
			BackgroundStyle("regularMaterial"),
		swiftui.Divider(),
		swiftui.HStackSpaced(0,
			swiftui.VStackSpaced(14,
				swiftui.GroupBox("Favorites",
					swiftui.VStackSpaced(8,
						bookmarkButton("Go", "bolt.fill", "https://go.dev", loadAddress),
						bookmarkButton("SwiftUI", "sparkles", "https://developer.apple.com/xcode/swiftui/", loadAddress),
						bookmarkButton("GitHub", "chevron.left.forwardslash.chevron.right", "https://github.com/tmc/swiftui", loadAddress),
						bookmarkButton("Search Go FFI", "magnifyingglass", "purego ffi patterns", loadAddress),
					).Padding(10),
				).MaxFrame(-1, 0),
				swiftui.GroupBox("Quick Notes",
					swiftui.VStackSpaced(10,
						infoLine("Navigation", "Home, reload, search"),
						infoLine("Gestures", "Swipe + zoom"),
						infoLine("Intent", "Host shell around the page"),
					).Padding(10),
				).MaxFrame(-1, 0),
				swiftui.Spacer(),
			).Padding(16).
				Frame(260, 0).
				BackgroundStyle("windowBackground"),
			swiftui.Divider(),
			webView.
				WebViewBackForwardNavigationGestures(swiftui.WebViewBehaviorEnabled).
				WebViewMagnificationGestures(swiftui.WebViewBehaviorEnabled).
				WebViewContentBackground(swiftui.WebViewContentBackgroundHidden).
				MaxFrame(-1, -1),
		).MaxFrame(-1, -1),
		swiftui.Divider(),
		swiftui.HStack(
			swiftui.Text("Address").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Text(urlState.Get()).
				Font(swiftui.FontCaption).
				MonospacedDigit().
				ForegroundStyleNamed("secondary"),
		).Padding(10).
			BackgroundStyle("regularMaterial"),
	))
}

func bookmarkButton(label, icon, target string, load func(string)) swiftui.View {
	return swiftui.HStack(
		swiftui.Image(icon).
			ForegroundStyle(0.35, 0.65, 1.0, 1.0).
			ImageScale(swiftui.ImageScaleSmall).
			Frame(16, 0),
		swiftui.Button(label, func() {
			load(target)
		}).
			ButtonStyle(swiftui.ButtonStyleBorderless),
		swiftui.Spacer(),
	)
}

func infoLine(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightMedium),
	)
}
