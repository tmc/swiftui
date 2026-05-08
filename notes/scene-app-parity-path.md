# Scene/App Parity Path

This is the only active execution note for deeper app and scene host work.

Goal:

- keep the current public `Scene` surface,
- replace runner-owned startup and orchestration with host-owned behavior, and
- keep docs honest until the host really owns those semantics.

Non-goals:

- no fake `@Environment` parity bolted onto the existing runner
- no second public scene API beside the current one
- no parity claims justified only by codegen or report coverage

## Current Baseline

Shipped already:

- `RunScenes`, `WindowGroup`, `Window`, `MenuBarExtra`, `Settings`, and
  `DocumentGroupWithHandle`
- runner-owned document workflows, scene runtime state, and action capability
  reporting
- per-instance `WindowGroup` runtime state
- baseline File/Edit/Window app-shell behavior

Still not owned by the host:

- startup and top-level scene orchestration
- lifecycle/environment semantics beyond the current runner state model
- unrestricted scene graph behavior

## Completion Bar

This track is complete only when all of the following are true:

1. the current public `Scene` APIs stay intact,
2. ordinary scene apps no longer depend on the generated scene-plan runner for
   startup and window orchestration,
3. lifecycle and scene action semantics come from the host implementation
   rather than runner-injected capability state,
4. `examples/scenes` and `examples/workbench` still prove the same product
   stories, and
5. `appledocs` reports the result as shipped runtime ownership, not
   generator-ready parity.

## Execution Order

Work this track in order. Do not skip ahead.

### Phase 1: Freeze The Current Contract

Goal:

- make the current runner contract explicit before replacing it

`swiftui` work:

- write down the invariants currently relied on by `RunScenes`:
  - scene identity
  - window/document/menu-bar counts
  - lifecycle callbacks
  - document action routing
  - restoration IDs and visibility policy
- add tests for those invariants where they are still implicit

Likely files:

- `scene_model.go`
- `scene_model_test.go`
- `lib.go`
- `internal/swift/Sources/bridge_scene_plan.swift`

`appledocs` work:

- keep the report language explicit that the current state is runner-owned
- do not move scene families into a false parity bucket during this phase

Exit criteria:

- the repo has a written contract for the current runner behavior
- tests fail if the current runtime semantics accidentally drift

### Phase 2: Introduce A Host Boundary

Goal:

- separate "scene description" from "scene host implementation"

`swiftui` work:

- introduce one internal host boundary that owns:
  - startup
  - window creation
  - document routing
  - scene action wiring
  - lifecycle publication
- keep `_SUIRunScenePlan` behind that boundary during the transition

Likely files:

- `lib.go`
- `scene_model.go`
- `scene_model_test.go`
- new internal host adapter files in `swiftui`

Rules:

- no public API churn
- no second runner path visible to callers
- the current runner becomes one implementation detail behind the host boundary

Exit criteria:

- `RunScenes` talks to an internal host abstraction instead of directly to the
  scene-plan runner
- existing examples and tests still pass unchanged

### Phase 3: Move Lifecycle And Action Semantics Behind The Host

Goal:

- stop treating scene actions and lifecycle as runner-specific glue

`swiftui` work:

- move action publication and lifecycle publication behind the host boundary
- keep borrowed `SceneActions` and existing runtime state types
- remove assumptions that capability strings are the public source of truth

Likely files:

- `scene_model.go`
- `scene_model_test.go`
- `callback.go`
- `internal/swift/Sources/bridge_scene_plan.swift`
- `internal/swift/Sources/bridge_app.gen.swift`

`appledocs` work:

- keep docs phrased as host-owned runtime semantics until the host cutover is
  done
- ensure report summaries do not regress into "scene parity complete" language

Exit criteria:

- lifecycle and action tests target the host boundary, not the scene-plan
  implementation
- the public docs can describe action and lifecycle semantics without referring
  to runner capability injection

### Phase 4: Host-Owned Startup And Orchestration

Goal:

- replace the generated scene-plan runner as the primary startup/orchestration
  path

`swiftui` work:

- move startup, window/document orchestration, and menu-bar setup into the host
  implementation
- preserve current behavior for:
  - `WindowGroup`
  - `Window`
  - `DocumentGroupWithHandle`
  - `Settings`
  - `MenuBarExtra`

Likely files:

- `lib.go`
- `internal/swift/Sources/bridge_app.gen.swift`
- `internal/swift/Sources/bridge_scene_plan.swift`
- scene runtime tests and examples

Rules:

- land one cutover path, not a permanent dual-runtime arrangement
- if a scene family cannot be carried across honestly, stop and document the
  blocker before widening the public surface

Exit criteria:

- ordinary app startup and window orchestration run through the host path
- `examples/scenes` still behaves like a normal document app shell
- no example copy claims native parity before the host really owns it

### Phase 5: Cleanup And Report Closure

Goal:

- remove temporary transition language and close the note cleanly

`swiftui` work:

- delete obsolete runner-only assumptions and dead transition helpers
- tighten `swiftui-runtime-roadmap.md` and
  `world-class-macos-apps-design.md` to describe the shipped host model

`appledocs` work:

- reclassify scene families from runner-owned to host-owned runtime coverage
- keep native SwiftUI symbol parity separate from shipped host ownership

Exit criteria:

- this note can be retired
- the repo-level roadmap treats scene/app host ownership as shipped, not as
  "later work"

## Stop Conditions

Stop the work and update the note instead of continuing if any of these happen:

1. the only path forward requires public API churn in the current `Scene`
   surface,
2. host ownership cannot preserve `DocumentGroupWithHandle` semantics without a
   new public abstraction, or
3. the examples stop reading like normal app shells and start reading like a
   runtime experiment again.

If a stop condition is hit, record the exact blocker and the smallest possible
follow-on note. Do not fall back to vague future-work language.

## Validation

Run after each phase:

```sh
GOWORK=off go test .
GOWORK=off go build ./examples/scenes ./examples/workbench
(cd internal/swift && swift build -c release --quiet --product SwiftUIBridge)
bash ./examples/build-flagship.sh
```

When user-visible scene behavior changes, rerun the relevant live AX/manual
verification for `examples/scenes`.
