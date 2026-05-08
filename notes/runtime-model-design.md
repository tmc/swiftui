# Runtime Model Design

Status: historical.

This was the pre-closeout design sketch for the first runtime-backed SwiftUI
state families. Most of the work described here is already shipped:

- `NavigationPathState`
- date-selection and date-range state
- timer state
- curated and native-backed table / outline state
- borrowed scene actions
- runner-owned scene runtime state
- explicit text selection state

Do not treat this file as active planning.

Use the current notes instead:

- `swiftui-runtime-roadmap.md`
  Current repo-level status, execution order, and completion bar.
- `scene-app-parity-path.md`
  Active plan for deeper scene/app host work.
- `table-outline-native-parity-path.md`
  Reopen criteria and bounded follow-on work for data surfaces.
- `layout-runtime-parity.md`
  Reopen criteria and bounded follow-on work for layout runtime depth.
- `swiftui-binding-gaps.md`
  `appledocs`-facing backlog and report mapping.

If a new runtime model is needed, add it to one of the active notes above or
create a new bounded execution note with:

1. a concrete product case,
2. a named public API target,
3. one flagship example,
4. behavioral tests, and
5. explicit exit criteria.
