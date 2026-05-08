# World-Class macOS Apps Design

This note defines the product bar for `github.com/tmc/swiftui`.

The goal is not raw SwiftUI symbol parity. The goal is that a Go developer can
ship a serious macOS app without treating AppKit as the real product API for
ordinary app structure.

This note is not an open-ended planning memo. It is the product contract for
the active roadmap.

## What "World-Class" Means

`swiftui` is world-class for macOS when all of the following are true:

1. App shell
   Windows, settings, commands, and common menus can be authored from public Go
   APIs.
2. Document and file workflows
   Open, save, export, import, revert, and close are available without custom
   AppKit glue.
3. Desktop interaction fidelity
   Focus, accessibility, selection, and command routing behave like a real Mac
   app.
4. Data-heavy credibility
   Tables and outlines are strong enough for editors, dashboards, and
   inspector-style tools.
5. API discipline
   The public API stays concrete and explicit about bridge, curated, and
   runtime-owned semantics.
6. Honest docs
   Notes, examples, and `appledocs` report output describe the same boundary.

## Shipped Baseline

The current repo already meets the closeout bar for:

- runner-owned app shell with commands, menus, settings, and menu-bar extras
- document scenes and file workflows
- scene runtime state and per-instance `WindowGroup` reporting
- baseline Edit and Window menu ownership
- policy-backed text input and explicit text selection state
- curated and native-backed data surfaces
- placement-aware layout models
- concrete media and transfer payloads
- focus and accessibility proof surfaces

That is enough to call the current macOS-runtime push credible.

## Active Completion Programs

There are only two active product programs left.

### 1. Host-Owned Scene/App Runtime

Why it matters:

- the biggest remaining gap is still startup/orchestration ownership, not menu,
  document, layout, or media support

Execution note:

- `scene-app-parity-path.md`

Product bar:

- the host, not the generated scene-plan runner, owns ordinary app startup,
  window orchestration, and lifecycle semantics

Stop if:

- the only path forward requires a second public `Scene` API or fake
  environment shims

### 2. Performance Discipline

Why it matters:

- a world-class app runtime cannot feel slower than the native controls it
  wraps

Execution note:

- `performance-optimization.md`

Product bar:

- bridge-specific hot paths have benchmarks and no obvious text-entry or
  scene-update regressions

Stop if:

- a proposed optimization widens product surface instead of removing bridge
  overhead

## Closed By Default

These are not active programs. Reopen only on concrete failure.

### Accessibility And Focus Depth

Current bar:

- shipped focus routing primitives, accessibility metadata, proof surfaces, and
  explicit text selection ownership

Reopen only if:

- a real app flow cannot be expressed with the current focus and AX model

### Native Data-Surface Parity

Current bar:

- the curated plus native-backed table/outline story is already strong enough
  to ship

Reopen only if:

- a specific desktop workflow fails both existing example paths

### Media And Transfer Generalization

Current bar:

- concrete share, paste, drop, and PhotosPicker surfaces are shipped

Reopen only if:

- a real product needs richer content exchange that can still stay concrete and
  Go-native

### Layout Follow-On

Current bar:

- tagged and placement-aware layout models are shipped and example-backed

Reopen only if:

- a named layout in a real example or product cannot be expressed by the
  current model set

## Architectural Rules

These rules stay load-bearing:

1. Prefer product semantics over symbol count.
2. Prefer concrete Go models over Swift protocol mirrors.
3. Keep bridge, curated, and runtime-owned lanes legible.
4. Do not claim native SwiftUI ownership before the runtime actually owns it.
5. Every major runtime claim must have one proof surface, tests, and honest
   docs.

## Review Bar

The repo can claim "world-class" only if a reviewer can build and believe:

- a normal multi-window app with commands and settings
- a document or content app with real file workflows
- a keyboard- and accessibility-credible desktop UI
- a data-heavy desktop tool that does not read like a fallback widget demo

The reviewer should also be able to name the remaining gap precisely. "We will
do more later" is not an acceptable answer.
