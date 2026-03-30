package swiftui

import (
	"errors"
	"testing"
)

func TestOpenWindowAction(t *testing.T) {
	t.Run("unavailable", func(t *testing.T) {
		var action OpenWindowAction
		if err := action.Open("inspector"); !errors.Is(err, ErrActionUnavailable) {
			t.Fatalf("Open() error = %v, want %v", err, ErrActionUnavailable)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		action := OpenWindowAction(func(id string) error { return nil })
		if err := action.Open(""); !errors.Is(err, errEmptyWindowID) {
			t.Fatalf("Open() error = %v, want %v", err, errEmptyWindowID)
		}
	})

	t.Run("calls function", func(t *testing.T) {
		var got string
		action := OpenWindowAction(func(id string) error {
			got = id
			return nil
		})
		if err := action.Open("inspector"); err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		if got != "inspector" {
			t.Fatalf("Open() id = %q, want %q", got, "inspector")
		}
	})
}

func TestOpenDocumentAction(t *testing.T) {
	t.Run("unavailable", func(t *testing.T) {
		var action OpenDocumentAction
		if err := action.Open("/tmp/doc.txt"); !errors.Is(err, ErrActionUnavailable) {
			t.Fatalf("Open() error = %v, want %v", err, ErrActionUnavailable)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		action := OpenDocumentAction(func(path string) error { return nil })
		if err := action.Open(""); !errors.Is(err, errEmptyDocumentPath) {
			t.Fatalf("Open() error = %v, want %v", err, errEmptyDocumentPath)
		}
	})

	t.Run("calls function", func(t *testing.T) {
		var got string
		action := OpenDocumentAction(func(path string) error {
			got = path
			return nil
		})
		if err := action.Open("/tmp/doc.txt"); err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		if got != "/tmp/doc.txt" {
			t.Fatalf("Open() path = %q, want %q", got, "/tmp/doc.txt")
		}
	})
}

func TestRefreshAction(t *testing.T) {
	t.Run("unavailable", func(t *testing.T) {
		var action RefreshAction
		if err := action.Refresh(); !errors.Is(err, ErrActionUnavailable) {
			t.Fatalf("Refresh() error = %v, want %v", err, ErrActionUnavailable)
		}
	})

	t.Run("calls function", func(t *testing.T) {
		called := false
		action := RefreshAction(func() error {
			called = true
			return nil
		})
		if err := action.Refresh(); err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}
		if !called {
			t.Fatal("Refresh() did not call the action")
		}
	})
}

func TestOpenImmersiveSpaceAction(t *testing.T) {
	t.Run("unavailable", func(t *testing.T) {
		var action OpenImmersiveSpaceAction
		if err := action.Open("space.main"); !errors.Is(err, ErrActionUnavailable) {
			t.Fatalf("Open() error = %v, want %v", err, ErrActionUnavailable)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		action := OpenImmersiveSpaceAction(func(id string) error { return nil })
		if err := action.Open(""); !errors.Is(err, errEmptyImmersiveSpaceID) {
			t.Fatalf("Open() error = %v, want %v", err, errEmptyImmersiveSpaceID)
		}
	})

	t.Run("calls function", func(t *testing.T) {
		var got string
		action := OpenImmersiveSpaceAction(func(id string) error {
			got = id
			return nil
		})
		if err := action.Open("space.main"); err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		if got != "space.main" {
			t.Fatalf("Open() id = %q, want %q", got, "space.main")
		}
	})
}
