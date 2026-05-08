package swiftui

import "testing"

func TestSelectableTextUsesNativeBridge(t *testing.T) {
	ensureTextRenderLibFuncs()
	old := _SUISelectableText
	t.Cleanup(func() { _SUISelectableText = old })

	var got string
	_SUISelectableText = func(textC *byte) uintptr {
		got = gostringFast(textC)
		return 0x1234
	}

	view := SelectableText("hello")
	if got != "hello" {
		t.Fatalf("SelectableText forwarded %q, want %q", got, "hello")
	}
	if view.Pointer() != 0x1234 {
		t.Fatalf("SelectableText pointer = %#x, want %#x", view.Pointer(), uintptr(0x1234))
	}
}

func TestViewOnTapGestureCountUsesCount(t *testing.T) {
	ensureSurfaceExtraLibFuncs()
	old := _SUIViewOnTapGestureCount
	t.Cleanup(func() { _SUIViewOnTapGestureCount = old })

	var gotCount int32
	_SUIViewOnTapGestureCount = func(ptr uintptr, count int32, callbackID uintptr) uintptr {
		gotCount = count
		return ptr + callbackID
	}

	view := View{ptr: 100, retained: nil}.OnTapGestureCount(2, func() {})
	if gotCount != 2 {
		t.Fatalf("tap count = %d, want 2", gotCount)
	}
	if view.Pointer() == 0 {
		t.Fatal("OnTapGestureCount returned zero pointer")
	}
}

func TestViewAnimationWithDurationForwardsDuration(t *testing.T) {
	ensureSurfaceExtraLibFuncs()
	old := _SUIViewAnimationWithDuration
	t.Cleanup(func() { _SUIViewAnimationWithDuration = old })

	var gotDuration float64
	_SUIViewAnimationWithDuration = func(ptr uintptr, kind int32, duration float64) uintptr {
		gotDuration = duration
		return ptr + 1
	}

	view := View{ptr: 41, retained: nil}.AnimationWithDuration(AnimationEaseInOut, 0.2)
	if gotDuration != 0.2 {
		t.Fatalf("duration = %v, want 0.2", gotDuration)
	}
	if view.Pointer() != 42 {
		t.Fatalf("AnimationWithDuration pointer = %#x, want %#x", view.Pointer(), uintptr(42))
	}
}

func TestViewEnvironmentOpenURLForwardsCallback(t *testing.T) {
	ensureOpenURLLibFuncs()
	old := _SUIViewEnvironmentOpenURL
	t.Cleanup(func() { _SUIViewEnvironmentOpenURL = old })

	var (
		gotPtr uintptr
		gotID  uintptr
	)
	_SUIViewEnvironmentOpenURL = func(ptr uintptr, callbackID uintptr) uintptr {
		gotPtr = ptr
		gotID = callbackID
		return ptr + 1
	}

	view := View{ptr: 41, retained: nil}.EnvironmentOpenURL(func(string) bool { return true })
	if gotPtr != 41 {
		t.Fatalf("EnvironmentOpenURL ptr = %#x, want %#x", gotPtr, uintptr(41))
	}
	if gotID == 0 {
		t.Fatal("EnvironmentOpenURL registered zero callback id")
	}
	t.Cleanup(func() { unregisterCallback(gotID) })
	if view.Pointer() != 42 {
		t.Fatalf("EnvironmentOpenURL pointer = %#x, want %#x", view.Pointer(), uintptr(42))
	}
}

func TestIntStateSetAnimatedWithDurationUsesTimedSetter(t *testing.T) {
	ensureSurfaceExtraLibFuncs()
	old := _SUIStateSetIntAnimatedWithDuration
	t.Cleanup(func() { _SUIStateSetIntAnimatedWithDuration = old })

	var (
		gotValue    int32
		gotKind     int32
		gotDuration float64
	)
	_SUIStateSetIntAnimatedWithDuration = func(ptr uintptr, value int32, kind int32, duration float64) {
		gotValue = value
		gotKind = kind
		gotDuration = duration
	}

	state := &IntState{ptr: 7}
	state.SetAnimatedWithDuration(11, AnimationEaseOut, 0.15)
	if gotValue != 11 || gotKind != int32(AnimationEaseOut) || gotDuration != 0.15 {
		t.Fatalf("timed setter = (%d, %d, %v), want (%d, %d, %v)", gotValue, gotKind, gotDuration, 11, int32(AnimationEaseOut), 0.15)
	}
}
