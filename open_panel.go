package swiftui

// OpenPanel presents an NSOpenPanel and reports the selected file path.
//
// Curated surface.
func OpenPanel(label string, onPick func(path string)) View {
	ensureExtraLibFuncs()
	if label == "" {
		label = "Choose File"
	}
	callbackID := registerStringCallback(func(path string) bool {
		if onPick != nil {
			onPick(path)
		}
		return true
	})
	var ptr uintptr
	withCString(label, func(labelC *byte) {
		ptr = _SUIOpenPanel(labelC, callbackID)
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(callbackID)
	return ret
}
