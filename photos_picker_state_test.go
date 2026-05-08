package swiftui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPhotosPickerSelectionState(t *testing.T) {
	state := NewPhotosPickerSelectionState(
		PhotosPickerItem{ID: "b", Filename: "b.jpg", UTType: "public.jpeg"},
		PhotosPickerItem{ID: "a", Filename: "a.png", UTType: "public.png"},
		PhotosPickerItem{Filename: "skip.jpg"},
	)
	defer state.Release()

	if got, want := state.Count(), 2; got != want {
		t.Fatalf("Count() = %d, want %d", got, want)
	}
	if got, want := state.Items(), []PhotosPickerItem{
		{ID: "a", Filename: "a.png", UTType: "public.png", MediaKind: "image"},
		{ID: "b", Filename: "b.jpg", UTType: "public.jpeg", MediaKind: "image"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Items() = %#v, want %#v", got, want)
	}

	state.Add(PhotosPickerItem{ID: "c", Filename: "c.heic", UTType: "public.heic"})
	if !state.Has("c") {
		t.Fatal("expected c to be selected")
	}

	state.Remove("b")
	if state.Has("b") {
		t.Fatal("expected b to be removed")
	}

	state.Clear()
	if got := state.Count(); got != 0 {
		t.Fatalf("Count() after Clear = %d, want 0", got)
	}
}

func TestPhotosPickerSelectionStateReplaceByID(t *testing.T) {
	state := NewPhotosPickerSelectionState(PhotosPickerItem{ID: "poster", Filename: "poster.png", UTType: "public.png"})
	defer state.Release()

	state.Add(PhotosPickerItem{ID: "poster", Filename: "poster.heic", UTType: "public.heic"})
	items := state.Items()
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(Items()) = %d, want %d", got, want)
	}
	if got, want := items[0].Filename, "poster.heic"; got != want {
		t.Fatalf("Items()[0].Filename = %q, want %q", got, want)
	}
	if got, want := items[0].MediaKind, "image"; got != want {
		t.Fatalf("Items()[0].MediaKind = %q, want %q", got, want)
	}
	if revState := state.RevisionState(); revState != nil && revState.Get() == 0 {
		t.Fatal("RevisionState() should advance")
	}
	if countState := state.CountState(); countState != nil && countState.Get() != 1 {
		t.Fatalf("CountState() = %d, want 1", countState.Get())
	}
}

func TestPhotosPickerSelectionStateRevisionAndCounts(t *testing.T) {
	state := NewPhotosPickerSelectionState()
	defer state.Release()

	if got, want := state.Revision(), 1; got != want {
		t.Fatalf("Revision() after init = %d, want %d", got, want)
	}
	if got, want := state.Count(), 0; got != want {
		t.Fatalf("Count() after init = %d, want %d", got, want)
	}

	state.Add(PhotosPickerItem{ID: "poster", Filename: "poster.png", UTType: "public.png"})
	if got, want := state.Revision(), 2; got != want {
		t.Fatalf("Revision() after add = %d, want %d", got, want)
	}
	if got, want := state.Count(), 1; got != want {
		t.Fatalf("Count() after add = %d, want %d", got, want)
	}

	state.Remove("poster")
	if got, want := state.Revision(), 3; got != want {
		t.Fatalf("Revision() after remove = %d, want %d", got, want)
	}
	if got, want := state.Count(), 0; got != want {
		t.Fatalf("Count() after remove = %d, want %d", got, want)
	}

	state.Clear()
	if got, want := state.Revision(), 3; got != want {
		t.Fatalf("Revision() after clear with no items = %d, want %d", got, want)
	}
	if got, want := state.Count(), 0; got != want {
		t.Fatalf("Count() after clear = %d, want %d", got, want)
	}
}

func TestPhotosPickerSelectionStateToggle(t *testing.T) {
	state := NewPhotosPickerSelectionState()
	defer state.Release()

	item := PhotosPickerItem{ID: "poster", Filename: "poster.png", UTType: "public.png"}
	state.Toggle(item)
	if !state.Has("poster") {
		t.Fatal("Toggle() should add poster")
	}
	if got, want := state.Count(), 1; got != want {
		t.Fatalf("Count() after add toggle = %d, want %d", got, want)
	}

	state.Toggle(item)
	if state.Has("poster") {
		t.Fatal("Toggle() should remove poster")
	}
	if got, want := state.Count(), 0; got != want {
		t.Fatalf("Count() after remove toggle = %d, want %d", got, want)
	}
}

func TestPhotosPickerSelectionStatePreservesOrder(t *testing.T) {
	state := NewPhotosPickerSelectionState(
		PhotosPickerItem{ID: "clip", Filename: "clip.mov", UTType: "public.movie", Order: 1},
		PhotosPickerItem{ID: "poster", Filename: "poster.png", UTType: "public.png", Order: 0},
	)
	defer state.Release()

	got := state.Items()
	want := []PhotosPickerItem{
		{ID: "poster", Filename: "poster.png", UTType: "public.png", MediaKind: "image", Order: 0},
		{ID: "clip", Filename: "clip.mov", UTType: "public.movie", MediaKind: "video", Order: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Items() = %#v, want %#v", got, want)
	}
}

func TestPhotosPickerLazyFileHandleLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "asset.txt")
	wantData := []byte("lazy asset payload")
	if err := os.WriteFile(path, wantData, 0o600); err != nil {
		t.Fatalf("WriteFile(%s) = %v", path, err)
	}

	handle := NewPhotosPickerLazyFileHandle(path)
	if handle == nil {
		t.Fatal("NewPhotosPickerLazyFileHandle() = nil, want handle")
	}
	gotData, err := handle.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if !reflect.DeepEqual(gotData, wantData) {
		t.Fatalf("Load() = %q, want %q", gotData, wantData)
	}
}

func TestPhotosPickerSelectionStatePreservesLazyFileHandle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "poster.txt")
	if err := os.WriteFile(path, []byte("poster payload"), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) = %v", path, err)
	}

	state := NewPhotosPickerSelectionState(
		PhotosPickerItem{
			ID:        "poster",
			Filename:  "poster.txt",
			UTType:    "public.plain-text",
			LazyFile:  NewPhotosPickerLazyFileHandle(path),
			MediaKind: "image",
		},
	)
	defer state.Release()

	items := state.Items()
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(Items()) = %d, want %d", got, want)
	}
	if items[0].LazyFile == nil {
		t.Fatal("Items()[0].LazyFile = nil, want lazy file handle")
	}
	gotData, err := items[0].LazyFile.Load()
	if err != nil {
		t.Fatalf("Items()[0].LazyFile.Load() = %v", err)
	}
	if got, want := string(gotData), "poster payload"; got != want {
		t.Fatalf("Items()[0].LazyFile.Load() = %q, want %q", got, want)
	}
}

func TestPhotosPickerMenuNilSelection(t *testing.T) {
	view := PhotosPickerMenu("Choose", nil, PhotosPickerItem{ID: "poster", Filename: "poster.png", UTType: "public.png"})
	if view.ptr == 0 {
		t.Fatal("PhotosPickerMenu(nil) should still return a fallback view")
	}
	view.Release()
}
