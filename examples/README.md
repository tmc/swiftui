# Examples

This directory has many demos, but a small set should be treated as the
current product-quality surface for the bridge.

## Flagship Examples

- `workbench`
  The catalog-style shell for validating the current SwiftUI surface in one
  app. Use this first when checking navigation, split views, curated table and
  outline state, timer state, and current runtime gaps.

- `bridge-coverage`
  Smoke surface for generated and runtime-backed bridge coverage. It exercises
  `Refreshable`, text and payload paste, share, concrete drag/drop payloads,
  curated grids, gradients, and safe-area chrome.

- `scenes`
  Flagship app-shell example for the current runner-owned scene surface. It
  combines `WindowGroup`, `DocumentGroupWithHandle`, `Settings`, app command
  menus, auxiliary windows, and a menu bar extra in one multi-window app
  shell. Use this first to validate runner-owned menus/commands, settings,
  focus-sensitive enablement, document/window actions, approved close
  lifecycle callbacks, recent-document and revert flows, per-instance
  `WindowGroup` runtime state, and scene restoration, not full SwiftUI `App` /
  `Scene` graph parity.

- `settings`
  App-shell example for the concrete `Settings(...)` scene. Use this to
  validate the dedicated settings surface in isolation. The flagship
  command/menu proof now lives in `scenes`.

- `form`
  Flagship text-entry surface for `TextFieldPolicy`, `SecureFieldPolicy`, and
  `TextEditorPolicy`, plus explicit focus routing, validation state, and
  macOS-native edit-responder behavior such as Select All, Cut, Copy, and
  Paste.

- `table-outline`
  Curated data-surface example for `TableModelView`, `OutlineModel`,
  selection, sorting, reveal, activation, and detail panes driven by model
  state plus stable AX inspection targets on the summary and detail regions.
  Use this to validate the stronger curated table/outline path, not native
  `Table` / `OutlineGroup` parity.

- `native-table-outline`
  Native-backed data-surface example for `NativeTableModel`,
  `NativeOutlineModel`, explicit selection and expansion state, reusable
  column visibility and width state, and denser desktop-style behavior. Use
  this to validate the additive native-backed layer, not raw SwiftUI generic
  `Table` / `OutlineGroup` parity. The native example now also carries stable
  AX identifiers on its summary, detail, and activation regions to match the
  curated proof surface.

- `layout`
  Constrained layout example for `LayoutSpec` / `AnyLayout` v1, including
  row, curated grid, and flow layout switching, plus the explicit
  `CustomLayout(...)` / tagged-layout models and placement presets for
  featured grids and primary/secondary shells. Use this to validate the
  concrete Go-native layout runtime, not raw SwiftUI `Layout` /
  `LayoutValueKey` parity.

- `accessibility`
  Accessibility and focus example for the current rotor model, stable
  accessibility identifiers, accessibility value/state metadata, and a
  menu-driven keyboard focus target.

- `media-transfer`
  Concrete media and transferable example for text and payload paste, share,
  drag/drop, and the curated `PhotosPicker(...)` surface backed by
  `PhotosPickerSelectionState`, including optional lazy file-backed sample
  assets.

- `charts`
  Primary charting example. Use this to validate that SwiftUI-side work does
  not regress the separate `charts` bridge.

## Coverage Map

- Scenes: `scenes`, `workbench`
- Settings scene: `settings`
- Text input: `form`, `settings`
- Table / outline: `table-outline`, `workbench`
- Native-backed table / outline: `native-table-outline`
- Layout v1: `layout`, `workbench`
- Accessibility rotor: `accessibility`
- Media / paste / share / drag-drop / photos: `media-transfer`, `bridge-coverage`
- Refreshable: `bridge-coverage`, `workbench` notes
- Charts: `charts`

## Validation

From the repo root:

```bash
./examples/build-flagship.sh
```

For AX and screenshot work, build or launch named binaries through:

```bash
notes/examples/launch-example.sh ./examples/workbench workbench-ui
notes/examples/launch-example.sh --foreground ./examples/workbench workbench-ui
```

The launch workflow is documented in:

- [`notes/examples/launch-path.md`](/Users/tmc/go/src/github.com/tmc/swiftui/notes/examples/launch-path.md)

## Policy

When a new runtime-backed SwiftUI family lands, it should have:

- one dedicated example if the feature is large enough to deserve it
- one mention or route in `workbench` if the feature is central to the current
  bridge story
- no claim of SwiftUI `App` / `Scene` parity unless the runtime actually owns it
