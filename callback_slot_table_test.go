// Copyright 2026 The swiftui Authors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package swiftui

import (
	"sync"
	"sync/atomic"
	"testing"
)

// TestCallbackTableRegisterLookup exercises the basic round-trip so that a
// registered callback is reachable via its returned ID.
func TestCallbackTableRegisterLookup(t *testing.T) {
	tab := newCallbackTable[func() int]()
	id := tab.register(func() int { return 42 })
	if id == 0 {
		t.Fatal("register returned zero id")
	}
	fn := tab.lookup(id)
	if fn == nil {
		t.Fatalf("lookup(%d) returned nil", id)
	}
	if got := fn(); got != 42 {
		t.Fatalf("fn() = %d, want 42", got)
	}
	// Zero id is always the nil sentinel.
	if tab.lookup(0) != nil {
		t.Fatal("lookup(0) returned non-nil fn")
	}
	// Out-of-range id is a no-op, not a panic.
	if tab.lookup(id + 100) != nil {
		t.Fatal("lookup(out-of-range) returned non-nil fn")
	}
}

// TestCallbackTableUnregisterClearsSlot verifies that unregister nils the
// slot so subsequent dispatches see the zero value.
func TestCallbackTableUnregisterClearsSlot(t *testing.T) {
	tab := newCallbackTable[func()]()
	id := tab.register(func() {})
	if tab.lookup(id) == nil {
		t.Fatal("lookup after register returned nil")
	}
	tab.unregister(id)
	if tab.lookup(id) != nil {
		t.Fatal("lookup after unregister returned non-nil fn")
	}
	// Unregistering again is a no-op.
	tab.unregister(id)
	// Unregistering id 0 is a no-op.
	tab.unregister(0)
	// Unregistering an out-of-range id is a no-op.
	tab.unregister(id + 1000)
}

// TestCallbackTableFreeListReuse verifies that an unregistered slot gets
// reused by the next register, so the backing slice does not grow
// monotonically across a long-running app.
func TestCallbackTableFreeListReuse(t *testing.T) {
	tab := newCallbackTable[func()]()
	id1 := tab.register(func() {})
	id2 := tab.register(func() {})
	if id1 == id2 {
		t.Fatalf("register returned duplicate ids %d/%d", id1, id2)
	}
	lenBefore := tab.len()
	tab.unregister(id1)
	id3 := tab.register(func() {})
	if id3 != id1 {
		t.Fatalf("id3 = %d, want reused %d", id3, id1)
	}
	if tab.len() != lenBefore {
		t.Fatalf("len() = %d after reuse, want %d", tab.len(), lenBefore)
	}
	_ = id2
}

// TestCallbackTableDoubleUnregisterIdempotent guards against the bug where a
// double unregister would double-push the slot index onto the free list,
// causing two subsequent registers to clobber each other on the same slot.
// The retained-release cleanup path in lib_test.go exercises this ordering,
// and the lifecycle test had real callbacks clobbered before this was fixed.
func TestCallbackTableDoubleUnregisterIdempotent(t *testing.T) {
	tab := newCallbackTable[func() int]()
	id := tab.register(func() int { return 1 })
	tab.unregister(id)
	tab.unregister(id) // must be a no-op; the slot is already free.

	// Register two new callbacks; they must land in distinct slots. Without
	// idempotent unregister, the free list would contain [idx, idx] and the
	// second register would reuse the same slot, clobbering the first.
	idA := tab.register(func() int { return 100 })
	idB := tab.register(func() int { return 200 })
	if idA == idB {
		t.Fatalf("duplicate ids after double-unregister: idA=%d idB=%d", idA, idB)
	}
	if fn := tab.lookup(idA); fn == nil || fn() != 100 {
		t.Fatalf("idA callback lost after double-unregister")
	}
	if fn := tab.lookup(idB); fn == nil || fn() != 200 {
		t.Fatalf("idB callback lost after double-unregister")
	}
}

// TestCallbackTableConcurrentRegisterAndDispatch hammers register, lookup and
// unregister from many goroutines. Intended to run under `go test -race` to
// surface any data race in the publish/consume sequence.
func TestCallbackTableConcurrentRegisterAndDispatch(t *testing.T) {
	tab := newCallbackTable[func() int]()
	const workers = 8
	const per = 2000
	var invocations atomic.Int64
	mk := func(v int) func() int {
		return func() int {
			invocations.Add(1)
			return v
		}
	}
	// Seed the table so dispatchers always have something to look at.
	seedIDs := make([]uintptr, 0, 32)
	for i := 0; i < cap(seedIDs); i++ {
		seedIDs = append(seedIDs, tab.register(mk(i)))
	}

	var wg sync.WaitGroup
	wg.Add(workers * 2)

	// Registrar goroutines register+unregister in a loop.
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			for i := 0; i < per; i++ {
				id := tab.register(mk(w*per + i))
				tab.unregister(id)
			}
		}(w)
	}
	// Dispatcher goroutines fire the seed ids lock-free.
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < per; i++ {
				for _, id := range seedIDs {
					if fn := tab.lookup(id); fn != nil {
						fn()
					}
				}
			}
		}()
	}
	wg.Wait()

	// Every dispatcher iteration invokes every seed id.
	want := int64(workers * per * len(seedIDs))
	if got := invocations.Load(); got < want {
		t.Fatalf("invocations = %d, want >= %d", got, want)
	}

	// Clean up seed ids so the table is empty for later tests.
	for _, id := range seedIDs {
		tab.unregister(id)
	}
}

// TestCallbackTableDispatchAfterGrowth covers the scenario where the slice
// is grown (copy-on-write) while a dispatcher is reading an older snapshot.
// After growth both old and new ids must resolve correctly.
func TestCallbackTableDispatchAfterGrowth(t *testing.T) {
	tab := newCallbackTable[func() int]()
	// Force the slice past its initial capacity so growth happens.
	const total = 64
	ids := make([]uintptr, total)
	for i := 0; i < total; i++ {
		v := i
		ids[i] = tab.register(func() int { return v })
	}
	for i, id := range ids {
		fn := tab.lookup(id)
		if fn == nil {
			t.Fatalf("lookup(%d) returned nil", id)
		}
		if got := fn(); got != i {
			t.Fatalf("id=%d fn() = %d, want %d", id, got, i)
		}
	}
}

// TestRegisterCallbackNilReturnsZero documents that the package-level
// register helpers return id 0 when fn is nil, so retained handles can
// safely call addCallbackID without tracking a real slot.
func TestRegisterCallbackNilReturnsZero(t *testing.T) {
	if id := registerCallback(nil); id != 0 {
		t.Fatalf("registerCallback(nil) = %d, want 0", id)
	}
	if id := registerBoolCallback(nil); id != 0 {
		t.Fatalf("registerBoolCallback(nil) = %d, want 0", id)
	}
	if id := registerStringCallback(nil); id != 0 {
		t.Fatalf("registerStringCallback(nil) = %d, want 0", id)
	}
	if id := registerPasteCallback(nil); id != 0 {
		t.Fatalf("registerPasteCallback(nil) = %d, want 0", id)
	}
	if id := registerHoverCallback(nil); id != 0 {
		t.Fatalf("registerHoverCallback(nil) = %d, want 0", id)
	}
	if id := registerCommandCallback(nil); id != 0 {
		t.Fatalf("registerCommandCallback(nil) = %d, want 0", id)
	}
	if id := registerViewBuilder(nil); id != 0 {
		t.Fatalf("registerViewBuilder(nil) = %d, want 0", id)
	}
	if id := registerFloatViewBuilder(nil); id != 0 {
		t.Fatalf("registerFloatViewBuilder(nil) = %d, want 0", id)
	}
	if id := registerGeometryBuilder(nil); id != 0 {
		t.Fatalf("registerGeometryBuilder(nil) = %d, want 0", id)
	}
}
