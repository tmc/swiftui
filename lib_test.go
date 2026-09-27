package swiftui

import "testing"

// TestAccessibilityIdentifierCallsTheBridge guards against the identifier being
// dropped on the floor. The method used to return the receiver unchanged when
// _SUIAccessibilityIdentifier was nil, which the dylib made routine: the symbol
// was missing from every shipped bridge, so every AccessibilityIdentifier call
// was a silent no-op and AX-driven tests saw unlabelled views. The bridge now
// exports it, the guard is gone, and a missing symbol panics in the stub.
func TestAccessibilityIdentifierCallsTheBridge(t *testing.T) {
	old := _SUIAccessibilityIdentifier
	t.Cleanup(func() {
		_SUIAccessibilityIdentifier = old
	})

	var got string
	_SUIAccessibilityIdentifier = func(_ uintptr, identifier *byte) uintptr {
		got = cStringToGoString(identifier)
		return 456
	}
	v := View{ptr: 123}
	if ret := v.AccessibilityIdentifier("status"); ret.ptr != 456 {
		t.Fatalf("ptr = %d, want 456", ret.ptr)
	}
	if got != "status" {
		t.Fatalf("identifier = %q, want %q", got, "status")
	}
}

// TestBridgeExportsEveryRequiredSymbol fails when the committed dylib is older
// than these bindings. A missing symbol builds clean and panics only when the
// call is made, so nothing else catches it.
func TestBridgeExportsEveryRequiredSymbol(t *testing.T) {
	if err := Err(); err != nil {
		t.Skipf("bridge not loaded: %v", err)
	}
	if missing := MissingSymbols(); len(missing) > 0 {
		t.Fatalf("dylib is missing %d symbol(s) the bindings register: %v", len(missing), missing)
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
