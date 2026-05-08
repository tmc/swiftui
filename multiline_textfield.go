package swiftui

import "sync"

var (
	multilineTextFieldLibOnce sync.Once

	_SUIMultilineTextFieldCallbacks func(*byte, uintptr, int32, int32, uintptr, uintptr) uintptr
)

func ensureMultilineTextFieldLibFuncs() {
	multilineTextFieldLibOnce.Do(func() {
		if libHandle != 0 {
			tryRegisterLibFunc(&_SUIMultilineTextFieldCallbacks, libHandle, "SUIMultilineTextFieldCallbacks")
		}
		setMultilineTextFieldUnavailableStubs()
	})
}

func setMultilineTextFieldUnavailableStubs() {
	if _SUIMultilineTextFieldCallbacks == nil {
		_SUIMultilineTextFieldCallbacks = func(placeholder *byte, state uintptr, _ int32, _ int32, onChangeID uintptr, onSubmitID uintptr) uintptr {
			return _SUITextFieldCallbacks(placeholder, state, onChangeID, onSubmitID)
		}
	}
}

func normalizeMultilineTextFieldLines(minLines int, maxLines int) (int32, int32) {
	if minLines < 1 {
		minLines = 1
	}
	if maxLines < minLines {
		maxLines = minLines
	}
	return int32(minLines), int32(maxLines)
}

// MultilineTextField creates a multiline text field bound to a StringState.
//
// The control grows from minLines through maxLines using SwiftUI's
// TextField(..., axis: .vertical) behavior.
func MultilineTextField(placeholder string, state *StringState, minLines int, maxLines int, onSubmit func()) View {
	return MultilineTextFieldCallbacks(placeholder, state, minLines, maxLines, nil, onSubmit)
}

// MultilineTextFieldCallbacks creates a multiline text field bound to a
// StringState with change and submit callbacks.
func MultilineTextFieldCallbacks(placeholder string, state *StringState, minLines int, maxLines int, onChange func(), onSubmit func()) View {
	ensureMultilineTextFieldLibFuncs()
	min, max := normalizeMultilineTextFieldLines(minLines, maxLines)
	onChangeID := registerCallback(onChange)
	onSubmitID := registerCallback(onSubmit)
	var ptr uintptr
	withCString(placeholder, func(placeholderC *byte) {
		ptr = _SUIMultilineTextFieldCallbacks(placeholderC, state.ptr, min, max, onChangeID, onSubmitID)
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(onChangeID)
	ret.retained.addCallbackID(onSubmitID)
	return ret
}
