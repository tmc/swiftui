# SwiftUI Runtime Roadmap

This note is the current repo-level workboard for
`github.com/tmc/swiftui`.

The goal is not raw SwiftUI symbol parity. The goal is a Go-first macOS app
runtime that is strong enough for normal desktop and content applications
without falling back to Swift or AppKit for ordinary app structure.

Updated: April 16, 2026.

## Current Status

Shipped and credible now:

- runner-owned app shell, commands, menus, and settings
- `WindowGroup`, `Window`, `MenuBarExtra`, and
  `DocumentGroupWithHandle`
- document workflows, recent documents, and restoration identity
- explicit scene runtime state, including per-instance window reporting
- curated and additive native-backed table / outline surfaces
- concrete placement-aware layout models
- concrete share / paste / drop payloads and PhotosPicker metadata
- AppKit-backed text entry with explicit selection ownership

The note set no longer uses anonymous future buckets.

Every remaining item must be one of:

- an active execution track with a next milestone,
- a closed track with explicit reopen criteria, or
- maintenance work that keeps docs, examples, and reports in sync.

## Completion Bar

The current macOS-runtime push is complete when:

1. the active execution tracks below are either shipped or explicitly deferred
   with a named blocker,
2. no flagship example needs Swift/AppKit escape hatches for ordinary app
   structure,
3. docs, examples, and `appledocs` report output describe the same ownership
   boundary, and
4. no remaining gap is described only as "future parity" without a concrete
   completion path.

## Canonical Workboard

### Active Track 1: Scene/App Host Ownership

Status: active.

What remains:

- replace runner-owned startup and orchestration with host-owned behavior
- keep the current public `Scene` APIs intact
- preserve document, menu, and multi-window semantics during the cutover

Execution note:

- `notes/scene-app-parity-path.md`

Done when:

- ordinary scene apps no longer depend on the generated scene-plan runner for
  startup and orchestration
- lifecycle and scene-action semantics come from the host implementation
- `examples/scenes` and `examples/workbench` still prove the same product
  story without API churn

### Active Track 2: Bridge Performance And Input Fidelity

Status: active.

What remains:

- remove obvious bridge-specific latency from text entry, state updates, and
  callback hot paths
- add benchmark gates so regressions are caught before they ship

Execution note:

- `notes/performance-optimization.md`

Done when:

- the active performance phases have benchmark coverage and exit criteria
- flagship typing and scene flows no longer show bridge-specific regressions
  relative to the native controls they wrap

### Active Track 3: Docs, Examples, And Report Honesty

Status: always active maintenance.

What remains:

- keep examples aligned with the shipped runtime
- keep `appledocs` coverage buckets aligned with the shipped runtime
- retire or redirect stale planning notes instead of letting them drift

Done when:

- the note set has one clear owner per active track
- stale planning notes are marked historical or superseded
- examples, notes, and report output use the same boundary language

## Closed Tracks With Reopen Criteria

These are not active workstreams. Reopen them only if the named trigger
appears.

### Text/Editor Depth

Current shipped bar:

- policy-backed text input
- baseline Edit and Window menus
- explicit `TextSelectionState`
- AppKit-backed `TextField`, `SecureField`, and `TextEditor` ownership

Reopen only if:

- a real product needs richer editor callbacks, selection affinity, or other
  semantics that the current controls cannot express

When reopened:

- add one explicit state family at a time
- update `examples/form`
- add tests before widening another editor surface

### Native Data-Surface Parity

Current shipped bar:

- curated `TableModel` / `OutlineModel`
- additive `NativeTableModel` / `NativeOutlineModel`
- flagship examples for both paths

Reopen only if:

- a concrete desktop workflow fails both the curated and the additive
  native-backed path

Execution note:

- `notes/table-outline-native-parity-path.md`

### Layout Follow-On

Current shipped bar:

- tagged layout metadata
- fixed-key placement metadata
- placement-aware layout helpers

Reopen only if:

- a real layout cannot be modeled with the current tagged/placement runtime

Execution note:

- `notes/layout-runtime-parity.md`

### Media/Transfer Generalization

Current shipped bar:

- concrete share / paste / drop payloads
- PhotosPicker metadata plus lazy file-backed sample assets

Reopen only if:

- a real product needs richer content exchange that still fits a concrete
  Go-native API

Do not reopen for:

- raw `Transferable` mirroring
- generic share-sheet parity slogans

## Execution Order

Do work in this order:

1. scene/app host ownership
2. bridge performance and input fidelity
3. docs/examples/report maintenance
4. reopen a closed track only if a flagship example or real product case fails

## Flagship Examples

These are the proof surfaces for the runtime roadmap:

- `examples/workbench`
- `examples/scenes`
- `examples/table-outline`
- `examples/native-table-outline`
- `examples/layout`
- `examples/media-transfer`
- `examples/accessibility`
- `examples/bridge-coverage`
- `examples/form`

Every runtime claim in this note should be visible in at least one of those
examples.

## Validation

Run at the end of runtime work:

```sh
GOWORK=off go test .
GOWORK=off go build ./examples/scenes ./examples/layout \
  ./examples/native-table-outline ./examples/table-outline \
  ./examples/accessibility ./examples/media-transfer \
  ./examples/bridge-coverage ./examples/form
(cd internal/swift && swift build -c release --quiet --product SwiftUIBridge)
bash ./examples/build-flagship.sh
```

When user-visible behavior changes, rerun the relevant live AX/manual
verification for the affected flagship example.
