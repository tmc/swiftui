package swiftui

import "errors"

// ErrActionUnavailable reports that a borrowed environment action is not bound.
var ErrActionUnavailable = errors.New("swiftui: action unavailable")

var (
	errEmptyWindowID         = errors.New("swiftui: empty window id")
	errEmptyDocumentPath     = errors.New("swiftui: empty document path")
	errEmptyImmersiveSpaceID = errors.New("swiftui: empty immersive space id")
)

// OpenWindowAction opens a scene or window provided by the SwiftUI environment.
//
// The zero value is not usable. These handles are borrowed capabilities, not
// owned state, so they do not require Release.
type OpenWindowAction func(id string) error

// Open opens the named window or scene.
func (a OpenWindowAction) Open(id string) error {
	if a == nil {
		return ErrActionUnavailable
	}
	if id == "" {
		return errEmptyWindowID
	}
	return a(id)
}

// OpenDocumentAction opens a document provided by the SwiftUI environment.
//
// The zero value is not usable. These handles are borrowed capabilities, not
// owned state, so they do not require Release.
type OpenDocumentAction func(path string) error

// Open opens the document at path.
func (a OpenDocumentAction) Open(path string) error {
	if a == nil {
		return ErrActionUnavailable
	}
	if path == "" {
		return errEmptyDocumentPath
	}
	return a(path)
}

// RefreshAction refreshes the current scene.
//
// The zero value is not usable. These handles are borrowed capabilities, not
// owned state, so they do not require Release.
type RefreshAction func() error

// Refresh asks SwiftUI to refresh the current scene.
func (a RefreshAction) Refresh() error {
	if a == nil {
		return ErrActionUnavailable
	}
	return a()
}

// OpenImmersiveSpaceAction opens a named immersive space.
//
// The zero value is not usable. These handles are borrowed capabilities, not
// owned state, so they do not require Release.
type OpenImmersiveSpaceAction func(id string) error

// Open opens the named immersive space.
func (a OpenImmersiveSpaceAction) Open(id string) error {
	if a == nil {
		return ErrActionUnavailable
	}
	if id == "" {
		return errEmptyImmersiveSpaceID
	}
	return a(id)
}
