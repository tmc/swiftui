# Bridge Generator Go-Side Drift (Pre-C1 Charter)

Charter note opened 2026-04-16. Scope: the Go-side analog of the Swift-side
convergence migration charted in `notes/bridge-generator-convergence.md`.
Sibling note, not sub-of.

**Status: preliminary scope capture only.** Tracks the problem, does not yet
propose a solution. The C1 tranche opens after the `perf/2026-q2` tag lands
and P7 closes. Until then, this note is the record of scope + the bucket-A/B/C
analog pre-audit hints that an ω-agent team will use when C1 begins.

## 1. Problem

The Swift-side convergence note documents emission collisions in
`bridge_*.gen.swift`. A parallel phenomenon exists on the Go side: over the
P1–P6 perf arc, hand-edits accumulated in generator-produced Go files
(`lib.go`, `callback.go`, `state.go`, `view.go`, `views.go`, etc., plus every
subpackage's `generate.go` and `views.go`). The template has not kept pace,
so running `go generate ./...` on any a2ui tip regenerates these files to
their pre-arc template shape, reverting the perf-arc improvements and
breaking Go-side callers.

Surfaced during B1 review: after B1 correctly narrows Swift-side emission, a
clean regen on swiftui @ 22c653c produces:

- Swift build: 0 errors (B1 Swift-side scope is complete).
- Go build of `examples/scenes`: **FAIL** — `bridge_extra.go:83: undefined:
  retainedOwned` and 8 more similar errors, because `lib.go` regen reverted
  the P6c `retained` → `retainedOwned` + `retainedTransient` split back to
  pre-P6c `type retained`, and every Go-side caller that expects the split
  types breaks.

P6c is the most visible but not the only drift. The full regen produces:

- **34 files changed, +1108 insertions, -889 deletions, net +219 lines.**
- Root-package files with meaningful drift: `app.go`, `callback.go`,
  `doc.go`, `font.go`, `generate.go`, `glass.go`, `lib.go`, `render.go`,
  `state.go`, `view.go`, `views.go`, `webview.go`, plus the four generated
  Swift files (covered by the Swift-side convergence note, not this one).
- Subpackage regen noise: nine subpackages (arkit, avkit, charts,
  localauth, quicklook, scenekit, spritekit, translation, workoutkit) each
  have a `generate.go` (+7 lines) and `views.go` (+2/-1 lines) drift plus
  a `charts/lib.go` (+2/-1) drift. Uniform shape — likely a single
  template-site change times N packages.

## 2. Why This Is Not Part Of The Swift-Side Convergence

`notes/bridge-generator-convergence.md` scopes to `bridge_*.swift`
hand-written files on the swiftui side vs. template emission in the
appledocs `swiftui_templates.go`. The ω1–ω5 audit population was
deliberately Swift-side-only. The Go-side emitter (`swiftui_go_emitter.go`
in appledocs, driving a different set of templates and Go-side catalog
shapes) has its own collision pattern that the Swift audits do not cover.

Keeping the two tranches separate is the cleaner design:

- Disjoint audit populations (Swift `bridge_*.swift` vs Go root-package +
  subpackages).
- Disjoint emitters on the appledocs side (Swift templates in one block,
  Go templates in another).
- Disjoint completion criteria (Swift regen-idempotent vs Go
  regen-idempotent).
- Neither blocks P7 individually; P7 touches hand-written dispatch sites
  in Go and neither Swift nor Go regen is a precondition for P7 work.

## 3. Scope Capture

Measured on a clean worktree: detached HEAD at swiftui a2ui `22c653c`,
`GOWORK=off go generate ./...`, applegen built from appledocs
`2026-01-15-clean` tip (with B1 landed).

### Root-package Go files

These are the primary C1 audit targets. Per-file `+/-` counts from the
clean regen diff, plus a pre-audit hint based on the shape of the drift.

| File | `+` | `-` | Shape | Pre-audit hint |
|---|---|---|---|---|
| `app.go` | 17 | 0 | Pure additive | **Bucket C likely.** Hand-additions only; template has no corresponding content. |
| `callback.go` | 150 | 112 | Two-way | **Mix A/B likely.** P3 callback slot table + P6 retained split propagation. Template has an older callback shape. |
| `doc.go` | 53 | 15 | Mostly additive | **Mostly C.** Docstring additions with a few template-owned lines revised. |
| `font.go` | 4 | 4 | Two-way (small) | **Bucket A likely.** Small uniform rename, probably the retained→retainedOwned propagation. |
| `generate.go` | 7 | 0 | Pure additive | **Bucket C.** Hand-added `go:generate` directives + comments. |
| `glass.go` | 10 | 10 | Two-way (small) | **Bucket A likely.** Retained-type propagation. |
| `lib.go` | 68 | 112 | Two-way (large) | **Mix A/B/C.** P6c retained split (bucket A — template has `retained`, hand wants `retainedOwned`/`retainedTransient`), plus other accumulated hand-edits. Biggest audit surface in the root package. |
| `render.go` | 13 | 1 | Mostly additive | **Mostly C.** |
| `state.go` | 31 | 66 | Two-way | **Mix A/B.** P2 dirty-skip + P2-tail infrastructure landed as hand-edits; template still emits pre-P2 state shape. |
| `view.go` | 183 | 186 | Two-way (large) | **Mix A/B.** View modifier surface extended significantly across the arc. |
| `views.go` | 219 | 145 | Two-way (large) | **Mix A/B.** View constructor surface with P6a/P6c touches. |
| `webview.go` | 2 | 2 | Two-way (trivial) | **Bucket A.** Retained rename propagation. |

**Subtotal**: ~12 root-package files, ~750 lines of drift across both directions.

### Subpackage drift

Nine subpackages each have two drifting files (`generate.go`, `views.go`),
plus `charts/lib.go`. All diffs are small and uniform across the set,
suggesting a single template-site change replicated across every subpackage:

| Subpackage | Files drifted | Shape |
|---|---|---|
| arkit | generate.go (+7), views.go (+2/-1) | Additive generate + tiny views rename |
| avkit | generate.go (+7), views.go (+2/-1) | Same shape |
| charts | generate.go (+7), lib.go (+2/-1) | Same shape |
| localauth | generate.go (+7), views.go (+2/-1) | Same shape |
| quicklook | generate.go (+7), views.go (+2/-1) | Same shape |
| scenekit | generate.go (+7), views.go (+2/-1) | Same shape |
| spritekit | generate.go (+7), views.go (+2/-1) | Same shape |
| translation | generate.go (+7), views.go (+2/-1) | Same shape |
| workoutkit | generate.go (+7), views.go (+2/-1) | Same shape |

**Subpackage pre-audit hint**: the `generate.go` hand-additions are
**bucket C** (additive `go:generate` / comment content); the `views.go`
drift is **bucket A** (uniform rename; likely retained propagation). One
C1 sub-commit probably covers all nine subpackages if the pattern holds.

### Ignored for this tranche

The four generated Swift files (`bridge_app.gen.swift`,
`bridge_helpers.gen.swift`, `bridge_modifiers.gen.swift`,
`bridge_views.gen.swift`) drift on every regen because the Swift-side
convergence note covers them. Not re-audited here.

## 4. Pre-Audit Bucket Hints

Based on the shapes above, C1 ω-agent assignments will likely land as:

- **Bucket A (collision, template-owned-and-emits-wrong-thing)**: P6c
  retained split propagation across `lib.go`, `font.go`, `glass.go`,
  `webview.go`, and subpackage `views.go`. Uniform rename. Largest
  symbol count but smallest audit effort per symbol.
- **Bucket B (template-could-emit-but-does-not)**: P3 callback slot
  table (template emits old mutex-map shape), P2 dirty-skip + P2-tail
  state infrastructure in `state.go`, modifier chain packing bits in
  `view.go`/`views.go`. Biggest port effort per C3+ sub-commit.
- **Bucket C (hand-written-legitimate)**: `app.go` additions,
  `doc.go` additions, `render.go` additions, subpackage `generate.go`
  `go:generate` directives and comments. Permanent hand surface; does
  not port.
- **Bucket D**: none expected based on the diff shapes, but the audit
  should confirm.

The total audit population is roughly **~40 distinct symbols** across
~12 root files + 9 subpackages (estimated; C1's ω agents produce the
exact count).

## 5. C1 Charter Shape (Post-P7)

To be designed when C1 opens. Expected parallel to the Swift-side
convergence tranche:

1. **C1 Audit**: fresh ω-agent team (C1.ω1 root-package files, C1.ω2
   subpackages, C1.ω3 appledocs Go-emitter template review). Per-symbol
   A/B/C classification. Output:
   `notes/generator-gaps.md` updated with Go-side bucket entries.
2. **C2 Narrowing**: if the Go-emitter template has collisions
   analogous to the Swift-side ones, gate with a `emit_go_converged_surface`
   flag (default false) mirroring B1's pattern. May not be needed —
   the Go-side collision might be "template emits wrong content" rather
   than "template emits extra content", which is a port problem not a
   narrow problem.
3. **C3+ Port/Prune**: per-file, sequential on the regen baseline, one
   commit per file. Port bucket-A content to template (most common
   path), port bucket-B content where the symbol belongs in the
   template, keep bucket-C as hand-written with an audit-trail entry.
4. **C4 Close**: regen on a clean swiftui tree produces no diff. Tag
   this note retired.

## 6. Relationship To Other Notes

- `notes/bridge-generator-convergence.md`: **sibling**, not sub-of. Same
  discipline (gated emission, per-file prune, bucket A/B/C). Disjoint
  audit populations. Both block the long-run regen-idempotent goal;
  neither blocks P7.
- `notes/performance-optimization.md` P1–P6 ledger: the drift
  captured here is the record of what P1–P6 added that the template
  didn't absorb. The perf wins themselves are intact on the a2ui tree;
  this note exists so they don't get lost on the next regen.
- `notes/generator-gaps.md`: C1 audit output lands here as a new
  "Go-side drift" section, analogous to the existing Swift-side
  "Priority" entries.

## 7. Freeze Policy

Until C1 opens, **do not run `go generate ./...` on swiftui**. This
freezes the Go-side tree in the P1–P6-winning state. Any new work
before C1 opens should:

- land as hand-edits to the generated Go files (same protocol that
  produced the current drift — acceptable given the freeze is
  time-bounded), OR
- land as new hand-written Go files that the generator has no notion
  of, OR
- land as appledocs template changes that are NOT regenerated onto
  swiftui until C1 protocol covers them.

P7 (main-thread coalescing) fits the "hand-edits to generated Go
files" path. Post-P7, C1 ports those edits into the template as part
of the normal C3+ protocol.

## 8. Stop Conditions

Stop C1 and re-scope if any of these happen:

1. The Go-emitter template turns out to have architectural
   limitations that prevent emitting the bucket-B content (e.g., the
   template structure can't represent `atomic.Pointer[T]` state cache
   field types). In that case, mark those symbols bucket C and move on.
2. Porting bucket-A content changes perf characteristics measured on
   the P1–P6 benchstat ledger. The port must preserve perf. If it
   can't, either the port is wrong or the perf win was non-template
   (some hand-written version of the algorithm) — audit and either
   fix the port or mark that perf path bucket C.
3. The subpackage drift turns out to not be uniform (audit produces
   surprises). In that case, split subpackage audit into per-package
   sub-audits.

## 9. Outputs This Note Does NOT Produce

This is a scope-capture note. It does not:

- Propose a specific template change.
- Classify any symbol as definitively bucket A/B/C (the hints in §4
  are pre-audit estimates, not verdicts).
- Set a timeline for C1.
- Block P7.

When C1 opens post-P7, this note is the starting context. The
ω-agent audit either confirms the pre-audit hints or revises them;
either outcome is acceptable.
