package swiftui

import (
	"errors"
	"testing"
)

func TestSceneActions(t *testing.T) {
	t.Run("availability", func(t *testing.T) {
		var actions SceneActions
		if actions.Available() {
			t.Fatal("zero SceneActions should not be available")
		}
	})

	t.Run("open window", func(t *testing.T) {
		var got string
		actions := SceneActions{
			Window: OpenWindowAction(func(id string) error {
				got = id
				return nil
			}),
		}
		if err := actions.OpenWindow("inspector"); err != nil {
			t.Fatalf("OpenWindow() error = %v", err)
		}
		if got != "inspector" {
			t.Fatalf("OpenWindow() id = %q, want %q", got, "inspector")
		}
	})

	t.Run("open document unavailable", func(t *testing.T) {
		var actions SceneActions
		if err := actions.OpenDocument("/tmp/doc.txt"); !errors.Is(err, ErrActionUnavailable) {
			t.Fatalf("OpenDocument() error = %v, want %v", err, ErrActionUnavailable)
		}
	})

	t.Run("refresh", func(t *testing.T) {
		called := false
		actions := SceneActions{
			RefreshScene: RefreshAction(func() error {
				called = true
				return nil
			}),
		}
		if err := actions.Refresh(); err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}
		if !called {
			t.Fatal("Refresh() did not call the action")
		}
	})

	t.Run("immersive space", func(t *testing.T) {
		var got string
		actions := SceneActions{
			ImmersiveSpace: OpenImmersiveSpaceAction(func(id string) error {
				got = id
				return nil
			}),
		}
		if err := actions.OpenImmersiveSpace("space.main"); err != nil {
			t.Fatalf("OpenImmersiveSpace() error = %v", err)
		}
		if got != "space.main" {
			t.Fatalf("OpenImmersiveSpace() id = %q, want %q", got, "space.main")
		}
	})
}
