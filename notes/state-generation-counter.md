# State Generation Counter (`_SUIStateGen`) — Design Note

This is the design for the P2 tail: a Swift-side generation counter that
lets Go-side `Get()` skip the FFI cross when the cached value is still
current. It closes the one outstanding MISS from the P2 ledger
(`BenchmarkStringStateGetRepeated` ≤ 10 ns/op).

Scope: small. One Swift-side counter per `BridgedXxxState`, one new
`@_cdecl` accessor per shape, one Go-side cache consultation on `Get`.
No new public Go surface. No new wire type.

## Problem

After P2, state `Set` is alloc-free and skips the FFI cross when the
value is unchanged. But `Get` still crosses the FFI every call:

```
func (s *IntState) Get() int { return int(_SUIStateGetInt(s.ptr)) }
```

Even though the Go-side cache *has* the current value after the first
`Set`, `Get` cannot trust the cache, because Swift-side mutations
(two-way `Binding` writes from `TextField`, `Slider`,
`ColorPicker`, `DatePicker`) go through `s.value = newValue` on the
Swift object *without* calling back into Go. If Go returned the cached
value, it would miss those Swift-initiated writes.

The result is that steady-state `Get` in a SwiftUI body recompute still
pays the full FFI cross per call. `StringStateGetRepeated` measures
1.3 µs/op after P2, driven entirely by the Swift→Go round trip and the
string allocation. P3 cannot help here — the callback fast path is a
different edge. P1 already did the string marshaling.

## Solution

Give each `BridgedXxxState` a monotonically-increasing `gen: UInt32`
counter, bumped on every write (including two-way Binding writes).
Expose one read-only `@_cdecl` per shape:

```swift
@_cdecl("SUIStateGen")
public func SUIStateGen(_ ref: UnsafeMutableRawPointer) -> UInt32 {
    Unmanaged<AnyBridgedState>.fromOpaque(ref).takeUnretainedValue().gen
}
```

Go-side `Get` then becomes:

```go
func (s *IntState) Get() int {
    swiftGen := _SUIStateGen(s.ptr)
    if swiftGen == s.cache.lastSeenGen {
        return s.cache.value
    }
    v := int(_SUIStateGetInt(s.ptr))
    s.cache.value = v
    s.cache.lastSeenGen = swiftGen
    return v
}
```

The cached value is returned when `swiftGen` matches the last seen
generation. The FFI cross still happens, but it is **one cheap `UInt32`
load** instead of a string conversion plus allocation plus copy. Cost
drops from ~1.3 µs/op to an estimated ~5–15 ns/op for the generation
read plus the branch, which meets the ≤ 10 ns/op charter bar assuming
the bridge call overhead is in that range — if not, the MISS framing
updates but the direction is still right.

Cost model per `Get`:
- cache hit: one Swift→Go FFI for `SUIStateGen` + UInt32 compare.
- cache miss: one Swift→Go FFI for `SUIStateGen` + one for the typed
  get + cache update. Two FFI calls vs. today's one, but this path only
  runs on actual mutation, not on every body recompute.

The two-FFI miss cost is acceptable because in steady state (slider not
being dragged, text field idle), `Get` is the hot path and cache hits
dominate.

## Counter Design

### Per-state, not global

A global counter would thrash every state's cache on every unrelated
write. Per-state keeps each state independent and matches how SwiftUI's
own observation works.

### `UInt32`, not `UInt64`

- U32 is enough: at 1 bump per 10 ns (unreasonably high), a U32 wraps
  in ~43 seconds. At 1 bump per µs (realistic for a dragged slider),
  it wraps in ~71 minutes. Wraparound is safe because we compare for
  equality, not ordering.
- U32 packs into one register on the Swift side and matches Go
  `uint32` without platform-dependent width.

Alternative considered: `UInt64` with fetch-and-increment via atomics.
Rejected because it doubles the read cost on older hardware and we do
not need the larger range.

### Bump semantics

The counter must bump on **every** write path that can change `value`,
including:

- `SUIStateSetInt`, `SUIStateSetString`, `SUIStateSetColor*`, all
  typed setters
- animated setters (`SUIStateSetIntAnimatedWith`)
- two-way Binding writes from SwiftUI controls (`$text` on
  `TextField`, `$value` on `Slider`, etc.). These route through
  `suiStringBinding` and similar helpers in
  `bridge_helpers.gen.swift`; those `set` closures must also bump the
  counter.

Missing any write path reintroduces cache staleness. The test plan must
cover each write entry point.

### Bump ordering

Bump **after** the value write, not before. If `gen` is visible as
"changed" while `value` still holds the old data, a Go-side reader could
load the new gen, race the value read, and see stale data.

```swift
s.value = newValue
s.gen &+= 1    // bump after write; &+= for explicit wraparound
```

In practice this is fine because all state mutations happen on
MainActor today, but the ordering invariant is worth documenting — if
coalescing work (P7) ever moves writes off-main, the invariant becomes
load-bearing.

### Initial value

`gen` starts at `0`. The Go-side cache should start at a sentinel that
cannot equal any valid `gen`. Simplest: a separate `valid bool` on the
cache that starts `false` and flips to `true` on first populate. This
avoids choosing between "0 means fresh" and "0 means unread" confusion.

```go
type intStateCache struct {
    mu           sync.Mutex // or atomic fields — match P2 shape
    value        int
    lastSeenGen  uint32
    valid        bool
}
```

Match P2's lock-free shape where possible. The string cache already
uses `atomic.Pointer[string]`; the generation counter can sit next to
it as an `atomic.Uint32`.

## Bulk Variant — Deferred

The original P2 charter mentioned `_SUIStateGenBulk(ids, out)` for
amortizing generation reads across many states in a single body
recompute. Deferred unless it fits naturally because:

- a single `SUIStateGen` call is already cheap (one pointer load +
  one UInt32 return),
- Bulk requires a new wire type (pointer array + length) and a
  Go-side slice pool — cost not justified by a single-digit-ns
  per-state saving,
- if Get-heavy workloads show FFI cost dominating a body recompute in
  profiles, revisit — but do not build speculatively.

If it is ever built: follow the P4 packed-wire pattern (a single
allocated `[]uint32` in Go, reused per body recompute) rather than
inventing a new marshaling shape.

## Go-side Shape

Add one field to each `*StateCache` struct in `state_cache.go`:

```go
type intStateCache struct {
    current      atomic.Int64
    lastSeenGen  atomic.Uint32
    valid        atomic.Bool
}
```

`Get` becomes:

```go
func (s *IntState) Get() int {
    gen := _SUIStateGen(s.ptr)
    if s.cache.valid.Load() && gen == s.cache.lastSeenGen.Load() {
        return int(s.cache.current.Load())
    }
    v := int(_SUIStateGetInt(s.ptr))
    s.cache.current.Store(int64(v))
    s.cache.lastSeenGen.Store(gen)
    s.cache.valid.Store(true)
    return v
}
```

Same shape for Bool / Float / Color / Date / String. `String` uses the
existing `atomic.Pointer[string]` plus a sibling `atomic.Uint32` for
the generation.

Atomics are independent (generation and value may momentarily disagree
during a cache update race), but that is safe: on disagreement we take
the miss path and re-read from Swift, which is the same as the first
read after any mutation. Exact consistency is not required — we only
need to *eventually* catch Swift-side writes, which the counter bump
guarantees.

## Swift-side Shape

Add one `@Observable`-compatible generation field to each
`BridgedXxxState`:

```swift
@Observable
final class BridgedIntState: @unchecked Sendable {
    var value: Int
    private(set) var gen: UInt32 = 0
    init(_ v: Int) { self.value = v }

    func setAndBump(_ v: Int) {
        self.value = v
        self.gen &+= 1
    }
}
```

Every call site currently doing `s.value = newValue` switches to
`s.setAndBump(newValue)`. The helper is the only place the bump
lives, so there is one audit point for correctness.

For two-way Bindings (`suiStringBinding`), the Binding's `set` closure
calls `setAndBump` instead of assigning `value` directly.

## Test Plan

Behavioral tests, not just benchmarks.

1. **Cache hit**: `state.Set(v); _ = state.Get(); _ = state.Get()` —
   second `Get` should not call `_SUIStateGetXxx`. Verify by
   instrumenting the FFI call count in a test build, or by checking
   allocation counts (`AllocsPerRun`) since only the miss path
   allocates for strings.

2. **Go-side Set invalidates**: `state.Set(v1); state.Set(v2);
   state.Get()` — second `Get` must see `v2`.

3. **Swift-side Set via Binding invalidates**: simulate a TextField
   write path. Hard to do in isolation; proxy test is to call the
   `setAndBump` helper directly from a test hook and verify `Get`
   observes the new value.

4. **Generation wraparound**: bump the counter 2³² + 1 times. Verify
   `Get` still behaves correctly. In practice this test uses a
   shortened counter (e.g. UInt8) in a unit test, not the real U32.

5. **Concurrent readers during write**: fan out N goroutines calling
   `Get`, while another goroutine calls `Set`. Under `-race`. All
   readers must eventually observe the written value; no data race on
   the cache fields.

6. **No-mutation baseline**: 1M `Get()` calls on a never-mutated state
   should hit the cache every time after the first one.

## Benchmarks

Extend `performance_benchmark_test.go` with:

- `BenchmarkStateGenReadHot` — steady-state `Get()` after initial
  populate. This should land ≤ 10 ns/op on the reference bar.
- `BenchmarkStateGenReadMiss` — alternating `Set` + `Get` to force
  misses. Upper bound on the two-FFI miss cost.
- Keep existing `BenchmarkStringStateGetRepeated` as the closeout
  signal for the P2 ledger.

Commit bench artifacts under `testdata/perf/bench-p2tail-before.txt`
and `bench-p2tail-after.txt`.

## Non-Goals

- no new public Go API (internal cache mechanism only),
- no new SwiftUI surface (counter is an implementation detail of
  `BridgedXxxState`),
- no bulk variant unless a profile demands it,
- no changes to coalescing semantics (P7 territory),
- no changes to `State.Set` semantics — the "safe from any goroutine,
  dispatches to MainActor" contract documented in `doc.go` holds.

## Sequencing

1. Land generator-side template changes first (`swiftui_templates.go`
   in appledocs) so the next applegen regen emits the counter field
   and `setAndBump` helper automatically. Without this, the P2 tail
   edits live on top of a generated file and get clobbered — the same
   risk P2 already carries (`generator-gaps.md` bullet #6).
2. Regenerate `bridge_state.gen.swift`. Verify counter field is
   present on every `BridgedXxxState`.
3. Add `@_cdecl("SUIStateGen")` accessor (one per shape, or one
   generic — evaluate during implementation).
4. Add Go-side `_SUIStateGen` FFI var + cache consultation in each
   `Get`.
5. Benchstat. Ship.

## Exit Criteria

- `BenchmarkStringStateGetRepeated` ≤ 10 ns/op hot, or the MISS
  framing is updated with a measured reason why the FFI floor prevents
  it,
- all behavioral tests pass, including `-race`,
- no new hand-written entries in `generator-gaps.md` (the template
  change ports the whole pattern),
- flagship examples still pass, no user-visible behavior change.

## Stop Conditions

Stop and record the blocker instead of continuing if:

1. the two-way Binding bump sites cannot be audited exhaustively
   (missing one reintroduces stale reads),
2. the atomic cache shape requires widening beyond what P2 already
   committed,
3. `SUIStateGen`'s measured cost is above ~20 ns/op on the reference
   bar, which would mean the FFI floor itself is the bottleneck and
   `Get` caching has hit diminishing returns.

## Relationship To Other Notes

- closes the P2 tail referenced in `performance-optimization.md`
  Phase P2 exit criteria,
- depends on the generator port tracked in `generator-gaps.md`
  Priority bullet #6,
- unrelated to P3 (callback dispatch) and does not touch the
  callback slot table,
- does not block P4 (packed wire) — P4 can start in parallel if we
  decide to, but doing this tail first leaves P4 with a clean HIT
  table.

## Outcome — Stop Condition #3 Triggered

Landed partially. The design shipped as infrastructure — `AnyBridgedState`
protocol, per-state `gen: UInt32` counter, `setAndBump` / `bumpGen`
helpers, `@_cdecl("SUIStateGen")` accessor, every write path rewired
including 13+ two-way Binding set closures — but the Go-side cache
consultation was **not** landed because stop condition #3 fired during
measurement.

Measured (reference bar, M4 Max, macOS 26.4, Go 1.26.2):

- `_SUIStateGen` FFI cost: **~300 ns/op with 3 allocs** (design budget
  was ≤20 ns/op).
- Go-side cache consultation: **net regression** on int/bool/float hot
  reads (~227 → ~500 ns). Marginal win on strings.

Root cause: the purego FFI floor itself is the bottleneck, not the
absence of caching. A 300 ns/op counter read cannot usefully gate a
~230 ns/op typed get. The whole cache-consultation premise assumed the
counter read would be in the low-single-digit ns range; that assumption
did not survive measurement.

Surprise data point: `_SUIStateGen` (returning `UInt32`) measured
~33% slower than `_SUIStateGetInt` (returning `Int32`) on the same
path. The purego `reflect.MakeFunc` unsigned-return path is slower
than the signed path. This is tracked as a measurement caveat in
`performance-optimization.md`.

What remains useful from the infrastructure land:

- the counter and `setAndBump` audit point stay in the generated Swift
  surface — they cost ~zero and give any future FFI swap (cgo,
  syscall-stubs, improved purego) a ready consumer,
- the counter is an observable write epoch, usable for debug/sync
  assertions even without Go-side caching,
- a future batched-read API (`_SUIStateGenBulk`) could amortize the
  FFI cost across many states in one cross; that was explicitly
  deferred in the original design and stays deferred.

`BenchmarkStringStateGetRepeated ≤ 10 ns/op` remains an **explicit MISS
in the P2 ledger**, root-caused to the purego FFI floor. It is not
listed as "pending work"; closing it requires changing the FFI layer,
not more caching.

This note is kept for the infrastructure reference. Do not treat the
"execution order" milestones as open work — they were either shipped
(A, B, C) or deliberately abandoned based on measurement (D, E).

See `/tmp/collab-p2tail-summary.md` for the full measurement report
(not checked in; ephemeral handoff file).
