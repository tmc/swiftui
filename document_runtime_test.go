package swiftui

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestDocumentWorkflowScenePlanFields(t *testing.T) {
	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		Session: DocumentSession{
			ID:          "docs",
			DisplayName: "Quarterly Review",
			Path:        "/tmp/review.md",
		},
		Dirty: true,
	}).WithDocumentWorkflow(DocumentWorkflow{
		Open:   func(string) error { return nil },
		Save:   func(DocumentSession, string) error { return nil },
		Export: func(DocumentSession, string) error { return nil },
		Import: func(string) error { return nil },
		Close:  func(DocumentSession) error { return nil },
	})

	plan, err := planScenes([]Scene{scene})
	if err != nil {
		t.Fatalf("planScenes() error = %v", err)
	}
	planJSON, _, err := plan.marshal()
	if err != nil {
		t.Fatalf("plan.marshal() error = %v", err)
	}
	var decoded sceneRunPlan
	if err := json.Unmarshal([]byte(planJSON), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(planJSON) error = %v", err)
	}
	if got, want := decoded.Scenes[0].DocumentDisplayName, "Quarterly Review"; got != want {
		t.Fatalf("decoded.Scenes[0].DocumentDisplayName = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[0].DocumentPath, "/tmp/review.md"; got != want {
		t.Fatalf("decoded.Scenes[0].DocumentPath = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[0].DocumentDirty, true; got != want {
		t.Fatalf("decoded.Scenes[0].DocumentDirty = %v, want %v", got, want)
	}
	if decoded.Scenes[0].DocumentOpenCallbackID == 0 || decoded.Scenes[0].DocumentSaveCallbackID == 0 || decoded.Scenes[0].DocumentExportCallbackID == 0 || decoded.Scenes[0].DocumentImportCallbackID == 0 || decoded.Scenes[0].DocumentCloseCallbackID == 0 || decoded.Scenes[0].DocumentDirtyCallbackID == 0 {
		t.Fatalf("document callback ids = %+v, want all non-zero", decoded.Scenes[0])
	}
}

func TestDocumentHandleActionsUseRunnerOperations(t *testing.T) {
	old := _SUIRunSceneDocumentOperation
	oldPath := _SUIRunSceneDocumentPathOperation
	defer func() {
		_SUIRunSceneDocumentOperation = old
		_SUIRunSceneDocumentPathOperation = oldPath
	}()

	var got []string
	_SUIRunSceneDocumentOperation = func(sceneID, operation *byte) int32 {
		got = append(got, cStringToGo(sceneID)+":"+cStringToGo(operation))
		return 1
	}
	_SUIRunSceneDocumentPathOperation = func(sceneID, operation, path *byte) int32 {
		got = append(got, cStringToGo(sceneID)+":"+cStringToGo(operation)+":"+cStringToGo(path))
		return 1
	}

	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		Path: "/tmp/review.md",
	}).WithDocumentWorkflow(DocumentWorkflow{
		Open:   func(string) error { return nil },
		Save:   func(DocumentSession, string) error { return nil },
		Export: func(DocumentSession, string) error { return nil },
		Import: func(string) error { return nil },
	})

	handle := scene.Handle()
	ops := []struct {
		name string
		fn   func() error
	}{
		{"open", handle.Open},
		{"save", handle.Save},
		{"saveAs", handle.SaveAs},
		{"export", handle.Export},
		{"import", handle.Import},
		{"close", handle.Close},
	}
	for _, op := range ops {
		if op.fn == nil {
			t.Fatalf("%s action = nil, want bound runner action", op.name)
		}
		if err := op.fn(); err != nil {
			t.Fatalf("%s action error = %v", op.name, err)
		}
	}
	if handle.OpenPath == nil {
		t.Fatal("openPath action = nil, want bound runner action")
	}
	if err := handle.OpenPath("/tmp/strategy.md"); err != nil {
		t.Fatalf("openPath action error = %v", err)
	}
	if handle.Revert == nil {
		t.Fatal("revert action = nil, want bound runner action")
	}
	if err := handle.Revert(); err != nil {
		t.Fatalf("revert action error = %v", err)
	}

	want := []string{
		"docs:open",
		"docs:save",
		"docs:saveAs",
		"docs:export",
		"docs:import",
		"docs:close",
		"docs:openPath:/tmp/strategy.md",
		"docs:revert",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runner operations = %v, want %v", got, want)
	}
}

func TestDocumentSceneSetDirtySyncsBridgeState(t *testing.T) {
	old := _SUIUpdateSceneDocumentState
	defer func() { _SUIUpdateSceneDocumentState = old }()

	var (
		gotSceneID     string
		gotDisplayName string
		gotPath        string
		gotDirty       int32
	)
	_SUIUpdateSceneDocumentState = func(sceneID, displayName, path *byte, dirty int32) {
		gotSceneID = cStringToGo(sceneID)
		gotDisplayName = cStringToGo(displayName)
		gotPath = cStringToGo(path)
		gotDirty = dirty
	}

	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		ID:          "docs",
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
	})

	if !scene.runtime.handleSceneActionEvent("available") {
		t.Fatal("handleSceneActionEvent(available) = false, want true")
	}

	scene.SetDirty(true)

	if got, want := scene.Handle().Dirty, true; got != want {
		t.Fatalf("scene.Handle().Dirty = %v, want %v", got, want)
	}
	if got, want := gotSceneID, "docs"; got != want {
		t.Fatalf("synced scene id = %q, want %q", got, want)
	}
	if got, want := gotDisplayName, "Quarterly Review"; got != want {
		t.Fatalf("synced display name = %q, want %q", got, want)
	}
	if got, want := gotPath, "/tmp/review.md"; got != want {
		t.Fatalf("synced path = %q, want %q", got, want)
	}
	if got, want := gotDirty, int32(1); got != want {
		t.Fatalf("synced dirty = %d, want %d", got, want)
	}
}

func TestSceneDocumentActionUsesWorkflowOpenWhenBound(t *testing.T) {
	var opened string
	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithDocumentWorkflow(DocumentWorkflow{
		Open: func(path string) error {
			opened = path
			return nil
		},
	})

	if !scene.runtime.handleSceneActionEvent("available:document") {
		t.Fatal("handleSceneActionEvent(available:document) = false, want true")
	}
	if err := scene.Actions().OpenDocument("/tmp/strategy.md"); err != nil {
		t.Fatalf("scene.Actions().OpenDocument() error = %v", err)
	}
	if got, want := opened, "/tmp/strategy.md"; got != want {
		t.Fatalf("workflow open path = %q, want %q", got, want)
	}
	if got, want := scene.Handle().Session.Path, "/tmp/strategy.md"; got != want {
		t.Fatalf("scene.Handle().Session.Path = %q, want %q", got, want)
	}
	if got, want := scene.Handle().Path, "/tmp/strategy.md"; got != want {
		t.Fatalf("scene.Handle().Path = %q, want %q", got, want)
	}
	if got, want := scene.Handle().Dirty, false; got != want {
		t.Fatalf("scene.Handle().Dirty = %v, want %v", got, want)
	}
}

func TestDocumentRecentDocumentsTrackCurrentSession(t *testing.T) {
	withDocumentRecentStorePathForTest(t)

	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
	}).WithDocumentWorkflow(DocumentWorkflow{
		Open: func(string) error { return nil },
		Save: func(DocumentSession, string) error { return nil },
	})

	if got, want := scene.Handle().RecentDocuments, []DocumentRecent{{
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("initial recent documents = %#v, want %#v", got, want)
	}
	if ok := scene.runtime.handleDocumentOpen("/tmp/strategy.md"); !ok {
		t.Fatal("handleDocumentOpen(/tmp/strategy.md) = false, want true")
	}
	if ok := scene.runtime.handleDocumentSave("/tmp/final.md"); !ok {
		t.Fatal("handleDocumentSave(/tmp/final.md) = false, want true")
	}
	got := scene.Handle().RecentDocuments
	want := []DocumentRecent{
		{DisplayName: "final.md", Path: "/tmp/final.md"},
		{DisplayName: "strategy.md", Path: "/tmp/strategy.md"},
		{DisplayName: "Quarterly Review", Path: "/tmp/review.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recent documents = %#v, want %#v", got, want)
	}
}

func TestDocumentRecentDocumentsPersistAcrossSceneReload(t *testing.T) {
	withDocumentRecentStorePathForTest(t)

	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
	}).WithDocumentWorkflow(DocumentWorkflow{
		Open: func(string) error { return nil },
		Save: func(DocumentSession, string) error { return nil },
	})
	if ok := scene.runtime.handleDocumentOpen("/tmp/strategy.md"); !ok {
		t.Fatal("handleDocumentOpen(/tmp/strategy.md) = false, want true")
	}
	if ok := scene.runtime.handleDocumentSave("/tmp/final.md"); !ok {
		t.Fatal("handleDocumentSave(/tmp/final.md) = false, want true")
	}

	reloaded := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView())
	want := []DocumentRecent{
		{DisplayName: "final.md", Path: "/tmp/final.md"},
		{DisplayName: "strategy.md", Path: "/tmp/strategy.md"},
		{DisplayName: "Quarterly Review", Path: "/tmp/review.md"},
	}
	if got := reloaded.Handle().RecentDocuments; !reflect.DeepEqual(got, want) {
		t.Fatalf("reloaded recent documents = %#v, want %#v", got, want)
	}
}

func TestDocumentSceneCloseUsesWorkflowCloseWhenBound(t *testing.T) {
	var got DocumentSession
	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
	}).WithDocumentWorkflow(DocumentWorkflow{
		Close: func(session DocumentSession) error {
			got = session
			return nil
		},
	})

	if !scene.runtime.handleDocumentClose() {
		t.Fatal("handleDocumentClose() = false, want true")
	}
	if got.DisplayName != "Quarterly Review" || got.Path != "/tmp/review.md" {
		t.Fatalf("close session = %#v, want display/path for current handle", got)
	}
}

func TestDocumentSceneCloseFailsWhenWorkflowCloseFails(t *testing.T) {
	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithDocumentWorkflow(DocumentWorkflow{
		Close: func(DocumentSession) error {
			return errors.New("boom")
		},
	})

	if scene.runtime.handleDocumentClose() {
		t.Fatal("handleDocumentClose() = true, want false on workflow error")
	}
}
