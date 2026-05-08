package swiftui

import "testing"

func TestViewFocusedUsesBoolStateBinding(t *testing.T) {
	oldFocused := _SUIViewFocused
	t.Cleanup(func() {
		_SUIViewFocused = oldFocused
	})

	var gotView uintptr
	var gotState uintptr
	_SUIViewFocused = func(viewPtr, statePtr uintptr) uintptr {
		gotView = viewPtr
		gotState = statePtr
		return 91
	}

	view := View{ptr: 17}
	state := &BoolState{ptr: 23}
	got := view.Focused(state)
	if got.ptr != 91 {
		t.Fatalf("Focused().ptr = %d, want 91", got.ptr)
	}
	if gotView != 17 {
		t.Fatalf("_SUIViewFocused view ptr = %d, want 17", gotView)
	}
	if gotState != 23 {
		t.Fatalf("_SUIViewFocused state ptr = %d, want 23", gotState)
	}
}

func TestViewAccessibilityValueUsesBridge(t *testing.T) {
	oldAccessibilityValue := _SUIViewAccessibilityValue
	t.Cleanup(func() {
		_SUIViewAccessibilityValue = oldAccessibilityValue
	})

	var gotView uintptr
	var gotValue string
	_SUIViewAccessibilityValue = func(viewPtr uintptr, valueC *byte) uintptr {
		gotView = viewPtr
		gotValue = cString(valueC)
		return 95
	}

	view := View{ptr: 17}
	got := view.AccessibilityValue("Current focus target: search field")
	if got.ptr != 95 {
		t.Fatalf("AccessibilityValue().ptr = %d, want 95", got.ptr)
	}
	if gotView != 17 {
		t.Fatalf("_SUIViewAccessibilityValue view ptr = %d, want 17", gotView)
	}
	if gotValue != "Current focus target: search field" {
		t.Fatalf("_SUIViewAccessibilityValue value = %q, want %q", gotValue, "Current focus target: search field")
	}
}

func TestViewFocusSectionUsesBridge(t *testing.T) {
	oldFocusSection := _SUIViewFocusSection
	t.Cleanup(func() {
		_SUIViewFocusSection = oldFocusSection
	})

	var gotView uintptr
	_SUIViewFocusSection = func(viewPtr uintptr) uintptr {
		gotView = viewPtr
		return 92
	}

	view := View{ptr: 17}
	got := view.FocusSection()
	if got.ptr != 92 {
		t.Fatalf("FocusSection().ptr = %d, want 92", got.ptr)
	}
	if gotView != 17 {
		t.Fatalf("_SUIViewFocusSection view ptr = %d, want 17", gotView)
	}
}

func TestViewFocusScopeIDUsesBridge(t *testing.T) {
	oldFocusScopeID := _SUIViewFocusScopeID
	t.Cleanup(func() {
		_SUIViewFocusScopeID = oldFocusScopeID
	})

	var gotView uintptr
	var gotNamespace uintptr
	_SUIViewFocusScopeID = func(viewPtr, namespacePtr uintptr) uintptr {
		gotView = viewPtr
		gotNamespace = namespacePtr
		return 93
	}

	view := View{ptr: 17}
	namespace := &FocusNamespace{ptr: 29}
	got := view.FocusScopeID(namespace)
	if got.ptr != 93 {
		t.Fatalf("FocusScopeID().ptr = %d, want 93", got.ptr)
	}
	if gotView != 17 {
		t.Fatalf("_SUIViewFocusScopeID view ptr = %d, want 17", gotView)
	}
	if gotNamespace != 29 {
		t.Fatalf("_SUIViewFocusScopeID namespace ptr = %d, want 29", gotNamespace)
	}
}

func TestViewPrefersDefaultFocusUsesBridge(t *testing.T) {
	oldPrefersDefaultFocus := _SUIViewPrefersDefaultFocus
	t.Cleanup(func() {
		_SUIViewPrefersDefaultFocus = oldPrefersDefaultFocus
	})

	var gotView uintptr
	var gotPreferred int32
	var gotNamespace uintptr
	_SUIViewPrefersDefaultFocus = func(viewPtr uintptr, preferred int32, namespacePtr uintptr) uintptr {
		gotView = viewPtr
		gotPreferred = preferred
		gotNamespace = namespacePtr
		return 94
	}

	view := View{ptr: 17}
	namespace := &FocusNamespace{ptr: 31}
	got := view.PrefersDefaultFocus(true, namespace)
	if got.ptr != 94 {
		t.Fatalf("PrefersDefaultFocus().ptr = %d, want 94", got.ptr)
	}
	if gotView != 17 {
		t.Fatalf("_SUIViewPrefersDefaultFocus view ptr = %d, want 17", gotView)
	}
	if gotPreferred != 1 {
		t.Fatalf("_SUIViewPrefersDefaultFocus preferred = %d, want 1", gotPreferred)
	}
	if gotNamespace != 31 {
		t.Fatalf("_SUIViewPrefersDefaultFocus namespace ptr = %d, want 31", gotNamespace)
	}
}

func TestNewFocusNamespaceUsesNamespaceBridge(t *testing.T) {
	oldCreate := _SUINamespaceCreate
	t.Cleanup(func() {
		_SUINamespaceCreate = oldCreate
	})

	_SUINamespaceCreate = func() uintptr { return 77 }

	ns := NewFocusNamespace()
	if ns == nil {
		t.Fatal("NewFocusNamespace() = nil, want namespace handle")
	}
	if ns.ptr != 77 {
		t.Fatalf("NewFocusNamespace().ptr = %d, want 77", ns.ptr)
	}
}
