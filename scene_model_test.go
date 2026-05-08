package swiftui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func withDocumentRecentStorePathForTest(t *testing.T) string {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "recent-documents.json")
	documentRecentStoreMu.Lock()
	oldPath := documentRecentStorePath
	oldLoaded := documentRecentStoreLoaded
	oldStore := documentRecentStore
	documentRecentStorePath = func() string { return storePath }
	documentRecentStoreLoaded = false
	documentRecentStore = nil
	documentRecentStoreMu.Unlock()
	t.Cleanup(func() {
		documentRecentStoreMu.Lock()
		documentRecentStorePath = oldPath
		documentRecentStoreLoaded = oldLoaded
		documentRecentStore = oldStore
		documentRecentStoreMu.Unlock()
	})
	return storePath
}

func TestPlanScenes(t *testing.T) {
	window := Window("inspector", AppConfig{Title: "Inspector", Width: 420, Height: 320}, View{})
	document := DocumentGroup("docs", DocumentConfig{Title: "Docs", Width: 900, Height: 700}, View{})
	menu := MenuBarExtra(MenuBarConfig{Label: "Status", SystemImage: "bolt", Width: 320, Height: 240}, View{})
	settings := Settings(AppConfig{Title: "Settings", Width: 660, Height: 620}, View{})

	plan, err := planScenes([]Scene{document, window, settings, menu})
	if err != nil {
		t.Fatalf("planScenes() error = %v", err)
	}
	if got, want := len(plan.specs), 4; got != want {
		t.Fatalf("len(plan.specs) = %d, want %d", got, want)
	}
	if got, want := plan.specs[0].id, "docs"; got != want {
		t.Fatalf("plan.specs[0].id = %q, want %q", got, want)
	}
	if got, want := plan.specs[1].id, "inspector"; got != want {
		t.Fatalf("plan.specs[1].id = %q, want %q", got, want)
	}
	if got, want := plan.specs[2].id, settingsSceneID; got != want {
		t.Fatalf("plan.specs[2].id = %q, want %q", got, want)
	}
	if got, want := plan.specs[3].menuConfig.Label, "Status"; got != want {
		t.Fatalf("plan.specs[3].menuConfig.Label = %q, want %q", got, want)
	}
}

func TestPlanScenesErrors(t *testing.T) {
	menu := MenuBarExtra(MenuBarConfig{Label: "Status"}, View{})

	tests := []struct {
		name   string
		scenes []Scene
		want   error
	}{
		{"empty", nil, errNoScenes},
		{"nil", []Scene{nil}, errNilScene},
		{"empty window id", []Scene{Window("", AppConfig{Title: "Main"}, View{})}, errEmptySceneID},
		{"empty document id", []Scene{DocumentGroup("", DocumentConfig{Title: "Docs"}, View{})}, errEmptySceneID},
		{"empty document title", []Scene{DocumentGroup("docs", DocumentConfig{}, View{})}, errEmptyDocumentTitle},
		{"duplicate ids", []Scene{
			Window("shared", AppConfig{Title: "Main"}, View{}),
			DocumentGroup("shared", DocumentConfig{Title: "Docs"}, View{}),
		}, errDuplicateSceneID},
		{"multiple settings", []Scene{
			Settings(AppConfig{Title: "Settings"}, View{}),
			Settings(AppConfig{Title: "Preferences"}, View{}),
		}, errMultipleSettings},
		{"multiple menu extras", []Scene{menu, menu}, errMultipleMenuExtras},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := planScenes(tt.scenes)
			if !errors.Is(err, tt.want) {
				t.Fatalf("planScenes() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestScenePlanMarshal(t *testing.T) {
	restoreFalse := false
	plan, err := planScenes([]Scene{
		DocumentGroup("docs", DocumentConfig{Title: "Docs", Width: 800, Height: 600}, View{ptr: 11}).WithRestorationID("docs.primary"),
		WindowGroup("inspector", AppConfig{Title: "Inspector", Width: 420, Height: 320}, View{ptr: 22}).
			WithRestorationID("docs.inspector").
			WithAuxiliaryWindowPolicy(AuxiliaryWindowOpenOnLaunch).
			WithVisibilityRestore(false),
		Settings(AppConfig{Title: "Settings", Width: 660, Height: 620}, View{ptr: 33}),
		MenuBarExtra(MenuBarConfig{Label: "Status", SystemImage: "bolt", Width: 240, Height: 160, OpenOnLaunch: true}, View{ptr: 44}),
	})
	if err != nil {
		t.Fatalf("planScenes() error = %v", err)
	}
	planJSON, views, err := plan.marshal()
	if err != nil {
		t.Fatalf("plan.marshal() error = %v", err)
	}
	if got, want := len(views), 4; got != want {
		t.Fatalf("len(views) = %d, want %d", got, want)
	}
	if got, want := views[1:], []uintptr{22, 33, 44}; !reflect.DeepEqual(got, want) {
		t.Fatalf("views[1:] = %v, want %v", got, want)
	}
	var decoded sceneRunPlan
	if err := json.Unmarshal([]byte(planJSON), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(planJSON) error = %v", err)
	}
	if got, want := len(decoded.Scenes), 4; got != want {
		t.Fatalf("len(decoded.Scenes) = %d, want %d", got, want)
	}
	if got, want := decoded.Scenes[1].ID, "inspector"; got != want {
		t.Fatalf("decoded.Scenes[1].ID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[0].RestorationID, "docs.primary"; got != want {
		t.Fatalf("decoded.Scenes[0].RestorationID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[0].OpenOnLaunch, true; got != want {
		t.Fatalf("decoded.Scenes[0].OpenOnLaunch = %v, want %v", got, want)
	}
	if got, want := decoded.Scenes[1].RestorationID, "docs.inspector"; got != want {
		t.Fatalf("decoded.Scenes[1].RestorationID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[1].MultipleInstances, true; got != want {
		t.Fatalf("decoded.Scenes[1].MultipleInstances = %v, want %v", got, want)
	}
	if got, want := decoded.Scenes[1].OpenOnLaunch, true; got != want {
		t.Fatalf("decoded.Scenes[1].OpenOnLaunch = %v, want %v", got, want)
	}
	if got, want := decoded.Scenes[1].RestoreVisibility, &restoreFalse; !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded.Scenes[1].RestoreVisibility = %v, want %v", got, want)
	}
	if got, want := decoded.Scenes[1].AuxiliaryWindowMode, string(AuxiliaryWindowOpenOnLaunch); got != want {
		t.Fatalf("decoded.Scenes[1].AuxiliaryWindowMode = %q, want %q", got, want)
	}
	if decoded.Scenes[1].ActionCallbackID == 0 {
		t.Fatal("decoded.Scenes[1].ActionCallbackID = 0, want non-zero")
	}
	if got, want := decoded.Scenes[2].Kind, "settings"; got != want {
		t.Fatalf("decoded.Scenes[2].Kind = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[2].ID, settingsSceneID; got != want {
		t.Fatalf("decoded.Scenes[2].ID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[2].OpenOnLaunch, false; got != want {
		t.Fatalf("decoded.Scenes[2].OpenOnLaunch = %v, want %v", got, want)
	}
	if got, want := decoded.Scenes[2].RestoreVisibility, &[]bool{false}[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded.Scenes[2].RestoreVisibility = %v, want false", got)
	}
	if got, want := decoded.Scenes[3].OpenOnLaunch, true; got != want {
		t.Fatalf("decoded.Scenes[2].OpenOnLaunch = %v, want %v", got, want)
	}
}

func TestDocumentConfigAppConfig(t *testing.T) {
	cfg := DocumentConfig{Title: "  Docs  ", Width: 800, Height: 600}
	app := cfg.AppConfig()
	if got, want := app.Title, "Docs"; got != want {
		t.Fatalf("AppConfig().Title = %q, want %q", got, want)
	}
	if got, want := app.Width, 800.0; got != want {
		t.Fatalf("AppConfig().Width = %v, want %v", got, want)
	}
	if got, want := app.Height, 600.0; got != want {
		t.Fatalf("AppConfig().Height = %v, want %v", got, want)
	}
}

func TestSceneSpecs(t *testing.T) {
	restoreFalse := false
	window := WindowGroup("inspector", AppConfig{Title: "Inspector"}, View{}).
		WithRestorationID("main.inspector").
		WithAuxiliaryWindowPolicy(AuxiliaryWindowOpenOnLaunch).
		WithVisibilityRestore(false).
		sceneSpec()
	if got, want := window.kind, sceneWindow; got != want {
		t.Fatalf("window kind = %v, want %v", got, want)
	}
	if got, want := window.id, "inspector"; got != want {
		t.Fatalf("window id = %q, want %q", got, want)
	}
	if got, want := window.restorationID, "main.inspector"; got != want {
		t.Fatalf("window restorationID = %q, want %q", got, want)
	}
	if got, want := window.auxiliaryWindowPolicy, AuxiliaryWindowOpenOnLaunch; got != want {
		t.Fatalf("window auxiliaryWindowPolicy = %q, want %q", got, want)
	}
	if got, want := window.restoreVisibility, &restoreFalse; !reflect.DeepEqual(got, want) {
		t.Fatalf("window restoreVisibility = %v, want %v", got, want)
	}
	if got, want := window.multipleInstances, true; got != want {
		t.Fatalf("window multipleInstances = %v, want %v", got, want)
	}

	singleton := Window("singleton", AppConfig{Title: "Window"}, View{}).sceneSpec()
	if got, want := singleton.multipleInstances, false; got != want {
		t.Fatalf("singleton multipleInstances = %v, want %v", got, want)
	}

	document := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, View{}).
		WithRestorationID("docs.primary").
		sceneSpec()
	if got, want := document.kind, sceneDocument; got != want {
		t.Fatalf("document kind = %v, want %v", got, want)
	}
	if got, want := document.id, "docs"; got != want {
		t.Fatalf("document id = %q, want %q", got, want)
	}
	if got, want := document.restorationID, "docs.primary"; got != want {
		t.Fatalf("document restorationID = %q, want %q", got, want)
	}

	menu := MenuBarExtra(MenuBarConfig{Label: "Agent"}, View{}).sceneSpec()
	if got, want := menu.kind, sceneMenuBar; got != want {
		t.Fatalf("menu kind = %v, want %v", got, want)
	}
	if got, want := menu.menuConfig.Label, "Agent"; got != want {
		t.Fatalf("menu label = %q, want %q", got, want)
	}

	settings := Settings(AppConfig{}, View{}).sceneSpec()
	if got, want := settings.kind, sceneSettings; got != want {
		t.Fatalf("settings kind = %v, want %v", got, want)
	}
	if got, want := settings.id, settingsSceneID; got != want {
		t.Fatalf("settings id = %q, want %q", got, want)
	}
}

func TestDocumentGroupHandleBuilder(t *testing.T) {
	scene := DocumentGroupWithHandle("docs", DocumentConfig{
		Title:  "Docs",
		Width:  800,
		Height: 600,
	}, func(handle DocumentHandle, actions SceneActions) View {
		if handle.ID != "docs" {
			t.Fatalf("handle.ID = %q, want docs", handle.ID)
		}
		if handle.DisplayName != "Docs" {
			t.Fatalf("handle.DisplayName = %q, want Docs", handle.DisplayName)
		}
		if actions.Available() {
			t.Fatal("constructor-time scene actions should still be unavailable before runner binding")
		}
		return EmptyView()
	})

	if got, want := scene.ID(), "docs"; got != want {
		t.Fatalf("scene.ID() = %q, want %q", got, want)
	}
	if got, want := scene.Config().Title, "Docs"; got != want {
		t.Fatalf("scene.Config().Title = %q, want %q", got, want)
	}
	if got, want := scene.Handle().DisplayName, "Docs"; got != want {
		t.Fatalf("scene.Handle().DisplayName = %q, want %q", got, want)
	}
	if got, want := scene.RestorationID(), "docs"; got != want {
		t.Fatalf("scene.RestorationID() = %q, want %q", got, want)
	}
	if scene.Actions().Available() {
		t.Fatal("scene.Actions() should stay unavailable before the scene runner injects actions")
	}
}

func TestDocumentGroupSetHandleTracksRecentDocuments(t *testing.T) {
	withDocumentRecentStorePathForTest(t)

	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView())
	scene.SetHandle(DocumentHandle{
		Session: DocumentSession{
			DisplayName: " Quarterly Review ",
			Path:        " /tmp/review.md ",
		},
		RecentDocuments: []DocumentRecent{
			{DisplayName: " ", Path: " /tmp/review.md "},
			{Path: " /tmp/product-strategy.md "},
			{Path: ""},
		},
	})

	handle := scene.Handle()
	if got, want := handle.Session.DisplayName, "Quarterly Review"; got != want {
		t.Fatalf("handle.Session.DisplayName = %q, want %q", got, want)
	}
	if got, want := handle.Session.Path, "/tmp/review.md"; got != want {
		t.Fatalf("handle.Session.Path = %q, want %q", got, want)
	}
	if got, want := handle.RecentDocuments, []DocumentRecent{
		{DisplayName: "Quarterly Review", Path: "/tmp/review.md"},
		{DisplayName: "product-strategy.md", Path: "/tmp/product-strategy.md"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handle.RecentDocuments = %#v, want %#v", got, want)
	}
}

func TestDocumentGroupRestoresRecentDocumentsAcrossRelaunch(t *testing.T) {
	withDocumentRecentStorePathForTest(t)

	first := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
	}).WithDocumentWorkflow(DocumentWorkflow{
		Open: func(string) error { return nil },
		Save: func(DocumentSession, string) error { return nil },
	})
	if ok := first.runtime.handleDocumentOpen("/tmp/product-strategy.md"); !ok {
		t.Fatal("handleDocumentOpen(/tmp/product-strategy.md) = false, want true")
	}

	want := []DocumentRecent{
		{DisplayName: "product-strategy.md", Path: "/tmp/product-strategy.md"},
		{DisplayName: "Quarterly Review", Path: "/tmp/review.md"},
	}
	if got := first.Handle().RecentDocuments; !reflect.DeepEqual(got, want) {
		t.Fatalf("first scene recent documents = %#v, want %#v", got, want)
	}

	second := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView())
	if got := second.Handle().RecentDocuments; !reflect.DeepEqual(got, want) {
		t.Fatalf("restored recent documents = %#v, want %#v", got, want)
	}
}

func TestDocumentRecentStoreKeyIncludesAppIdentity(t *testing.T) {
	oldArgs := append([]string(nil), os.Args...)
	t.Cleanup(func() {
		os.Args = oldArgs
	})

	os.Args = []string{"/tmp/scenes"}
	if got, want := normalizeDocumentRecentStoreKey("docs"), "scenes:docs"; got != want {
		t.Fatalf("normalizeDocumentRecentStoreKey() = %q, want %q", got, want)
	}

	os.Args = []string{"/tmp/workbench"}
	if got, want := normalizeDocumentRecentStoreKey("docs"), "workbench:docs"; got != want {
		t.Fatalf("normalizeDocumentRecentStoreKey() with second app = %q, want %q", got, want)
	}
}

func TestDocumentGroupWithHandleRestoresRecentDocumentsBeforeBuilder(t *testing.T) {
	withDocumentRecentStorePathForTest(t)

	seed := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView()).WithHandle(DocumentHandle{
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
	}).WithDocumentWorkflow(DocumentWorkflow{
		Open: func(string) error { return nil },
		Save: func(DocumentSession, string) error { return nil },
	})
	if ok := seed.runtime.handleDocumentOpen("/tmp/product-strategy.md"); !ok {
		t.Fatal("handleDocumentOpen(/tmp/product-strategy.md) = false, want true")
	}

	want := []DocumentRecent{
		{DisplayName: "product-strategy.md", Path: "/tmp/product-strategy.md"},
		{DisplayName: "Quarterly Review", Path: "/tmp/review.md"},
	}

	var got []DocumentRecent
	scene := DocumentGroupWithHandle("docs", DocumentConfig{Title: "Docs"}, func(handle DocumentHandle, _ SceneActions) View {
		got = append([]DocumentRecent(nil), handle.RecentDocuments...)
		return EmptyView()
	})

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("builder recent documents = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(scene.Handle().RecentDocuments, want) {
		t.Fatalf("scene recent documents = %#v, want %#v", scene.Handle().RecentDocuments, want)
	}
}

func TestDocumentGroupMatchesHandleIdentity(t *testing.T) {
	scene := DocumentGroup(" docs ", DocumentConfig{Title: "  Docs  "}, EmptyView())
	if got, want := scene.ID(), "docs"; got != want {
		t.Fatalf("scene.ID() = %q, want %q", got, want)
	}
	if got, want := scene.Config().Title, "Docs"; got != want {
		t.Fatalf("scene.Config().Title = %q, want %q", got, want)
	}
	if got, want := scene.Handle().ID, "docs"; got != want {
		t.Fatalf("scene.Handle().ID = %q, want %q", got, want)
	}
	if got, want := scene.Handle().DisplayName, "Docs"; got != want {
		t.Fatalf("scene.Handle().DisplayName = %q, want %q", got, want)
	}
	if got, want := scene.RestorationID(), "docs"; got != want {
		t.Fatalf("scene.RestorationID() = %q, want %q", got, want)
	}
}

func TestWindowGroupRestorationAndAuxiliaryDefaults(t *testing.T) {
	scene := Window("inspector", AppConfig{Title: "Inspector"}, EmptyView())
	if got, want := scene.RestorationID(), "inspector"; got != want {
		t.Fatalf("scene.RestorationID() = %q, want %q", got, want)
	}
	if got, want := scene.AuxiliaryWindowPolicy(), AuxiliaryWindowManual; got != want {
		t.Fatalf("scene.AuxiliaryWindowPolicy() = %q, want %q", got, want)
	}
}

func TestWindowGroupRuntimeState(t *testing.T) {
	restoreFalse := false
	scene := WindowGroup("inspector", AppConfig{Title: "Inspector"}, EmptyView()).
		WithRestorationID("main.inspector").
		WithAuxiliaryWindowPolicy(AuxiliaryWindowOpenOnLaunch).
		WithVisibilityRestore(false)
	state := scene.RuntimeState()
	if got, want := state.Kind, "window"; got != want {
		t.Fatalf("state.Kind = %q, want %q", got, want)
	}
	if got, want := state.ID, "inspector"; got != want {
		t.Fatalf("state.ID = %q, want %q", got, want)
	}
	if got, want := state.RestorationID, "main.inspector"; got != want {
		t.Fatalf("state.RestorationID = %q, want %q", got, want)
	}
	if got, want := state.Lifecycle, SceneLifecycleInactive; got != want {
		t.Fatalf("state.Lifecycle = %q, want %q", got, want)
	}
	if state.Live {
		t.Fatal("state.Live = true, want false before runner activation")
	}
	if got, want := state.WindowInstanceCount, 0; got != want {
		t.Fatalf("state.WindowInstanceCount = %d, want %d before runner activation", got, want)
	}
	if got := state.FocusedWindowInstanceID; got != "" {
		t.Fatalf("state.FocusedWindowInstanceID = %q, want empty before runner activation", got)
	}
	if len(state.WindowInstances) != 0 {
		t.Fatalf("len(state.WindowInstances) = %d, want 0 before runner activation", len(state.WindowInstances))
	}
	if state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = true, want false before runner binding")
	}
	if got, want := state.AuxiliaryWindowPolicy, AuxiliaryWindowOpenOnLaunch; got != want {
		t.Fatalf("state.AuxiliaryWindowPolicy = %q, want %q", got, want)
	}
	if got, want := state.RestoreVisibility, &restoreFalse; !reflect.DeepEqual(got, want) {
		t.Fatalf("state.RestoreVisibility = %v, want %v", got, want)
	}
	if state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = true, want false")
	}
	if scene.runtime == nil {
		t.Fatal("scene.runtime = nil, want runtime")
	}
	if ok := scene.runtime.handleSceneActionEvent("available:window,document,refresh;count=3;instance=inspector.instance.2;visible=0"); !ok {
		t.Fatal("handleSceneActionEvent(available:window,document,refresh;count=3;instance=inspector.instance.2;visible=0) = false, want true")
	}
	state = scene.RuntimeState()
	if got, want := state.Lifecycle, SceneLifecycleInactive; got != want {
		t.Fatalf("state.Lifecycle after available = %q, want %q", got, want)
	}
	if !state.Live {
		t.Fatal("state.Live = false after available, want true")
	}
	if got, want := state.WindowInstanceCount, 3; got != want {
		t.Fatalf("state.WindowInstanceCount after available = %d, want %d", got, want)
	}
	if got := state.FocusedWindowInstanceID; got != "" {
		t.Fatalf("state.FocusedWindowInstanceID after available = %q, want empty", got)
	}
	if got, want := state.WindowInstances, []SceneWindowInstanceState{
		{ID: "inspector.instance.2", Visible: false, Focused: false},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("state.WindowInstances after available = %#v, want %#v", got, want)
	}
	if !state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = false after runner binding, want true")
	}
	if ok := scene.runtime.handleSceneActionEvent("focused:count=3;instance=inspector.instance.1;visible=1"); !ok {
		t.Fatal("handleSceneActionEvent(focused:count=3;instance=inspector.instance.1;visible=1) = false, want true")
	}
	state = scene.RuntimeState()
	if got, want := state.Lifecycle, SceneLifecycleActive; got != want {
		t.Fatalf("state.Lifecycle after focused = %q, want %q", got, want)
	}
	if got, want := state.FocusedWindowInstanceID, "inspector.instance.1"; got != want {
		t.Fatalf("state.FocusedWindowInstanceID after focused = %q, want %q", got, want)
	}
	if got, want := state.WindowInstances, []SceneWindowInstanceState{
		{ID: "inspector.instance.1", Visible: true, Focused: true},
		{ID: "inspector.instance.2", Visible: false, Focused: false},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("state.WindowInstances after focused = %#v, want %#v", got, want)
	}
	if ok := scene.runtime.handleSceneActionEvent("blurred:count=3;instance=inspector.instance.1;visible=1"); !ok {
		t.Fatal("handleSceneActionEvent(blurred:count=3;instance=inspector.instance.1;visible=1) = false, want true")
	}
	state = scene.RuntimeState()
	if got, want := state.Lifecycle, SceneLifecycleInactive; got != want {
		t.Fatalf("state.Lifecycle after blurred = %q, want %q", got, want)
	}
	if !state.Live {
		t.Fatal("state.Live = false after blurred, want true")
	}
	if got := state.FocusedWindowInstanceID; got != "" {
		t.Fatalf("state.FocusedWindowInstanceID after blurred = %q, want empty", got)
	}
	if got, want := state.WindowInstances, []SceneWindowInstanceState{
		{ID: "inspector.instance.1", Visible: true, Focused: false},
		{ID: "inspector.instance.2", Visible: false, Focused: false},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("state.WindowInstances after blurred = %#v, want %#v", got, want)
	}
	if !state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = false after blurred, want true")
	}
	if ok := scene.runtime.handleSceneActionEvent("blurred:count=1;instance=inspector.instance.2;visible=0"); !ok {
		t.Fatal("handleSceneActionEvent(blurred:count=1;instance=inspector.instance.2;visible=0) = false, want true")
	}
	state = scene.RuntimeState()
	if got, want := state.WindowInstanceCount, 1; got != want {
		t.Fatalf("state.WindowInstanceCount after hidden close = %d, want %d", got, want)
	}
	if got, want := state.WindowInstances, []SceneWindowInstanceState{
		{ID: "inspector.instance.1", Visible: true, Focused: false},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("state.WindowInstances after hidden close = %#v, want %#v", got, want)
	}
	if ok := scene.runtime.handleSceneActionEvent("unavailable:count=0;instance=inspector.instance.1;visible=0"); !ok {
		t.Fatal("handleSceneActionEvent(unavailable:count=0;instance=inspector.instance.1;visible=0) = false, want true")
	}
	state = scene.RuntimeState()
	if state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = true after unavailable, want false")
	}
	if got, want := state.WindowInstanceCount, 0; got != want {
		t.Fatalf("state.WindowInstanceCount after unavailable = %d, want %d", got, want)
	}
	if got := state.FocusedWindowInstanceID; got != "" {
		t.Fatalf("state.FocusedWindowInstanceID after unavailable = %q, want empty", got)
	}
	if len(state.WindowInstances) != 0 {
		t.Fatalf("len(state.WindowInstances) after unavailable = %d, want 0", len(state.WindowInstances))
	}
}

func TestWindowGroupWithActionsReportsBorrowedCapabilities(t *testing.T) {
	scene := Window("inspector", AppConfig{Title: "Inspector"}, EmptyView()).
		WithActions(SceneActions{
			RefreshScene: RefreshAction(func() error { return nil }),
		})

	state := scene.RuntimeState()
	if !state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = false with manual actions, want true")
	}
	if state.Actions.RefreshScene == nil {
		t.Fatal("state.Actions.RefreshScene = nil, want manual action bound")
	}
	if scene.Actions().RefreshScene == nil {
		t.Fatal("scene.Actions().RefreshScene = nil, want manual action bound")
	}
}

func TestWindowGroupRuntimeActionsMergeManualAndInjectedCapabilities(t *testing.T) {
	scene := Window("inspector", AppConfig{Title: "Inspector"}, EmptyView()).
		WithActions(SceneActions{
			Document: OpenDocumentAction(func(path string) error {
				if got, want := path, "/tmp/merged.txt"; got != want {
					t.Fatalf("manual document path = %q, want %q", got, want)
				}
				return nil
			}),
		})
	if scene.runtime == nil {
		t.Fatal("scene.runtime = nil, want runtime")
	}
	if ok := scene.runtime.handleSceneActionEvent("available:window,refresh,immersive"); !ok {
		t.Fatal("handleSceneActionEvent(available:window,refresh,immersive) = false, want true")
	}

	actions := scene.Actions()
	if actions.Window == nil {
		t.Fatal("actions.Window = nil, want injected window action")
	}
	if actions.RefreshScene == nil {
		t.Fatal("actions.RefreshScene = nil, want injected refresh action")
	}
	if actions.ImmersiveSpace == nil {
		t.Fatal("actions.ImmersiveSpace = nil, want injected immersive action")
	}
	if actions.Document == nil {
		t.Fatal("actions.Document = nil, want manual document action preserved")
	}
	if err := actions.OpenDocument("/tmp/merged.txt"); err != nil {
		t.Fatalf("actions.OpenDocument() error = %v", err)
	}
}

func TestWindowGroupStatus(t *testing.T) {
	restoreFalse := false
	scene := Window("inspector", AppConfig{Title: "Inspector"}, EmptyView()).
		WithRestorationID("main.inspector").
		WithAuxiliaryWindowPolicy(AuxiliaryWindowOpenOnLaunch).
		WithVisibilityRestore(false)
	status := scene.Status()
	if got, want := status.Kind, "window"; got != want {
		t.Fatalf("status.Kind = %q, want %q", got, want)
	}
	if got, want := status.ID, "inspector"; got != want {
		t.Fatalf("status.ID = %q, want %q", got, want)
	}
	if status.Active {
		t.Fatal("status.Active = true, want false before runner activation")
	}
	if got, want := status.RestorationID, "main.inspector"; got != want {
		t.Fatalf("status.RestorationID = %q, want %q", got, want)
	}
	if got, want := status.RestoresVisibility, &restoreFalse; !reflect.DeepEqual(got, want) {
		t.Fatalf("status.RestoresVisibility = %v, want %v", got, want)
	}
	if got, want := status.AuxiliaryPolicy, AuxiliaryWindowOpenOnLaunch; got != want {
		t.Fatalf("status.AuxiliaryPolicy = %q, want %q", got, want)
	}
	if scene.runtime == nil {
		t.Fatal("scene.runtime = nil, want runtime")
	}
	if ok := scene.runtime.handleSceneActionEvent("available"); !ok {
		t.Fatal("handleSceneActionEvent(available) = false, want true")
	}
	status = scene.Status()
	if !status.Active {
		t.Fatal("status.Active = false after available, want true")
	}
	if ok := scene.runtime.handleSceneActionEvent("blurred"); !ok {
		t.Fatal("handleSceneActionEvent(blurred) = false, want true")
	}
	status = scene.Status()
	if status.Active {
		t.Fatal("status.Active = true after blurred, want false")
	}
	if ok := scene.runtime.handleSceneActionEvent("focused"); !ok {
		t.Fatal("handleSceneActionEvent(focused) = false, want true")
	}
	status = scene.Status()
	if !status.Active {
		t.Fatal("status.Active = false after focused, want true")
	}
	if ok := scene.runtime.handleSceneActionEvent("unavailable"); !ok {
		t.Fatal("handleSceneActionEvent(unavailable) = false, want true")
	}
	status = scene.Status()
	if status.Active {
		t.Fatal("status.Active = true after unavailable, want false")
	}
}

func TestScenePlanMarshalPreservesExplicitRestorationIDs(t *testing.T) {
	plan, err := planScenes([]Scene{
		Window("window.id", AppConfig{Title: "Window"}, View{ptr: 11}).
			WithRestorationID("window.restore"),
		DocumentGroup("document.id", DocumentConfig{Title: "Document"}, View{ptr: 22}).
			WithRestorationID("document.restore"),
	})
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
	if got, want := len(decoded.Scenes), 2; got != want {
		t.Fatalf("len(decoded.Scenes) = %d, want %d", got, want)
	}
	if got, want := decoded.Scenes[0].ID, "window.id"; got != want {
		t.Fatalf("decoded.Scenes[0].ID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[0].RestorationID, "window.restore"; got != want {
		t.Fatalf("decoded.Scenes[0].RestorationID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[1].ID, "document.id"; got != want {
		t.Fatalf("decoded.Scenes[1].ID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[1].RestorationID, "document.restore"; got != want {
		t.Fatalf("decoded.Scenes[1].RestorationID = %q, want %q", got, want)
	}
}

func TestScenePlanMarshalLifecycleCallbacks(t *testing.T) {
	var launched, activated, resigned, terminated bool
	allowTerminate := true
	shouldTerminateCalls := 0

	lifecycle := AppLifecycle{
		OnLaunched: func() {
			launched = true
		},
		OnActivate: func() {
			activated = true
		},
		OnResignActive: func() {
			resigned = true
		},
		ShouldTerminate: func() bool {
			shouldTerminateCalls++
			return allowTerminate
		},
		OnTerminate: func() {
			terminated = true
		},
	}

	plan, err := planScenes([]Scene{
		Window("window.id", AppConfig{Title: "Window"}, View{ptr: 11}),
	})
	if err != nil {
		t.Fatalf("planScenes() error = %v", err)
	}
	WithLifecycle(lifecycle).applySceneOption(&plan)

	planJSON, _, err := plan.marshal()
	if err != nil {
		t.Fatalf("plan.marshal() error = %v", err)
	}
	var decoded sceneRunPlan
	if err := json.Unmarshal([]byte(planJSON), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(planJSON) error = %v", err)
	}
	if decoded.Lifecycle == nil {
		t.Fatal("decoded.Lifecycle = nil, want lifecycle callbacks")
	}
	if decoded.Lifecycle.DidFinishLaunchingCallbackID == 0 {
		t.Fatal("decoded.Lifecycle.DidFinishLaunchingCallbackID = 0, want non-zero")
	}
	if decoded.Lifecycle.DidBecomeActiveCallbackID == 0 {
		t.Fatal("decoded.Lifecycle.DidBecomeActiveCallbackID = 0, want non-zero")
	}
	if decoded.Lifecycle.DidResignActiveCallbackID == 0 {
		t.Fatal("decoded.Lifecycle.DidResignActiveCallbackID = 0, want non-zero")
	}
	if decoded.Lifecycle.ShouldTerminateCallbackID == 0 {
		t.Fatal("decoded.Lifecycle.ShouldTerminateCallbackID = 0, want non-zero")
	}
	if decoded.Lifecycle.WillTerminateCallbackID == 0 {
		t.Fatal("decoded.Lifecycle.WillTerminateCallbackID = 0, want non-zero")
	}

	buttonCallbackTrampoline(uintptr(decoded.Lifecycle.DidFinishLaunchingCallbackID))
	if !launched {
		t.Fatal("OnLaunched callback did not fire")
	}
	buttonCallbackTrampoline(uintptr(decoded.Lifecycle.DidBecomeActiveCallbackID))
	if !activated {
		t.Fatal("OnActivate callback did not fire")
	}
	buttonCallbackTrampoline(uintptr(decoded.Lifecycle.DidResignActiveCallbackID))
	if !resigned {
		t.Fatal("OnResignActive callback did not fire")
	}
	buttonCallbackTrampoline(uintptr(decoded.Lifecycle.WillTerminateCallbackID))
	if !terminated {
		t.Fatal("OnTerminate callback did not fire")
	}

	if got := commandCallbackTrampoline(uintptr(decoded.Lifecycle.ShouldTerminateCallbackID)); got != 1 {
		t.Fatalf("ShouldTerminate returned %d, want 1", got)
	}
	if shouldTerminateCalls != 1 {
		t.Fatalf("ShouldTerminate call count = %d, want 1", shouldTerminateCalls)
	}
	allowTerminate = false
	if got := commandCallbackTrampoline(uintptr(decoded.Lifecycle.ShouldTerminateCallbackID)); got != 0 {
		t.Fatalf("ShouldTerminate returned %d, want 0", got)
	}
	if shouldTerminateCalls != 2 {
		t.Fatalf("ShouldTerminate call count = %d, want 2", shouldTerminateCalls)
	}
}

func TestScenePlanMarshalAuxiliaryWindowDefaultsToManual(t *testing.T) {
	plan, err := planScenes([]Scene{
		DocumentGroup("docs", DocumentConfig{Title: "Docs", Width: 800, Height: 600}, View{ptr: 11}),
		Window("inspector", AppConfig{Title: "Inspector", Width: 420, Height: 320}, View{ptr: 22}),
	})
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
	if got, want := decoded.Scenes[0].RestorationID, "docs"; got != want {
		t.Fatalf("decoded.Scenes[0].RestorationID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[1].RestorationID, "inspector"; got != want {
		t.Fatalf("decoded.Scenes[1].RestorationID = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[1].AuxiliaryWindowMode, string(AuxiliaryWindowManual); got != want {
		t.Fatalf("decoded.Scenes[1].AuxiliaryWindowMode = %q, want %q", got, want)
	}
	if !strings.Contains(planJSON, `"openOnLaunch":false`) {
		t.Fatalf("planJSON missing explicit false openOnLaunch: %s", planJSON)
	}
	if decoded.Scenes[1].OpenOnLaunch {
		t.Fatal("decoded.Scenes[1].OpenOnLaunch = true, want false for manual auxiliary window")
	}
	if decoded.Scenes[1].ActionCallbackID == 0 {
		t.Fatal("decoded.Scenes[1].ActionCallbackID = 0, want non-zero")
	}
}

func TestScenePlanMarshalSettingsOnlyOpensOnLaunch(t *testing.T) {
	plan, err := planScenes([]Scene{
		Settings(AppConfig{Title: "Settings", Width: 660, Height: 620}, View{ptr: 55}),
	})
	if err != nil {
		t.Fatalf("planScenes() error = %v", err)
	}
	planJSON, views, err := plan.marshal()
	if err != nil {
		t.Fatalf("plan.marshal() error = %v", err)
	}
	if got, want := views, []uintptr{55}; !reflect.DeepEqual(got, want) {
		t.Fatalf("views = %v, want %v", got, want)
	}
	var decoded sceneRunPlan
	if err := json.Unmarshal([]byte(planJSON), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(planJSON) error = %v", err)
	}
	if got, want := len(decoded.Scenes), 1; got != want {
		t.Fatalf("len(decoded.Scenes) = %d, want %d", got, want)
	}
	if got, want := decoded.Scenes[0].Kind, "settings"; got != want {
		t.Fatalf("decoded.Scenes[0].Kind = %q, want %q", got, want)
	}
	if got, want := decoded.Scenes[0].OpenOnLaunch, true; got != want {
		t.Fatalf("decoded.Scenes[0].OpenOnLaunch = %v, want %v", got, want)
	}
}

func TestDocumentGroupWithHandleAndActions(t *testing.T) {
	var gotHandle DocumentHandle
	var gotAvailable bool
	oldDynamicView := _SUIDynamicView
	t.Cleanup(func() {
		_SUIDynamicView = oldDynamicView
	})
	_SUIDynamicView = func(_ uintptr, builderID uintptr) uintptr {
		return viewBuilderCallbackTrampoline(builderID, 0)
	}
	scene := DocumentGroupWithHandle("docs", DocumentConfig{Title: "Docs"}, func(handle DocumentHandle, actions SceneActions) View {
		gotHandle = handle
		gotAvailable = actions.Available()
		return EmptyView()
	}).WithHandle(DocumentHandle{
		ID:          "docs",
		DisplayName: "Quarterly Review",
		Path:        "/tmp/review.md",
		Dirty:       true,
	}).WithActions(SceneActions{
		RefreshScene: RefreshAction(func() error { return nil }),
	})

	spec := scene.sceneSpec()
	if got, want := spec.documentHandle.DisplayName, "Quarterly Review"; got != want {
		t.Fatalf("sceneSpec().documentHandle.DisplayName = %q, want %q", got, want)
	}
	if got, want := spec.documentHandle.Path, "/tmp/review.md"; got != want {
		t.Fatalf("sceneSpec().documentHandle.Path = %q, want %q", got, want)
	}
	if !spec.documentActions.Available() {
		t.Fatal("sceneSpec().documentActions should be available")
	}
	if gotHandle.DisplayName != "Quarterly Review" || gotHandle.Path != "/tmp/review.md" || !gotHandle.Dirty {
		t.Fatalf("builder handle = %+v, want updated handle", gotHandle)
	}
	if !gotAvailable {
		t.Fatal("builder actions should reflect WithActions bindings")
	}
}

func TestWindowGroupWithStatus(t *testing.T) {
	var gotStatus SceneStatus
	oldDynamicView := _SUIDynamicView
	t.Cleanup(func() {
		_SUIDynamicView = oldDynamicView
	})
	_SUIDynamicView = func(_ uintptr, builderID uintptr) uintptr {
		return viewBuilderCallbackTrampoline(builderID, 0)
	}

	scene := WindowGroupWithStatus("inspector", AppConfig{Title: "Inspector"}, func(status SceneStatus) View {
		gotStatus = status
		return EmptyView()
	}).WithRestorationID("main.inspector").WithAuxiliaryWindowPolicy(AuxiliaryWindowOpenOnLaunch)

	spec := scene.sceneSpec()
	if got, want := gotStatus.ID, "inspector"; got != want {
		t.Fatalf("builder status ID = %q, want %q", got, want)
	}
	if got, want := gotStatus.RestorationID, "main.inspector"; got != want {
		t.Fatalf("builder status RestorationID = %q, want %q", got, want)
	}
	if got, want := gotStatus.Active, false; got != want {
		t.Fatalf("builder status Active = %v, want %v", got, want)
	}
	if spec.appView.ptr == 0 {
		t.Fatal("sceneSpec().appView.ptr = 0, want dynamic builder view")
	}

	if scene.runtime == nil {
		t.Fatal("scene.runtime = nil, want runtime")
	}
	if ok := scene.runtime.handleSceneActionEvent("available"); !ok {
		t.Fatal("handleSceneActionEvent(available) = false, want true")
	}
	_ = scene.sceneSpec()
	if got, want := gotStatus.Active, true; got != want {
		t.Fatalf("builder status Active after available = %v, want %v", got, want)
	}
}

func TestDocumentGroupRuntimeState(t *testing.T) {
	scene := DocumentGroupWithHandle("docs", DocumentConfig{Title: "Docs"}, func(DocumentHandle, SceneActions) View {
		return EmptyView()
	})
	state := scene.RuntimeState()
	if got, want := state.Kind, "document"; got != want {
		t.Fatalf("state.Kind = %q, want %q", got, want)
	}
	if got, want := state.ID, "docs"; got != want {
		t.Fatalf("state.ID = %q, want %q", got, want)
	}
	if got, want := state.RestorationID, "docs"; got != want {
		t.Fatalf("state.RestorationID = %q, want %q", got, want)
	}
	if got, want := state.Lifecycle, SceneLifecycleInactive; got != want {
		t.Fatalf("state.Lifecycle = %q, want %q", got, want)
	}
	if state.Live {
		t.Fatal("state.Live = true, want false before runner activation")
	}
	if state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = true, want false before runner binding")
	}
	if got, want := state.Handle.DisplayName, "Docs"; got != want {
		t.Fatalf("state.Handle.DisplayName = %q, want %q", got, want)
	}
	if got, want := state.Handle.ID, "docs"; got != want {
		t.Fatalf("state.Handle.ID = %q, want %q", got, want)
	}
	if !scene.runtime.handleSceneActionEvent("available:window,document,refresh,immersive") {
		t.Fatal("handleSceneActionEvent(available:window,document,refresh,immersive) = false, want true")
	}
	state = scene.RuntimeState()
	if got, want := state.Lifecycle, SceneLifecycleActive; got != want {
		t.Fatalf("state.Lifecycle after availability = %q, want %q", got, want)
	}
	if !state.Live {
		t.Fatal("state.Live = false after runner activation, want true")
	}
	if !state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = false after runner binding, want true")
	}
	if !scene.runtime.handleSceneActionEvent("blurred") {
		t.Fatal("handleSceneActionEvent(blurred) = false, want true")
	}
	state = scene.RuntimeState()
	if got, want := state.Lifecycle, SceneLifecycleInactive; got != want {
		t.Fatalf("state.Lifecycle after blurred = %q, want %q", got, want)
	}
	if !state.Live {
		t.Fatal("state.Live = false after blurred, want true")
	}
	if !state.ActionsAvailable() {
		t.Fatal("state.ActionsAvailable() = false after blurred, want true")
	}
}

func TestDocumentGroupStatus(t *testing.T) {
	restoreTrue := true
	scene := DocumentGroupWithHandle("docs", DocumentConfig{Title: "Docs"}, func(DocumentHandle, SceneActions) View {
		return EmptyView()
	}).WithRestorationID("docs.primary").WithVisibilityRestore(true)

	status := scene.Status()
	if got, want := status.Kind, "document"; got != want {
		t.Fatalf("status.Kind = %q, want %q", got, want)
	}
	if got, want := status.ID, "docs"; got != want {
		t.Fatalf("status.ID = %q, want %q", got, want)
	}
	if status.Active {
		t.Fatal("status.Active = true before availability, want false")
	}
	if got, want := status.RestorationID, "docs.primary"; got != want {
		t.Fatalf("status.RestorationID = %q, want %q", got, want)
	}
	if got, want := status.RestoresVisibility, &restoreTrue; !reflect.DeepEqual(got, want) {
		t.Fatalf("status.RestoresVisibility = %v, want %v", got, want)
	}
	if got, want := status.AuxiliaryPolicy, AuxiliaryWindowManual; got != want {
		t.Fatalf("status.AuxiliaryPolicy = %q, want %q", got, want)
	}

	if !scene.runtime.handleSceneActionEvent("available:window,document,refresh,immersive") {
		t.Fatal("handleSceneActionEvent(available:window,document,refresh,immersive) = false, want true")
	}
	status = scene.Status()
	if !status.Active {
		t.Fatal("status.Active = false after availability, want true")
	}
	if !scene.runtime.handleSceneActionEvent("blurred") {
		t.Fatal("handleSceneActionEvent(blurred) = false, want true")
	}
	status = scene.Status()
	if status.Active {
		t.Fatal("status.Active = true after blurred, want false")
	}
}

func TestDocumentGroupSceneActionHelpers(t *testing.T) {
	scene := DocumentGroupWithHandle("docs", DocumentConfig{Title: "Docs"}, func(DocumentHandle, SceneActions) View {
		return EmptyView()
	}).
		WithWindowAction(OpenWindowAction(func(string) error { return nil })).
		WithDocumentAction(OpenDocumentAction(func(string) error { return nil })).
		WithRefreshAction(RefreshAction(func() error { return nil }))

	if !scene.Actions().Available() {
		t.Fatal("scene.Actions() should be available")
	}
	if scene.Actions().Window == nil || scene.Actions().Document == nil || scene.Actions().RefreshScene == nil {
		t.Fatal("expected helper-bound actions to be present")
	}
}

func TestDocumentGroupCurrentRunnerActions(t *testing.T) {
	old := _SUIOpenSceneWindow
	defer func() { _SUIOpenSceneWindow = old }()
	_SUIOpenSceneWindow = func(id *byte) int32 {
		switch cStringToGo(id) {
		case "inspector", "demo.space":
			return 1
		default:
			return 0
		}
	}

	scene := DocumentGroupWithHandle("docs", DocumentConfig{Title: "Docs"}, func(DocumentHandle, SceneActions) View {
		return EmptyView()
	})
	if scene.runtime == nil {
		t.Fatal("scene runtime should be available")
	}
	if scene.Active() {
		t.Fatal("scene.Active() = true before lifecycle availability, want false")
	}
	if !scene.runtime.handleSceneActionEvent("available:window,document,refresh,immersive") {
		t.Fatal("handleSceneActionEvent(available:window,document,refresh,immersive) = false, want true")
	}
	if !scene.Active() {
		t.Fatal("scene.Active() = false after lifecycle availability, want true")
	}
	if err := scene.Actions().Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if err := scene.Actions().OpenDocument("/tmp/current-runner.md"); err != nil {
		t.Fatalf("OpenDocument() error = %v", err)
	}
	handle := scene.Handle()
	if got, want := handle.Path, "/tmp/current-runner.md"; got != want {
		t.Fatalf("scene.Handle().Path = %q, want %q", got, want)
	}
	if got, want := handle.DisplayName, "current-runner.md"; got != want {
		t.Fatalf("scene.Handle().DisplayName = %q, want %q", got, want)
	}
	if err := scene.Actions().OpenImmersiveSpace("demo.space"); err != nil {
		t.Fatalf("OpenImmersiveSpace() error = %v", err)
	}
	if err := scene.Actions().OpenWindow("inspector"); err != nil {
		t.Fatalf("OpenWindow(inspector) error = %v, want nil", err)
	}
	if err := scene.Actions().OpenWindow("missing"); err == nil {
		t.Fatal("OpenWindow(missing) unexpectedly succeeded without a registered scene")
	}
	if !scene.runtime.handleSceneActionEvent("unavailable") {
		t.Fatal("handleSceneActionEvent(unavailable) = false, want true")
	}
	if scene.Active() {
		t.Fatal("scene.Active() = true after lifecycle unavailability, want false")
	}
	if scene.Actions().Available() {
		t.Fatal("scene.Actions() should be unavailable after scene disappear")
	}
}

func TestPlanScenesBindsOpenWindowAction(t *testing.T) {
	old := _SUIOpenSceneWindow
	defer func() { _SUIOpenSceneWindow = old }()

	var got string
	_SUIOpenSceneWindow = func(id *byte) int32 {
		got = cStringToGo(id)
		return 1
	}

	scene := DocumentGroup("docs", DocumentConfig{Title: "Docs"}, EmptyView())
	plan, err := planScenes([]Scene{
		scene,
		Window("inspector", AppConfig{Title: "Inspector"}, EmptyView()),
	})
	if err != nil {
		t.Fatalf("planScenes() error = %v", err)
	}
	doc := scene
	if doc.runtime == nil {
		t.Fatal("scene runtime should be available")
	}
	if !doc.runtime.handleSceneActionEvent("available:window,document,refresh,immersive") {
		t.Fatal("handleSceneActionEvent(available:window,document,refresh,immersive) = false, want true")
	}
	if plan.specs[0].actionCallbackID == 0 {
		t.Fatal("plan.specs[0].actionCallbackID = 0, want non-zero")
	}
	if err := doc.Actions().OpenWindow("inspector"); err != nil {
		t.Fatalf("OpenWindow() error = %v", err)
	}
	if got != "inspector" {
		t.Fatalf("open window target = %q, want inspector", got)
	}
}

func TestParseSceneActionEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		wantKind sceneActionEventKind
		wantCaps sceneActionCapabilities
		wantOK   bool
	}{
		{
			name:     "legacy available",
			event:    "available",
			wantKind: sceneActionEventAvailable,
			wantCaps: defaultSceneActionCapabilities(),
			wantOK:   true,
		},
		{
			name:     "capability payload",
			event:    "available:window,document,refresh",
			wantKind: sceneActionEventAvailable,
			wantCaps: sceneActionCapabilities{Window: true, Document: true, Refresh: true, Count: 1},
			wantOK:   true,
		},
		{
			name:     "capability payload with immersive",
			event:    "available:window,document,refresh,immersive",
			wantKind: sceneActionEventAvailable,
			wantCaps: sceneActionCapabilities{Window: true, Document: true, Refresh: true, Immersive: true, Count: 1},
			wantOK:   true,
		},
		{
			name:     "capability payload with count",
			event:    "available:window,document,refresh;count=3",
			wantKind: sceneActionEventAvailable,
			wantCaps: sceneActionCapabilities{Window: true, Document: true, Refresh: true, Count: 3},
			wantOK:   true,
		},
		{
			name:     "capability payload with instance visibility",
			event:    "available:window,document,refresh;count=3;instance=inspector.instance.2;visible=0",
			wantKind: sceneActionEventAvailable,
			wantCaps: sceneActionCapabilities{Window: true, Document: true, Refresh: true, Count: 3, Instance: "inspector.instance.2", Visible: boolPtr(false)},
			wantOK:   true,
		},
		{
			name:     "explicit unavailable",
			event:    "unavailable",
			wantKind: sceneActionEventUnavailable,
			wantCaps: sceneActionCapabilities{Count: 0},
			wantOK:   true,
		},
		{
			name:     "focused",
			event:    "focused",
			wantKind: sceneActionEventFocused,
			wantCaps: sceneActionCapabilities{Count: 1},
			wantOK:   true,
		},
		{
			name:     "focused instance payload",
			event:    "focused:count=2;instance=inspector.instance.1;visible=1",
			wantKind: sceneActionEventFocused,
			wantCaps: sceneActionCapabilities{Count: 2, Instance: "inspector.instance.1", Visible: boolPtr(true)},
			wantOK:   true,
		},
		{
			name:     "blurred",
			event:    "blurred",
			wantKind: sceneActionEventBlurred,
			wantCaps: sceneActionCapabilities{Count: 1},
			wantOK:   true,
		},
		{
			name:     "unknown capability rejected",
			event:    "available:window,teleport",
			wantKind: sceneActionEventUnknown,
			wantCaps: sceneActionCapabilities{},
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotCaps, gotOK := parseSceneActionEvent(tt.event)
			if gotKind != tt.wantKind || !reflect.DeepEqual(gotCaps, tt.wantCaps) || gotOK != tt.wantOK {
				t.Fatalf("parseSceneActionEvent(%q) = (%v, %+v, %v), want (%v, %+v, %v)", tt.event, gotKind, gotCaps, gotOK, tt.wantKind, tt.wantCaps, tt.wantOK)
			}
		})
	}
}

func TestScenePlanAvailabilityInjectsImmersiveSpaceWhenAdvertised(t *testing.T) {
	old := _SUIOpenSceneWindow
	defer func() { _SUIOpenSceneWindow = old }()
	_SUIOpenSceneWindow = func(id *byte) int32 {
		if cStringToGo(id) == "demo.space" {
			return 1
		}
		return 0
	}

	scene := DocumentGroupWithHandle("docs", DocumentConfig{Title: "Docs"}, func(DocumentHandle, SceneActions) View {
		return EmptyView()
	})
	if scene.runtime == nil {
		t.Fatal("scene runtime should be available")
	}
	if !scene.runtime.handleSceneActionEvent("available:window,document,refresh,immersive") {
		t.Fatal("handleSceneActionEvent(available:window,document,refresh,immersive) = false, want true")
	}
	if scene.Actions().ImmersiveSpace == nil {
		t.Fatal("scene.Actions().ImmersiveSpace = nil, want available when the runner advertises immersive preview")
	}
	if err := scene.Actions().OpenImmersiveSpace("demo.space"); err != nil {
		t.Fatalf("OpenImmersiveSpace() error = %v", err)
	}
}

func TestRunScenesMenuBarOnlyCommandsUseSceneRunner(t *testing.T) {
	old := _SUIRunScenePlan
	defer func() { _SUIRunScenePlan = old }()

	var (
		gotPlanJSON  string
		gotViewPtr   uintptr
		gotViewCount int32
	)
	_SUIRunScenePlan = func(plan *byte, viewRefs *uintptr, viewCount int32) {
		gotPlanJSON = cStringToGo(plan)
		gotViewCount = viewCount
		if viewRefs != nil {
			gotViewPtr = *viewRefs
		}
	}

	err := RunScenes(
		MenuBarExtra(MenuBarConfig{Label: "Status", SystemImage: "bolt"}, View{ptr: 44}),
		WithCommands(Commands(
			StandardEditMenu(),
			CommandMenu{
				Title: "Help",
				Items: []CommandItem{
					{
						Title:  "Explain Runner",
						Action: func(CommandContext) {},
					},
				},
			},
		)),
	)
	if err != nil {
		t.Fatalf("RunScenes() error = %v", err)
	}
	if gotPlanJSON == "" {
		t.Fatal("RunScenes() did not invoke the scene-plan runner")
	}
	if got, want := gotViewCount, int32(1); got != want {
		t.Fatalf("view count = %d, want %d", got, want)
	}
	if got, want := gotViewPtr, uintptr(44); got != want {
		t.Fatalf("view ptr = %d, want %d", got, want)
	}

	var run sceneRunPlan
	if err := json.Unmarshal([]byte(gotPlanJSON), &run); err != nil {
		t.Fatalf("json.Unmarshal(planJSON) error = %v", err)
	}
	if got, want := len(run.Scenes), 1; got != want {
		t.Fatalf("len(run.Scenes) = %d, want %d", got, want)
	}
	if got, want := run.Scenes[0].Kind, "menuBar"; got != want {
		t.Fatalf("run.Scenes[0].Kind = %q, want %q", got, want)
	}
	if got, want := run.Scenes[0].Label, "Status"; got != want {
		t.Fatalf("run.Scenes[0].Label = %q, want %q", got, want)
	}
	if got, want := len(run.Commands), 2; got != want {
		t.Fatalf("len(run.Commands) = %d, want %d", got, want)
	}
	if got, want := run.Commands[0].Title, "Edit"; got != want {
		t.Fatalf("run.Commands[0].Title = %q, want %q", got, want)
	}
	if got, want := run.Commands[0].Items[0].SystemAction, string(CommandSystemUndo); got != want {
		t.Fatalf("run.Commands[0].Items[0].SystemAction = %q, want %q", got, want)
	}
	if got, want := run.Commands[1].Title, "Help"; got != want {
		t.Fatalf("run.Commands[1].Title = %q, want %q", got, want)
	}
	if run.Commands[1].Items[0].ActionCallbackID == 0 {
		t.Fatal("help command ActionCallbackID = 0, want non-zero")
	}
}

func cStringToGo(p *byte) string {
	if p == nil {
		return ""
	}
	buf := make([]byte, 0, 16)
	for *p != 0 {
		buf = append(buf, *p)
		p = (*byte)(unsafe.Add(unsafe.Pointer(p), 1))
	}
	return string(buf)
}
