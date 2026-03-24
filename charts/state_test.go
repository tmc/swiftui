package charts

import (
	"testing"
	"time"

	"github.com/tmc/swiftui"
)

type optionalNumberChange struct {
	value float64
	ok    bool
}

func waitOptionalNumberChange(t *testing.T, ch <-chan optionalNumberChange) optionalNumberChange {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for state change")
		return optionalNumberChange{}
	}
}

func TestTypedNilSelectionBindingsAreOmitted(t *testing.T) {
	x := NewOptionalNumberState(0, false)
	defer x.Release()

	var y *OptionalNumberState

	spec := Chart(
		PointMark(XFloat("Step", 1), YFloat("Score", 0.5)),
	).ChartXSelection(x).
		ChartYSelection(y).
		ChartOverlay(CrosshairOverlay(x, y, swiftui.RGB(0.2, 0.4, 0.6), 1)).
		ChartOverlay(ReadoutOverlay(x, y, LabelAlignmentTopLeading, FixedFormat(1), FixedFormat(1))).
		builder.toSpec()

	if spec.YSelection != nil {
		t.Fatalf("y selection = %#v, want nil", spec.YSelection)
	}
	if got, want := len(spec.Overlays), 2; got != want {
		t.Fatalf("overlays = %d, want %d", got, want)
	}
	for i, overlay := range spec.Overlays {
		if overlay.YState != nil {
			t.Fatalf("overlay %d y state = %#v, want nil", i, overlay.YState)
		}
	}
}

func TestZZZOptionalNumberStateOnChange(t *testing.T) {
	state := NewOptionalNumberState(0, false)
	defer state.Release()

	changes := make(chan optionalNumberChange, 4)
	cancel := state.OnChange(func(value float64, ok bool) {
		changes <- optionalNumberChange{value: value, ok: ok}
	})
	defer cancel()

	state.Set(3.5)
	if got := waitOptionalNumberChange(t, changes); got != (optionalNumberChange{value: 3.5, ok: true}) {
		t.Fatalf("first change = %#v, want value 3.5 present", got)
	}

	state.Clear()
	if got := waitOptionalNumberChange(t, changes); got != (optionalNumberChange{value: 0, ok: false}) {
		t.Fatalf("second change = %#v, want cleared value", got)
	}

	cancel()
	state.Set(7.25)
	select {
	case got := <-changes:
		t.Fatalf("change after cancel = %#v, want none", got)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestOptionalNumberStateGetUsesMirroredSnapshotOffThread(t *testing.T) {
	state := NewOptionalNumberState(0, false)
	defer state.Release()

	stateChangeCallbackTrampoline(state.ptr, stateKindOptionalNumber, 9.25, 0, 1)

	gotCh := make(chan optionalNumberChange, 1)
	go func() {
		value, ok := state.Get()
		gotCh <- optionalNumberChange{value: value, ok: ok}
	}()

	select {
	case got := <-gotCh:
		if got != (optionalNumberChange{value: 9.25, ok: true}) {
			t.Fatalf("background Get = %#v, want mirrored bridge snapshot", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for background Get")
	}
}
