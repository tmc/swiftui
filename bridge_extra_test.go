package swiftui

import (
	"reflect"
	"testing"
)

func TestExtraBridgeRegistersA2UISymbols(t *testing.T) {
	if loadErr != nil {
		t.Fatalf("load bridge: %v", loadErr)
	}
	if libHandle == 0 {
		t.Fatal("bridge dylib not loaded")
	}
	ensureExtraLibFuncs()
	if _SUIAsyncImageFit == nil {
		t.Fatal("SUIAsyncImageFit not registered")
	}
	if _SUIDatePickerBounded == nil {
		t.Fatal("SUIDatePickerBounded not registered")
	}
	if _SUIDatePickerBoundedMode == nil {
		t.Fatal("SUIDatePickerBoundedMode not registered")
	}
	if _SUITextFieldPolicy == nil {
		t.Fatal("SUITextFieldPolicy not registered")
	}
	if _SUIStateCreateStringList == nil {
		t.Fatal("SUIStateCreateStringList not registered")
	}
	if _SUISearchableMultiPicker == nil {
		t.Fatal("SUISearchableMultiPicker not registered")
	}
	if _SUIViewMaxFrameAligned == nil {
		t.Fatal("SUIViewMaxFrameAligned not registered")
	}
}

func TestStringListStateRoundTrip(t *testing.T) {
	state := NewStringListState([]string{"alpha", "beta"})
	defer state.Release()

	if got := state.Get(); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("Get() = %v, want [alpha beta]", got)
	}

	state.Set([]string{"beta", "gamma"})
	if got := state.Get(); !reflect.DeepEqual(got, []string{"beta", "gamma"}) {
		t.Fatalf("Get() after Set = %v, want [beta gamma]", got)
	}
}
