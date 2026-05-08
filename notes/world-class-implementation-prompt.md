# Implementation Prompt: World-Class macOS Apps

Status: superseded.

Do not use this file as the active implementation brief. It predates the
current closeout pass and still assumes an older execution model that is no
longer the source of truth.

When briefing work now, use the active note that matches the task:

- `swiftui-runtime-roadmap.md`
  Repo-level workboard and execution order.
- `scene-app-parity-path.md`
  Active execution plan for deeper app/scene host ownership.
- `performance-optimization.md`
  Active execution plan for bridge hot paths and regression gates.
- `table-outline-native-parity-path.md`
  Closed-by-default follow-on note for data surfaces.
- `layout-runtime-parity.md`
  Closed-by-default follow-on note for layout runtime depth.
- `swiftui-binding-gaps.md`
  Backlog/report mapping for `appledocs`.

If a new implementation prompt is needed, generate it from one of those notes
and include:

1. the exact milestone or phase being worked,
2. the files and examples in scope,
3. the validation commands to run, and
4. the explicit done bar and stop conditions.
