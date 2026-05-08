# AXMCP Snags

These notes come from driving `/Volumes/tmc/go/src/github.com/tmc/swiftui/examples/workbench`
and from inspecting the real Codex desktop app.
They are meant to be handoff material for improving AXMCP against SwiftUI-heavy UIs.

## Repro Context

- Build: `go build -o /tmp/workbench-ui ./examples/workbench`
- Run: `/tmp/workbench-ui`
- App title: `Surface Workbench`
- Key structure:
  - left sidebar: `SelectableList`
  - center content: mix of `ScrollView`, `SelectableList`, `NavigationStack`
  - right inspector: `Form` with `SectionExpanded`

## Concrete Snags

1. `ax_find` and `ax_click` by visible sidebar text did not work for `SelectableList` rows.
   - Visible OCR text included entries like `Planner`, `Documents`, and `Runtime Gaps`.
   - AX lookup did not expose those labels.
   - `ax_find(app=\"workbench-ui\", contains=\"Planner\")` returned only unrelated candidates like toolbar buttons.

2. OCR and AX trees disagreed on what text existed.
   - `ax_ocr` could see sidebar row text and truncated labels like `Pla...`, `Do...`, `Na...`.
   - `ax_tree` showed the sidebar as:
     - `AXScrollArea`
     - `AXOutline desc="Sidebar"`
     - a sequence of `AXRow` and `AXCell`
   - Those rows/cells had no accessible text values.

3. Focus was ambiguous because there were multiple `AXOutline` elements.
   - One outline was the real app sidebar.
   - Another outline lived inside the center panel (`Selection-Driven Workspace`).
   - `ax_focus` reported the inner outline:
     - `AXOutline bounds=(740,-1374 280x280)`
   - Arrow-key navigation therefore targeted the wrong list, or at least appeared to.

4. `ax_focus` was unstable after some interactions.
   - At one point it returned:
     - `no focused element and no main window found (app might be in background or has no standard UI)`
   - The window was still visibly present and interactive.

5. Window bounds from `ax_list_windows` were not useful.
   - It reported the window title correctly.
   - It returned zero bounds:
     - `x=0,y=0,width=0,height=0`
   - The deeper AX tree later showed nonzero bounds for the same window.

6. AX bounds were in a coordinate space that was hard to use directly.
   - Many elements had negative coordinates, for example:
     - `AXWindow title="Surface Workbench" bounds=(551,-1848 1460x1004)`
     - `AXOutline desc="Sidebar" bounds=(577,-1452 104x542)`
   - OCR coordinates were in a normal screenshot-local space.
   - This mismatch made coordinate-based fallback clicking brittle.

7. Current fallback required mixing tools manually.
   - Successful route targeting likely needs:
     - `ax_tree` to locate the correct `AXOutline`
     - OCR to understand visible row labels
     - manual coordinate math against the outline bounds
   - There is no direct raw-coordinate click helper in the current flow.

8. Sidebar row labels were truncated visually.
   - OCR often saw `Pla...`, `Do...`, `Na...`, etc.
   - Even if AX text lookup worked, exact-text targeting would still be fragile.

9. `ax_click` did not always behave like a real click.
   - In the workbench sidebar, AXMCP sometimes reported a successful click on a route label, but the UI behaved as if the pointer only moved or hovered.
   - This was visible when trying to change routes: AXMCP printed a click result, yet selection did not change until using a different hit point inside the row.
   - Another agent should verify whether `ax_click` is always delivering a full press/release event, or whether some SwiftUI surfaces only receive pointer movement from the current implementation.

10. OCR-backed text lookup missed visible toolbar text in Codex.
   - In the real Codex app, `ax_ocr(app=\"Codex\", json=true, find=\"Hand off\")` succeeded, but `ax_ocr(app=\"Codex\", json=true, find=\"Push\")` failed even though `Push` was visibly present in the same toolbar strip.
   - This makes OCR-driven targeting unreliable for compact toolbar controls, especially when labels sit next to icons, dividers, or rounded button chrome.
   - Another agent should check whether the OCR pipeline is dropping short labels, low-contrast text, or text embedded inside segmented/compound toolbar controls.

11. True right-click flow on Codex thread rows still was not reproducible through the current automation stack.
   - Semantic AX lookup could expose some row-adjacent affordances in earlier passes, but not a direct context-menu entry point for OCR-only thread rows.
   - External fallback using `cliclick rc:x,y` was blocked by missing Accessibility permission for the calling process.
   - Result: we could confirm hover affordances like `Expand agent threads`, but not an OS-level row context menu.

## Hover-Diff Workflow In Codex

During the live Codex UI pass, the working route for capturing real hover states was:

1. `ax_list_windows(app="Codex")` to get the live window bounds and `window_id`.
2. `ax_screenshot(app="Codex", window="Codex")` to confirm the visible state before targeting.
3. `ax_find(app="Codex", window="Codex", role="AXButton")` to locate controls that were truly AX-addressable.
4. `ax_action_screenshot(... action="hover" ...)` or `ax_ocr_hover(...)` to move the pointer onto the target.
5. `/usr/sbin/screencapture -o -l <window_id>` to save real before/after PNGs because the AXMCP image previews were not durable artifacts.
6. A small Pillow script to crop around the target bounds and build a readable diff image.

That route produced usable hover evidence for:

- `Commit`
- `Hide sidebar`

The live workflow was still awkward for native automation:

1. `ax_action_screenshot` could prove that pixels changed, but it did not save `before`, `after`, and `diff` files directly.
   - The fallback had to use native `screencapture` plus a separate crop/diff step.

2. There was no explicit "clear hover" or window-local hover-by-coordinate primitive.
   - Capturing a neutral before-state meant hovering some unrelated far-away control or visible OCR text.

3. Hover targeting by substring was ambiguous in compact SwiftUI chrome.
   - Queries like `New thread` or `Playground 2` could match the wrong nested control instead of the intended row or header.

4. Whole-window diffs were too noisy during a live agent session.
   - Codex transcript content kept changing while captures ran, so only cropped target-region diffs were trustworthy.

5. AX and OCR still covered different parts of the UI.
   - Toolbar buttons like `Commit` and `Hide sidebar` were AX-addressable.
   - Left-rail rows and footer pills were often visible but not cleanly targetable through AX.

6. OCR remained weak on compact, low-contrast toolbar/footer text.
   - It was good enough for some labels and bad enough to miss or misread others in the same strip.

7. Tooltip capture needed explicit settle timing.
   - `Hide sidebar` only became useful once the hover waited long enough for `Toggle sidebar` and `⌘B` to appear.

## What Helped

- `ax_screenshot(app="workbench-ui")` was the most reliable way to see what was actually on screen.
- `ax_tree(app="workbench-ui", depth=8)` exposed the real structure, including the sidebar outline.
- `ax_ocr(app="workbench-ui", json=true)` was useful for locating visible text and approximate positions.
- `ax_find(app="Codex", window="Codex", role="AXButton")` was the fastest way to separate genuinely targetable toolbar controls from OCR-only surfaces.
- `ax_action_screenshot` was useful as a quick proof that hover changed pixels, even when it was not sufficient as the final artifact pipeline.
- `ax_ocr_hover` was a workable fallback for moving the pointer onto visible text when AX targeting was unavailable.
- Shadowless `screencapture -o -l <window_id>` was the cleanest external fallback because it produced window-only PNGs in the same coordinate space as `ax_list_windows`.

## Improvements To Consider

1. Add a direct click-by-coordinate tool for app-local coordinates.
   - OCR already gives usable local coordinates.
   - The missing piece is a reliable way to click those coordinates without anchoring off another AX element.

2. Add a helper to enumerate row text for SwiftUI `List`/`Outline` structures.
   - If AX text is missing, fall back to OCR text associated with row bounds.
   - Expose rows as actionable items instead of raw `AXRow`/`AXCell`.

3. Add a way to disambiguate multiple outlines/lists.
   - Example desired query:
     - "find outlines under the left split-group"
     - "focus sidebar outline"

4. Normalize or explain coordinate spaces.
   - AX tree bounds and OCR bounds should be easier to relate.
   - At minimum, expose whether coordinates are screen, window, or local.

5. Improve `ax_find` fallback reporting for custom SwiftUI content.
   - Current output is too sparse when text exists visually but not in AX.
   - It should suggest OCR-backed candidates when AX lookup fails.

6. Add a "click OCR match" flow.
   - Example:
     - OCR find `Planner`
     - click the center of that OCR match inside the app window

7. Add native file output to the screenshot and diff tools.
   - `ax_screenshot`, `ax_action_screenshot`, and `ax_ocr_action_diff` should be able to save artifacts directly.
   - Ideal output is a small bundle or directory containing `before.png`, `after.png`, and `diff.png`.

8. Add a native hover-by-coordinate tool in window-local space.
   - `ax_window_click` exists now, but there is no sibling hover primitive for local coordinates.
   - OCR already returns good local coordinates for many visible labels.

9. Add an explicit "clear hover" or "hover window background" action.
   - Hover-diff capture needs a reproducible neutral state before the target hover begins.

10. Add exact-match and stable-handle targeting for hover actions.
    - Substring lookup is not enough when `New thread` can match multiple nested controls.
    - A good pattern would be:
      - `ax_find(...)` returns a stable target id
      - `ax_hover(target_id=...)` reuses that exact match

11. Add target-scoped diff capture.
    - Let hover/click diff tools crop to the target bounds plus padding instead of diffing the full window.
    - This avoids noise from unrelated live content updates.

12. Add a unified AX-or-OCR target resolver.
    - If AX lookup fails or is ambiguous, the tool should be able to fall back to OCR within the same region and return one best target with bounds and confidence.

13. Add tooltip-aware settle options.
    - Hover capture often needs a second phase:
      - immediate hover state
      - delayed tooltip state
    - The tool should support both without requiring shell loops.

14. Add a native right-click or `AXShowMenu` path for OCR-backed targets.
    - Current tooling can left-click OCR matches, but not open a row context menu when the row itself is only OCR-visible.
    - This is especially relevant for apps like Codex where thread rows can expose nested controls and contextual actions.

## Minimal Repro Cases Worth Automating

1. Select a route in the left `SelectableList` sidebar of `examples/workbench`.
2. Verify selection in the center outline inside `Selection-Driven Workspace`.
3. Toggle the segmented column visibility control.
4. Read the right inspector sections and headings.
5. In Codex, hover `Commit` and save `before` / `after` / `diff` artifacts without using external `screencapture` or Pillow.
6. In Codex, hover `Hide sidebar` and verify that the delayed tooltip shows `Toggle sidebar` and `⌘B`.
7. In Codex, disambiguate `New thread`-like substring collisions and hover the intended control exactly.
8. In Codex, open a row context menu for `Plan next experiments` or a similar thread without relying on external tools like `cliclick`.

The first case is still the weakest general SwiftUI-navigation target.
The Codex hover-diff cases are the clearest end-to-end repros for making screenshot-based UI evidence native in AXMCP.
