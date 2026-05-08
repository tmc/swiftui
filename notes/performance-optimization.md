# SwiftUI Bindings Performance Optimization

This note is the performance charter for `github.com/tmc/swiftui`. It is the
counterpart to `swiftui-runtime-roadmap.md` and `swiftui-binding-gaps.md`: those
govern *surface*, this governs *speed*. It exists because, in the Jacob Bartlett
"SwiftUI vs UIKit in iOS 26" experiment (Mar 2026), a complex scrolling feed in
SwiftUI burned **~100% CPU at rest**, dropped **3.4 hitches/sec**, and held
**248MB** resident — while the UIKit version held **11% CPU**, **0.7
hitches/sec**, and **92MB**. That gap exists even in a pure-Swift program. A Go
binding that adds a second language boundary on top of SwiftUI does *not* get to
start the race with more slack.

Our job is to make the Go → bridge → SwiftUI path add as close to zero overhead
as is achievable, so that any user-visible performance ceiling comes from
SwiftUI itself, not from us.

Updated: April 16, 2026.

## Out of Scope

This note deliberately does *not* cover:

- beating SwiftUI's own body-recomputation / diff / layout cost — that is
  Apple's floor and our ceiling
- rewriting purego or replacing our FFI approach — the bridge shape is fixed
- cross-platform (iOS / tvOS / visionOS) performance — macOS is the current
  target, the mobile story reopens if and when we ship there
- native SwiftUI `Table` / `OutlineGroup` perf (tracked separately in the
  runtime roadmap's closed data-surface follow-on note)
- any perf work that requires reopening a closed runtime track just to
  widen coverage

If a proposed optimization requires one of the above, it does not belong in
this note — open a separate design review.

## Guiding Principles

These principles are load-bearing. Code review should push back on anything
that violates them.

1. **The bridge is the budget.** SwiftUI is expensive enough on its own. Every
   microsecond we spend in Go marshaling, in `withCString` copies, in callback
   map lookups, or in defensive slice copies, is a microsecond stolen from the
   8.33ms 120Hz frame budget. We do not get to spend the budget twice.
2. **Zero allocation on the hot path.** State `Set`, modifier chaining, list
   reconfiguration, and callback dispatch must not allocate in steady state.
   Allocations are permitted on construction and on user-initiated mutation; not
   on per-frame, per-row, or per-tick paths.
3. **Coalesce, don't chase.** Every Go-side state write should not mean a
   synchronous Swift-side re-evaluation. Batch, debounce, or pass through
   `withAnimation` / `withTransaction` when the caller's intent is one logical
   update.
4. **The Go type system is the runtime.** We do not round-trip JSON to express
   slices of strings, enum kinds, or structured config to Swift. We pass packed
   binary when we cross the boundary. JSON is for debugging.
5. **Main-thread safety is not a performance free-pass.** Main-thread dispatch
   is ~5µs on a warm queue. Calling it per row in a 1000-row update costs a
   frame. Hoist the dispatch; don't sprinkle it.
6. **Measure before tuning. Measure after tuning.** Every perf claim in this
   note must be backed by a reproducible benchmark in `*_benchmark_test.go` or
   a reproducible trace in `examples/perflab`.

## Current Baseline (Hot Paths Audited)

I walked the tree with a focus on the per-frame and per-mutation paths. The
places where we pay overhead that SwiftUI does not:

### 1. `withCString` copies the whole string every call

`withCString` at `views.go:1221` (plus duplicates in every subpackage's
`views.go`) is the one-and-only outbound string path:

```go
func withCString(s string, fn func(*byte)) {
    b := append([]byte(s), 0)
    fn((*byte)(unsafe.Pointer(&b[0])))
    runtime.KeepAlive(b)
}
```

Every single bridge call that takes a string (`StringState.Set`, every
modifier label, every `Text` construction, every menu item) allocates a fresh
byte slice and copies the full string. For a frame that sets 30 labels, that
is 30 tiny allocations hitting the GC. `withCString` is called thousands of
times in a single scene configuration. We are on purego, not cgo — the copy
is *not* required by the FFI; it is required only by the absence of a pinned
`unsafe.StringData` path.

### 2. `cString` / `cStringToGoString` grow via `append`

Both `render.go:42` (`cString`) and `callback.go:197` (`cStringToGoString`)
walk the C string byte-by-byte with `append(buf, b)`. For a 200-char
identifier the slice grows 5 times (1→2→4→...→256) before the final
`string(buf)` copy — so a single inbound string can allocate twice and copy
three times. The two functions are textual duplicates and should converge on
one helper.

### 3. State Get/Set is one FFI call per scalar

`IntState.Get()` → `_SUIStateGetInt` every call. If application code reads the
same state 4 times in a `body` recomputation, it crosses the bridge 4 times.
SwiftUI on the other side caches its own values; ours doesn't.

### 4. Callback registration holds a global `sync.Mutex`

Every Swift→Go callback does:

```go
callbackMu.Lock()
fn := callbackMap[id]
callbackMu.Unlock()
if fn != nil { fn() }
```

This is fine at 60Hz with 10 buttons. It is not fine at 120Hz with a scrolling
feed issuing hover + appear + disappear + gesture tick callbacks per cell.
Under contention, `sync.Mutex` on the call site plus a second lock in the
`registerX` path becomes a hot single point.

### 5. `TableModel.Rows()` / `SelectedIDs()` defensive-copy on every read

```go
func (m *TableModel[T]) Rows() []T {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return append([]T(nil), m.rows...)
}
```

Sensible for safety, expensive when the bridge walks all rows to render a
diff. For a 1024-row dataset that's a ~32KB copy per read.

### 6. `SetRows` sorts, syncs selection, bumps revision synchronously

The path in `table_model.go:137` re-allocates the slice, re-sorts if sort is
active, re-synchronizes the selection set, and pushes a revision-bump
`IntState` — all under one write lock. A paginated append that adds 20 rows
does the work of a full reset.

### 7. View construction has no pooling

Every `Text("...")`, every `VStack(...)`, every modifier is a fresh bridge
allocation. A list cell with 6 child views + 4 modifiers = 10 allocations on
each render. SwiftUI's own view graph is value-semantic and cheap; ours goes
through a retain-counted `retained` struct + `sync.Once` per node.

### 8. Scene-plan runner bridges on every scene mutation

`bridge_scene_plan.swift` handles the runner-owned model; any time a
document's dirty flag changes, we cross the boundary. Scene-level metadata
updates should batch.

## Performance Targets

These targets are the bar. They are enforced with benchmarks in
`*_benchmark_test.go` and the `examples/perflab` harness.

| Operation | Target | Rationale |
|---|---|---|
| `StringState.Set(shortString)` | ≤ 200ns, 0 allocs (steady state) | Called per keystroke in policy fields |
| `IntState.Set(v)` where `v` unchanged | ≤ 30ns, 0 allocs, no bridge cross | SwiftUI views should not re-eval on no-op |
| `TableModel.SetRows(1024 rows)` | ≤ 150µs, ≤ 1 alloc per unique row ID | Paginated append should be O(delta) |
| `TableModel.Rows()` read | 0 allocs when caller promises read-only | Defensive copy is opt-in |
| Modifier chain of 10 on a leaf view | ≤ 1µs Go-side, ≤ 10 bridge calls | Single frame at 120Hz = 8333µs |
| Callback dispatch (Swift → Go → return) | ≤ 150ns, 0 allocs | Scroll-driven hover at 120Hz |
| Scene mutation (dirty flag toggle) | ≤ 500ns, 0 bridge cross if unchanged | Save-state churn must not wake SwiftUI |
| Bridge dylib load + first frame | ≤ 50ms cold | Launch latency is visible |
| Steady-state CPU on idle window | ≤ 1% on M-series | No work should happen with no input |

These are aggressive. They are also achievable on an A19/M4 once the hot path
is dealloc-free.

### Measurement methodology

Targets are only comparable under a pinned environment. The reference bar is:

- Hardware: M4 Pro, 48 GB, on AC power, charged ≥ 50%, in Performance mode
- OS: macOS 26.x current minor
- Go: 1.26.x current point release
- Swift: release configuration, `swift build -c release --product SwiftUIBridge`
- Build flags: Go release (`-ldflags=-s -w` off — we want symbols for pprof)
- Thermals: baseline with a 60-second idle before each run to let fans settle
- `go test -bench` with `-benchtime=2s -count=5` minimum
- Power: no other foreground AppKit app; Activity Monitor idle <3%

Other machines are allowed, but baseline comparisons (`benchstat`) must use
the same machine / OS / toolchain as the committed `bench-baseline.txt`. Drift
across hardware shows up as noise that erodes the gate.

**What the Go-side benchmarks do *not* measure.** Callback and state
benchmarks call Go trampolines directly (e.g. `buttonCallbackTrampoline(id)`),
not through `purego.NewCallback`. The purego calling-convention bridge that
Swift actually traverses on every Swift→Go dispatch has its own floor
(register save/restore, stack switch, Go scheduler handoff) that we neither
control nor measure here. The ns/op numbers in these benchmarks are therefore
a **lower bound** on what a live SwiftUI app sees, not an end-to-end Swift→Go
cost. An honest end-to-end number requires Instruments against a running
flagship example under load — which is tracked as per-tranche live
verification, not as a CI benchmark.

**purego return-type path matters.** Measured during P2-tail: an FFI
function returning `uint32` (`_SUIStateGen`) lands ~33% slower than an
otherwise-identical function returning `int32` (`_SUIStateGetInt`) on the
same path. The purego `reflect.MakeFunc` unsigned-return code path is
slower than the signed path. When a new `@_cdecl` is being added and the
signedness is semantic-free (a generation counter, a handle token, an
opaque ID), **prefer `int32` / `int64` over `uint32` / `uint64`**.
Unsignedness is worth paying for only when the caller legitimately needs
the full unsigned range. This caveat is per-call and compounds on hot
paths — it was enough to move `SUIStateGen` from a useful optimization to
a net regression once combined with the rest of the FFI floor.

## Execution Phases

These phases are ordered so that (a) early phases remove allocations that
later phases would otherwise amplify, and (b) each phase's exit criteria
are measurable in isolation. Do them in order unless a profile shows a later
phase blocking a concrete product scenario. Skipping is allowed with
justification; reordering silently is not.

Expected per-phase payoff (rough; revise with real numbers as landed):

| Tranche | Hot path touched | Expected steady-state win |
|---|---|---|
| P1 | String boundary | ~40–60% of per-frame Go allocs |
| P2 | State dirty-skip | Eliminates idle FFI traffic |
| P3 | Callback dispatch | Removes mutex from 120Hz path |
| P4 | Packed wire | Removes JSON from per-frame paths |
| P5 | Collection deltas | Paginated append O(n) → O(delta) |
| P6 | View pooling | Per-cell alloc floor drops |
| P7 | Main-thread coalesce | Removes per-row `dispatch_async` |
| P8 | Rendering primitives | Only if P1–P7 still leave us behind |

### Phase P1: String Boundary

Goal: eliminate the `append`-based C-string marshaling on both directions.

1. Introduce a package-private `cstringBuf` pool backed by `sync.Pool` that
   hands out scratch `[]byte` of capacity ≥ `len(s)+1`. Return on `defer`.
   Expose `withCStringPooled(s string, fn func(*byte))` as a drop-in
   replacement. Keep `runtime.KeepAlive` on the pooled buffer for the duration
   of `fn` to prevent GC collection of the pointer handed to Swift.
2. For the inbound path, add `gostringFast(*byte) string` that does one pass
   to measure length (pointer arithmetic as today), then one `make` and one
   `memmove` — no `append` growth. For known-short identifiers (row IDs,
   path tokens) carve a small-string fast path that hands the caller a value
   via `unsafe.String(&b[0], n)` when the buffer is newly allocated and
   unshared.
3. Add a `cstringView(*byte) []byte` that returns a zero-copy slice for
   *immediate use only* (not retained). Use in callback paths that only
   compare or look up a map key.
4. Collapse the two textual duplicates (`render.go:42` `cString`,
   `callback.go:197` `cStringToGoString`) onto the new `gostringFast`.
5. Migrate `state.go`, the root `views.go:1221` `withCString`, and every
   subpackage `views.go:withCString` to the pooled versions.

Exit criteria:

- `BenchmarkStringStateSet/short` shows 0 allocs/op.
- `BenchmarkStringStateGet/short` shows ≤ 1 alloc/op (the final string).
- `go test -run=xxx -benchmem ./...` shows a measurable drop in
  alloc rate across the package.

### Phase P2: State Dirty-Skip and Packed Readback

Goal: make `State.Set(same)` free and `State.Get()` cheaper than an FFI call.

1. Cache the last-known value in the Go-side `State` struct. `Set(v)` compares
   with the cache and no-ops on equality. `Get()` returns the cache unless
   it's been explicitly invalidated.
2. For invalidation: Swift currently already owns the source of truth for
   state. Add a Swift-side monotonic "generation" counter per state object.
   `Get()` does one cheap `_SUIStateGen(ptr)` call; if the cached generation
   matches, return the cached value. Otherwise re-fetch.
3. Batch the `_SUIStateGen` check: for `N` states observed in one `body`, a
   single `_SUIStateGenBulk(ids []uintptr, out []uint32)` call beats `N` FFI
   crossings.
4. For `StringState` specifically, cache both the Go string *and* the UTF-8
   byte buffer so `Set` can memcmp on the byte level before copying.

Exit criteria:

- `BenchmarkIntStateSetSame` shows ≤ 30ns/op and 0 FFI calls.
- `BenchmarkStringStateGetRepeated` shows ≤ 10ns/op after warmup.
- The scenes example's idle CPU drops below 1% on M-series.

### Phase P3: Callback Fast Path

Goal: remove the mutex and map lookup from the hot callback path.

1. Replace `map[uintptr]func()` with a slot table: callbacks are registered
   into a `[]func()` indexed by the returned ID. Registration takes a lock;
   dispatch does not (the slice header is read with `atomic.LoadPointer`).
2. Partition by callback shape so each trampoline does one typed call with no
   map indirection. The existing `boolCallbackMap`, `stringCallbackMap`, etc.
   become parallel slot tables.
3. Unregistration nils the slot and pushes the index onto a free-list. The
   trampoline checks for nil and no-ops.
4. For "single-writer, many-callers" patterns (one scroll delegate, one hover
   handler per view), inline the closure in the bridge handle struct and avoid
   going through the slot table at all.

Exit criteria:

- `BenchmarkCallbackDispatch` shows ≤ 150ns/op, 0 allocs, 0 lock contention
  under `go test -cpu=8`.
- Hover-heavy example's CPU drops visibly at 120Hz scroll.

### Phase P4: Packed Wire Format

Goal: stop passing JSON across the boundary for structured data.

1. Audit every `withCString(marshalStringSlice(...))` call site. String slices
   are passed today as JSON-encoded strings; replace with a simple length-prefix
   format (`[count:uint32][len:uint32 bytes...]*`) on a pooled scratch buffer.
2. For structured payloads (`ShareItem`, `DropPayload`, `PastePayload`), define
   a versioned binary layout and a Swift-side decoder. Keep JSON as an
   opt-in debug format.
3. For enum kinds (`AnimationKind`, `PlacementPreset`, column kinds), they are
   already `int32` on the wire — confirm no accidental `fmt.Sprintf` is
   inserted by generated templates.

Exit criteria:

- `marshalStringSlice` is no longer on any hot path. `grep -n marshalStringSlice`
  shows only debug / non-frame callers.
- A 100-item string slice Set is ≥ 5x faster than the JSON baseline.

### Phase P5: Collection Mutation Deltas

Goal: `TableModel.SetRows` becomes O(delta), not O(n).

1. Add `TableModel.ApplyDelta(Delta[T])` where `Delta` is:

   ```go
   type Delta[T any] struct {
       Insert []Indexed[T]   // index, row
       Remove []int          // indices, descending
       Update []Indexed[T]
       Move   []Move         // from, to
   }
   ```

2. `SetRows(rows)` keeps its current semantics but internally diffs against
   the previous slice using row IDs and calls `ApplyDelta`. For the paginated
   append case (common in Codex/transcript/feed shells), the diff is cheap.
3. Expose `TableModel.Append(rows ...T)` and `TableModel.Insert(index, rows
   ...T)` as direct, non-diffing fast paths.
4. Swift side: on `ApplyDelta`, call SwiftUI's animated diff API with the
   indices directly — no string-ID round trip.
5. Same treatment for `OutlineModel`, `NavigationPathState`,
   `DateSelectionState`.

Exit criteria:

- `BenchmarkTableModelAppend/1024+20` is ≥ 10x faster than `SetRows(1044)`.
- `examples/codex-clone` transcript append no longer reallocates the row
  slice per message.

### Phase P6: View Construction Pooling

Goal: remove allocation from modifier chains and cell construction.

1. Introduce a per-goroutine view-construction arena: a `sync.Pool` of
   `*viewBuilder` with a scratch capacity for the common case of ≤ 16 children
   and ≤ 8 modifiers. `Reset()` drops references without releasing capacity.
2. Leaf builders (`Text`, `Image`, `Spacer`) return stack-sized values that
   promote to the pool only when composed into a container.
3. Modifier chains (`.padding(8).foregroundColor(...)`) materialize as a
   single bridge call that takes a packed modifier list, not one call per
   modifier. This is already the shape in `bridge_modifiers.gen.swift`; the
   Go side is the one currently fan-calling.
4. Audit `retained` + `sync.Once` per node. `sync.Once` is ~16 bytes plus a
   CAS; for views rebuilt every frame that is overhead we pay for no benefit.
   Do *not* remove `sync.Once` outright — it protects against
   double-release on borrowed handles. Instead, split the `retained` type:
   - `retainedOwned` for handles that cross API boundaries and may be
     released concurrently (keeps `sync.Once`)
   - `retainedTransient` for per-frame view nodes that are single-goroutine
     and deterministically released at end-of-frame (plain `released bool`,
     no `sync.Once`)
   This is a type-system distinction, not a runtime check.

Exit criteria:

- `BenchmarkLeafText` shows ≤ 1 alloc/op (the View handle itself).
- `BenchmarkModifierChain/10` shows ≤ 2 allocs/op, ≤ 1 bridge call.
- Generated code for modifier chains emits a single packed call where safe.

### Phase P7: Main-Thread Dispatch Coalescing

Goal: stop burning ~5µs of main-queue hop per small mutation.

1. Introduce a `bridgeCommandQueue` that accumulates mutations on non-main
   threads and flushes them on the next main-thread turn (or synchronously if
   already on main).
2. Scene / document / table / state mutations all route through the queue so
   that `N` mutations in one event tick = 1 main-thread hop.
3. Track queue depth with a debug counter; expose via `SwiftUIDebug.Stats()`.
4. Do not batch across frame boundaries (bounded by CVDisplayLink / CFRunLoop).

Exit criteria:

- Bulk `TableModel` updates no longer show per-row `dispatch_async` in
  Instruments Points-of-Interest.
- `examples/bridge-coverage` share/drop scenario shows one main-thread hop per
  user gesture.

### Phase P8: Rendering Primitives (later)

Only after P1–P7 land. The SwiftUI performance ceiling at that point is
SwiftUI's own body-recomputation + diff + layout cost, which we do not own.
Possible but speculative:

- `LazyVStack` / `LazyHStack` / `List` tagged with explicit `id:` so SwiftUI's
  diff is O(1) per row, not identity-based.
- `drawingGroup()` and `.compositingGroup()` hints on known-expensive subtrees,
  exposed as curated modifier.
- Image decode off the main thread via a Go-owned decode pool feeding an
  NSCache-like Swift-side cache (parallel to Jacob's test setup).
- `PhaseAnimator` (already in the `promote` bucket) to let the runtime express
  animated state transitions without Go-side per-tick callbacks.

These are not on the critical path. Consider them only if profiling after
P1–P7 still shows the Go binding as the bottleneck.

## Benchmarks and Regression Gates

The only way to keep performance is to measure it every CI run.

### Required benchmarks

`table_outline_benchmark_test.go` already exists with `BenchmarkTableModelSetRows`,
`BenchmarkOutlineModelSetRoots`, and the native variants. Extend those rather
than replacing, and add `performance_benchmark_test.go` in the package root
for the rest:

- `BenchmarkStringStateSet/short`, `BenchmarkStringStateSet/long`,
  `BenchmarkStringStateSet/same` (no-op path)
- `BenchmarkStringStateGet/short`, `BenchmarkStringStateGetRepeated`
- `BenchmarkIntStateSet`, `BenchmarkIntStateSetSame`
- `BenchmarkTableModelSetRows/{64,1024,16384}` — *extend* existing bench with
  size subtests rather than duplicating
- `BenchmarkTableModelAppend/1024+20` (new, for P5)
- `BenchmarkCallbackDispatch`, `BenchmarkCallbackRegister`
- `BenchmarkModifierChain/{1,5,10}`
- `BenchmarkSceneUpdateDirtyFlag`
- `BenchmarkStringMarshalSlice/{10,100,1000}`

Every bench reports allocations (`b.ReportAllocs()`). Allocations are
tracked, not just ns/op. Never run the reference suite under `-race` — the
race detector skews both wall time and alloc accounting. Use a separate
`go test -race` pass for correctness only.

### examples/perflab

Add `examples/perflab` modeled after the Jacob Bartlett scroll test:

- paginated list of ≥ 500 rows
- each cell: heading text, subheading text, 2 badges, 1 auto-playing
  gradient/shape animation, 1 gesture target
- variable cell height (forces SwiftUI layout recompute)
- runs as a normal `RunScenes` scene

Record a trace harness script that opens Instruments with the Animation
Hitches + Points of Interest + Allocations instruments, scrolls
programmatically for 20s, and writes a JSON summary. Store the
summary artifact per CI run; fail if hitch rate regresses > 20% vs. baseline.

### Regression gate

Add a CI-friendly subcommand:

```sh
GOWORK=off go test -run=xxx -bench=Bench -benchmem -count=5 ./... \
  | tee bench-current.txt
benchstat bench-baseline.txt bench-current.txt
```

Any `ns/op` regression > 10% or `allocs/op` regression > 0 (for benchmarks
declared at 0) fails the gate. `bench-baseline.txt` is checked in and updated
deliberately, not automatically.

## Anti-Patterns to Reject in Review

These patterns should never land. If a PR introduces any, flag them.

- `withCString(fmt.Sprintf(...))` — two allocations where one would do.
  Precompute, or format into a reusable buffer.
- `json.Marshal` on a per-frame or per-row path. JSON is for debug and
  persistence, not for the bridge boundary.
- `defer mu.Unlock()` inside a trampoline called from Swift. Inline the
  unlock; the defer cost shows up at 120Hz.
- `append([]T(nil), s...)` as a "safety copy" on every read. If the caller
  is read-only, hand them a read-only view; reserve the copy for true mutation
  handoff.
- New public API that takes or returns a string where an integer token would
  do. Row IDs, path tokens, and enum kinds should be `int32`/`uint32` on the
  wire; strings are for user-visible labels only.
- A Swift-side `DispatchQueue.main.async` inside a loop. Hoist above the
  loop; accept a closure instead.
- A new `sync.Once` on a short-lived object. `sync.Once` is 16 bytes and a
  CAS; for per-frame objects it's overhead. Prefer an explicit lifecycle.

## Relationship to the Runtime Roadmap

This note is orthogonal to `swiftui-runtime-roadmap.md`:

- The runtime roadmap says *what surface to ship.*
- This note says *how fast that surface must be.*

When a runtime track is considered complete, it is not actually complete
until the matching perf target in this note is hit or explicitly deferred.
Scene/app parity, native table/outline parity, and richer text/editor parity
should each include a perf checklist entry before close.

## Close Conditions

This note is retired when:

1. Every P1–P7 phase is either landed or explicitly deferred with a named
   reason.
2. `examples/perflab` has a steady-state trace showing the Go binding at < 5%
   of total frame cost vs. SwiftUI body recomputation.
3. The CI regression gate has been enforcing for at least one release cycle
   without a bypass.

Until then, this is the document reviewers cite when pushing back on perf
regressions.

## Validation

The canonical check at the end of a perf phase:

```sh
GOWORK=off go test -run=xxx -bench=Bench -benchmem -count=5 ./... \
  > bench-current.txt
benchstat bench-baseline.txt bench-current.txt
(cd internal/swift && swift build -c release --quiet --product SwiftUIBridge)
bash ./examples/build-flagship.sh
# Optional but strongly recommended for P5+P7:
#   open examples/perflab in Instruments, scroll, capture trace.
```

Update `bench-baseline.txt` in the same PR that lands the improvement. Do not
update it as a separate housekeeping commit — the diff is the record of
earned improvement.
