package swiftui

import (
	"sync"
	"sync/atomic"
	"testing"
)

// stubViewable satisfies Viewable with a caller-provided pointer, so the
// tests can exercise the arena without a loaded Swift bridge.
type stubViewable struct{ p uintptr }

func (s stubViewable) viewPtr() uintptr { return s.p }

func TestViewBuilderInlineCapacityNoOverflow(t *testing.T) {
	b := acquireViewBuilder()
	t.Cleanup(func() { releaseViewBuilder(b) })

	const n = 16
	ptrs := b.scratchPtrs(n)
	if len(ptrs) != n {
		t.Fatalf("len(ptrs) = %d, want %d", len(ptrs), n)
	}
	if cap(ptrs) != n {
		t.Fatalf("cap(ptrs) = %d, want %d (three-index slice keeps capacity pinned)", cap(ptrs), n)
	}
	if b.overflow != nil {
		t.Fatalf("overflow populated for inline case: %v", b.overflow)
	}
	// Slice must alias the inline array so writes are visible through b.children.
	for i := range ptrs {
		ptrs[i] = uintptr(i + 1)
	}
	for i, got := range b.children {
		if got != uintptr(i+1) {
			t.Fatalf("b.children[%d] = %d, want %d (slice does not alias inline array)", i, got, i+1)
		}
	}
}

func TestViewBuilderOverflowUsesSpillBuffer(t *testing.T) {
	b := acquireViewBuilder()
	t.Cleanup(func() { releaseViewBuilder(b) })

	const n = 64
	ptrs := b.scratchPtrs(n)
	if len(ptrs) != n {
		t.Fatalf("len(ptrs) = %d, want %d", len(ptrs), n)
	}
	if b.overflow == nil || len(b.overflow) != n {
		t.Fatalf("overflow not populated: len=%d want %d", len(b.overflow), n)
	}
	// Inline array stays at zero — spill path does not touch it.
	for i, v := range b.children {
		if v != 0 {
			t.Fatalf("b.children[%d] = %d, want 0 (inline array dirtied on spill)", i, v)
		}
	}
	// Write through the spill slice; subsequent scratchPtrs call on the
	// same builder at the same size must reuse the same backing array.
	for i := range ptrs {
		ptrs[i] = uintptr(0x1000 + i)
	}
	backing := &b.overflow[0]

	b.Reset()

	ptrs2 := b.scratchPtrs(n)
	if &b.overflow[0] != backing {
		t.Fatalf("Reset dropped overflow backing buffer; want reuse across Reset")
	}
	for i, v := range ptrs2 {
		if v != 0 {
			t.Fatalf("ptrs2[%d] = %d, want 0 (Reset failed to zero overflow)", i, v)
		}
	}
}

func TestViewBuilderResetClearsInlineSlots(t *testing.T) {
	b := acquireViewBuilder()
	t.Cleanup(func() { releaseViewBuilder(b) })

	ptrs := b.scratchPtrs(8)
	for i := range ptrs {
		ptrs[i] = uintptr(0xdeadbeef)
	}
	// Poke a modifier slot so Reset must walk it too.
	b.modifiers[0] = modifier{op: 1, flags: 2, a: 3, b: 4}
	b.nmods = 1

	b.Reset()

	if b.nchildren != 0 {
		t.Fatalf("nchildren = %d, want 0", b.nchildren)
	}
	if b.nmods != 0 {
		t.Fatalf("nmods = %d, want 0", b.nmods)
	}
	for i := 0; i < 8; i++ {
		if b.children[i] != 0 {
			t.Fatalf("b.children[%d] = %d, want 0 (Reset leaked reference)", i, b.children[i])
		}
	}
	if (b.modifiers[0] != modifier{}) {
		t.Fatalf("b.modifiers[0] = %+v, want zero value", b.modifiers[0])
	}
}

func TestAcquireChildScratchFillsFromViewable(t *testing.T) {
	children := []Viewable{
		stubViewable{p: 1},
		stubViewable{p: 2},
		stubViewable{p: 3},
	}
	b, ptrs := acquireChildScratch(children)
	t.Cleanup(func() { releaseViewBuilder(b) })

	if len(ptrs) != 3 {
		t.Fatalf("len(ptrs) = %d, want 3", len(ptrs))
	}
	for i, want := range []uintptr{1, 2, 3} {
		if ptrs[i] != want {
			t.Fatalf("ptrs[%d] = %d, want %d", i, ptrs[i], want)
		}
	}
}

func TestAcquireChildScratchEscapesOverCap(t *testing.T) {
	const n = 24 // > len([16]uintptr)
	children := make([]Viewable, n)
	for i := range children {
		children[i] = stubViewable{p: uintptr(i + 1)}
	}
	b, ptrs := acquireChildScratch(children)
	t.Cleanup(func() { releaseViewBuilder(b) })

	if len(ptrs) != n {
		t.Fatalf("len(ptrs) = %d, want %d", len(ptrs), n)
	}
	for i := 0; i < n; i++ {
		if ptrs[i] != uintptr(i+1) {
			t.Fatalf("ptrs[%d] = %d, want %d (overflow fill is wrong)", i, ptrs[i], i+1)
		}
	}
	if b.overflow == nil {
		t.Fatalf("overflow slice not engaged for n = %d", n)
	}
}

func TestViewBuilderArenaIsolationAcrossGoroutines(t *testing.T) {
	// Each goroutine acquires its own builder, writes a recognisable
	// pattern, verifies the builder is private, then releases. The test
	// fails if any two goroutines observe the same scratch slot at the
	// same time.
	const workers = 32
	const perWorker = 200
	var (
		wg       sync.WaitGroup
		failures atomic.Int64
	)
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for iter := 0; iter < perWorker; iter++ {
				children := make([]Viewable, 5)
				for i := range children {
					children[i] = stubViewable{p: uintptr(id*1000 + iter*10 + i + 1)}
				}
				b, ptrs := acquireChildScratch(children)
				for i, v := range ptrs {
					if v != uintptr(id*1000+iter*10+i+1) {
						failures.Add(1)
					}
				}
				releaseViewBuilder(b)
			}
		}(w)
	}
	wg.Wait()
	if f := failures.Load(); f != 0 {
		t.Fatalf("cross-goroutine scratch contamination: %d bad reads", f)
	}
}

func TestReleaseDropsOversizedOverflow(t *testing.T) {
	b := acquireViewBuilder()
	// Grow overflow past the 128-entry drop threshold.
	_ = b.scratchPtrs(256)
	if cap(b.overflow) <= 128 {
		t.Fatalf("cap(overflow) = %d, want > 128 to test drop path", cap(b.overflow))
	}
	releaseViewBuilder(b)

	// Re-acquire and verify the large buffer was not pinned. sync.Pool may
	// return a fresh builder or the same one with overflow cleared; either
	// way the overflow slice must not retain the oversize buffer.
	b2 := acquireViewBuilder()
	t.Cleanup(func() { releaseViewBuilder(b2) })
	if cap(b2.overflow) > 128 {
		t.Fatalf("cap(overflow) = %d after release; oversized buffer leaked back into pool", cap(b2.overflow))
	}
}

func TestContainerEscapeProducesValidView(t *testing.T) {
	// Regression test for the P6a escape path: the arena must correctly
	// hand a heap-backed []uintptr to the bridge when len(children) > 16.
	// We can't assert on the resulting Swift view from Go-only tests, but
	// we can assert that the generated ptrs slice matches the inputs and
	// that no child pointer is silently dropped or duplicated.
	const n = 20
	children := make([]Viewable, n)
	for i := range children {
		children[i] = stubViewable{p: uintptr(0x1000 + i)}
	}
	b, ptrs := acquireChildScratch(children)
	t.Cleanup(func() { releaseViewBuilder(b) })

	seen := make(map[uintptr]int, n)
	for _, p := range ptrs {
		seen[p]++
	}
	for i := 0; i < n; i++ {
		want := uintptr(0x1000 + i)
		if seen[want] != 1 {
			t.Fatalf("ptr 0x%x appeared %d times in spill slice, want 1", want, seen[want])
		}
	}
}

func TestModifierSlotCapacityReachesEight(t *testing.T) {
	// P6b will consume the [8]modifier slot; lock the capacity here so a
	// later tranche that edits viewBuilder.modifiers doesn't silently
	// shrink the scratch block.
	b := acquireViewBuilder()
	t.Cleanup(func() { releaseViewBuilder(b) })
	if got := len(b.modifiers); got != 8 {
		t.Fatalf("len(modifiers) = %d, want 8 (reserved for tranche P6b)", got)
	}
}
