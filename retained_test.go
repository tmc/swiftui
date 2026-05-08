package swiftui

import (
	"sync"
	"testing"
)

// setupRetainedBridgeStubs swaps in a fake libHandle and _SUIRelease so the
// real retain/release machinery runs without the bridge dylib. Returns a
// restore closure plus a *int counting _SUIRelease invocations.
func setupRetainedBridgeStubs(t *testing.T) (restore func(), releaseCalls *int) {
	t.Helper()
	oldHandle := libHandle
	oldRelease := _SUIRelease
	libHandle = 1
	calls := 0
	releaseCalls = &calls
	_SUIRelease = func(uintptr) { calls++ }
	return func() {
		libHandle = oldHandle
		_SUIRelease = oldRelease
	}, releaseCalls
}

func TestRetainedTransientReleaseIsIdempotent(t *testing.T) {
	restore, releaseCalls := setupRetainedBridgeStubs(t)
	defer restore()

	id := registerCallback(func() {})
	t.Cleanup(func() { unregisterCallback(id) })

	r := newRetainedTransient(123)
	if r == nil {
		t.Fatal("newRetainedTransient returned nil")
	}
	r.addCallbackID(id)
	r.release()
	r.release() // second release is a no-op

	if *releaseCalls != 1 {
		t.Fatalf("_SUIRelease calls = %d, want 1", *releaseCalls)
	}
	if fn := buttonCallbacks.lookup(id); fn != nil {
		t.Fatalf("callback %d still registered after release", id)
	}
	if r.ptr != 0 {
		t.Fatalf("ptr = %#x, want 0 after release", r.ptr)
	}
	if !r.released {
		t.Fatal("released flag not set")
	}
}

func TestRetainedTransientNilSafe(t *testing.T) {
	var r *retainedTransient
	// Must not panic. Both methods are documented as nil-safe.
	r.release()
	r.addCallbackID(1)
}

func TestRetainedTransientSkipsWhenLibNotLoaded(t *testing.T) {
	// With libHandle == 0 (as in normal tests before bridge stubs), the
	// constructor should return nil rather than fabricating a handle that
	// will never be releasable.
	oldHandle := libHandle
	libHandle = 0
	t.Cleanup(func() { libHandle = oldHandle })

	if r := newRetainedTransient(42); r != nil {
		t.Fatalf("newRetainedTransient returned %v, want nil when libHandle == 0", r)
	}
}

func TestRetainedOwnedConcurrentReleaseRaceSafe(t *testing.T) {
	// Exercised primarily under `go test -race`: multiple goroutines racing
	// to release the same handle. sync.Once guarantees _SUIRelease fires
	// exactly once, and the callback cleanup happens exactly once.
	restore, releaseCalls := setupRetainedBridgeStubs(t)
	defer restore()

	const goroutines = 16
	const iters = 64
	for iter := 0; iter < iters; iter++ {
		id := registerCallback(func() {})

		r := newRetainedOwned(uintptr(iter) + 1)
		if r == nil {
			t.Fatal("newRetainedOwned returned nil")
		}
		r.addCallbackID(id)

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()
				r.release()
			}()
		}
		wg.Wait()

		if fn := buttonCallbacks.lookup(id); fn != nil {
			t.Fatalf("iter %d: callback still registered after concurrent release", iter)
		}
	}

	if *releaseCalls != iters {
		t.Fatalf("_SUIRelease calls = %d, want %d (one per iteration)", *releaseCalls, iters)
	}
}

func TestRetainedAddCallbackIDTracking(t *testing.T) {
	restore, _ := setupRetainedBridgeStubs(t)
	defer restore()

	t.Run("owned", func(t *testing.T) {
		ids := []uintptr{
			registerCallback(func() {}),
			registerCallback(func() {}),
			registerCallback(func() {}),
		}
		r := newRetainedOwned(1)
		for _, id := range ids {
			r.addCallbackID(id)
		}
		// addCallbackID(0) is a documented no-op.
		r.addCallbackID(0)
		if got, want := len(r.callbackIDs), len(ids); got != want {
			t.Fatalf("callbackIDs len = %d, want %d", got, want)
		}
		r.release()
		for _, id := range ids {
			if fn := buttonCallbacks.lookup(id); fn != nil {
				t.Fatalf("callback %d still registered after release", id)
			}
		}
	})

	t.Run("transient", func(t *testing.T) {
		ids := []uintptr{
			registerCallback(func() {}),
			registerCallback(func() {}),
		}
		r := newRetainedTransient(2)
		for _, id := range ids {
			r.addCallbackID(id)
		}
		r.addCallbackID(0)
		if got, want := len(r.callbackIDs), len(ids); got != want {
			t.Fatalf("callbackIDs len = %d, want %d", got, want)
		}
		r.release()
		for _, id := range ids {
			if fn := buttonCallbacks.lookup(id); fn != nil {
				t.Fatalf("callback %d still registered after release", id)
			}
		}
	})
}

func TestRetainedTransientSecondReleaseDoesNotDoubleUnregister(t *testing.T) {
	// If the second release were to re-enter the callback loop we'd either
	// unregister an id that has been reused or hit a nil map lookup. The
	// released bool must gate both the callback cleanup and the _SUIRelease
	// call.
	restore, releaseCalls := setupRetainedBridgeStubs(t)
	defer restore()

	id := registerCallback(func() {})
	t.Cleanup(func() { unregisterCallback(id) })

	r := newRetainedTransient(7)
	r.addCallbackID(id)
	r.release()

	// Register a new callback that reuses the slot; a buggy second release
	// that looped over callbackIDs again would wipe it out.
	id2 := registerCallback(func() {})
	t.Cleanup(func() { unregisterCallback(id2) })

	r.release()

	if fn := buttonCallbacks.lookup(id2); fn == nil {
		t.Fatal("second retainedTransient.release wiped an unrelated callback id")
	}
	if *releaseCalls != 1 {
		t.Fatalf("_SUIRelease calls = %d after double release, want 1", *releaseCalls)
	}
}
