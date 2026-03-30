package swiftui

import (
	"reflect"
	"testing"
)

func TestNavigationPathState(t *testing.T) {
	state := NewNavigationPathStateWith(" inbox ", "", "thread")

	got := state.Get()
	want := []string{"inbox", "thread"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("initial path = %v, want %v", got, want)
	}

	got[0] = "mutated"
	if again := state.Get(); !reflect.DeepEqual(again, want) {
		t.Fatalf("path copy leaked mutation: got %v, want %v", again, want)
	}

	state.Push(" detail ")
	if got, want := state.Depth(), 3; got != want {
		t.Fatalf("depth = %d, want %d", got, want)
	}
	if got, want := state.String(), "inbox/thread/detail"; got != want {
		t.Fatalf("string = %q, want %q", got, want)
	}
	if got, ok := state.Current(); !ok || got != "detail" {
		t.Fatalf("current = %q, %v, want detail, true", got, ok)
	}
	if !state.Pop() {
		t.Fatalf("pop returned false")
	}
	if got, ok := state.Current(); !ok || got != "thread" {
		t.Fatalf("current after pop = %q, %v, want thread, true", got, ok)
	}

	state.Clear()
	if got, want := state.Depth(), 0; got != want {
		t.Fatalf("depth after clear = %d, want %d", got, want)
	}
	if got, ok := state.Current(); ok || got != "" {
		t.Fatalf("current after clear = %q, %v, want empty, false", got, ok)
	}
	if state.Pop() {
		t.Fatalf("pop on empty path returned true")
	}
	if got := state.Revision(); got == 0 {
		t.Fatalf("revision = %d, want non-zero after mutations", got)
	}
	if revState := state.RevisionState(); revState != nil && revState.Get() != state.Revision() {
		t.Fatalf("revision state = %d, want %d", revState.Get(), state.Revision())
	}
}

func TestCompactColumnState(t *testing.T) {
	state := NewCompactColumnState(NavigationSplitViewColumnContent)
	t.Cleanup(state.Release)

	if got, want := state.Get(), NavigationSplitViewColumnContent; got != want {
		t.Fatalf("get = %v, want %v", got, want)
	}
	if got, want := state.Visibility(), NavigationSplitViewVisibilityDoubleColumn; got != want {
		t.Fatalf("visibility = %v, want %v", got, want)
	}
	if got, want := state.VisibilityState().Get(), int(NavigationSplitViewVisibilityDoubleColumn); got != want {
		t.Fatalf("visibility state = %d, want %d", got, want)
	}

	state.Set(NavigationSplitViewColumnSidebar)
	if got, want := state.Get(), NavigationSplitViewColumnSidebar; got != want {
		t.Fatalf("sidebar get = %v, want %v", got, want)
	}
	if got, want := state.Visibility(), NavigationSplitViewVisibilityAll; got != want {
		t.Fatalf("sidebar visibility = %v, want %v", got, want)
	}

	state.Set(NavigationSplitViewColumnDetail)
	if got, want := state.Get(), NavigationSplitViewColumnDetail; got != want {
		t.Fatalf("detail get = %v, want %v", got, want)
	}
	if got, want := state.Visibility(), NavigationSplitViewVisibilityDetailOnly; got != want {
		t.Fatalf("detail visibility = %v, want %v", got, want)
	}

	state.Set(NavigationSplitViewColumnAutomatic)
	if got, want := state.Get(), NavigationSplitViewColumnAutomatic; got != want {
		t.Fatalf("automatic get = %v, want %v", got, want)
	}
	if got, want := state.Visibility(), NavigationSplitViewVisibilityAutomatic; got != want {
		t.Fatalf("automatic visibility = %v, want %v", got, want)
	}
}
