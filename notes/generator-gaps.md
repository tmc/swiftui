# Generator Gaps

> **Last spot-checked: 2026-04-16.** `_SUILinearGradient`, `_SUIPasteButton`,
> `_SUIRunScenePlan`, and the callback map section were still hand-written in
> `lib.go` / `callback.go` at that date. The T1 entries are historical —
> T1 landed, but the rows document what the emitter still needs to cover the
> next time hand-authored symbols get consolidated. Re-verify rows before
> citing them; do not assume a row is current just because the table still
> lists it.

---

Hand-written additions to generated files that should eventually be emitted by
applegen. Each entry names the generated file, the symbol or block that was
added by hand, and which hand-written extension file currently owns it.

## Go side

### lib.go — function var declarations + registrations

These vars and their `tryRegisterLibFunc` calls were added to `lib.go` by hand
because applegen does not yet emit them. They live in `lib.go` because Go
function vars must be in scope for the single `init()` that registers them.

| Symbol | Category | Notes |
|---|---|---|
| `_SUILinearGradient` | Views | Gradient view constructors |
| `_SUIRadialGradient` | Views | |
| `_SUIMeshGradient4` | Views | |
| `_SUIPasteButton` | Views | Paste button |
| `_SUIShareLinkURL` | Views | Share link variants |
| `_SUIShareLinkItem` | Views | |
| `_SUIViewRefreshable` | Modifiers | Pull-to-refresh |
| `_SUIViewDraggableText` | Modifiers | Drag-and-drop |
| `_SUIViewDraggableURL` | Modifiers | |
| `_SUIViewDraggableFileURL` | Modifiers | |
| `_SUIViewDropDestinationText` | Modifiers | |
| `_SUIViewDropDestinationURL` | Modifiers | |
| `_SUIViewDropDestinationFileURL` | Modifiers | |
| `_SUIViewSafeAreaInset` | Modifiers | Safe area inset modifier |
| `_SUIViewAccessibilityRotor` | Modifiers | Accessibility rotor JSON |
| `_SUIRunScenePlan` | Runtime | Scene plan entry point |
| `_SUIOpenSceneWindow` | Runtime | Open scene window by ID |
| `_SUIAccessibilityIdentifier` | Modifiers | Accessibility identifier (T1) |
| `_SUISetStringCallback` | Callbacks | String callback trampoline |
| `_SUIRegisterCommandCallback` | Callbacks | Command callback trampoline (T1) |
| `_SUIUpdateMenuItemEnabled` | Callbacks | Menu item enable/disable (T1) |

### callback.go — callback types + trampolines

| Symbol | Category | Notes |
|---|---|---|
| `stringCallbackMap` | Callback map | `func(string) bool` — used by scene action callbacks |
| `registerStringCallback` | Registration | |
| `stringCallbackTrampoline` | Trampoline | Returns `int32` |
| `stringCallbackPtr` | Pointer | `purego.NewCallback` |
| `commandCallbackMap` | Callback map | `func() int32` — used by menu actions, enabled checks (T1) |
| `registerCommandCallback` | Registration | (T1) |
| `commandCallbackTrampoline` | Trampoline | Returns `int32` (T1) |
| `commandCallbackPtr` | Pointer | `purego.NewCallback` (T1) |
| `cStringToGoString` | Helper | C string → Go string conversion |

## Swift side

### bridge_app.gen.swift — scene plan + window management

The following should eventually be generated. Currently hand-written in
`bridge_scene_plan.swift`:

| Symbol | Category | Notes |
|---|---|---|
| `SUIScenePlanPayload` | Wire type | JSON-decoded scene plan |
| `SUIScenePlanScene` | Wire type | Per-scene spec |
| `SUISceneRunnerDelegate` | Delegate | NSApplicationDelegate for scene plan |
| `SUISceneWindowDelegate` | Delegate | NSWindowDelegate per scene |
| `SUIRunScenePlan` | @_cdecl | Entry point for scene plan runner |
| `SUIOpenSceneWindow` | @_cdecl | Open scene window by ID |
| `SUIInstallSceneWindow` | Helper | Window creation and management |
| `SUIRevealSceneWindow` | Helper | Show/hide scene window |
| `SUIConfigureSceneMenuBar` | Helper | Menu bar extra popover setup |
| `SUIInstallQuitMenu` | Helper | Basic Quit menu for SUIRun |
| `SUIInstallAppMenu` | Helper | App menu with optional Settings |

### bridge_app.gen.swift — quit menu hook

`SUIRun` and `SUIRunWithMenuBar` should call `SUIInstallQuitMenu()` after
setting up the delegate, but they are in the generated file. The generator
should emit this call.

### bridge_helpers.gen.swift — callback globals

Currently hand-written in `bridge_commands.swift`:

| Symbol | Category | Notes |
|---|---|---|
| `_SUIStringCallback` | Callback global | String callback function pointer |
| `SUISetStringCallback` | @_cdecl | Registration function |
| `_SUICommandCallback` | Callback global | Command callback function pointer (T1) |
| `SUIRegisterCommandCallback` | @_cdecl | Registration function (T1) |

### bridge_modifiers.gen.swift — missing modifiers

Currently hand-written in `bridge_a2ui_extra.swift`:

| Symbol | Category | Notes |
|---|---|---|
| `SUIAccessibilityIdentifier` | @_cdecl | `.accessibilityIdentifier()` modifier (T1) |

## T1-specific additions (command menus, lifecycle)

Currently hand-written in `bridge_commands.swift` and `bridge_scene_plan.swift`:

| Symbol | File | Notes |
|---|---|---|
| `SUICommandGroup` | bridge_scene_plan.swift | Wire type for command menu groups |
| `SUICommandItem` | bridge_scene_plan.swift | Wire type for menu items |
| `SUILifecycleCallbacks` | bridge_scene_plan.swift | Wire type for lifecycle callback IDs |
| `SUICommandCoordinator` | bridge_commands.swift | NSMenuDelegate handling command dispatch |
| `SUIBuildMenuItems` | bridge_commands.swift | Recursive NSMenuItem tree builder |
| `SUIInstallCommandMenus` | bridge_commands.swift | Full command menu installation |
| `SUIUpdateMenuItemEnabled` | bridge_commands.swift | Imperative menu item enable/disable |

## Priority for generator improvement

1. **Callback types**: `stringCallbackMap`/`commandCallbackMap` follow the exact
   same pattern as `boolCallbackMap`. The generator should handle new callback
   shapes given a signature spec.

2. **View + modifier function vars**: The generator already emits most of these.
   The gaps (gradients, drag-drop, refreshable, safe area, accessibility rotor)
   are likely missing from the applegen input spec.

3. **Scene plan wire types**: If the scene plan JSON schema is formalized, the
   generator could emit `SUIScenePlanPayload` and related types.

4. **Quit menu hook**: The generator should emit `SUIInstallQuitMenu()` calls
   in `SUIRun` and `SUIRunWithMenuBar`.

5. **Pooled withCString in subpackages**: The root package's `withCString`
   forwards to `withCStringPooled` (sync.Pool-backed scratch) as of P1. The
   applegen-generated `withCString` in `arkit`, `avkit`, `charts`, `charts3d`,
   `localauth`, `quicklook`, `scenekit`, `spritekit`, `translation`,
   `workoutkit` still uses `append([]byte(s), 0)` and allocates per call.
   The durable fix is in the applegen template; each subpackage currently
   carries a one-line `TODO(perf P1 subpackage rollout)` flag.

6. **P2 state dirty-skip in the generator**: P2 lands a Go-side value cache
   and no-op-on-equal `Set` path in `state.go`, which is `Code generated by
   applegen; DO NOT EDIT.` The cache fields live in hand-written
   `state_cache.go`, but the generated `state.go` forwarders were edited to
   read-and-compare the cache before crossing the bridge. The next applegen
   regen will clobber those forwarder edits unless the template in
   `appledocs/internal/swiftbridge/swiftui_templates.go` is updated first to
   emit the cache-check shape directly. Port must land BEFORE the next
   regen, not after.
