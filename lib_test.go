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
	if ok {
		t.Fatalf("callback %d still registered after release", id)
	}
}
