package swiftui

import "testing"

func TestAccessibilityIdentifierMissingSymbolFallback(t *testing.T) {
	old := _SUIAccessibilityIdentifier
	t.Cleanup(func() {
		_SUIAccessibilityIdentifier = old
	})

	v := View{ptr: 123}
	_SUIAccessibilityIdentifier = nil
	if got := v.AccessibilityIdentifier("status"); got.ptr != v.ptr {
		t.Fatalf("nil symbol ptr = %d, want %d", got.ptr, v.ptr)
	}

	var called bool
	_SUIAccessibilityIdentifier = func(uintptr, *byte) uintptr {
		called = true
		return 0
	}
	_ = v.AccessibilityIdentifier("status")
	if !called {
		t.Fatal("registered fallback was not called")
	}
}

func TestRetainedReleaseIsIdempotent(t *testing.T) {
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

	r := &retained{ptr: 123, callbackIDs: []uintptr{id}}
	r.release()
	r.release()

	if releaseCalls != 1 {
		t.Fatalf("release calls = %d, want 1", releaseCalls)
	}
	callbackMu.Lock()
	_, ok := callbackMap[id]
	callbackMu.Unlock()
	if !ok {
		t.Fatalf("callback %d unregistered while Swift may still reference it", id)
	}
}
