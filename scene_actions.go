package swiftui

// SceneActions bundles scene-scoped borrowed capabilities.
//
// Runtime surface.
//
// The zero value is not usable because each action is optional and may be
// unavailable until SwiftUI injects the environment capability.
type SceneActions struct {
	Window         OpenWindowAction
	Document       OpenDocumentAction
	RefreshScene   RefreshAction
	ImmersiveSpace OpenImmersiveSpaceAction
}

// OpenWindow opens a named scene or window.
func (a SceneActions) OpenWindow(id string) error {
	return a.Window.Open(id)
}

// OpenDocument opens a document path.
func (a SceneActions) OpenDocument(path string) error {
	return a.Document.Open(path)
}

// Refresh asks SwiftUI to refresh the current scene.
func (a SceneActions) Refresh() error {
	return a.RefreshScene.Refresh()
}

// OpenImmersiveSpace opens a named immersive space.
func (a SceneActions) OpenImmersiveSpace(id string) error {
	return a.ImmersiveSpace.Open(id)
}

// Available reports whether any borrowed capability is bound.
func (a SceneActions) Available() bool {
	return a.Window != nil || a.Document != nil || a.RefreshScene != nil || a.ImmersiveSpace != nil
}
