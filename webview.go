package swiftui

// WebPage is a retained handle to a Swift WebPage object.
//
// Deprecated: Use webkit.NewWebPage from swift-bridge/webkit instead.
// This type provides only basic navigation; the webkit package exposes
// the full WebPage API (Title, URL, EstimatedProgress, JavaScript, PDF
// export, and more). To embed a web page in SwiftUI, use
// swiftui.ViewFromPointer(webkit.NewWebView(page)).
type WebPage struct {
	ptr      uintptr
	retained *retained
}

// Pointer returns the underlying Swift WebPage object pointer.
func (p *WebPage) Pointer() uintptr {
	if p == nil {
		return 0
	}
	return p.ptr
}

// NewWebPage creates a new WebPage handle.
//
// Deprecated: Use webkit.NewWebPage from swift-bridge/webkit instead.
func NewWebPage() *WebPage {
	ptr := _SUIWebPageCreate()
	return &WebPage{ptr: ptr, retained: newRetained(ptr)}
}

// NewWebPageURL creates a WebPage and starts loading the given URL.
//
// Deprecated: Use webkit.NewWebPageWithURL from swift-bridge/webkit instead.
func NewWebPageURL(url string) *WebPage {
	page := NewWebPage()
	page.LoadURL(url)
	return page
}

// LoadURL loads a URL string on the page.
//
// Deprecated: Use webkit.WebPage.LoadURL from swift-bridge/webkit instead.
func (p *WebPage) LoadURL(url string) {
	if p == nil || p.ptr == 0 {
		return
	}
	withCString(url, func(urlC *byte) {
		_SUIWebPageLoadURL(p.ptr, urlC)
	})
}

// Reload reloads the current page.
//
// Deprecated: Use webkit.WebPage.Reload from swift-bridge/webkit instead.
func (p *WebPage) Reload() {
	if p == nil || p.ptr == 0 {
		return
	}
	_SUIWebPageReload(p.ptr)
}

// Release decrements the underlying Swift retain count for this page.
func (p *WebPage) Release() {
	if p == nil || p.retained == nil {
		return
	}
	p.retained.release()
	p.retained = nil
	p.ptr = 0
}
