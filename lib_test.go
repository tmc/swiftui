package swiftui

import "testing"

func TestRetainedExplicitReleaseIsIdempotent(t *testing.T) {
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
	r.explicitRelease()
	r.explicitRelease()

	if releaseCalls != 1 {
		t.Fatalf("release calls = %d, want 1", releaseCalls)
	}
	callbackMu.Lock()
	_, ok := callbackMap[id]
	callbackMu.Unlock()
	if ok {
		t.Fatalf("callback %d still registered after explicitRelease", id)
	}
}

func TestRetainedFinalizerReleaseDoesNotFreeCallbacksOrSwiftObject(t *testing.T) {
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

	if releaseCalls != 0 {
		t.Fatalf("release calls = %d, want 0", releaseCalls)
	}
	callbackMu.Lock()
	_, ok := callbackMap[id]
	callbackMu.Unlock()
	if !ok {
		t.Fatalf("callback %d removed by finalizer release", id)
	}
	if r.ptr != 0 {
		t.Fatalf("retained ptr = %d, want 0", r.ptr)
	}
}
