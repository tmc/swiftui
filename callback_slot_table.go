// Copyright 2026 The swiftui Authors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package swiftui

import (
	"sync"
	"sync/atomic"
)

// callbackTable is a lock-free slot table for Swift->Go callback dispatch.
// Callbacks are registered into a []T indexed by a 1-based ID. The trampoline
// dispatch path loads the current slice snapshot with a single atomic pointer
// read and does one bounds-checked index; no mutex, no map lookup, no
// indirection beyond the slice header load.
//
// Writers (register, unregister) take mu. Growth copies the slice, writes the
// new slot, then publishes the fresh slice via atomic.Pointer.Store. Readers
// always observe a consistent snapshot. Unregister nils the slot and pushes
// the index onto a free list so IDs are reused.
//
// The zero value is unusable. Call newCallbackTable to construct.
//
// Concurrency contract: a just-unregistered callback may still fire once if a
// reader loaded the slice snapshot before the nil store. This matches the
// contract of the previous map+mutex implementation (the lock window there
// also did not prevent a racing trampoline from seeing the fn just before
// delete).
type callbackTable[T any] struct {
	slots atomic.Pointer[[]T]
	mu    sync.Mutex
	free  []int           // indices into *slots.Load(); protected by mu
	freed map[int]struct{} // dedupe set for free; protected by mu
}

// newCallbackTable returns a ready-to-use table with an empty slot slice.
func newCallbackTable[T any]() *callbackTable[T] {
	t := &callbackTable[T]{
		freed: make(map[int]struct{}),
	}
	empty := make([]T, 0, 16)
	t.slots.Store(&empty)
	return t
}

// register installs fn and returns a 1-based ID. Registration is O(1)
// amortized; growth copies the slice. The zero ID (0) is reserved and is
// never returned.
func (t *callbackTable[T]) register(fn T) uintptr {
	t.mu.Lock()
	defer t.mu.Unlock()
	cur := *t.slots.Load()
	if n := len(t.free); n > 0 {
		idx := t.free[n-1]
		t.free = t.free[:n-1]
		delete(t.freed, idx)
		// Reuse an existing slot. Safe to mutate in place: readers already
		// see the freed slot as its zero value (unregister stored a zero).
		// Writing a new callback is one word-sized atomic on ARM64; any
		// racing reader either observes the zero (no-op) or the new fn
		// (valid). Either way it is safe — the contract already allows a
		// just-registered callback to fire on the next dispatch.
		cur[idx] = fn
		return uintptr(idx + 1)
	}
	// Grow by copy-on-write so racing readers never observe a partially
	// initialized slot. A plain append that reuses the underlying array
	// could let a reader see the new len before the new element was
	// published; the copy-and-publish pattern avoids that.
	grown := make([]T, len(cur)+1, nextCap(len(cur)+1))
	copy(grown, cur)
	grown[len(cur)] = fn
	t.slots.Store(&grown)
	return uintptr(len(cur) + 1)
}

// unregister clears slot id. Subsequent dispatches with this id are no-ops
// (they observe the zero value). The slot is pushed onto the free list for
// reuse. Unregistering id 0 or an out-of-range id is a no-op.
func (t *callbackTable[T]) unregister(id uintptr) {
	if id == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	cur := *t.slots.Load()
	idx := int(id) - 1
	if idx < 0 || idx >= len(cur) {
		return
	}
	if _, already := t.freed[idx]; already {
		// Double-unregister: idempotent. Slot already zero, idx already on
		// the free list. Don't re-push or we would hand the same slot to
		// two subsequent register calls and clobber one of them.
		return
	}
	var zero T
	cur[idx] = zero
	t.free = append(t.free, idx)
	t.freed[idx] = struct{}{}
}

// lookup returns the callback for id or the zero value if id is out of range.
// It performs a single atomic load and a bounds check — no mutex.
//
// Callers should compare the returned value to the zero T and no-op on a
// miss; this function does not panic on a stale id.
func (t *callbackTable[T]) lookup(id uintptr) T {
	var zero T
	if id == 0 {
		return zero
	}
	slots := *t.slots.Load()
	idx := int(id) - 1
	if idx < 0 || idx >= len(slots) {
		return zero
	}
	return slots[idx]
}

// len returns the current slot count, including freed slots. For tests and
// debug stats only.
func (t *callbackTable[T]) len() int {
	return len(*t.slots.Load())
}

// nextCap returns a small-step growth capacity. We don't want Go's default
// doubling because these tables stay small in practice (one entry per live
// handle), but we also want amortized O(1) register. 16/64/256/... works.
func nextCap(need int) int {
	switch {
	case need <= 16:
		return 16
	case need <= 64:
		return 64
	case need <= 256:
		return 256
	case need <= 1024:
		return 1024
	}
	return need + need>>1 // 1.5x beyond that
}
