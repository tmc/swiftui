package swiftui

import "testing"

func TestKeyboardCaptureRegistersStringCallback(t *testing.T) {
	oldHandle := libHandle
	oldKeyboardCapture := _SUIKeyboardCapture
	oldRelease := _SUIRelease
	t.Cleanup(func() {
		libHandle = oldHandle
		_SUIKeyboardCapture = oldKeyboardCapture
		_SUIRelease = oldRelease
	})

	libHandle = 1
	_SUIRelease = func(uintptr) {}

	var gotID uintptr
	_SUIKeyboardCapture = func(id uintptr) uintptr {
		gotID = id
		return 123
	}

	view := KeyboardCapture(func(string) bool { return true })
	if view.ptr != 123 {
		t.Fatalf("KeyboardCapture ptr = %d, want 123", view.ptr)
	}
	if gotID == 0 {
		t.Fatal("KeyboardCapture registered callback ID 0")
	}
	if fn := stringCallbacks.lookup(gotID); fn == nil {
		t.Fatalf("string callback %d was not registered", gotID)
	}

	view.Release()
	if fn := stringCallbacks.lookup(gotID); fn != nil {
		t.Fatalf("string callback %d still registered after release", gotID)
	}
}
