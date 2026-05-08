package swiftui

import (
	"runtime"
	"sync"
)

var (
	textRenderLibOnce sync.Once

	_SUISelectableText     func(*byte) uintptr
	_SUIInlineMarkdownText func(*byte) uintptr
)

func ensureTextRenderLibFuncs() {
	textRenderLibOnce.Do(func() {
		if libHandle != 0 {
			tryRegisterLibFunc(&_SUISelectableText, libHandle, "SUISelectableText")
			tryRegisterLibFunc(&_SUIInlineMarkdownText, libHandle, "SUIInlineMarkdownText")
		}
		setTextRenderUnavailableStubs()
	})
}

func setTextRenderUnavailableStubs() {
	if _SUISelectableText == nil {
		_SUISelectableText = func(textC *byte) uintptr {
			return _SUIText(textC)
		}
	}
	if _SUIInlineMarkdownText == nil {
		_SUIInlineMarkdownText = func(textC *byte) uintptr {
			return _SUISelectableText(textC)
		}
	}
}

// SelectableText creates a read-only text view with native selection enabled.
func SelectableText(text string) View {
	ensureTextRenderLibFuncs()
	var ptr uintptr
	withCString(text, func(textC *byte) {
		ptr = _SUISelectableText(textC)
	})
	return View{ptr: ptr, retained: newRetainedOwned(ptr)}
}

func markdownInlineText(source string) View {
	ensureTextRenderLibFuncs()
	var ptr uintptr
	withCString(source, func(sourceC *byte) {
		ptr = _SUIInlineMarkdownText(sourceC)
	})
	return View{ptr: ptr, retained: newRetainedOwned(ptr)}
}

func keepAliveViews(views ...View) {
	for _, view := range views {
		runtime.KeepAlive(view.retained)
	}
}
