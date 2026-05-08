# Bridge Generator Convergence (Pre-P7 Tranche)

Charter note opened 2026-04-16. Scope: the migration required before P7
(main-thread coalescing) can start, driven by the fact that the appledocs
template now emits enough of the bridge surface that it collides with
hand-written Swift files that previously filled those gaps.

This note fixes the plan. Implementation detail lives in the B1 / B2 / B3+
commit messages when they land.

## 1. Problem

`internal/swiftbridge/swiftui_templates.go` in `github.com/tmc/appledocs` has
grown incrementally across the P1-P6 / T1 arc. Each template increment was
safe in isolation because hand-written `bridge_*.swift` files in
`github.com/tmc/swiftui` owned the corresponding surface and the template
was silent about it.

That regime has ended. On the current `2026-01-15-clean` tip, `go generate
./...` now produces output that overlaps with hand-written files, and the
Swift compiler reports the overlap as hard errors rather than silent
re-exports:

- `bridge_a2ui_extra.swift:1070` — invalid redeclaration of
  `SUIAccessibilityIdentifier`. The template now emits the
  `.accessibilityIdentifier()` @_cdecl; the hand-written copy was landed
  in T1 when the template did not yet cover it.
- `bridge_a2ui_extra.swift:277` — ambiguous use of `_SUIStringCallback`.
  The generated `bridge_helpers.gen.swift` and the hand-written
  `bridge_commands.swift` both declare it.
- `bridge_app.gen.swift:17` — `SUISceneRunnerDelegate` ambiguous. Type is
  defined in `bridge_scene_plan.swift` (hand-written) and re-emitted by
  the template into `bridge_app.gen.swift`.

These are three different species of the same problem. The incremental
"revert one hand-written symbol at a time" regime that sustained P1-P6
has crossed a threshold where per-symbol hand-fixes no longer converge.
A coordinated migration is required.

The migration is not large, but it is load-bearing for P7: P7 touches
dispatch sites across the bridge, and running P7 against a bridge whose
emission surface is not settled would compound two separate design
problems into one failure mode. P7 is formally blocked behind this
tranche (see §8).

## 2. Guiding Principles

1. **Narrow-first, prune-incrementally.** Do not delete hand-written
   files until the template's emission is proven to produce a working
   Swift build on its own. The default state of the template gate is
   "emit nothing new" so narrowing is a pure regression-safety commit.
2. **One atomic commit per migration step.** B1 is one commit. Each
   pruned hand-written file in B3+ is one commit. No multi-file
   big-bang.
3. **Reversibility at every step.** The emission gate is a boolean per
   emission site. Any narrowing can be flipped back on by a single
   commit if the Swift build regresses.
4. **Stop conditions are explicit.** If a hand-written symbol cannot be
   ported to the template without observable behavior loss (Apple-API
   glue that does not factor, debug helpers that the generator has no
   notion of, etc.), it stays hand-written forever and is documented
   as bucket C in the audit. Bucket C is a legitimate terminal state,
   not a defeat.
5. **Regen parity, not regen coverage, is the success metric.** The
   bar for closing this tranche is "regen produces a Swift build that
   passes the same flagship-example set the pre-tranche tip passed",
   not "every symbol is template-owned".

## 3. Phase B1: Template Narrowing via Emission Gate

Single atomic commit on `github.com/tmc/appledocs`. Content:

1. Add a template flag `emit_converged_surface` defaulting to `false`
   on every emission site that currently collides with hand-written
   Swift. Emission sites identified by the three known collisions:
   - `SUIAccessibilityIdentifier` @_cdecl in the modifiers template.
   - `_SUIStringCallback` global and `SUISetStringCallback` @_cdecl in
     the helpers template.
   - `SUISceneRunnerDelegate` / `SUISceneWindowDelegate` /
     `SUIRunScenePlan` / `SUIOpenSceneWindow` / `SUIInstallSceneWindow`
     / `SUIRevealSceneWindow` / `SUIConfigureSceneMenuBar` /
     `SUIInstallQuitMenu` / `SUIInstallAppMenu` in the scene-plan
     emission block of the app template.
   - Three scene-plan *shadow symbols* flagged by ω2's audit but not
     yet in `notes/generator-gaps.md`: `suiSceneShouldRestoreVisibility`,
     `suiSceneShouldOpenOnLaunch`, `suiSceneVisibleKey`. Hand-written
     and template copies diverge on key derivation (hand-written calls
     `suiScenePersistenceKey(scene)`; template uses the raw `scene.id`),
     so the collision is semantic as well as syntactic. B1 must gate
     these alongside the named scene-plan symbols above or the narrowed
     regen will still produce divergent persistence behavior.

   Additional emission sites may be gated based on B2's findings, but
   B1 covers at minimum these three.

2. Flag is `false` by default. Regen on the current hand-written file
   set produces the same `bridge_*.gen.swift` output the pre-B1 tip
   produced — i.e. without the symbols now claimed by
   `bridge_a2ui_extra.swift`, `bridge_commands.swift`,
   `bridge_scene_plan.swift`.

3. Flag values live next to the emission sites, not in a centralized
   config. Design reason: one pruning commit touches one flag site
   and produces one regen diff. Centralizing the flags would couple
   unrelated prune commits.

### B1 exit criteria

- `go generate ./...` on `github.com/tmc/swiftui` with the narrowed
  template produces a clean Swift build.
- No hand-written Swift file has been modified.
- No prune sub-commits have landed yet.
- `swiftui` tests that were green on `2026-01-15-clean` tip are still
  green.

B1 is explicitly a non-functional commit: it changes generator output
only in the specific narrowing, not in anything else.

## 4. Phase B2: Audit

Goal: enumerate every hand-written `bridge_*.swift` file and every
symbol inside it, classify per symbol, and consolidate into
`notes/generator-gaps.md` (or an equivalent Priority section there).

Four buckets per symbol:

- **A. Template-owned-and-emitted.** Template currently emits this
  symbol; hand-written file also defines it; collision exists.
  Immediate narrowing candidate. These are the symbols B1 gates off.
- **B. Template-owned-but-not-emitted.** Template has the structure
  and could emit this symbol, but does not today. Future B3+ target:
  the template gets taught to emit, then the hand-written copy is
  pruned.
- **C. Hand-written-legitimate.** Apple-API glue that does not factor,
  `@MainActor` coordinators, AppKit delegate adapters, NSMenu
  builders, debug helpers, `@_cdecl` wrappers for symbols the
  generator has no notion of. Stays hand-written forever. The audit
  entry for bucket C records *why* it cannot be ported.
- **D. Unclear.** Escalate. Bucket D must be drained before B3+
  closes for the symbol in question.

### Per-file audit scope

Files in `internal/swift/Sources/` named `bridge_*.swift` that are
*not* `bridge_*.gen.swift`:

- `bridge_a2ui_extra.swift` — ω1 audit agent.
- `bridge_scene_plan.swift` — ω2 audit agent.
- `bridge_commands.swift` — ω3 audit agent.
- `bridge_packed_wire.swift` — NOT currently assigned to an ω agent.
  Flagged here so B2 assigns coverage before closing.
- Any other `bridge_*.swift` that is not `.gen.swift` — inspect
  directory at B2 kickoff and assign.

Each ω agent writes `/tmp/convergence-audit-<id>.md` with the
per-symbol bucket and the reason. Consolidation into
`notes/generator-gaps.md` happens after all agents return; the format
mirrors the existing tables in that file.

### B2 exit criteria

- Every symbol in every hand-written `bridge_*.swift` is classified
  A/B/C.
- Bucket D is empty.
- `notes/generator-gaps.md` Priority section reflects the full
  classification, with per-symbol ownership decision.
- Dropped-stash policy (§7) is reflected in the audit where relevant.

B2 does not change code. It is a documentation commit.

## 5. Phase B3+: Incremental Pruning

One sub-commit per hand-written file. Sub-commits are sequential on
the regen baseline: each one changes what the next regen produces, so
parallel execution is not safe.

Per-file protocol:

1. **Confirm the audit.** Re-read the bucket A + B entries for this
   file. If bucket D resurfaces, back out and drain it first.
2. **Port bucket B gaps into the template.** For every symbol still
   in bucket B for this file, extend the template so it can emit the
   symbol. This is the only point in the whole tranche that can
   introduce genuinely new emission.
3. **Flip the gate.** Set `emit_converged_surface` to `true` for each
   emission site corresponding to a bucket A or newly-ported bucket B
   symbol in this file.
4. **Regen.** `go generate ./...` in `github.com/tmc/swiftui`.
   Template now emits the symbols previously owned by the
   hand-written file.
5. **Delete / narrow the hand-written file.** If every symbol in the
   file is bucket A or B, delete the file. If the file has bucket C
   residue, narrow it to just the bucket C symbols.
6. **Verify.** Swift build clean, swiftui tests pass, one flagship
   example (`examples/scenes`, `examples/glass`, or equivalent) still
   builds and runs.
7. **Commit atomically.** One commit per file. Commit message names
   the file pruned, the symbols migrated, and the bucket C residue
   (if any).

### B3+ ordering

No strict ordering is mandated, but the recommended order is from
smallest to largest hand-written file, so that any template-side
regression shows up early against a small migration diff:

1. `bridge_a2ui_extra.swift` — bucket A is largest here (the three
   collisions all originate in this file or its neighbors); prune
   first because the audit is cheapest.
2. `bridge_commands.swift` — T1 surface; template has partial
   coverage (ω3 measured 25/25 symbols template-owned for a wholesale
   prune). Prune is blocked on one concrete template port: the
   generator's `suiSystemSelector` must grow four `NSSelector` cases
   — `closeWindow`, `minimizeWindow`, `zoomWindow`, `bringAllToFront`
   — that the hand-written version emits for the no-coordinator
   `SUIInstallDefaultMenus` path. Without those cases the template's
   narrower selector coverage would silently drop four window-action
   fallbacks. Ordering inside the commands prune is therefore
   port-first, prune-second: the selector port lands as the first
   sub-commit in the commands sequence, and the wholesale prune waits
   until the template's selector coverage matches hand-written.
3. `bridge_scene_plan.swift` — largest bucket A / B surface; wire types
   and scene-plan runner delegate. Two legitimate outcomes, both
   ratified:

   - **Option A: multi-commit template port.** Teach the template to
     emit scene-runner / scene-window / install-scene-window behavior
     with full parity, then prune `bridge_scene_plan.swift`. Meaningful
     work; the prune is the biggest template change in the tranche.
   - **Option B: permanent suppression.** The ~24 bucket-A collisions
     stay suppressed via the B1 flag indefinitely;
     `bridge_scene_plan.swift` stays hand-written as a stable long-term
     owner. `notes/generator-gaps.md` records this as an intentional
     terminal state (bucket C-equivalent for the file as a whole), not
     a TODO.

   Both outcomes are legitimate. Option B is preferred if the template
   is not otherwise gaining scene-runner complexity; Option A is
   preferred if a broader scene-runner template emission is already
   planned for other reasons (cross-platform support, scene-plan code
   sharing with iOS/visionOS when those come online). FF2A10CD picks
   at B3+ time based on scope budget.
4. `bridge_packed_wire.swift` and any other unassigned files —
   per B2 assignment.

### B3+ exit criteria

- Every hand-written `bridge_*.swift` file either deleted or narrowed
  to bucket C only.
- `notes/generator-gaps.md` updated to reflect the final ownership
  state (bucket C symbols remain listed; bucket A / B symbols are
  removed from the gaps list).
- Regen is idempotent: `go generate ./...` on a clean tree produces no
  diff.

## 6. Retained Hand-Written Residue

After B3+, the hand-written surface that stays is the bucket C set.
Examples expected to land in bucket C based on the current
understanding:

- `SUICommandCoordinator` (AppKit `NSMenuDelegate` conformance that
  the generator has no notion of).
- `SUIBuildMenuItems` recursive NSMenuItem tree builder.
- AppKit-backed helpers for scene-plan drag-and-drop, pasteboard
  routing, and NSWindow delegate plumbing.
- Any `@MainActor`-scoped singletons with non-obvious lifecycle.

Bucket C is not a TODO. It is a legitimate terminal state. Adding to
bucket C in later work is allowed; the ownership boundary is "the
generator cannot or should not emit this", not "the generator has not
gotten to it yet".

## 7. Dropped-Stash Policy

Two dangling commits from the prior session carry content that
interacts with this tranche.

- `446b069` — **ABANDON.** Bucket A content: work-in-progress toward
  the same goal the post-TextFieldSelection template now emits.
  Captured by the current `2026-01-15-clean` tip; re-applying would
  re-introduce the collisions B1 is narrowing. Do not re-stage.
- `b9ebd5d` — **RE-STAGE AS HAND-WRITTEN (bucket C).** Contains:
  - `@MainActor enum SUIRegexMatcher`
  - `struct BridgedPolicyTextField`
  - `struct BridgedPolicySecureField`
  - `struct BridgedPolicyTextEditor`
  - `@_cdecl SUIStateCreateStringList`

  This content is novel, has no template counterpart, and is
  legitimate hand-written surface. Re-apply on top of B1's narrowed
  regen as a separate commit. Record in `notes/generator-gaps.md`
  bucket C.

Re-staging `b9ebd5d` is gated on B1 landing; doing it before B1 would
reintroduce the collisions B1 narrows against.

## 8. Relationship to P7

P7 (main-thread coalescing) is formally blocked behind this tranche.

Rationale: P7 modifies dispatch sites across the bridge. Those
dispatch sites currently live in a mix of generated and hand-written
Swift. Running P7 against that mix would mean:

- Some dispatch changes land in hand-written files that B3+ will
  later delete, producing merge hell.
- Some dispatch changes land in template emission sites that B1/B3+
  is actively moving under P7's feet.
- The P7 regression signature would be indistinguishable from a
  convergence regression.

P7 work resumes after B3+ closes. The P1-P6 bench ledger remains the
baseline; no convergence commit alters perf characteristics, so the
P7 baseline is unchanged.

## 9. Close Conditions

This tranche retires when all of the following are true:

1. B1 has landed on `github.com/tmc/appledocs` with a green regen.
2. B2 audit is reflected in `notes/generator-gaps.md` with zero
   bucket D entries.
3. Every `bridge_*.swift` hand-written file is either deleted or
   narrowed to bucket C only.
4. Regen on a clean `github.com/tmc/swiftui` tree is idempotent
   (no diff).
5. Flagship examples (`examples/scenes` at minimum, plus whatever
   the ω audits identify as load-bearing) build and run.
6. `b9ebd5d` re-staged as bucket C or explicitly documented as
   abandoned.
7. P7 charter opens against the post-convergence baseline.

When those are true, this note is not deleted — it stays as the
historical charter for why the tranche existed. The Priority entries
in `notes/generator-gaps.md` that name bucket A / B symbols get
removed; bucket C entries stay.

## 10. Benchmarks / Regression Gates

Convergence does not alter perf characteristics. B1 changes generator
output shape, not runtime behavior; B3+ moves symbols between files
without changing their semantics. The existing P1-P6 bench ledger
remains load-bearing and is the P7 baseline.

Regression gates for each commit in this tranche:

- **B1 commit:** regen on current `swiftui` tip produces clean Swift
  build; `swiftui` test suite green; no flagship example regresses.
- **B3+ per-file commits:** same regen-and-test gate, plus the
  specific flagship example exercising the migrated symbols
  (scenes/commands for scene-plan pruning, a2ui-extra-dependent
  example for `bridge_a2ui_extra.swift` pruning, etc.).

No new benchmarks are introduced by this tranche. If a bucket B port
to the template changes dispatch shape in a way that touches hot
paths (unlikely — the hot paths are P1-P6 territory), the owning
commit must include a benchstat against the P1-P6 ledger and keep the
delta within noise.

---

Owners: `github.com/tmc/appledocs` for B1 and B3+ template work;
`github.com/tmc/swiftui` for prune commits and regen. ω1/ω2/ω3 sibling
agents are writing their audits to `/tmp/convergence-audit-*.md` in
parallel with this charter and will be consolidated into
`notes/generator-gaps.md` after B1 lands.
