package swiftui

import "testing"

func TestBridgeRegistersGeneratedLayoutSymbols(t *testing.T) {
	if loadErr != nil {
		t.Fatalf("load bridge: %v", loadErr)
	}
	if libHandle == 0 {
		t.Fatal("bridge dylib not loaded")
	}
	if _SUIVStackAligned == nil {
		t.Fatal("SUIVStackAligned not registered")
	}
	if _SUIHStackAlignedSpaced == nil {
		t.Fatal("SUIHStackAlignedSpaced not registered")
	}
	if _SUIViewLabelsHidden == nil {
		t.Fatal("SUIViewLabelsHidden not registered")
	}
	if _SUIShareLinkItem == nil {
		t.Fatal("SUIShareLinkItem not registered")
	}
	if _SUIViewAccessibilityRotor == nil {
		t.Fatal("SUIViewAccessibilityRotor not registered")
	}
	if _SUIViewAccessibilityValue == nil {
		t.Fatal("SUIViewAccessibilityValue not registered")
	}
	if _SUITextFieldCallbacks == nil {
		t.Fatal("SUITextFieldCallbacks not registered")
	}
	if _SUIViewFocused == nil {
		t.Fatal("SUIViewFocused not registered")
	}
	if _SUISecureFieldCallbacks == nil {
		t.Fatal("SUISecureFieldCallbacks not registered")
	}
	if _SUITextEditorOnChange == nil {
		t.Fatal("SUITextEditorOnChange not registered")
	}
	if _SUIViewScrollContentBackgroundHidden == nil {
		t.Fatal("SUIViewScrollContentBackgroundHidden not registered")
	}
}

func TestRetainedOwnedReleaseIsIdempotent(t *testing.T) {
	oldHandle := libHandle
	oldRelease := _SUIRelease
	t.Cleanup(func() {
		libHandle = oldHandle
		_SUIRelease = oldRelease
	})

	libHandle = 1
	var releaseCalls int
	_SUIRelease = func(uintptr) { releaseCalls++ }

	id := registerCallback(func() {})
	t.Cleanup(func() { unregisterCallback(id) })

	r := &retainedOwned{ptr: 123, callbackIDs: []uintptr{id}}
	r.release()
	r.release()

	if releaseCalls != 1 {
		t.Fatalf("release calls = %d, want 1", releaseCalls)
	}
	if fn := buttonCallbacks.lookup(id); fn != nil {
		t.Fatalf("callback %d still registered after release", id)
	}
}
