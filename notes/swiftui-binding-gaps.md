# SwiftUI Binding Gaps

This note tracks the current SwiftUI binding gaps separately from the generator design review and the demo work.

Update on March 29, 2026:

- Landed since this snapshot:
  - `MenuView`
  - `DisclosureGroupView`
  - `SectionExpandedView`
  - `ScrollViewReader(position, anchor, content)`
  - Bool-backed `SheetPresented`, `AlertPresented`, `ConfirmationDialogPresented`, `PopoverPresented`, and `FullScreenCoverPresented`
  - `ToolbarItem`, `ToolbarRole`, `KeyboardShortcut`, `Searchable`, `PointerStyle`, and `ID`
  - curated `OutlineGroup` and `Table` helpers on the Go side
- Important caveat:
  - `HoverEffect` remains intentionally absent because `View.hoverEffect(_:)` is unavailable on macOS.
  - `Table` is currently a curated table-layout helper, not a direct bridge to SwiftUI's generic native `Table`.

Source snapshot:

- Report file: `/tmp/swiftui-report.json`
- Generated on: `2026-03-29`
- Command:

```bash
cd /Volumes/tmc/go/src/github.com/tmc/appledocs
go run ./cmd/applegen swiftui-report SwiftUI --output /tmp/swiftui-report.json
```

## Snapshot

- Total analyzed symbols: `2774`
- Already supported: `1592`
- Deferred: `1053`
- Unsupported: `129`
- Deferred and already on a plausible codegen path: `351`
- Promotion candidates worth adding to the public catalog: `9`

The important distinction is that not every deferred or unsupported symbol is a missing product feature. A large share of the backlog is either already covered by curated APIs, internal implementation detail, or blocked on runtime-model work rather than emitter work.

## Codegen-Ready Buckets

These are the deferred symbols that already have a plausible lowering path.

- `opaque_return_erasure` (`193`)
  These are mostly `some View` / opaque-return APIs that could be lowered through the existing `AnyView`-style bridge path.
  Representative APIs: `MenuButton.menuButtonStyle(_:)`, `NavigationLink.isDetailLink(_:)`, `NavigationView.navigationViewStyle(_:)`, `Section.collapsible(_:)`, `Text.textVariant(_:)`.

- `concrete_specialization` (`139`)
  These are generic APIs that need curated concrete overlays instead of trying to expose open generics directly.
  Representative APIs: `OutlineGroup.init(_:children:)`, `PhaseAnimator.init(_:content:animation:)`, `PhaseAnimator.init(_:trigger:content:animation:)`, `SliderTickContentForEach.init(_:id:content:)`, `Text.init(timerInterval:pauseTime:countsDown:showsHours:)`.

- `existing_async_bridge` (`9`)
  These fit the existing async bridge machinery and mostly need SwiftUI-specific emitter wiring.
  Representative APIs: `OpenDocumentAction.callAsFunction(at:)`, `OpenWindowAction.callAsFunction(...)`, `OpenImmersiveSpaceAction.callAsFunction(...)`, `RefreshAction.callAsFunction()`, `Shader.compile(as:)`.

- `optional_primitive_pair` (`10`)
  These are optional scalar values that want a `value + ok` style bridge model.
  Representative APIs: `ProgressViewStyleConfiguration.fractionCompleted`, `ViewSpacing.width`, `ViewSpacing.height`, `ViewSpacing.minLength`.

## Promotion Split

The `351` codegen-ready symbols are not all equal. The report now splits them by what should happen next.

- `covered_by_catalog` (`36`)
  These are effectively closed already.
  Representative APIs: `NavigationSplitView` initializers, `List(selection:content:)`, `Picker(selection:content:)`, `MenuBarExtra` initializers.

- `implementation_detail` (`250`)
  These are real symbols, but they are not good public Go APIs and should usually stay internal or omitted.
  Representative APIs: `ViewBuilder.buildEither(...)`, `ToolbarContentBuilder.buildEither(...)`, `makeBody(configuration:)`, `body`, style internals, and optional layout plumbing.

- `promote` (`9`)
  These are the remaining codegen-ready items that look worth adding to the curated surface:
  - `PhaseAnimator.init(_:content:animation:)`
  - `PhaseAnimator.init(_:trigger:content:animation:)`
  - `SliderTickContentForEach.init(_:id:content:)`
  - `Text.init(timerInterval:pauseTime:countsDown:showsHours:)`
  - `MenuButton.menuButtonStyle(_:)`
  - `NavigationLink.isDetailLink(_:)`
  - `NavigationView.navigationViewStyle(_:)`
  - `Section.collapsible(_:)`
  - `Text.textVariant(_:)`

- `requires_runtime_model` (`56`)
  These are not just emitter work. They need new Go-side abstractions or runtime support.
  Representative APIs:
  - `MultiDatePicker` selection initializers
  - `NavigationStack.init(path:root:)`
  - `NavigationSplitView` initializers with `preferredCompactColumn`
  - native `OutlineGroup.init(_:children:)`
  - timer-backed `ProgressView` initializers
  - `ShareLink` initializers
  - `Table` / `TableColumn` / `TableColumnForEach`
  - `TextSelection`
  - `OpenDocumentAction`, `OpenWindowAction`, `OpenImmersiveSpaceAction`, `RefreshAction`
  - `TimeDataSource`

## Deferred But Not Codegen-Ready Yet

These buckets still need type-model or policy work before they are good codegen candidates.

- `policy_unresolved_type` (`398`)
  The parser/classifier still cannot model some SwiftUI-heavy parameter or return types cleanly.
  Representative APIs: `KeyframeAnimator`, `KeyframeTrack`, `LabeledContent`, `LazyHStack`, `LazyVStack`, `List.init(content:)`, `Menu.init(_:content:)`.

- `policy_generic_specialization` (`196`)
  These are generic APIs that do not yet have an obvious curated concrete overlay.
  Representative APIs: `KeyframeTimeline`, `LinearGradient.init(stops:...)`, `MeshGradient`, `Picker.init(sources:selection:...)`, `RadialGradient.init(stops:...)`, generic `Slider` initializers.

- `policy_nested_type` (`72`)
  These are APIs that lean on nested Swift types that the bridge does not flatten yet.
  Representative APIs: `LabeledContent.init(_:value:format:)`, several `Slider` initializers, several `Stepper` initializers.

- `policy_collection` (`38`)
  These need typed slice or dictionary adapters on the bridge boundary.
  Representative APIs: `LazyHGrid`, `LazyVGrid`, `LinearGradient.init(colors:...)`, `MeshGradient.init(points:colors:...)`, `PasteButton`, `RadialGradient.init(colors:...)`.

## Unsupported Nominal Types

There are `127` unsupported symbols that currently show up as unclassified nominal types. These are mostly top-level SwiftUI declarations rather than immediately actionable missing bindings.

Representative examples:

- `NSHostingController`
- `NSHostingView`
- `UIHostingController`
- `List`
- `Menu`
- `MenuBarExtra`
- `LazyVStack`
- `LazyHGrid`
- `Label`
- `LabeledContent`

In practice, many of these are already partially represented through curated constructors or modifiers. They should not all be treated as “missing APIs to generate next.”

## Practical Next Moves

If the goal is to grow coverage without losing the curated shape:

1. Burn down the `promote` bucket first.
2. Keep `covered_by_catalog` and `implementation_detail` closed.
3. Treat `requires_runtime_model` as design work, not emitter backlog.
4. Revisit `policy_unresolved_type`, `policy_generic_specialization`, `policy_nested_type`, and `policy_collection` only after there is a concrete API shape worth promoting.

## Codex Clone Priorities

The standalone Codex shell in `examples/codex-clone` is a useful product-shaped audit because it exercises shell chrome, left-rail state, nested agent rows, contextual review UI, and footer menus in one place.

### Already present and usable

- `Help` for tooltip text (`view.go:563`)
- `Focusable` for keyboard-focus eligibility (`view.go:704`)
- `OnHover`, `OnHoverPhase`, and `OnHoverLocation` for hover callbacks (`view.go:822`, `view.go:833`, `view.go:844`)
- `ContextMenu` for row actions (`view.go:953`)
- `ListRowBackground` for selected-row styling inside real lists (`view.go:1035`)
- `Popover` and `PopoverPresented` for footer menus and Bool-backed popovers (`view.go:1089`, `view.go:1097`)
- `SelectableList` for selection-backed sidebars (`views.go:268`)
- `SectionExpanded`, `DisclosureGroupView`, and `SectionExpandedView` for BoolState-backed disclosure and custom-label collapsible sections (`views.go:306`, `views.go:316`, `views.go:324`)
- `DynamicView` and `DynamicBoolView` for state-driven subtree rebuilds (`views.go:474`, `views.go:494`)
- `NavigationSplitView`, triple-column variants, and visibility-backed variants (`views.go:541`, `views.go:549`, `views.go:558`, `views.go:566`)
- `Menu`, `MenuView`, and `PickerMenu` for overflow and option menus (`views.go:575`, `views.go:585`, `views.go:603`)
- `ToolbarRole`, `ToolbarItem`, `KeyboardShortcut`, `Searchable`, and `PointerStyle` for shell chrome and interaction polish (`view.go:744`, `view.go:751`, `view.go:759`, `view.go:769`, `view.go:779`)
- `SheetPresented`, `AlertPresented`, `ConfirmationDialogPresented`, `PopoverPresented`, and `FullScreenCoverPresented` for BoolState-backed presentation (`view.go:922`, `view.go:930`, `view.go:942`, `view.go:1097`, `view.go:1113`)
- `ScrollViewReader` plus `ID` for IntState-backed programmatic scrolling (`views.go:205`, `view.go:793`)
- `DefaultScrollAnchor`, `ScrollTargetBehavior`, `ScrollTargetLayout`, and `ScrollBounceBehavior` for semantic scroll positioning and snap/bounce policy (`view.go`)
- Curated `OutlineGroup` and `Table` helpers for hierarchical rows and equal-width column layouts (`views.go:741`, `views.go:780`)

For the Codex clone specifically, this means the manual shell can now move more of its sidebar disclosure, footer menu trigger, toolbar action, search field, and Bool-backed popover/dialog wiring onto built-in bridge surface instead of carrying custom scaffolding for each of those patterns.

### Closed since the previous pass

These items were previously called out as missing and are now closed on the Go side:

- Toolbar shell support is no longer blocked. `ToolbarRole` and `ToolbarItem` are exported in `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:744` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:751`.
- Keyboard shortcut binding is now exported in `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:759`.
- Search-field integration is now exported in `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:769`.
- Pointer styling is now exported in `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:779`.
- Custom-label menu and disclosure shells are now exported in `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:316`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:324`, and `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:585`.
- Bool-backed presentation state is no longer a gap for sheets, alerts, confirmation dialogs, popovers, or full-screen covers (`/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:922`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:930`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:942`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:1097`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:1113`).
- Programmatic scroll-to is now available through `ScrollViewReader` plus `ID` in `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:205` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:793`.
- Scroll snapping and bounce policy are now exported through `DefaultScrollAnchor`, `ScrollTargetBehavior`, `ScrollTargetLayout`, and `ScrollBounceBehavior` in `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go`.
- Curated hierarchical and table helpers now exist in `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:741` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:780`.

### Types already modeled but missing public wrappers

- `HoverEffectKind` exists (`view.go:346`), but there is still no public hover-effect modifier.

This bucket is much smaller now. On macOS, the main remaining “types already modeled” gap is hover-effect styling, and that one is intentionally absent because `View.hoverEffect(_:)` is unavailable on macOS.

### Still missing or still design work

- `SafeAreaInset`
- Rich transcript presentation helpers for markdown, code blocks, badges, and diff rows
- Focus state bindings beyond simple `Focusable(bool)`
- Native `OutlineGroup` / `Table` parity beyond the current curated Go helpers

For the Codex clone, the biggest remaining blockers are richer transcript primitives, safe-area-aware shell layout, and more capable focus management. Nested disclosure and programmatic scroll-to are no longer first-order blockers, even though the current outline/table helpers are curated rather than one-to-one SwiftUI API mirrors.

### Existing bindings the clone should adopt before adding more surface

Some of the current manual shell work is not blocked on new bindings.

- The sidebar rows in `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:376` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:416` are hand-rolled buttons. `SelectableList` plus `Tag` in `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:268` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:786` would give the rail a real selection model instead of custom selected-state bookkeeping.
- The clone currently toggles agent expansion manually in `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:407` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:472`. `DisclosureGroupView`, `SectionExpandedView`, or the curated `OutlineGroup` in `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:316`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:324`, and `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:741` can now cover custom-label disclosure without hand-building the full tree shell.
- The sidebar filter at `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:261` is still a plain text field. `Searchable` in `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:769` now gives the shell a semantic search field.
- Footer menus in `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:688` still use manual pill buttons and per-menu state. `MenuView`, `PopoverPresented`, `Help`, `OnHoverPhase`, and `PointerStyle` in `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:585`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:1097`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:563`, `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:833`, and `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:779` can now replace a lot of that custom chrome.
- Transcript and review panes can now adopt `ScrollViewReader` plus `ID` from `/Volumes/tmc/go/src/github.com/tmc/swiftui/views.go:205` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:793` instead of keeping scroll position entirely implicit.

### Remaining ergonomic gaps exposed by the clone

The Codex shell does not just need more symbols. A few remaining places still want either broader modifiers or richer higher-level widgets.

- `HoverEffectKind` exists in `/Volumes/tmc/go/src/github.com/tmc/swiftui/view.go:346`, but there is still no public `HoverEffect` modifier to pair with the existing hover callbacks.
- `OutlineGroup` and `Table` now exist as curated Go helpers, but they are not native SwiftUI `OutlineGroup` / `Table` bindings. They are good enough for custom-label tree rows and equal-width column layouts, but they do not provide native table sorting, built-in column resize, or lazy tree disclosure.

### Binding additions with the highest Codex payoff

If the goal is to make `examples/codex-clone` substantially cleaner without overexpanding the public surface, the highest-payoff additions are:

1. Rich transcript primitives for markdown, code blocks, badges, and diff rows, so `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:570` and `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/codex-clone/shell.go:631` do not need every message and review row spelled out as custom layout.
2. `SafeAreaInset`, so footer and accessory chrome can attach semantically instead of being packed into manual outer stacks.
3. Focus state bindings beyond simple `Focusable(bool)`, so keyboard navigation can move through the shell without manual focus bookkeeping.
4. If platform scope ever expands beyond macOS, revisit `HoverEffect`. On macOS it remains intentionally absent because the underlying SwiftUI modifier is unavailable.
5. If native platform parity becomes important, a second pass on true SwiftUI `OutlineGroup` / `Table` bindings rather than the current curated helpers.
