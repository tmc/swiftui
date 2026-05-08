# SwiftUI Binding Gaps — Execution Plan

This is the active execution plan for closing the actionable subset of
`swiftui-binding-gaps.md`. That note stays the scorecard and classifier; this
one is the ordered work list.

## Charter

Close two concrete buckets without expanding the public surface beyond them:

1. the `promote` bucket (9 items) from `swiftui-binding-gaps.md`, and
2. the three highest-payoff ergonomic gaps exposed by `examples/codex-clone`.

Everything else in the classifier note stays deferred.

## Out Of Scope

These stay on the classifier note, not this plan:

- native SwiftUI `Table` / `OutlineGroup` parity — see
  `table-outline-native-parity-path.md`
- `KeyframeAnimator`, `KeyframeTimeline`, `KeyframeTrack`, and native
  `MeshGradient` constructor families
- full `FocusState` parity beyond token-level focus coordination
- full SwiftUI `App` / `Scene` environment-action parity — see
  `scene-app-parity-path.md`
- `HoverEffect` on macOS (underlying modifier unavailable)
- anything requiring public API churn in the current `Scene`, `View`, or
  state-binding surface

## Completion Bar

This plan is complete when all of the following are true:

1. every item in Tranche B1 ships with a Go wrapper, a Swift `@_cdecl` shim
   where needed, and a test,
2. Tranche B2 items either ship or are explicitly deferred with a named
   follow-on note,
3. Tranche B3 retires at least the sidebar, disclosure, search, menu, and
   scroll-position scaffolding in `examples/codex-clone`, and
4. `swiftui-binding-gaps.md` classifier counts show the `promote` bucket at 0
   or explicitly deferred.

## Execution Order

Work these tranches in order. Do not start B2 before B1 is closed or
explicitly carved.

### Tranche B1 — Promote Bucket

Ordered by flagship-example payoff, not SwiftUI symbol count.

1. `Text.init(timerInterval:pauseTime:countsDown:showsHours:)`
   - Exercises an existing timer example need. Land before the animation
     pair so the timer story reads cleanly.
2. `PhaseAnimator.init(_:content:animation:)` and
   `PhaseAnimator.init(_:trigger:content:animation:)`
   - Land together. One Go constructor per variant, one shared Swift shim.
3. `SliderTickContentForEach.init(_:id:content:)`
   - Curated concrete overlay. No generic `ForEach` expansion.
4. `Section.collapsible(_:)`
   - Thin modifier on existing `Section` surface. Keep `SectionExpandedView`
     untouched.
5. Modifier flight: `Text.textVariant(_:)`,
   `MenuButton.menuButtonStyle(_:)`, `NavigationLink.isDetailLink(_:)`,
   `NavigationView.navigationViewStyle(_:)`
   - Four thin modifier passes. Land in one PR. No per-item design step.

Per-item rules:

- one Go wrapper + one Swift `@_cdecl` shim where the bridge does not already
  expose the needed call,
- one test that drives the wrapper through the existing view model,
- no example churn unless the wrapper exposes new behavior callers must see,
- if an item turns out to need a new runtime-model type, stop and move it to
  Tranche B2 instead of widening scope.

Exit criteria:

- all nine items closed in the generator report `promote` bucket,
- no new unexported-protocol or untyped-string escape hatches in the public
  surface.

### Tranche B2 — Codex Clone Payoff

These are the three remaining ergonomic gaps exposed by
`examples/codex-clone`.

1. Non-markdown transcript primitives
   - Badges and diff rows for the transcript and review panes.
   - Short design pass in `examples/codex-clone/design/` before code:
     decide whether these are one curated `TranscriptRowModel` or two
     smaller widgets.
   - Markdown and code-block rendering is **not** in this tranche — it
     moved to its own execution note,
     `notes/markdown-rendering-path.md`, because the design surface
     (inline formatting, code-block theming, selection, link handling,
     image policy) is larger than one B2 item.

2. Focus coordination beyond `Focusable(bool)`
   - Token-based focus routing across the shell. Keep the bar explicit: this
     is not full `FocusState` parity.
   - Decide whether this is a new `FocusToken` type on top of
     `NewFocusNamespace` / `PrefersDefaultFocus`, or a curated extension of
     the Bool-backed `Focused(...)` path.
   - Design step lands before code. One short note, not a full track.

3. Safe-area-aware shell layout
   - Check first whether the existing `SafeAreaInset` modifier already
     covers the case with a documentation/example change. If it does, this
     becomes a Tranche B3 item (adoption, not new surface).
   - If it genuinely needs a new modifier, land one concrete modifier that
     the codex clone consumes in the same PR.

Rules:

- each item gets a short design note (1–2 pages) before code,
- do not ship a B2 item unless the codex clone actually adopts it,
- if a B2 item grows into a runtime-model track (new state type, new host
  behavior), stop and spin a dedicated execution note instead of widening
  this plan.

Exit criteria:

- each B2 item is either shipped and adopted by the clone, or explicitly
  deferred with a linked follow-on note,
- no B2 item leaves behind a half-shipped modifier the clone does not use.

### Tranche B3 — Curated Surface Adoption

No new bindings. Rewrite the existing `examples/codex-clone` shell to use
surface that already exists but is not yet adopted. Proves the current
surface is actually usable and retires manual scaffolding.

Targets (all in `examples/codex-clone/shell.go`):

- sidebar rows (lines ~376, ~416) → `SelectableList` + `Tag`
- agent-row expansion toggles (lines ~407, ~472) → `DisclosureGroupView`,
  `SectionExpandedView`, or curated `OutlineGroup`
- sidebar filter (line ~261) → `Searchable`
- footer menus (line ~688) → `MenuView` + `PopoverPresented` + `Help` +
  `OnHoverPhase` + `PointerStyle`
- transcript and review scroll anchors (lines ~570, ~631) →
  `ScrollViewReader` + `ID`

Rules:

- one PR per target area, not one mega-rewrite,
- each PR must keep the clone running under `go run ./examples/codex-clone`
  and pass the existing smoke test,
- no new public surface here. If a rewrite turns up a missing modifier,
  stop and route it back to B1 or B2.

Exit criteria:

- the codex clone no longer hand-rolls the patterns above,
- manual selected-state, expansion, search, menu, and scroll bookkeeping is
  gone from `shell.go`.

## Stop Conditions

Stop the plan and record the blocker instead of continuing if any of these
happen:

1. a B1 item cannot be modeled without a new runtime-model type,
2. a B2 design step concludes the right answer is a new execution track (for
   example, richer focus state, or a full transcript DSL),
3. a B3 rewrite would require public API churn to the current curated
   surface,
4. the codex clone stops being representative of the target product shape
   and starts reading like a bindings test harness.

If a stop condition is hit, record the exact blocker and the smallest
possible follow-on note. Do not fall back to vague future-work language.

## Validation

Run after each tranche:

```sh
GOWORK=off go test .
GOWORK=off go build ./examples/codex-clone
(cd internal/swift && swift build -c release --quiet --product SwiftUIBridge)
bash ./examples/build-flagship.sh
```

After each B1 item or B3 PR, rerun the codex-clone under
`go run ./examples/codex-clone` and confirm the touched flow still behaves.

For any B2 change that alters user-visible focus, scroll, or transcript
behavior, rerun the relevant live AX/manual pass for the clone.

## Relationship To The Classifier Note

`swiftui-binding-gaps.md` stays the scorecard:

- bucket counts (`promote`, `covered_by_catalog`, `implementation_detail`,
  `requires_runtime_model`, policy buckets) are updated there, not here,
- curated-vs-native split stays there,
- retire this plan when B1 closes and B2/B3 either ship or are explicitly
  deferred with linked notes; leave the classifier note in place so future
  passes still have a counted baseline.
