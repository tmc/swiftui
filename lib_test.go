package swiftui

import "testing"

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
