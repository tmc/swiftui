package swiftui

import (
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// The state caches are pure Go structs with no Swift-side dependency, so
// they can be exercised without a live bridge. The goal of these tests is
// to confirm dirty-skip semantics directly (checkSet/observe) rather than
// to round-trip values through Swift. Tests that drive live state objects
// are covered by the existing bench harness and integration suites.

func TestIntStateCacheCheckSet(t *testing.T) {
	var c intStateCache
	if c.checkSet(7) {
		t.Fatalf("first checkSet must fall through (no cached value yet)")
	}
	if !c.checkSet(7) {
		t.Fatalf("second checkSet with same value must report a hit")
	}
	if c.checkSet(8) {
		t.Fatalf("checkSet with different value must fall through")
	}
	if !c.checkSet(8) {
		t.Fatalf("checkSet with same value after update must report a hit")
	}
}

func TestIntStateCacheObserveRefreshes(t *testing.T) {
	var c intStateCache
	c.observe(42)
	if !c.checkSet(42) {
		t.Fatalf("observe should seed the cache so the next Set(v) is a hit")
	}
	c.observe(99)
	if !c.checkSet(99) {
		t.Fatalf("observe should update the cache with new values")
	}
	if c.checkSet(42) {
		t.Fatalf("cache must fall through for values that are no longer current")
	}
}

func TestBoolStateCacheCheckSet(t *testing.T) {
	var c boolStateCache
	if c.checkSet(false) {
		t.Fatalf("first checkSet(false) must fall through (no cached value)")
	}
	if !c.checkSet(false) {
		t.Fatalf("repeat checkSet(false) must report a hit")
	}
	if c.checkSet(true) {
		t.Fatalf("toggle must fall through")
	}
	if !c.checkSet(true) {
		t.Fatalf("repeat checkSet(true) must report a hit")
	}
}

func TestFloatStateCacheCheckSet(t *testing.T) {
	var c floatStateCache
	if c.checkSet(1.5) {
		t.Fatalf("first checkSet must fall through")
	}
	if !c.checkSet(1.5) {
		t.Fatalf("repeat same value must be a hit")
	}
	if c.checkSet(1.6) {
		t.Fatalf("different value must fall through")
	}
}

func TestFloatStateCacheNaNAlwaysFFIs(t *testing.T) {
	var c floatStateCache
	if c.checkSet(math.NaN()) {
		t.Fatalf("first NaN must fall through")
	}
	// Back-to-back NaNs must still fall through because NaN != NaN per
	// IEEE-754; otherwise a caller could silently drop a request.
	if c.checkSet(math.NaN()) {
		t.Fatalf("second NaN must also fall through")
	}
	if c.checkSet(0.0) {
		t.Fatalf("after NaN, a real value must fall through")
	}
	if !c.checkSet(0.0) {
		t.Fatalf("repeat real value must hit")
	}
}

func TestColorStateCacheCheckSet(t *testing.T) {
	var c colorStateCache
	if c.checkSet(0.1, 0.2, 0.3, 1.0) {
		t.Fatalf("first checkSet must fall through")
	}
	if !c.checkSet(0.1, 0.2, 0.3, 1.0) {
		t.Fatalf("same tuple must hit")
	}
	if c.checkSet(0.1, 0.2, 0.3, 0.5) {
		t.Fatalf("different alpha must fall through")
	}
	if !c.checkSet(0.1, 0.2, 0.3, 0.5) {
		t.Fatalf("same updated tuple must hit")
	}
}

func TestStringStateCacheCheckSet(t *testing.T) {
	var c stringStateCache
	if c.checkSet("hello") {
		t.Fatalf("first checkSet must fall through")
	}
	if !c.checkSet("hello") {
		t.Fatalf("repeat same string must hit")
	}
	if c.checkSet("world") {
		t.Fatalf("different string must fall through")
	}
	if !c.checkSet("world") {
		t.Fatalf("repeat updated string must hit")
	}
	c.observe("external")
	if !c.checkSet("external") {
		t.Fatalf("observe must seed the cache for a following Set")
	}
}

// TestStringStateCacheEmptyStringDistinct verifies that the sentinel
// "no value set" state is distinct from "cached value is empty". This
// matters because Go string zero value is "".
func TestStringStateCacheEmptyStringDistinct(t *testing.T) {
	var c stringStateCache
	if c.checkSet("") {
		t.Fatalf("first checkSet(\"\") on a zero-value cache must fall through")
	}
	if !c.checkSet("") {
		t.Fatalf("repeat checkSet(\"\") must hit once cache is seeded")
	}
}

// TestIntStateCacheZeroValueDistinct mirrors the empty-string guarantee
// for integers: a fresh cache treats Set(0) as "do FFI" because the zero
// value of value atomic.Int64 would otherwise alias with "value is 0".
func TestIntStateCacheZeroValueDistinct(t *testing.T) {
	var c intStateCache
	if c.checkSet(0) {
		t.Fatalf("first checkSet(0) on a zero-value cache must fall through")
	}
	if !c.checkSet(0) {
		t.Fatalf("repeat checkSet(0) must hit once cache is seeded")
	}
}

// TestIntStateConcurrentSetSafe is a stress check that concurrent Setters
// on a shared IntState do not corrupt the cache. The final cached value
// must be one of the inputs (not a torn read) and must match the last
// value that actually reached the Swift side. Because Set is the only
// mutation path, final observation must also match.
//
// We do NOT use the live bridge here; a bare cache is enough to expose any
// torn writes.
func TestIntStateCacheConcurrentCheckSet(t *testing.T) {
	var c intStateCache
	const goroutines = 8
	const iters = 10000
	var wg sync.WaitGroup
	wg.Add(goroutines)
	var ffiCount int64
	values := []int64{1, 2, 3, 4, 5, 6, 7, 8}
	for g := 0; g < goroutines; g++ {
		v := values[g]
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				if !c.checkSet(v) {
					atomic.AddInt64(&ffiCount, 1)
				}
			}
		}()
	}
	wg.Wait()
	// If two goroutines race, each may see a miss, but the cache never
	// stores a value that isn't one of the inputs.
	final := c.value.Load()
	found := false
	for _, v := range values {
		if v == final {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("final cached value %d is not one of the concurrent inputs", final)
	}
	// At least one goroutine must have FFI'd; otherwise the cache is
	// pretending every Set is a hit which would be a correctness bug.
	if ffiCount == 0 {
		t.Fatalf("expected at least one FFI call through the cache; got 0")
	}
	runtime.GC()
}

// TestStringStateCacheConcurrentCheckSet covers the mutex-guarded string
// cache. Same contract as the int variant.
func TestStringStateCacheConcurrentCheckSet(t *testing.T) {
	var c stringStateCache
	const goroutines = 8
	const iters = 5000
	values := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	var wg sync.WaitGroup
	wg.Add(goroutines)
	var ffiCount int64
	for g := 0; g < goroutines; g++ {
		v := values[g]
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				if !c.checkSet(v) {
					atomic.AddInt64(&ffiCount, 1)
				}
			}
		}()
	}
	wg.Wait()
	finalPtr := c.p.Load()
	if finalPtr == nil {
		t.Fatalf("expected cache to hold a value after concurrent writers")
	}
	final := *finalPtr
	found := false
	for _, v := range values {
		if v == final {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("final cached string %q is not one of the concurrent inputs", final)
	}
	if ffiCount == 0 {
		t.Fatalf("expected at least one FFI call through the cache; got 0")
	}
}

// TestIntStateSetSameNoAllocation confirms the cache-hit path allocates
// nothing. This test uses the live IntState; if it fails after P2 lands,
// we've introduced a hidden alloc in the forwarder.
func TestIntStateSetSameNoAllocation(t *testing.T) {
	s := NewIntState(5)
	defer s.Release()
	// Warm.
	s.Set(5)
	allocs := testing.AllocsPerRun(100, func() {
		s.Set(5)
	})
	if allocs != 0 {
		t.Fatalf("Set(sameValue) allocates %v objects; want 0", allocs)
	}
}

// TestStringStateSetSameNoAllocation likewise for strings.
func TestStringStateSetSameNoAllocation(t *testing.T) {
	const v = "cache-hit"
	s := NewStringState(v)
	defer s.Release()
	s.Set(v)
	allocs := testing.AllocsPerRun(100, func() {
		s.Set(v)
	})
	if allocs != 0 {
		t.Fatalf("StringState.Set(sameValue) allocates %v objects; want 0", allocs)
	}
}
