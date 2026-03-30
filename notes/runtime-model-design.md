# Runtime Model Design

This note captures the next runtime-backed SwiftUI gaps to close in `swiftui`.
It is intentionally separate from the `appledocs` report work. The goal is to
add a small set of owned state types and borrowed environment actions that
unlock the current example surface without widening the curated binding catalog.

The design is implementation-oriented:

- keep the current generated view surface
- add a small number of hand-written runtime model types in `swiftui`
- make the ownership rules explicit
- prefer narrow, composable types over a single large catch-all model

## Scope

The first tranche covers:

- path-based `NavigationStack`
- compact-column state for `NavigationSplitView`
- multi-date selection for `MultiDatePicker`
- date-range selection for planner and filter flows
- table row, selection, and sort state
- environment action handles for window, document, refresh, and immersive-space flows
- timer / countdown state for status and progress views

These are the remaining gaps that can be modeled cleanly in Go without asking
the generated binding catalog to mirror all of SwiftUI.

## Proposed Public Types

### Navigation Path

`NavigationPathState` should own a stack of stable path tokens for
`NavigationStack(path:)`.

Suggested shape:

```go
type NavigationPathState struct {
	// opaque bridge handle
}

func NewNavigationPathState() *NavigationPathState
func NewNavigationPathStateWith(path ...string) *NavigationPathState

func (s *NavigationPathState) Get() []string
func (s *NavigationPathState) Set(path []string)
func (s *NavigationPathState) Push(segment string)
func (s *NavigationPathState) Pop() bool
func (s *NavigationPathState) Clear()
func (s *NavigationPathState) Release()
```

Model it as a slice of stable string tokens in v1. That is enough for router
flows in the examples and keeps the bridge serializable. If a future use case
needs typed payloads, add an encoder layer later rather than baking generics
into the first version.

### Compact Column

`CompactColumnState` should represent the preferred compact column for
`NavigationSplitView`.

Suggested enum:

```go
type NavigationSplitViewColumnKind int32

const (
	NavigationSplitViewColumnAutomatic NavigationSplitViewColumnKind = 0
	NavigationSplitViewColumnSidebar   NavigationSplitViewColumnKind = 1
	NavigationSplitViewColumnContent   NavigationSplitViewColumnKind = 2
	NavigationSplitViewColumnDetail    NavigationSplitViewColumnKind = 3
)
```

Suggested state wrapper:

```go
type CompactColumnState struct {
	// opaque bridge handle
}

func NewCompactColumnState(initial NavigationSplitViewColumnKind) *CompactColumnState
func (s *CompactColumnState) Get() NavigationSplitViewColumnKind
func (s *CompactColumnState) Set(v NavigationSplitViewColumnKind)
func (s *CompactColumnState) Release()
```

This keeps the current split-view shell explicit instead of encoding compact
column choice in ad hoc route state.

### Date Selection

`DateSelectionState` should back `MultiDatePicker` and any other multi-day
selection surface.

Suggested shape:

```go
type DateSelectionState struct {
	// opaque bridge handle
}

func NewDateSelectionState(initial ...time.Time) *DateSelectionState

func (s *DateSelectionState) Get() []time.Time
func (s *DateSelectionState) Set(dates []time.Time)
func (s *DateSelectionState) Add(date time.Time)
func (s *DateSelectionState) Remove(date time.Time)
func (s *DateSelectionState) Clear()
func (s *DateSelectionState) Release()
```

The state should normalize dates to a canonical day key before crossing the
bridge. The current charts package already uses date-selection and date-range
state, so the implementation should reuse the same normalization rules where
possible.

### Date Range

`DateRangeState` should cover planner-like intervals and filter windows.

Suggested shape:

```go
type DateRangeState struct {
	// opaque bridge handle
}

func NewDateRangeState(start, end time.Time, ok bool) *DateRangeState

func (s *DateRangeState) Get() (time.Time, time.Time, bool)
func (s *DateRangeState) Set(start, end time.Time)
func (s *DateRangeState) Clear()
func (s *DateRangeState) Release()
```

This mirrors the already-established `charts.DateRangeState` semantics, which
keeps planner behavior predictable and avoids inventing a second range API.

### Table Model

`TableModel` should own row identity, selection, and sort state for table-style
surfaces.

Use a generic shape if the package is willing to follow the same pattern used
elsewhere in the repo:

```go
type TableModel[T any] struct {
	// opaque bridge handle
}

func NewTableModel[T any](rows []T, id func(T) string) *TableModel[T]

func (m *TableModel[T]) Rows() []T
func (m *TableModel[T]) SetRows(rows []T)
func (m *TableModel[T]) SelectedID() (string, bool)
func (m *TableModel[T]) SelectID(id string)
func (m *TableModel[T]) ClearSelection()
func (m *TableModel[T]) Sort(less func(a, b T) bool)
func (m *TableModel[T]) Release()
```

If a non-generic first pass is easier to land, keep the same semantics but
store rows as opaque records with stable string IDs. The key requirement is
that the table state owns row order and selection, rather than pushing that
logic into every example.

### Environment Actions

Environment actions are not owned state. They are borrowed capability handles
that can be invoked while a scene is alive.

Suggested types:

```go
type OpenWindowAction struct{}
type OpenDocumentAction struct{}
type RefreshAction struct{}
type OpenImmersiveSpaceAction struct{}

func (a OpenWindowAction) Open(id string) error
func (a OpenDocumentAction) Open(path string) error
func (a RefreshAction) Refresh() error
func (a OpenImmersiveSpaceAction) Open(id string) error
```

Rules:

- zero value is unusable
- no public constructor; these come from generated environment wiring
- no `Release`, because the scene owns the underlying capability
- calls should marshal through the UI runtime, not expose raw bridge pointers

If the bridge needs a shared abstraction, keep it private and wrap it with the
four public handles above.

### Timer State

`TimerState` should combine the remaining time, running flag, and progress into
a single runtime object instead of splitting those concerns across separate
primitive states.

Suggested shape:

```go
type TimerState struct {
	// opaque bridge handle
}

func NewTimerState(total, remaining time.Duration, running bool) *TimerState

func (s *TimerState) Total() time.Duration
func (s *TimerState) Remaining() time.Duration
func (s *TimerState) Progress() float64
func (s *TimerState) Running() bool
func (s *TimerState) SetRemaining(v time.Duration)
func (s *TimerState) SetRunning(v bool)
func (s *TimerState) Reset()
func (s *TimerState) Release()
```

This gives the timer and pomodoro examples one state owner instead of a
hand-rolled trio of `IntState`, `FloatState`, and `BoolState`.

## Zero Value and Constructor Story

State-bearing types should follow the same rule as the existing `IntState`,
`BoolState`, and `DateState` types:

- the zero value is not usable
- constructors allocate the Swift-side bridge object
- `Release` is required when the state is no longer needed
- copying the Go value should not duplicate ownership

That keeps lifecycle behavior explicit and avoids accidental retain leaks.

The exception is the environment action family, which is borrowed rather than
owned:

- zero value is still not useful
- there is no public constructor
- there is no `Release`
- the enclosing scene or environment owns the capability

## Ownership and Lifecycle Rules

The bridge should treat these families differently:

1. Owned state objects (`NavigationPathState`, `CompactColumnState`,
   `DateSelectionState`, `DateRangeState`, `TableModel`, `TimerState`) retain
   the Swift object and must be released explicitly.
2. Borrowed action handles (`OpenWindowAction`, `OpenDocumentAction`,
   `RefreshAction`, `OpenImmersiveSpaceAction`) are scene-scoped capabilities.
   They should be passed through and invoked, but not retained as long-lived
   application state.
3. All state mutations should stay main-thread safe, because they feed SwiftUI
   view updates.
4. State should own canonicalization. Callers should pass plain Go values and
   not need to know how the bridge encodes path tokens, day keys, or row IDs.

## Which Examples This Unlocks

### Already in the tree

- `examples/workbench`
  - `NavigationPathState` unlocks the router path pattern instead of the current
    route-id-only fallback.
  - `CompactColumnState` gives the split-view shell an explicit compact-column
    model.
  - `DateSelectionState` and `DateRangeState` make the planner pane a real
    calendar model instead of bitmask/date shims.
  - `TimerState` collapses the current remaining/running/progress trio into one
    timer object.
  - `TableModel` is the right foundation for the data-grid surface.

- `examples/codex-clone`
  - `NavigationPathState` supports thread/workspace routing if the shell grows
    to a deeper navigation stack.
  - `OpenWindowAction` and `OpenDocumentAction` are the right fit for future
    workspace/window management.
  - `RefreshAction` unlocks a cleaner refresh story for thread or diff updates.

- `examples/timer`
  - `TimerState` replaces hand-managed countdown progress.

- `examples/pomodoro`
  - `TimerState` is the natural model for countdown, pause, and progress.

### Still to add or expand

- a multi-date planner example
- a multi-window document example
- a spreadsheet-style table example with selection and sort
- a share/refresh/immersive demo that exercises the environment action family

## Suggested Implementation Order

1. Navigation path and compact column state.
2. Date selection and date range state.
3. Timer state.
4. Table model.
5. Environment action handles.

That order gives the largest immediate example coverage with the smallest
surface area, and it keeps the implementation close to the existing `State`
pattern.
