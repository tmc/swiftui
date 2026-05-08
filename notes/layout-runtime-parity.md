## Layout Runtime Parity Path

Status: closed for the current macOS-runtime push.

The current layout runtime already shipped the intended v2 bar:

- `LayoutSpec` and `AnyLayout` as the stable entry point
- `CustomLayout(...)` over concrete Go-native models
- tagged child metadata
- fixed-key placement metadata
- placement-aware layout helpers
- flagship proof in `examples/layout`

This note is no longer an open-ended parity plan. There is no active layout
track until a concrete product case fails the shipped surface.

### Reopen Only If One Of These Fails

Reopen layout work only if all of the following are true:

1. the failing UI cannot be expressed with the current tagged or placement
   models,
2. the limitation shows up in a real example or product surface, not only in a
   symbol diff, and
3. the proposed fix can stay concrete and Go-native.

### Allowed Follow-On Work

If the note is reopened, the work must stay bounded to this sequence:

1. Write down the failing layout case in terms of one flagship example or one
   product surface.
2. Add one read-only measurement or placement-summary model that solves that
   case without exposing SwiftUI protocol conformance.
3. Update `examples/layout`, add table-driven tests, and tighten the docs to
   describe the new limit precisely.

### Do Not Do

Do not reopen layout work for:

- raw SwiftUI `Layout` protocol mirroring
- `LayoutValueKey` parity
- open-ended child placement callbacks
- cache hooks shaped like SwiftUI protocol methods
- parity claims justified only by symbol count

### Done Bar If Reopened

Call the reopened layout track complete when:

1. the named failing layout is covered by one concrete model,
2. `examples/layout` proves the new behavior,
3. behavioral tests cover normalization and resolution rules, and
4. the public API still reads as a concrete Go model instead of a Swift
   protocol mirror.

### Validation

For any reopened layout work:

```sh
GOWORK=off go test .
GOWORK=off go build ./examples/layout
(cd internal/swift && swift build -c release --quiet --product SwiftUIBridge)
```
