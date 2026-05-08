// Hand-written state-cache helpers, consumed by the generated state.go
// forwarders. See notes/performance-optimization.md, phase P2.
//
// Scope: keeps Go-side "last value we wrote" caches so that
// State.Set(sameValue) can collapse to a no-op instead of crossing the FFI
// boundary. Eliminates the dominant idle-tick FFI churn described in the
// charter without expanding the Swift API surface.
//
// Correctness note (two-way bindings):
//
// SwiftUI Bindings (TextField text, Slider value, Toggle isOn, DatePicker
// selection, ColorPicker selection, etc.) can mutate the underlying
// BridgedXxxState.value on the Swift side without telling Go. That means a
// Go-side Set(cachedValue) skip is correct only against the last value Go
// itself observed -- it does not follow unrelated Swift writes. The
// documented contract:
//
//   * Set(v) skips the FFI when v == cachedValue. If Swift has drifted
//     (e.g. the user typed into a bound TextField), that drift is ignored
//     until Go explicitly calls Get.
//   * Get() always crosses the FFI boundary so Swift-side writes stay
//     visible, then refreshes the cache.
//
// Eliminating the Get() FFI requires a Swift-side monotonic generation
// counter (_SUIStateGen). That is deliberately a P2 follow-up -- see the
// P2 summary.

package swiftui

import (
	"math"
	"sync"
	"sync/atomic"
)

// intStateCache caches the last value Go wrote to an IntState.
// Reads and writes are atomic; Set's compare-and-skip uses Load+Store,
// which is safe across goroutines because the only ordering guarantee
// State.Set makes is eventual delivery to the MainActor -- two Go-side
// writers racing on the same state already had no total ordering.
type intStateCache struct {
	// value is the last int32 Go handed to the Swift side. Not
	// authoritative for Get (Swift may drift via two-way bindings).
	value atomic.Int64
	// hasValue marks the cache as populated; zero value of value is
	// ambiguous with "never Set". Avoids a spurious skip on the first
	// Set(0).
	hasValue atomic.Bool
}

// checkSet returns true if the cache already holds v (Set can be a no-op).
// Otherwise it updates the cache to v and returns false, meaning the
// caller must perform the FFI write.
func (c *intStateCache) checkSet(v int64) bool {
	if c.hasValue.Load() && c.value.Load() == v {
		return true
	}
	c.value.Store(v)
	c.hasValue.Store(true)
	return false
}

// observe refreshes the cache with a value returned from the Swift side.
func (c *intStateCache) observe(v int64) {
	c.value.Store(v)
	c.hasValue.Store(true)
}

// boolStateCache mirrors intStateCache for booleans.
type boolStateCache struct {
	value    atomic.Bool
	hasValue atomic.Bool
}

func (c *boolStateCache) checkSet(v bool) bool {
	if c.hasValue.Load() && c.value.Load() == v {
		return true
	}
	c.value.Store(v)
	c.hasValue.Store(true)
	return false
}

// floatStateCache caches a float64 via atomic.Uint64 + math.Float64bits.
// NaN is never treated as equal to anything, matching IEEE-754 semantics;
// that keeps Set(NaN) from being silently dropped.
type floatStateCache struct {
	bits     atomic.Uint64
	hasValue atomic.Bool
}

func (c *floatStateCache) checkSet(v float64) bool {
	if math.IsNaN(v) {
		c.bits.Store(math.Float64bits(v))
		c.hasValue.Store(true)
		return false
	}
	if c.hasValue.Load() {
		cached := math.Float64frombits(c.bits.Load())
		if !math.IsNaN(cached) && cached == v {
			return true
		}
	}
	c.bits.Store(math.Float64bits(v))
	c.hasValue.Store(true)
	return false
}

// colorStateCache holds four Doubles plus a guard flag. Writes race-safe
// under the same "no total ordering for concurrent Set" contract as the
// scalar caches; concurrent Set callers see atomic per-component stores.
type colorStateCache struct {
	mu                sync.Mutex
	r, g, b, a        float64
	hasValue          bool
}

func (c *colorStateCache) checkSet(r, g, b, a float64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hasValue && !math.IsNaN(r) && !math.IsNaN(g) && !math.IsNaN(b) && !math.IsNaN(a) &&
		c.r == r && c.g == g && c.b == b && c.a == a {
		return true
	}
	c.r, c.g, c.b, c.a = r, g, b, a
	c.hasValue = true
	return false
}

// stringStateCache holds the last Go string that was written. Reads use
// atomic.Pointer to keep the Set hot path lock-free; the nil pointer
// doubles as the "no value set yet" sentinel, so Set("") on a fresh
// state still crosses the bridge exactly once.
//
// Concurrent Setters race on the pointer swap but that already matches
// the State.Set concurrency contract (no total ordering across
// goroutines). Go string comparison is a single-word length check
// followed by a memcmp, so the read path is cheap.
type stringStateCache struct {
	p atomic.Pointer[string]
}

// checkSet returns true if v equals the cached string. Otherwise it
// installs v in the cache and returns false. The hit-path split keeps v
// from escaping to the heap on cache hits: taking &v in the miss-only
// update() helper means the escape analyzer treats v as heap-only when
// update is reached, leaving the common hit path alloc-free.
func (c *stringStateCache) checkSet(v string) bool {
	if cur := c.p.Load(); cur != nil && *cur == v {
		return true
	}
	c.update(v)
	return false
}

// update installs v in the cache. Extracted so checkSet's hit path does
// not force v to the heap.
//
//go:noinline
func (c *stringStateCache) update(v string) {
	c.p.Store(&v)
}

// observe refreshes the cache with a string returned from the Swift side.
//
//go:noinline
func (c *stringStateCache) observe(v string) {
	c.p.Store(&v)
}
