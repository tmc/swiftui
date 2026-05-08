package swiftui

import (
	"errors"
	"strings"
)

var (
	// ErrDocumentActionCanceled reports that a runner-owned document action was
	// canceled by the user.
	ErrDocumentActionCanceled = errors.New("swiftui: document action canceled")

	// ErrDocumentActionFailed reports that a runner-owned document action failed
	// or is unavailable in the current host state.
	ErrDocumentActionFailed = errors.New("swiftui: document action failed")
)

func runSceneDocumentOperation(sceneID, operation string) error {
	if sceneID == "" || operation == "" {
		return ErrDocumentActionFailed
	}
	var result int32
	withCString(sceneID, func(sceneIDC *byte) {
		withCString(operation, func(operationC *byte) {
			result = _SUIRunSceneDocumentOperation(sceneIDC, operationC)
		})
	})
	switch result {
	case 1:
		return nil
	case -1:
		return ErrDocumentActionCanceled
	default:
		return ErrDocumentActionFailed
	}
}

func runSceneDocumentPathOperation(sceneID, operation, path string) error {
	if sceneID == "" || operation == "" {
		return ErrDocumentActionFailed
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return errEmptyDocumentPath
	}
	var result int32
	withCString(sceneID, func(sceneIDC *byte) {
		withCString(operation, func(operationC *byte) {
			withCString(path, func(pathC *byte) {
				result = _SUIRunSceneDocumentPathOperation(sceneIDC, operationC, pathC)
			})
		})
	})
	switch result {
	case 1:
		return nil
	case -1:
		return ErrDocumentActionCanceled
	default:
		return ErrDocumentActionFailed
	}
}

func updateSceneDocumentState(sceneID string, handle DocumentHandle) {
	handle = normalizeDocumentHandle(handle)
	withCString(sceneID, func(sceneIDC *byte) {
		withCString(handle.Session.DisplayName, func(displayNameC *byte) {
			withCString(handle.Session.Path, func(pathC *byte) {
				var dirty int32
				if handle.Dirty {
					dirty = 1
				}
				_SUIUpdateSceneDocumentState(sceneIDC, displayNameC, pathC, dirty)
			})
		})
	})
}
