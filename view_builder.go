// Copyright (c) 2026 Travis Cline. All rights reserved.
//
// Tranche P6a: per-goroutine view-construction arena.
//
// Container view constructors (VStack, HStack, ZStack, List, Form, TabView,
// and their spaced/aligned/lazy variants) each need a transient []uintptr
// scratch buffer to marshal child pointers across the Swift bridge. Before
// P6a every call allocated a fresh slice via `make([]uintptr, len(children))`.
// On hot UI paths (e.g. body rebuilds per frame) that scratch is the largest
// single source of tracing-GC churn in the pure-Go half of the bridge.
//
// The arena below amortises the scratch slice over a per-goroutine pool of
// *viewBuilder values. Each builder owns two inline arrays — a [16]uintptr
// child scratch and a [8]modifier slot — sized to cover the 95th-percentile
// container without touching the heap. Containers whose child count exceeds
// the inline capacity spill to a heap-allocated slice; the pool machinery
// still owns the bookkeeping, so the common (≤16 children) case pays zero
// allocations for the scratch itself.
//
// The exported contract is deliberately narrow: callers never observe the
// pool, they invoke withContainerScratch and receive a ready-to-use []uintptr
// that remains valid for the duration of the callback. The callback must NOT
// leak the slice header — Reset zeroes references on release to avoid
// retaining Swift view pointers past the call.
//
// The modifier slot is plumbed for tranche P6b (modifier chain packing);
// this tranche does not consume it beyond making the capacity reachable.

package swiftui

import "sync"

// modifier is an opaque placeholder for tranche P6b (modifier chain packing).
// It is declared here so the arena can expose [8]modifier scratch without
// pulling in the as-yet-unbuilt modifier-packing machinery; P6b will flesh
// out the real representation and promote consumers off the placeholder.
type modifier struct {
	op    uint16
	flags uint16
	a, b  uint64
}

// viewBuilder is the per-goroutine scratch block for container construction.
//
// children holds child-view pointers marshalled across the Swift bridge.
// 16 is wide enough for the VStack/HStack/Form/List layouts found across
// the flagship examples and the swiftui-mlx-chat demo; larger containers
// spill to builder.overflow on demand.
//
// modifiers is reserved for tranche P6b's packed modifier chain. P6a never
// writes to it; the capacity exists so benchmarks can observe the final
// footprint of a builder without a size regression when P6b lands.
type viewBuilder struct {
	children  [16]uintptr
	nchildren int
	modifiers [8]modifier
	nmods     int
	// overflow backs heap-spill scratch when a container's child count
	// exceeds the inline array. Retained across Reset so subsequent
	// over-capacity containers on the same goroutine reuse the same
	// allocation. Cleared (nil'd out) only when the builder is put back
	// to the pool with an oversized spill to avoid pinning arbitrarily
	// large buffers.
	overflow []uintptr
}

var viewBuilderPool = sync.Pool{
	New: func() any { return &viewBuilder{} },
}

// acquireViewBuilder returns a builder with zeroed counters. Callers must
// call releaseViewBuilder exactly once on the returned value.
func acquireViewBuilder() *viewBuilder {
	return viewBuilderPool.Get().(*viewBuilder)
}

// releaseViewBuilder clears transient references and returns the builder to
// the pool. Scratch capacities are preserved so the next acquire reuses them.
// Overflow slices larger than a conservative cap (128 entries, i.e. 1 KiB on
// 64-bit) are dropped to prevent the pool from pinning unbounded memory after
// a single outsize container.
func releaseViewBuilder(b *viewBuilder) {
	b.Reset()
	// Drop pathologically large overflow buffers rather than caching them.
	// A 128-entry cap absorbs realistic lists/forms without retaining the
	// full buffer of a one-off 10k-element debug dump.
	if cap(b.overflow) > 128 {
		b.overflow = nil
	}
	viewBuilderPool.Put(b)
}

// Reset drops references without releasing scratch capacity. The inline
// children array is zeroed up through nchildren; the modifiers slots are
// zeroed up through nmods; overflow is truncated but retained.
func (b *viewBuilder) Reset() {
	for i := 0; i < b.nchildren && i < len(b.children); i++ {
		b.children[i] = 0
	}
	b.nchildren = 0
	for i := 0; i < b.nmods && i < len(b.modifiers); i++ {
		b.modifiers[i] = modifier{}
	}
	b.nmods = 0
	if b.overflow != nil {
		for i := range b.overflow {
			b.overflow[i] = 0
		}
		b.overflow = b.overflow[:0]
	}
}

// scratchPtrs returns a []uintptr of exactly n entries. For n ≤ 16 the slice
// aliases the inline array (no heap traffic). For n > 16 the slice is backed
// by the builder's overflow buffer, which is grown in place; the resulting
// slice header still escapes to the caller's frame but the backing array is
// reused across subsequent over-capacity calls on the same goroutine.
func (b *viewBuilder) scratchPtrs(n int) []uintptr {
	if n <= len(b.children) {
		b.nchildren = n
		return b.children[:n:n]
	}
	if cap(b.overflow) < n {
		b.overflow = make([]uintptr, n)
	} else {
		b.overflow = b.overflow[:n]
	}
	return b.overflow
}

// acquireChildScratch returns (builder, ptrs) where ptrs is a []uintptr of
// exactly len(children) entries populated from each child's viewPtr. The
// common case (≤16 children) uses the builder's inline array and performs
// zero heap allocations for the scratch itself; the >16 case falls back to
// the builder's overflow buffer.
//
// Callers MUST pair this with releaseViewBuilder(builder) once the returned
// ptrs slice is no longer needed — typically on the very next statement after
// the bridge call:
//
//	b, ptrs := acquireChildScratch(children)
//	var head *uintptr
//	if len(ptrs) > 0 {
//	    head = &ptrs[0]
//	}
//	ptr := _SUIVStack(head, int32(len(ptrs)))
//	releaseViewBuilder(b)
//
// The non-closure shape keeps the escape-analyser happy: the returned
// []uintptr header lives on the caller's stack frame (because its backing
// array lives on the builder, which the pool owns) and the surrounding
// local pointer variable (`ptr`) does not need to be heap-promoted.
func acquireChildScratch(children []Viewable) (*viewBuilder, []uintptr) {
	b := acquireViewBuilder()
	ptrs := b.scratchPtrs(len(children))
	for i, c := range children {
		ptrs[i] = c.viewPtr()
	}
	return b, ptrs
}
