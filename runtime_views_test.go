package swiftui

import (
	"reflect"
	"testing"
)

func TestNilSafePath(t *testing.T) {
	if got := nilSafePath(nil); got != nil {
		t.Fatalf("nilSafePath(nil) = %v, want nil", got)
	}

	path := NewNavigationPathStateWith(" inbox ", "", "thread")
	got := nilSafePath(path)
	want := []string{"inbox", "thread"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nilSafePath(path) = %v, want %v", got, want)
	}

	got[0] = "mutated"
	if again := nilSafePath(path); !reflect.DeepEqual(again, want) {
		t.Fatalf("nilSafePath leaked mutation: got %v, want %v", again, want)
	}
}

func TestNavigationStackPathNilBuilder(t *testing.T) {
	_ = NavigationStackPath(NewNavigationPathStateWith("thread"), nil)
}

func TestNavigationSplitViewPreferredCompactColumnNilState(t *testing.T) {
	sidebar := Text("Sidebar").AsView()
	detail := Text("Detail").AsView()
	content := Text("Content").AsView()

	oldSplit := _SUINavigationSplitView
	oldSplitTriple := _SUINavigationSplitViewTriple
	oldSplitVisibility := _SUINavigationSplitViewVisibility
	oldSplitTripleVisibility := _SUINavigationSplitViewTripleVisibility
	t.Cleanup(func() {
		_SUINavigationSplitView = oldSplit
		_SUINavigationSplitViewTriple = oldSplitTriple
		_SUINavigationSplitViewVisibility = oldSplitVisibility
		_SUINavigationSplitViewTripleVisibility = oldSplitTripleVisibility
	})

	var splitCalled, splitTripleCalled bool
	_SUINavigationSplitView = func(sidebarPtr, detailPtr uintptr) uintptr {
		splitCalled = true
		return sidebarPtr ^ detailPtr
	}
	_SUINavigationSplitViewTriple = func(sidebarPtr, contentPtr, detailPtr uintptr) uintptr {
		splitTripleCalled = true
		return sidebarPtr ^ contentPtr ^ detailPtr
	}
	_SUINavigationSplitViewVisibility = func(uintptr, uintptr, uintptr) uintptr {
		t.Fatal("NavigationSplitViewVisibility should not be used for nil compact state")
		return 0
	}
	_SUINavigationSplitViewTripleVisibility = func(uintptr, uintptr, uintptr, uintptr) uintptr {
		t.Fatal("NavigationSplitViewTripleVisibility should not be used for nil compact state")
		return 0
	}

	_ = NavigationSplitViewPreferredCompactColumn(nil, sidebar, detail)
	_ = NavigationSplitViewTriplePreferredCompactColumn(nil, sidebar, content, detail)
	if !splitCalled {
		t.Fatal("NavigationSplitView fallback was not called")
	}
	if !splitTripleCalled {
		t.Fatal("NavigationSplitViewTriple fallback was not called")
	}
}
