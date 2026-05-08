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
	if _SUIPhotosPicker == nil {
		t.Fatal("SUIPhotosPicker not registered")
	}
	if _SUIOpenPanel == nil {
		t.Fatal("SUIOpenPanel not registered")
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

func TestUnmarshalPhotosPickerItems(t *testing.T) {
	got := unmarshalPhotosPickerItems(`[{"id":"b","filename":"b.heic","utType":"public.heic","mediaKind":"image","order":1},{"id":"a","filename":"clip.mov","utType":"public.movie","order":0},{"id":"","filename":"skip","utType":"public.jpeg"}]`)
	want := []PhotosPickerItem{
		{ID: "a", Filename: "clip.mov", UTType: "public.movie", MediaKind: "video", Order: 0},
		{ID: "b", Filename: "b.heic", UTType: "public.heic", MediaKind: "image", Order: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unmarshalPhotosPickerItems() = %#v, want %#v", got, want)
	}
}
