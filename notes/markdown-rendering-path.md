# Markdown Rendering Path

This is a dedicated execution note for landing markdown rendering as a
first-class primitive in `swiftui`, separate from the Codex clone and
separate from the broader binding-gaps plan.

Markdown is pulled out of `swiftui-binding-gaps-plan.md` Tranche B2 because
the design surface (inline formatting, code blocks, selection, link
handling, image policy, streaming updates) is larger than one bucket item
and needs its own completion bar.

## Charter

Land a curated markdown view that is:

- good enough to render agent transcripts, review panes, README-style docs,
  and chat messages in flagship examples,
- honest about what it does and does not render (explicit supported subset,
  no silent downgrades),
- reusable across examples, not codex-clone-specific,
- buildable on the current curated surface without new public bridge churn
  unless a gap shows up mid-track.

## Non-Goals

- full CommonMark 0.31.2 conformance,
- full GitHub-Flavored Markdown including tables with alignment, task
  lists, and arbitrary HTML passthrough,
- a markdown editor (parsing only, not authoring),
- rich image rendering beyond the existing curated image surface,
- math/LaTeX blocks,
- custom markdown extensions for agent-tool-call formatting — those
  belong in the transcript row primitives, not here.

## Completion Bar

This track is complete when all of the following are true:

1. a `Markdown(source string)` view exists in the public surface with a
   documented supported subset,
2. at least three flagship examples render real content with it without
   per-example scaffolding,
3. the supported subset is covered by behavioral tests, not just
   snapshot strings,
4. streaming updates (append-only text) do not reflow already-rendered
   blocks that did not change,
5. selection, copy, and link activation work without manual wiring in the
   consuming example, and
6. the renderer is not the hot path's bottleneck on the performance
   benchmarks currently tracked in `performance-optimization.md`.

## Execution Order

Work these milestones in order.

### M1 — Supported Subset Decision

Goal:

- write down exactly what the renderer supports before any code.

Decide and record in this note:

- inline: bold, italic, inline code, links, line breaks,
- block: paragraph, heading levels 1–6, unordered list, ordered list,
  nested lists to a fixed depth, fenced code block with language hint,
  blockquote, horizontal rule,
- explicit exclusions: raw HTML, tables, task lists, images (beyond
  link-to-asset), footnotes, definition lists, strikethrough (decide
  yes/no), autolinks (decide yes/no), math.

Exit criteria:

- the supported subset is a list in this note, not an aspiration,
- every excluded feature has one reason given.

### M2 — Parser Boundary

Goal:

- pick a parser strategy and draw the Go/Swift boundary.

Decide:

- parse on the Go side and send a structured AST or display list over
  the bridge, **or**
- send raw source to Swift and let `AttributedString(markdown:)` do the
  rendering.

Likely call: Go-side parser with a small AST wire type, because:

- `AttributedString(markdown:)` only covers the inline subset, not block
  structure,
- streaming performance is easier to reason about with a Go-side parser,
- keeps the bridge narrow (one "render markdown AST" call, not N per-node
  calls).

But verify before committing. Prototype both for one non-trivial document
and compare bridge call count and frame-to-frame stability.

Exit criteria:

- one choice is written down with the prototype numbers that justified
  it,
- the wire type for the AST (or the fallback `AttributedString` shape) is
  frozen enough that M3 can build against it.

### M3 — Core Renderer

Goal:

- ship the renderer behind the subset chosen in M1, using the boundary
  chosen in M2.

`swiftui` work:

- `Markdown(source string) View` public constructor,
- internal parser + AST types (or `AttributedString` bridge helpers),
- selection and link activation wired through existing view surface,
- one Swift-side view that consumes the AST and renders using existing
  text primitives; do not introduce a second text rendering path.

Likely files:

- new `markdown.go` in the root package,
- new `internal/swift/Sources/bridge_markdown.swift` or equivalent,
- tests in `markdown_test.go` exercising the supported subset.

Rules:

- no public API churn to existing `Text`, `Label`, or scroll surface,
- no new hand-written entries added to `generator-gaps.md` without
  flagging them there,
- all rendering goes through existing text primitives — no parallel
  text stack.

Exit criteria:

- `Markdown(...)` renders the full M1 subset,
- behavioral tests cover each supported element at least once,
- no snapshot-only tests for rendered output; test the parsed structure
  and the produced view tree semantics.

### M4 — Streaming Stability

Goal:

- make append-only updates cheap enough that transcript streaming is
  fluid.

`swiftui` work:

- memoize block-level render output keyed on stable block identity,
- when the source grows by appended text, reuse prior block output for
  untouched prefix blocks,
- add a benchmark exercising a 10 KB transcript grown to 100 KB by
  append, measuring re-render cost per append.

Rules:

- do not change the public `Markdown(...)` signature,
- stay within the performance budget set in
  `performance-optimization.md`. If a tranche from that note is needed
  to make this milestone cheap enough, coordinate with that note before
  widening scope here.

Exit criteria:

- per-append re-render cost is sub-millisecond on the reference bar
  (M4 Pro, macOS 26.x, Go 1.26.x) for a 100 KB transcript,
- appended text does not reflow earlier rendered blocks.

### M5 — Flagship Adoption

Goal:

- prove the renderer on real content in more than the codex clone.

Adopt `Markdown(...)` in at least three flagship examples:

1. `examples/codex-clone` transcript pane (primary motivator),
2. one documentation/help pane in an existing flagship example that
   currently spells out headings and lists by hand,
3. one chat/message example, either an existing one or a small new
   example if none fits.

Rules:

- no example-specific renderer extensions,
- if an adopter needs something outside the M1 subset, stop and decide:
  either extend M1 explicitly, or reject the use case as out of scope,
- do not add a markdown feature for one adopter only.

Exit criteria:

- three adopters render real content without per-example scaffolding,
- the codex clone's hand-rolled transcript formatting is replaced by
  `Markdown(...)` for at least message bodies and code blocks,
- consumer code is shorter after adoption than before.

### M6 — Docs And Closeout

Goal:

- document the final surface and retire this note.

- document the supported subset and explicit exclusions in the
  `Markdown(...)` godoc,
- update `swiftui-binding-gaps.md` classifier counts if any symbol moved
  from deferred to shipped,
- update `swiftui-binding-gaps-plan.md` Tranche B2 to mark markdown as
  shipped and point at the shipped API, not at this note.

Exit criteria:

- this note can be retired,
- future markdown changes happen in code + godoc, not in a backlog
  note.

## Stop Conditions

Stop the track and record the blocker instead of continuing if any of
these happen:

1. M1 subset decisions cannot be made without knowing a specific
   product requirement that has not been written down,
2. M2 prototype shows that neither boundary choice hits acceptable
   performance with the current bridge shape,
3. M4 cannot meet the streaming budget without public API churn or
   without a dedicated tranche in `performance-optimization.md`,
4. M5 adopters consistently ask for features outside the M1 subset,
   which means the subset was wrong and should be re-decided before
   more shipping happens.

If a stop condition is hit, record the exact blocker in this note and
either revise the milestone or spin a focused follow-on note. Do not
fall back to vague future-work language.

## Validation

Run after each milestone:

```sh
GOWORK=off go test .
GOWORK=off go build ./examples/codex-clone
(cd internal/swift && swift build -c release --quiet --product SwiftUIBridge)
bash ./examples/build-flagship.sh
```

After M4 and M5, rerun the flagship adopters live and confirm the
rendered output matches the M1 subset without visible regressions.
