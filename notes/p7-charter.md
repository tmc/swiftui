# P7 Main-Thread Dispatch Coalescing — Charter

Tranche charter for the seventh perf-arc phase. Opens after the
bridge-generator Swift-side convergence (B1) lands, the dropped-stash
recovery (b9ebd5d) lands, and the tmc editorial sign-off on pending notes
clears. Closes on tag `perf/2026-q2`.

This charter supersedes the one-paragraph P7 stub in
`notes/performance-optimization.md` §Phase P7. It names the narrower scope
that emerged from the pre-P7 dispatch-site scan, plus the convergence-era
boundary conditions that the P7 agent team must respect.

## 1. Goal

Stop burning ~5µs of main-queue hop per animated-state mutation by folding
N mutations in one event tick into one `dispatch_async`. Preserve the
`withAnimation(kind)` grouping semantic — coalescing cannot lose animation
identity.

Non-animated state setters (`SUIStateSetInt`, `SUIStateSetBool`,
`SUIStateSetFloat`, etc.) already write directly on the calling thread
without any dispatch. They are **not** P7 targets; the win is entirely in
the animated path.

### Scope reframe vs. the original stub

The P7 stub in `performance-optimization.md` §Phase P7 implied a
cross-cutting amortizer ("scene / document / table / state mutations all
route through the queue"). The pre-audit scan (§3) shows the actual
hot-path dispatch population is **3 Swift-side sites**, all animated-state
setters — scene/document/table paths either write synchronously today or
already hop for reasons unrelated to animation. P7 is therefore a **narrow
animation coalescer**, not a general-purpose dispatch amortizer. The
realistic win is modest; the stop condition in §7 (#1) explicitly
anticipates "N < 5 coalescable writes per frame" as a reframe-or-defer
trigger. Ship P7 only if the measured N justifies it.

## 2. Scope

**Root-package dispatch sites only.** The five files the P7 agent team
may touch on the Go side:

- `state.go`
- `callback.go`
- `view.go`
- `views.go`
- `lib.go`

On the Swift side, the emission sites that P7 may affect are in the
generated surface — `bridge_state.gen.swift` (the 3 animated setters) and
possibly `bridge_helpers.gen.swift` (queue primitives). Hand-written Swift
files (`bridge_a2ui_extra.swift`, `bridge_scene_plan.swift`,
`bridge_commands.swift`) are **not** P7 targets; their dispatch sites are
low-frequency (menu updates, scene launch, sound play) and not on the hot
path.

### Subpackage exclusion

Of the 9 swiftui subpackages:

- arkit, avkit, localauth, quicklook, scenekit, spritekit, translation,
  workoutkit: zero state dispatch, nothing to coordinate. Safe.
- **charts**: has its own package-local `type retained struct`
  (`charts/bridge_extra.go:9`) and its own state types
  (`charts/state.go`: NumberState, DateState, OptionalNumberState). **Not
  in P7 scope.** If charts coalescing ever becomes necessary, it lands as
  a dedicated post-P7.charts sub-tranche using the same design pattern.

This constraint is load-bearing: the charts package-local retained shape
differs from the root-package P6c split, and threading P7 through charts
would hit a type mismatch that has no clean fix within this tranche.

### Frozen-regen constraint

P7 lands on **hand-written surface only**. No `go generate ./...` during
P7. The Swift-side convergence (B1) has narrowed emission; the Go-side
convergence (C1, per `notes/bridge-generator-go-drift.md`) opens post-P7.
Until C1, every regen blows away the P6c split and more.

If P7 requires template changes (e.g., adding a Swift-side bridge queue
primitive that lives in the template, not in a hand-written file), the
change lands on appledocs first, but **is not regenerated onto swiftui
during P7**. The regen happens as part of C1 prep, post-P7, post-tag.

## 3. Dispatch-Site Pre-Audit

Pre-scan of `DispatchQueue.main.async` sites in the Swift-side bridge:

| File | Count | Shape | P7 target? |
|---|---|---|---|
| `bridge_state.gen.swift` | 3 | Animated state setters: `SUIStateSetIntAnimatedWith`, `SUIStateSetFloatAnimatedWith`, `SUIStateSetBoolAnimatedWith`. Each wraps `s.setAndBump(v)` in `withAnimation(animationForKind(kind)) { ... }`. | **YES** — primary target. Coalescable within the same animation kind. |
| `bridge_app.gen.swift` | 4 | (1) Scene launch `openPopoverOnLaunch`. (2) Menu bar label `_statusItem?.button?.title`. (3) Styled menu bar label. (4) `NSSound(named:).play()`. | **NO** — low frequency, one-shot, UX-critical. Explicitly excluded. |
| `bridge_helpers.gen.swift` | 1 | `unregisterShape(ref)` registry cleanup. | **NO** — lifecycle-bound, already one-shot per view teardown. |
| `bridge_scene_plan.swift` | 1 | Scene plan runner startup. | **NO** — hand-written file, out of scope per §2. |
| `bridge_a2ui_extra.swift` | 1 | Inspect during P7 kickoff; likely not on hot path. | **LIKELY NO** — hand-written file, out of scope per §2. |

**Primary P7 target population: 3 Swift-side dispatch sites** (all in
`bridge_state.gen.swift`, all in the animated-state setter family).

### Preserved invariants

1. **`withAnimation(kind)` grouping**: coalesced mutations with different
   animation kinds cannot be folded into one `withAnimation` scope. The
   queue must partition by kind.
2. **Frame boundary**: do not batch across `CVDisplayLink` /
   `CFRunLoop` frame boundaries. The flush is bounded by the next
   main-thread turn.
3. **Synchronous fast path**: if the caller is already on the main
   thread, write directly. No queue detour.
4. **State.Set from any goroutine safety contract** (documented in
   `swiftui/doc.go:14`): coalescing cannot violate the "safe to call
   from any goroutine" guarantee. Every `Set` call returns immediately
   regardless of coalescing state.

## 4. Design Sketch

Per the `performance-optimization.md` §P7 stub, elaborated with the
pre-audit findings:

### bridgeCommandQueue

A main-thread-owned queue of pending state mutations. Not a general-purpose
mutation queue — specifically scoped to animated-state writes, which is
the only P7 target population.

```go
// Sketch only; final shape determined by P7 agent team.
type bridgeCommandQueue struct {
    // Partitioned by animation kind so withAnimation(kind) grouping is
    // preserved across coalesced mutations.
    pending map[animationKind][]pendingStateWrite
    // Armed iff a main-thread flush is scheduled.
    armed atomic.Bool
}
```

### Flush trigger

On the first `Set(animated)` call from a non-main thread in a frame, the
queue arms itself with a single `DispatchQueue.main.async` that calls the
flush. Subsequent `Set(animated)` calls in the same frame append to the
queue partition without triggering another main-queue hop.

The flush iterates partitions, wrapping each in `withAnimation(kind) { ...
all writes for this kind ... }`, then clears the queue and disarms.

### Synchronous fast path

If `Set(animated)` is called from the main thread (common case for
event-handler-driven mutations), skip the queue and write directly. One
branch, measurable via `SwiftUIDebug.Stats()`.

### Debug counter

Extend `SwiftUIDebug.Stats()` with:

- `bridgeCoalescedWrites`: total coalesced writes since process start
- `bridgeMainHops`: total main-thread hops since process start
- `bridgeCoalesceRatio`: derived, want > 1.0 for the win to be real

## 5. Exit Criteria

From the `performance-optimization.md` stub, plus one addition:

1. Bulk `TableModel` updates or rapid slider dragging no longer show per-row
   `dispatch_async` in Instruments Points-of-Interest.
2. `examples/animation` slider-drag + auto-cycle scenario shows **one
   main-thread hop per (gesture × distinct animation kind)** in
   Instruments traces — the charter narrative is "one hop per gesture",
   the literal measurement accepts one hop per animation kind because
   `withAnimation(kind)` cannot fold across kinds. A gesture that uses a
   single animation kind hits the one-hop bar; a gesture that mixes two
   kinds hits two hops (still a win over N hops, but not one). This is
   by design of the partitioning invariant in §3, not a P7 defect.

   *Flagship correction (2026-04-16)*: the earlier revision of this
   charter cited `examples/bridge-coverage`, but that example does not
   exercise animated-state setters at all (verified with
   `grep -n Animated examples/bridge-coverage/main.go` → zero hits).
   `examples/animation` is the correct choice because its slider drag
   and scene-cycle buttons emit multiple `SetAnimated` /
   `SetAnimatedWith` calls per frame across mixed animation curves —
   representative of the P7 target workload. Any Instruments
   verification should use `examples/animation`, not `bridge-coverage`.
3. **`bridgeCoalesceRatio` > 1.5** on the benchmark that stress-tests
   animated state mutations (new bench introduced as part of P7).
4. Benchstat against the P6 baseline shows no regression on non-animated
   state paths (`StateSet{Int,Bool,Float,Date}`, `StateSetSame*`,
   `CallbackDispatch*`). P7 touches the animated path, must not disturb
   non-animated hot paths.
5. `-race` clean on a stress test that interleaves `Set(animated)` calls
   from multiple goroutines.

## 6. Agent Team Shape

Assuming default parallel-agent discipline per the memory feedback
(parallelize tranches with agent teams). Three agents with disjoint file
ownership, matching the P6 a/b/c pattern:

- **P7.α (queue primitive)**: owns the Swift-side `bridgeCommandQueue`
  implementation in a new hand-written file (e.g.,
  `bridge_command_queue.swift`). No template changes during P7 per §2
  frozen-regen constraint. File ownership: new hand-written Swift.
- **P7.β (state-setter integration)**: owns the changes that route
  animated-state setters through the queue. Swift-side: modifies
  `bridge_state.gen.swift` as a hand-edit during P7 (acceptable per
  freeze policy; absorbed into C1 template later). Go-side:
  `state.go` if any queue-arming hooks need to land there. File
  ownership: `bridge_state.gen.swift`, `state.go`.
- **P7.γ (benchmarks + debug stats)**: owns the new benchmarks
  measuring coalesced throughput, plus the `SwiftUIDebug.Stats()`
  extensions. File ownership: `performance_benchmark_test.go`,
  wherever `SwiftUIDebug` is defined. No Swift changes.

Cross-agent coordination: P7.α must complete the queue primitive
before P7.β can integrate. P7.γ can start in parallel with P7.α
(benchmark design only needs the API sketch, not the implementation).

### Agent-ordering protocol

1. Orchestrator reviews this charter.
2. P7.α spawned first; writes queue primitive, verifies Swift build
   clean, commits.
3. P7.β spawned after P7.α lands; integrates animated setters.
4. P7.γ spawned in parallel with P7.β; writes benchmarks + debug
   stats. Runs benchstat post-integration.
5. Orchestrator consolidates, reviews the three commits, ff-merges
   onto a2ui.
6. Post-P7 close: tag `perf/2026-q2` with the rollup body.

## 7. Stop Conditions

Stop P7 and revise if:

1. Pre-audit-hit count for the animated setters under realistic load
   is lower than expected (e.g., < 5 coalescable writes per frame in
   flagship example traces). Coalescing N=1 is not a win. If the real
   N is 1 in practice, P7 is a nothing-burger; reframe or defer.
2. The `withAnimation(kind)` grouping invariant cannot be preserved
   without widening scope beyond the 3 animated setters.
3. A race surfaces between the queue-arming atomic and
   main-thread flush that cannot be closed with a simple lock-free
   pattern. At that point either re-architect with a mutex (accept
   the cost) or stop and document the blocker.
4. Swift-side bridge_state.gen.swift hand-edits become so invasive
   that C1 absorption looks unmanageable. If P7 modifies more than
   ~50 lines in that generated file, spin the bridge_state template
   port into its own sub-tranche before continuing.

## 8. Non-Goals

- No generalization of `bridgeCommandQueue` to non-animated writes.
  Non-animated writes don't dispatch and don't need coalescing.
- No cross-package dispatch unification. Subpackages out of scope per §2.
- No async / cancellation semantics. The queue is fire-and-forget.
  Mutations never fail and never return status.
- No opt-in / opt-out API. Callers don't know the queue exists; it's a
  bridge-internal optimization.
- No Swift `async/await` or Swift Concurrency integration. The queue
  lives below that layer; `DispatchQueue.main.async` is the primitive.
- No observability beyond `SwiftUIDebug.Stats()` counters. If a use
  case needs trace-level visibility, spin a follow-up tranche.

## 9. Relationship To Other Tranches

- **B1 (Swift-side convergence)**: landed at `cb0b384276`. Unblocks P7 by
  stabilizing the Swift-side emission surface. P7 can reference the
  narrowed template without regen concerns.
- **B2/B3+ (Swift-side pruning)**: proceed in parallel with P7 only if
  they touch hand-written Swift files **not** in P7's scope.
  bridge_scene_plan.swift and bridge_a2ui_extra.swift fit; touching
  bridge_state.gen.swift would conflict with P7.β.
- **b9ebd5d re-stage**: lands on swiftui a2ui before P7 opens. Adds 191
  lines to bridge_a2ui_extra.swift (`SUIRegexMatcher` etc.). Does not
  touch P7's files. Safe.
- **C1 Go-side convergence** (post-P7): absorbs any P7 hand-edits to
  `bridge_state.gen.swift` and any new `lib.go` additions during P7.
  Explicitly scoped to happen *after* P7 so coalescing changes don't
  churn during the convergence migration.
- **P8 rendering primitives**: only after P7 lands and the SwiftUI
  ceiling is visible.

## 10. Close Conditions

P7 retires when:

1. All five exit criteria in §5 are met.
2. All three agent commits landed on a2ui with clean rebase onto
   post-B1 tip.
3. `examples/animation` flagship example traces show one main-thread
   hop per (gesture × distinct animation kind) under Instruments
   verification. See §5 #2 for the flagship-correction context.
4. `notes/performance-optimization.md` §Phase P7 stub is replaced with a
   reference to this charter and the measured outcomes.
5. Rollup narrative at `/tmp/collab-rollup-p1-p6.md` is extended with a
   P7 row (coordination with 9108 for tag prep).
6. Tag `perf/2026-q2` lands on the P7-close commit; body = updated
   rollup narrative; tree = a2ui @ post-P7-close (includes
   `testdata/perf/bench-post-p6.txt` from the existing rollup plus any
   P7-specific bench artifacts).

This note is not deleted after P7 closes; it stays as the historical
record of what the tranche scoped against and what it measured.
Post-retirement, the tranche's fact rows move into the rollup narrative
and the charter row in `performance-optimization.md` gets updated to
"LAND".
