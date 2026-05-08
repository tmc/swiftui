package swiftui

import (
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

var (
	benchShortString = "svc-0100"
	benchLongString  = strings.Repeat("swiftui-perf-", 80) // ~1040 bytes
	bench128String   = strings.Repeat("x", 128)
	bench1024String  = strings.Repeat("y", 1024)
	bench16String    = "sixteen-chars-!!"
)

func BenchmarkStringStateSet(b *testing.B) {
	b.Run("short_new", func(b *testing.B) {
		s := NewStringState("")
		defer s.Release()
		vals := []string{
			"svc-0001", "svc-0002", "svc-0003", "svc-0004",
			"svc-0005", "svc-0006", "svc-0007", "svc-0008",
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Set(vals[i&7])
		}
	})
	b.Run("short_same", func(b *testing.B) {
		s := NewStringState(benchShortString)
		defer s.Release()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Set(benchShortString)
		}
	})
	b.Run("long_new", func(b *testing.B) {
		s := NewStringState("")
		defer s.Release()
		long1 := benchLongString
		long2 := strings.Repeat("z", len(benchLongString))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i&1 == 0 {
				s.Set(long1)
			} else {
				s.Set(long2)
			}
		}
	})
}

func BenchmarkStringStateGet(b *testing.B) {
	b.Run("short", func(b *testing.B) {
		s := NewStringState(benchShortString)
		defer s.Release()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = s.Get()
		}
	})
	b.Run("long", func(b *testing.B) {
		s := NewStringState(benchLongString)
		defer s.Release()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = s.Get()
		}
	})
}

func BenchmarkWithCString(b *testing.B) {
	b.Run("16", func(b *testing.B) {
		s := bench16String
		b.ReportAllocs()
		b.ResetTimer()
		var sink *byte
		for i := 0; i < b.N; i++ {
			withCString(s, func(p *byte) { sink = p })
		}
		_ = sink
	})
	b.Run("128", func(b *testing.B) {
		s := bench128String
		b.ReportAllocs()
		b.ResetTimer()
		var sink *byte
		for i := 0; i < b.N; i++ {
			withCString(s, func(p *byte) { sink = p })
		}
		_ = sink
	})
	b.Run("1024", func(b *testing.B) {
		s := bench1024String
		b.ReportAllocs()
		b.ResetTimer()
		var sink *byte
		for i := 0; i < b.N; i++ {
			withCString(s, func(p *byte) { sink = p })
		}
		_ = sink
	})
}

func BenchmarkCStringGoString(b *testing.B) {
	// Build NUL-terminated buffers we can hand to the inbound helper.
	mk := func(n int) *byte {
		buf := make([]byte, n+1)
		for i := 0; i < n; i++ {
			buf[i] = 'a' + byte(i%26)
		}
		buf[n] = 0
		return &buf[0]
	}
	run := func(b *testing.B, p *byte) {
		b.ReportAllocs()
		b.ResetTimer()
		var sink string
		for i := 0; i < b.N; i++ {
			sink = cStringToGoString(p)
		}
		_ = sink
	}
	b.Run("16", func(b *testing.B) {
		p := mk(16)
		run(b, p)
		runtimeKeepByte(p)
	})
	b.Run("128", func(b *testing.B) {
		p := mk(128)
		run(b, p)
		runtimeKeepByte(p)
	})
	b.Run("1024", func(b *testing.B) {
		p := mk(1024)
		run(b, p)
		runtimeKeepByte(p)
	})
}

func runtimeKeepByte(p *byte) { _ = unsafe.Pointer(p) }

// P2 benchmarks: state dirty-skip coverage.
//
// These measure the no-op-on-equal Set and repeated Get hot paths that
// P2 targets. SetSame variants should collapse to a cache hit with no FFI
// and no allocation; SetChanging variants exercise the miss path to make
// sure dirty-skip doesn't regress the common flip case.

func BenchmarkIntStateSetSame(b *testing.B) {
	s := NewIntState(42)
	defer s.Release()
	s.Set(42)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(42)
	}
}

func BenchmarkIntStateSetChanging(b *testing.B) {
	s := NewIntState(0)
	defer s.Release()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(i & 1)
	}
}

func BenchmarkStringStateSetSame(b *testing.B) {
	v := strings.Repeat("x", 32)
	s := NewStringState(v)
	defer s.Release()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(v)
	}
}

func BenchmarkStringStateGetRepeated(b *testing.B) {
	s := NewStringState(benchShortString)
	defer s.Release()
	// Warm the cache.
	_ = s.Get()
	b.ReportAllocs()
	b.ResetTimer()
	var sink string
	for i := 0; i < b.N; i++ {
		sink = s.Get()
		sink = s.Get()
		sink = s.Get()
		sink = s.Get()
	}
	_ = sink
}

func BenchmarkBoolStateSetSame(b *testing.B) {
	s := NewBoolState(true)
	defer s.Release()
	s.Set(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(true)
	}
}

func BenchmarkFloatStateSetSame(b *testing.B) {
	s := NewFloatState(1.5)
	defer s.Release()
	s.Set(1.5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(1.5)
	}
}

func BenchmarkDateStateSetSame(b *testing.B) {
	s := NewDateState(1_700_000_000)
	defer s.Release()
	s.Set(1_700_000_000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(1_700_000_000)
	}
}

// P2-tail benchmarks: generation-counter cache consultation in State.Get.
//
// BenchmarkStateGenReadHot measures the steady-state Get() cost when the
// Swift-side generation counter has not advanced since the last Get -- the
// expected common case inside a SwiftUI body recompute. The charter exit
// target for P2 is ≤ 10 ns/op hot; the as! cast inside _SUIStateGen adds
// ~2-5 ns on top of the bare FFI call, so the number reported here is the
// sum of that plus a UInt32 compare plus the branch.
//
// BenchmarkStateGenReadMiss alternates Set + Get to force every Get down
// the miss path. This is the upper bound on the two-FFI cost (gen read
// plus typed value read). It is documented separately so a regression in
// the miss path is easy to spot against the hot bench.
//
// BenchmarkStringStateGetRepeated lives above; it stays the closeout
// signal for the P2 MISS ledger entry.

func BenchmarkStateGenReadHot(b *testing.B) {
	b.Run("int", func(b *testing.B) {
		s := NewIntState(42)
		defer s.Release()
		_ = s.Get() // warm cache
		b.ReportAllocs()
		b.ResetTimer()
		var sink int
		for i := 0; i < b.N; i++ {
			sink = s.Get()
		}
		_ = sink
	})
	b.Run("string", func(b *testing.B) {
		s := NewStringState(benchShortString)
		defer s.Release()
		_ = s.Get()
		b.ReportAllocs()
		b.ResetTimer()
		var sink string
		for i := 0; i < b.N; i++ {
			sink = s.Get()
		}
		_ = sink
	})
	b.Run("float", func(b *testing.B) {
		s := NewFloatState(1.5)
		defer s.Release()
		_ = s.Get()
		b.ReportAllocs()
		b.ResetTimer()
		var sink float64
		for i := 0; i < b.N; i++ {
			sink = s.Get()
		}
		_ = sink
	})
	b.Run("bool", func(b *testing.B) {
		s := NewBoolState(true)
		defer s.Release()
		_ = s.Get()
		b.ReportAllocs()
		b.ResetTimer()
		var sink bool
		for i := 0; i < b.N; i++ {
			sink = s.Get()
		}
		_ = sink
	})
}

func BenchmarkStateGenReadMiss(b *testing.B) {
	b.Run("int", func(b *testing.B) {
		s := NewIntState(0)
		defer s.Release()
		b.ReportAllocs()
		b.ResetTimer()
		var sink int
		for i := 0; i < b.N; i++ {
			s.Set(i)
			sink = s.Get()
		}
		_ = sink
	})
	b.Run("string", func(b *testing.B) {
		s := NewStringState("")
		defer s.Release()
		vals := []string{"a", "b"}
		b.ReportAllocs()
		b.ResetTimer()
		var sink string
		for i := 0; i < b.N; i++ {
			s.Set(vals[i&1])
			sink = s.Get()
		}
		_ = sink
	})
}

// BenchmarkCallbackDispatch measures the plain trampoline path for a
// registered zero-argument Go callback. This is the hottest shape on the
// Swift→Go boundary (button taps, completion closures) and feeds the P3
// charter target of ≤ 150 ns/op, 0 allocs/op.
func BenchmarkCallbackDispatch(b *testing.B) {
	id := registerCallback(func() {})
	defer unregisterCallback(id)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buttonCallbackTrampoline(id)
	}
}

// BenchmarkCallbackDispatchParallel exercises the same trampoline from many
// goroutines concurrently to surface lock contention. A slot-table dispatch
// must not block readers, so this should report 0 allocs and scale with -cpu.
func BenchmarkCallbackDispatchParallel(b *testing.B) {
	id := registerCallback(func() {})
	defer unregisterCallback(id)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buttonCallbackTrampoline(id)
		}
	})
}

// BenchmarkCallbackRegister measures the register+unregister round-trip. This
// path does take a lock even in the slot-table model; it exists to detect
// regressions in the register/free-list path.
func BenchmarkCallbackRegister(b *testing.B) {
	fn := func() {}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := registerCallback(fn)
		unregisterCallback(id)
	}
}

// BenchmarkBoolCallbackDispatch covers the Bool-shape trampoline so the P3
// optimization can't regress it silently while focusing on the zero-arg path.
func BenchmarkBoolCallbackDispatch(b *testing.B) {
	id := registerBoolCallback(func(bool) {})
	defer unregisterCallback(id)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		boolCallbackTrampoline(id, 1)
	}
}

// BenchmarkStringCallbackDispatch feeds a fixed NUL-terminated buffer into the
// string-shape trampoline so we can observe dispatch overhead separately from
// the inbound string-decoding cost measured by BenchmarkCStringGoString.
func BenchmarkStringCallbackDispatch(b *testing.B) {
	id := registerStringCallback(func(string) bool { return true })
	defer unregisterCallback(id)
	buf := []byte("ok\x00")
	p := &buf[0]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stringCallbackTrampoline(id, p)
	}
	runtimeKeepByte(p)
}

// P4 benchmarks: packed wire format for string slices and structured payloads.
//
// Pre-P4 the string-slice state bridge marshals every Set/Get as JSON (see
// bridge_extra.go: NewStringListState, StringListState.Set). BenchmarkMarshal*
// isolates the Go-side encoder cost. BenchmarkStringListStateSet exercises the
// full Go→Swift→Go round-trip that the P4 charter targets (≥ 5× faster at 100
// items vs. JSON baseline).
func benchStringSlice(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		// Mix of ASCII and multi-byte to keep the encoder honest.
		if i%7 == 0 {
			out[i] = "item-" + strings.Repeat("ü", 3) + "-" + benchShortString
		} else {
			out[i] = "item-" + benchShortString + "-" + strings.Repeat("x", i%31)
		}
	}
	return out
}

func BenchmarkMarshalStringSlice(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		values := benchStringSlice(n)
		b.Run(benchSizeName(n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			var sink string
			for i := 0; i < b.N; i++ {
				sink = marshalStringSlice(values)
			}
			_ = sink
		})
	}
}

func BenchmarkStringListStateSet(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		values := benchStringSlice(n)
		b.Run(benchSizeName(n), func(b *testing.B) {
			s := NewStringListState(nil)
			defer s.Release()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Set(values)
			}
		})
	}
}

func BenchmarkMarshalChoiceOptions(b *testing.B) {
	for _, n := range []int{10, 100} {
		options := make([]ChoiceOption, n)
		for i := range options {
			options[i] = ChoiceOption{
				Label: "label-" + benchShortString,
				Value: "value-" + benchShortString,
			}
		}
		b.Run(benchSizeName(n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			var sink string
			for i := 0; i < b.N; i++ {
				sink = marshalChoiceOptions(options)
			}
			_ = sink
		})
	}
}

func benchSizeName(n int) string {
	switch n {
	case 1:
		return "1"
	case 5:
		return "5"
	case 10:
		return "10"
	case 100:
		return "100"
	case 1000:
		return "1000"
	default:
		return "n"
	}
}

// benchRetainedStubs installs non-zero libHandle and a no-op _SUIRelease so
// that newRetained/release exercise their real fast paths during benchmarking.
// Returns a restore closure.
func benchRetainedStubs(b *testing.B) func() {
	b.Helper()
	oldHandle := libHandle
	oldRelease := _SUIRelease
	libHandle = 1
	_SUIRelease = func(uintptr) {}
	return func() {
		libHandle = oldHandle
		_SUIRelease = oldRelease
	}
}

func BenchmarkRetainAcquireOwned(b *testing.B) {
	restore := benchRetainedStubs(b)
	defer restore()
	b.ReportAllocs()
	b.ResetTimer()
	var sink *retainedOwned
	for i := 0; i < b.N; i++ {
		sink = newRetainedOwned(uintptr(i) + 1)
	}
	_ = sink
}

func BenchmarkRetainAcquireTransient(b *testing.B) {
	restore := benchRetainedStubs(b)
	defer restore()
	b.ReportAllocs()
	b.ResetTimer()
	var sink *retainedTransient
	for i := 0; i < b.N; i++ {
		sink = newRetainedTransient(uintptr(i) + 1)
	}
	_ = sink
}

func BenchmarkRetainReleaseOwned(b *testing.B) {
	restore := benchRetainedStubs(b)
	defer restore()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := newRetainedOwned(uintptr(i) + 1)
		r.release()
	}
}

func BenchmarkRetainReleaseTransient(b *testing.B) {
	restore := benchRetainedStubs(b)
	defer restore()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := newRetainedTransient(uintptr(i) + 1)
		r.release()
	}
}

// P6a benchmarks: view-builder arena + leaf allocation profile.
//
// Pre-P6a state: every container constructor (VStack/HStack/ZStack and the
// lazy/aligned variants) allocates a fresh []uintptr scratch buffer per call
// via `make([]uintptr, len(children))`. Leaf constructors (Text, Image,
// Spacer) each allocate a *retained handle, which is the irreducible cost of
// the View value itself.
//
// Exit criteria (from performance-optimization.md § Phase P6):
//   - BenchmarkLeafText ≤ 1 alloc/op (the retained handle).
//   - BenchmarkContainerBuildReuse shows arena reuse amortising the scratch
//     slice cost relative to BenchmarkContainerBuild.

// BenchmarkLeafText measures the per-call cost of the simplest leaf view.
// A single *retained allocation is the expected floor.
func BenchmarkLeafText(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Text("hi")
	}
}

// BenchmarkLeafSpacer measures the zero-argument leaf path. Also bounded by
// the *retained allocation.
func BenchmarkLeafSpacer(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Spacer()
	}
}

// BenchmarkLeafImage measures an SF Symbol leaf. Adds a withCString pooled
// copy on top of the *retained allocation; no additional heap traffic is
// expected from the Go side.
func BenchmarkLeafImage(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Image("star.fill")
	}
}

// BenchmarkContainerBuild constructs fresh leaf children on every iteration,
// so the per-frame cost includes both the container scratch slice and the
// child retained handles. Sub-benchmarks sweep the inline-capacity boundary
// (≤16 children) and the escape path (>16 children).
func BenchmarkContainerBuild(b *testing.B) {
	cases := []struct {
		name string
		n    int
	}{
		{"shallow_2", 2},
		{"medium_8", 8},
		{"inline_16", 16},
		{"deep_32", 32},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				children := make([]Viewable, tc.n)
				for j := 0; j < tc.n; j++ {
					children[j] = Text("x")
				}
				_ = VStack(children...)
			}
		})
	}
}

// BenchmarkContainerBuildReuse holds the child slice constant across
// iterations so the leaf-allocation cost amortises to zero. The remaining
// per-op allocations are what the P6a arena targets: the container scratch
// []uintptr inside VStack.
func BenchmarkContainerBuildReuse(b *testing.B) {
	cases := []struct {
		name string
		n    int
	}{
		{"shallow_2", 2},
		{"medium_8", 8},
		{"inline_16", 16},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			children := make([]Viewable, tc.n)
			for j := 0; j < tc.n; j++ {
				children[j] = Text("x")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = VStack(children...)
			}
		})
	}
}

// BenchmarkContainerBuildNoReuse exercises the arena spill path: the child
// count exceeds the inline [16]uintptr scratch capacity, so each call must
// escape to the heap. This is the worst case for the pool because the
// Get+Put bookkeeping is strictly overhead.
func BenchmarkContainerBuildNoReuse(b *testing.B) {
	children := make([]Viewable, 32)
	for j := range children {
		children[j] = Text("x")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VStack(children...)
	}
}
// BenchmarkModifierChain measures the cost of applying a chain of N modifiers
// to a base view. Each element of the chain is a distinct modifier kind so
// the benchmark exercises both scalar and enum payloads.
//
// Pre-P6b: every chain step materialises as its own bridge call, allocates a
// new View and *retained sentinel. Post-P6b: chains of length >=2 flush as a
// single SUIApplyModifiers call with a pooled scratch buffer.
func BenchmarkModifierChain(b *testing.B) {
	for _, n := range []int{1, 5, 10} {
		n := n
		b.Run(benchSizeName(n), func(b *testing.B) {
			base := Text("chain").AsView()
			defer base.Release()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v := applyBenchChain(base, n)
				_ = v
			}
		})
	}
}

// applyBenchChain applies a fixed sequence of up to n modifiers to base.
// The sequence is deterministic and exercises a mix of scalar, RGBA, enum,
// and zero-payload modifier kinds to avoid biasing toward one path.
//
// Post-P6b this routes through ApplyModifiers which packs N>=2 chains into a
// single SUIApplyModifiers bridge call. For N==1 the legacy per-modifier
// @_cdecl entry is used (no encode/decode overhead).
func applyBenchChain(base View, n int) View {
	switch n {
	case 1:
		return ApplyModifiers(base, benchChain1...)
	case 5:
		return ApplyModifiers(base, benchChain5...)
	case 10:
		return ApplyModifiers(base, benchChain10...)
	default:
		return base
	}
}

// Pre-built modifier chains for BenchmarkModifierChain. Kept as package-level
// slices so the chain construction itself does not contribute to per-op
// allocations; the benchmark measures the cost of flushing the chain through
// the bridge.
var (
	benchChain1  = []Modifier{ModPadding(8)}
	benchChain5  = []Modifier{
		ModPadding(8),
		ModOpacity(0.9),
		ModForegroundRGBA(0.2, 0.4, 0.6, 1.0),
		ModCornerRadius(4),
		ModScaleEffect(1.05),
	}
	benchChain10 = []Modifier{
		ModPadding(8),
		ModOpacity(0.9),
		ModForegroundRGBA(0.2, 0.4, 0.6, 1.0),
		ModBackgroundRGBA(0.95, 0.95, 0.95, 1.0),
		ModCornerRadius(4),
		ModScaleEffect(1.05),
		ModRotationEffect(1.5),
		ModBlur(0.5),
		ModZIndex(2.0),
		ModLayoutPriority(1.0),
	}
)

// P7 benchmarks: animated-state mutation coalescing.
//
// These benches measure the three paths exercised by P7's main-thread
// coalescing queue (see notes/p7-charter.md §4):
//
//   1. enqueue   — caller is off-main; write is appended to a kind
//                  partition, the queue arms a single DispatchQueue.main.async
//                  flush for the frame (amortised across all enqueues).
//   2. applyInline — caller is on main; write is applied synchronously
//                  inside withAnimation(kind) { ... }, skipping the queue.
//   3. flush     — main-thread drain that wraps each kind partition in
//                  one withAnimation(kind) scope and applies all pending
//                  writes for that kind in insertion order.
//
// The Go test harness runs worker goroutines off the main OS thread, so
// SetAnimatedWith from a Benchmark always routes through path (1). That
// is the hot path P7 optimises; these benches measure the steady-state
// enqueue overhead and the contention profile under parallel load.
//
// Inline and flush behaviour is exercised indirectly: BenchmarkAnimated
// StateSetInline stubs the bridge at the Go side (like benchRetained
// Stubs) so the setter's Go-side overhead can be measured without Swift
// roundtrips. Flush latency is not a Go-side bench concern; it is
// observed via Instruments in examples/bridge-coverage and reported in
// the P7.γ summary.
//
// Counter readings use SwiftUIDebug.Stats(); the exit criterion #3 ratio
// is computed as coalescedWrites / mainHops (see the charter). Note that
// Go benchmarks running off the main thread cannot pump the main-queue
// runloop, so mainHops may remain zero across a bench iteration. The
// representative BenchmarkCoalesceRatio sleeps between iterations to
// model a realistic 60 Hz frame cadence and reads the counters as a
// process-global delta; see the bench docstring for the measurement
// protocol.

// benchAnimatedStubs installs non-zero libHandle and no-op stubs for the
// three animated state setters so the Go-side dispatch cost can be
// measured without the Swift bridge round-trip. Returns a restore
// closure that must be deferred. Mirrors benchRetainedStubs's shape.
func benchAnimatedStubs(b *testing.B) func() {
	b.Helper()
	oldHandle := libHandle
	oldInt := _SUIStateSetIntAnimatedWith
	oldFloat := _SUIStateSetFloatAnimatedWith
	oldBool := _SUIStateSetBoolAnimatedWith
	libHandle = 1
	_SUIStateSetIntAnimatedWith = func(uintptr, int32, int32) {}
	_SUIStateSetFloatAnimatedWith = func(uintptr, float64, int32) {}
	_SUIStateSetBoolAnimatedWith = func(uintptr, int32, int32) {}
	return func() {
		libHandle = oldHandle
		_SUIStateSetIntAnimatedWith = oldInt
		_SUIStateSetFloatAnimatedWith = oldFloat
		_SUIStateSetBoolAnimatedWith = oldBool
	}
}

// BenchmarkAnimatedStateSetSerial measures single-goroutine throughput
// through IntState.SetAnimatedWith. Sub-benchmarks sweep across 1, 2,
// and 4 distinct animation kinds: the queue partitions by kind, so the
// same-kind case is the hottest path (one partition, append only) and
// the multi-kind case exercises the partition-map bookkeeping.
//
// This bench does NOT stub the bridge: it measures the full enqueue
// path including the Swift-side NSLock append and the
// DispatchQueue.main.async scheduling amortised across b.N iterations.
// Use BenchmarkAnimatedStateSetInline for a Go-side-only measurement.
func BenchmarkAnimatedStateSetSerial(b *testing.B) {
	cases := []struct {
		name  string
		kinds []AnimationKind
	}{
		{"single_kind", []AnimationKind{AnimationEaseInOut}},
		{"two_kinds", []AnimationKind{AnimationEaseInOut, AnimationSpring}},
		{"four_kinds", []AnimationKind{AnimationEaseInOut, AnimationEaseIn, AnimationEaseOut, AnimationSpring}},
	}
	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			s := NewIntState(0)
			defer s.Release()
			k := tc.kinds
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.SetAnimatedWith(i, k[i%len(k)])
			}
		})
	}
}

// BenchmarkAnimatedStateSetParallel exercises the enqueue path from
// many goroutines concurrently to surface contention on the Swift-side
// queue lock. Sub-benchmarks mirror the serial bench's kind sweep.
// Zero allocations per op is the target; the Swift-side NSLock is the
// only synchronization on the hot path, so contention should show up
// as wall-time per op rather than GC pressure.
func BenchmarkAnimatedStateSetParallel(b *testing.B) {
	cases := []struct {
		name  string
		kinds []AnimationKind
	}{
		{"single_kind", []AnimationKind{AnimationEaseInOut}},
		{"two_kinds", []AnimationKind{AnimationEaseInOut, AnimationSpring}},
		{"four_kinds", []AnimationKind{AnimationEaseInOut, AnimationEaseIn, AnimationEaseOut, AnimationSpring}},
	}
	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			s := NewIntState(0)
			defer s.Release()
			k := tc.kinds
			b.ReportAllocs()
			b.ResetTimer()
			var counter int64
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					i := int(counter)
					counter++
					s.SetAnimatedWith(i, k[i%len(k)])
				}
			})
		})
	}
}

// BenchmarkAnimatedStateSetInline measures the Go-side overhead of the
// animated setter with the bridge stubbed out. This is the closest
// approximation to the Thread.isMainThread applyInline fast path from
// Go: the Go-side work (cache observe, kind int32 marshal) runs
// identically, but the Swift-side round-trip is replaced with a no-op
// function pointer. Charter exit #4 wants this path near-zero overhead
// vs. pre-P7; this bench is the Go-side signal.
func BenchmarkAnimatedStateSetInline(b *testing.B) {
	restore := benchAnimatedStubs(b)
	defer restore()
	// Stubs installed: s.SetAnimatedWith performs cache.observe + kind
	// int32 cast + stub fn call with no bridge round-trip.
	s := &IntState{ptr: 1, retained: nil}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SetAnimatedWith(i, AnimationEaseInOut)
	}
}

// BenchmarkCoalesceUpperBound is the synthetic ceiling bench. It issues
// N writes in a tight loop with no frame-boundary think time and reads
// the counter deltas. The resulting ratio is an upper bound on
// coalescing — real callers spread writes across frames, so this is
// NOT the stop-condition gate. See BenchmarkCoalesceRatio for the
// representative measurement.
//
// This bench exists to verify the queue partition is actually coalescing
// (ratio >> 1) under the best-case synthetic load, which is a sanity
// check on α's queue primitive. Reported as sub-benchmark "synthetic".
func BenchmarkCoalesceUpperBound(b *testing.B) {
	if loadErr != nil {
		b.Skipf("bridge dylib not loaded: %v", loadErr)
	}
	const writesPerIter = 32
	s := NewIntState(0)
	defer s.Release()
	var dbg SwiftUIDebug
	dbg.ResetStats()
	before := dbg.Stats()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < writesPerIter; j++ {
			s.SetAnimatedWith(i*writesPerIter+j, AnimationEaseInOut)
		}
	}
	b.StopTimer()
	after := dbg.Stats()
	coalesced := after.BridgeCoalescedWrites - before.BridgeCoalescedWrites
	hops := after.BridgeMainHops - before.BridgeMainHops
	b.ReportMetric(float64(coalesced), "coalescedWrites")
	b.ReportMetric(float64(hops), "mainHops")
	if hops > 0 {
		b.ReportMetric(float64(coalesced)/float64(hops), "coalesceRatio_synthetic")
	}
}

// BenchmarkCoalesceRatio is the representative coalescing bench used as
// the stop-condition gate (charter §7 #1: N ≥ 5 coalescable writes per
// frame). Design goals (from the pre-bench design review):
//
//   - Multiple writes per frame, not just one (would under-coalesce).
//   - Frame-boundary think time of ~16 ms between fan-out batches so the
//     main-queue flush has an opportunity to fire between batches (would
//     otherwise over-coalesce into one synthetic flush).
//   - Multi-goroutine fan-out to match realistic UI state-update shape
//     (animated setters typically fire from gesture handlers and async
//     IO completions, not one synchronous loop).
//   - Mix of same-kind and cross-kind writes so the partition map is
//     exercised and not just a single partition.
//
// The ratio reported here is over a b.N-iteration window; each iteration
// simulates one animation frame worth of writes. The bench reports the
// observed ratio as "coalesceRatio"; charter §5 exit #3 target is > 1.5.
//
// Caveat: the Go test harness does not drive the main-thread runloop, so
// the Swift-side flush may not fire within the bench window and
// mainHops may remain zero. When that happens, the bench reports
// coalesceRatio as NaN/Inf and the stop-condition gate defers to the
// flagship-example trace (see the P7.γ summary for that measurement).
func BenchmarkCoalesceRatio(b *testing.B) {
	if loadErr != nil {
		b.Skipf("bridge dylib not loaded: %v", loadErr)
	}
	const writesPerFrame = 10
	const goroutines = 4
	const frameThink = 16 * time.Millisecond
	kinds := []AnimationKind{AnimationEaseInOut, AnimationSpring}
	s := NewIntState(0)
	defer s.Release()
	var dbg SwiftUIDebug
	dbg.ResetStats()
	before := dbg.Stats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Fan out writesPerFrame writes across goroutines.
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func(base int) {
				defer wg.Done()
				// Distribute writes so the total equals writesPerFrame
				// exactly (no integer-division truncation loss). Each
				// goroutine owns indices [base, goroutines, 2*goroutines, ...)
				// up to writesPerFrame.
				for j := base; j < writesPerFrame; j += goroutines {
					// Mix same-kind and cross-kind writes within the frame
					// window so the queue partition map is exercised.
					k := kinds[(base+j)%len(kinds)]
					s.SetAnimatedWith(j, k)
				}
			}(g)
		}
		wg.Wait()
		// Frame-boundary think time: give the main-queue flush a chance
		// to fire before the next frame's writes land. On a live runloop
		// this drains pending writes; in a Go test harness the flush
		// accumulates across frames (see caveat in docstring).
		time.Sleep(frameThink)
	}
	b.StopTimer()
	after := dbg.Stats()
	coalesced := after.BridgeCoalescedWrites - before.BridgeCoalescedWrites
	hops := after.BridgeMainHops - before.BridgeMainHops
	b.ReportMetric(float64(coalesced), "coalescedWrites")
	b.ReportMetric(float64(hops), "mainHops")
	if hops > 0 {
		b.ReportMetric(float64(coalesced)/float64(hops), "coalesceRatio")
	}
	// Report the per-frame write count as a diagnostic for the
	// stop-condition gate. Charter §7 #1: N < 5 writes/frame ⇒ STOP.
	b.ReportMetric(float64(writesPerFrame), "writesPerFrame")
}
