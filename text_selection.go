package swiftui

import "sync"

// TextSelection describes one explicit text selection or caret position.
//
// Runtime surface.
//
// Offsets use UTF-16 code units so the range matches AppKit text-system
// selection semantics exactly.
type TextSelection struct {
	Start int
	End   int
}

// Caret reports whether the selection is collapsed to a single insertion point.
func (s TextSelection) Caret() bool {
	return s.Start == s.End
}

// Length reports the UTF-16 length of the current selection.
func (s TextSelection) Length() int {
	if s.End <= s.Start {
		return 0
	}
	return s.End - s.Start
}

// TextSelectionState owns explicit text selection state for editable controls.
//
// Bridge surface.
//
// The zero value is not usable. Use NewTextSelectionState.
type TextSelectionState struct {
	ptr      uintptr
	retained *retainedOwned
}

var (
	textSelectionLibOnce           sync.Once
	_SUIStateCreateTextSelection   func(int32, int32) uintptr
	_SUIStateGetTextSelectionStart func(uintptr) int32
	_SUIStateGetTextSelectionEnd   func(uintptr) int32
	_SUIStateSetTextSelection      func(uintptr, int32, int32)
)

func ensureTextSelectionBridgeFuncs() {
	textSelectionLibOnce.Do(func() {
		if libHandle != 0 {
			tryRegisterLibFunc(&_SUIStateCreateTextSelection, libHandle, "SUIStateCreateTextSelection")
			tryRegisterLibFunc(&_SUIStateGetTextSelectionStart, libHandle, "SUIStateGetTextSelectionStart")
			tryRegisterLibFunc(&_SUIStateGetTextSelectionEnd, libHandle, "SUIStateGetTextSelectionEnd")
			tryRegisterLibFunc(&_SUIStateSetTextSelection, libHandle, "SUIStateSetTextSelection")
		}
		setTextSelectionUnavailableStubs()
	})
}

func setTextSelectionUnavailableStubs() {
	stub := func(name string) {
		if loadErr != nil {
			panic("swiftui: " + name + ": " + loadErr.Error())
		}
		panic("swiftui: " + name + ": dylib not loaded")
	}
	if _SUIStateCreateTextSelection == nil {
		_SUIStateCreateTextSelection = func(int32, int32) uintptr {
			stub("SUIStateCreateTextSelection")
			return 0
		}
	}
	if _SUIStateGetTextSelectionStart == nil {
		_SUIStateGetTextSelectionStart = func(uintptr) int32 {
			stub("SUIStateGetTextSelectionStart")
			return 0
		}
	}
	if _SUIStateGetTextSelectionEnd == nil {
		_SUIStateGetTextSelectionEnd = func(uintptr) int32 {
			stub("SUIStateGetTextSelectionEnd")
			return 0
		}
	}
	if _SUIStateSetTextSelection == nil {
		_SUIStateSetTextSelection = func(uintptr, int32, int32) {
			stub("SUIStateSetTextSelection")
		}
	}
}

// NewTextSelectionState creates a new explicit text selection state.
func NewTextSelectionState(start, end int) *TextSelectionState {
	ensureTextSelectionBridgeFuncs()
	ptr := _SUIStateCreateTextSelection(int32(start), int32(end))
	return &TextSelectionState{ptr: ptr, retained: newRetainedOwned(ptr)}
}

// Get reports the current explicit selection range.
func (s *TextSelectionState) Get() TextSelection {
	if s == nil {
		return TextSelection{}
	}
	ensureTextSelectionBridgeFuncs()
	return TextSelection{
		Start: int(_SUIStateGetTextSelectionStart(s.ptr)),
		End:   int(_SUIStateGetTextSelectionEnd(s.ptr)),
	}
}

// Set replaces the current selection range.
func (s *TextSelectionState) Set(selection TextSelection) {
	if s == nil {
		return
	}
	ensureTextSelectionBridgeFuncs()
	_SUIStateSetTextSelection(s.ptr, int32(selection.Start), int32(selection.End))
}

// SetRange replaces the current selection range.
func (s *TextSelectionState) SetRange(start, end int) {
	s.Set(TextSelection{Start: start, End: end})
}

// Collapse moves the selection to a single caret position.
func (s *TextSelectionState) Collapse(offset int) {
	s.Set(TextSelection{Start: offset, End: offset})
}

// Release decrements the underlying Swift retain count.
func (s *TextSelectionState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}
