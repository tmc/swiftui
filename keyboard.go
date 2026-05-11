package swiftui

// KeyboardCapture creates a focusable view that forwards key presses to onKey.
func KeyboardCapture(onKey func(string) bool) View {
	callbackID := registerStringCallback(onKey)
	ptr := _SUIKeyboardCapture(callbackID)
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(callbackID)
	return ret
}
