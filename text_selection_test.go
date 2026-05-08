package swiftui

import "testing"

func TestTextSelectionStateRoundTrip(t *testing.T) {
	state := NewTextSelectionState(8, 2)
	defer state.Release()

	if got := state.Get(); got != (TextSelection{Start: 2, End: 8}) {
		t.Fatalf("Get() = %#v, want %#v", got, TextSelection{Start: 2, End: 8})
	}

	state.SetRange(4, 9)
	if got := state.Get(); got != (TextSelection{Start: 4, End: 9}) {
		t.Fatalf("Get() after SetRange = %#v, want %#v", got, TextSelection{Start: 4, End: 9})
	}

	state.Collapse(3)
	if got := state.Get(); got != (TextSelection{Start: 3, End: 3}) {
		t.Fatalf("Get() after Collapse = %#v, want %#v", got, TextSelection{Start: 3, End: 3})
	}
}

func TestTextSelectionHelpers(t *testing.T) {
	if got := (TextSelection{Start: 4, End: 9}).Length(); got != 5 {
		t.Fatalf("Length() = %d, want 5", got)
	}
	if !(TextSelection{Start: 7, End: 7}).Caret() {
		t.Fatal("Caret() = false, want true")
	}
}
