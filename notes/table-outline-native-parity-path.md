# Table / Outline Native-Parity Path

Status: closed unless a concrete desktop workflow fails the shipped surface.

The current data story is already good enough to ship:

- curated `TableModel` / `OutlineModel` for the default Go-first path
- additive `NativeTableModel` / `NativeOutlineModel` for denser desktop
  behavior
- flagship proof in `examples/table-outline` and
  `examples/native-table-outline`
- report support in `appledocs` that keeps native parity separate from the
  current curated/runtime story

This note is not permission to chase raw SwiftUI `Table` or `OutlineGroup`
symbol parity.

## Reopen Criteria

Reopen this note only if a real desktop scenario fails all three of these:

1. the curated model APIs,
2. the additive native-backed model APIs, and
3. the existing flagship examples as a product template.

The failing scenario must be written down before code starts. Good examples:

- a concrete multi-select workflow that the current models cannot express
- a column behavior that matters in a real desktop tool and cannot be modeled
  as explicit state
- a disclosure or reveal flow that requires deeper native behavior than the
  current additive layer exposes

Bad examples:

- a missing SwiftUI symbol with no product case
- a request to claim native parity without changing a real workflow

## Bounded Execution Order If Reopened

If reopened, do only this work:

1. Add one failing scenario to `examples/table-outline` or
   `examples/native-table-outline`.
2. Land one concrete model or runtime addition that fixes that scenario.
3. Add behavioral tests for the added state or behavior.
4. Update `appledocs` report language so the remaining gap stays honest.

Do not start a second native data-surface expansion until the first one is
proved by the example and tests.

## Do Not Do

Do not reopen this track for:

- raw SwiftUI generic `Table` / `TableColumn` mirroring
- automatic Go struct reflection as a public table model
- parity claims justified only by coverage counts
- additive API that is not exercised by the flagship examples

## Done Bar If Reopened

Call a reopened data-surface pass complete when:

1. the named failing workflow works in a flagship example,
2. the added API stays explicit about identity, selection, sort, expansion, or
   column state,
3. tests cover the new behavior, and
4. `appledocs` still reports true native parity as separate from the shipped
   curated/runtime surface.

## Validation

For any reopened work:

```sh
GOWORK=off go test .
GOWORK=off go build ./examples/table-outline ./examples/native-table-outline
(cd internal/swift && swift build -c release --quiet --product SwiftUIBridge)
```
