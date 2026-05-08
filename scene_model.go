package swiftui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	errNoScenes           = errors.New("swiftui: no scenes")
	errNilScene           = errors.New("swiftui: nil scene")
	errMultipleMenuExtras = errors.New("swiftui: multiple menu bar scenes are not supported by the current runner")
	errMultipleSettings   = errors.New("swiftui: multiple settings scenes are not supported by the current runner")
	errUnsupportedScene   = errors.New("swiftui: unsupported scene")
	errEmptySceneID       = errors.New("swiftui: scene id must not be empty")
	errEmptyDocumentTitle = errors.New("swiftui: document title must not be empty")
	errDuplicateSceneID   = errors.New("swiftui: duplicate scene id")
)

// Scene is a declarative application scene.
//
// Runtime surface.
//
// Current scenes lower onto the existing macOS runner. The API is intentionally
// narrower than SwiftUI's Scene protocol so it can grow without exposing raw
// protocol conformance across the bridge.
type Scene interface {
	sceneSpec() sceneSpec
}

type sceneKind int

const (
	sceneWindow sceneKind = iota + 1
	sceneDocument
	sceneMenuBar
	sceneSettings
)

// AuxiliaryWindowPolicy describes how the current scene-plan runner should
// launch auxiliary windows beyond the primary window/document scene.
//
// Runtime surface.
type AuxiliaryWindowPolicy string

const (
	// AuxiliaryWindowManual keeps an auxiliary window hidden until an explicit
	// scene action reveals it.
	AuxiliaryWindowManual AuxiliaryWindowPolicy = "manual"

	// AuxiliaryWindowOpenOnLaunch asks the current runner to show the auxiliary
	// window during launch.
	AuxiliaryWindowOpenOnLaunch AuxiliaryWindowPolicy = "openOnLaunch"
)

// DocumentConfig describes one document scene.
//
// Runtime surface.
type DocumentConfig struct {
	Title  string
	Width  int
	Height int
}

// AppConfig converts cfg into an application config for the current runner.
func (cfg DocumentConfig) AppConfig() AppConfig {
	title := strings.TrimSpace(cfg.Title)
	return AppConfig{
		Title:  title,
		Width:  float64(cfg.Width),
		Height: float64(cfg.Height),
	}
}

// DocumentSession describes the current read-only document identity.
//
// Runtime surface.
type DocumentSession struct {
	ID          string
	DisplayName string
	Path        string
}

// DocumentRecent describes one recent document path tracked by the current
// runner-owned document scene.
//
// Runtime surface.
type DocumentRecent struct {
	DisplayName string
	Path        string
}

// DocumentWorkflow describes runner-owned file workflow callbacks for one
// document scene.
//
// Runtime surface.
//
// The current runner owns panels, close-confirmation, and session/path
// mutation. Go code supplies the concrete file-system work through these
// callbacks and can observe approved document close/terminate paths through
// Close.
type DocumentWorkflow struct {
	Open   func(path string) error
	Save   func(session DocumentSession, path string) error
	Export func(session DocumentSession, path string) error
	Import func(path string) error
	// Close runs after dirty-state confirmation succeeds and before the runner
	// commits the close or terminate path for this document window.
	Close func(session DocumentSession) error
}

// DocumentHandle describes the current document-scene identity, mutable state,
// and runner-owned lifecycle actions.
//
// Runtime surface.
type DocumentHandle struct {
	Session DocumentSession

	// ID, DisplayName, and Path mirror Session for source compatibility.
	ID              string
	DisplayName     string
	Path            string
	Dirty           bool
	RecentDocuments []DocumentRecent

	Open     func() error
	OpenPath func(path string) error
	Save     func() error
	SaveAs   func() error
	Revert   func() error
	Export   func() error
	Import   func() error
	Close    func() error
}

// SceneLifecycleState reports the runner-owned lifecycle state for a scene.
//
// Runtime surface.
//
// The zero value is not meaningful; use one of the named states.
type SceneLifecycleState string

const (
	SceneLifecycleUnknown  SceneLifecycleState = "unknown"
	SceneLifecycleInactive SceneLifecycleState = "inactive"
	SceneLifecycleActive   SceneLifecycleState = "active"
)

// SceneRuntimeState reports runner-owned scene state that is explicit but not
// native SwiftUI App/Scene parity.
//
// Runtime surface.
type SceneRuntimeState struct {
	Kind                    string
	ID                      string
	RestorationID           string
	Lifecycle               SceneLifecycleState
	Live                    bool
	WindowInstanceCount     int
	FocusedWindowInstanceID string
	WindowInstances         []SceneWindowInstanceState
	RestoreVisibility       *bool
	AuxiliaryWindowPolicy   AuxiliaryWindowPolicy
	Handle                  DocumentHandle
	Actions                 SceneActions
}

// SceneWindowInstanceState reports one runner-owned window instance tracked for
// a window scene.
//
// Runtime surface.
type SceneWindowInstanceState struct {
	ID      string
	Visible bool
	Focused bool
}

// ActionsAvailable reports whether the runtime state currently has any
// borrowed scene-scoped actions.
func (s SceneRuntimeState) ActionsAvailable() bool {
	return s.Actions.Available()
}

// SceneStatus reports the concrete runner-backed status currently exposed for a
// scene.
//
// Runtime surface.
//
// This is intentionally smaller than full runtime state and does not claim
// native SwiftUI App or Scene parity. It exists so callers can query the
// current scene-plan runner for stable status fields such as lifecycle,
// restoration identity, visibility-restore policy, and auxiliary-window policy.
type SceneStatus struct {
	Kind               string
	ID                 string
	Active             bool
	RestorationID      string
	RestoresVisibility *bool
	AuxiliaryPolicy    AuxiliaryWindowPolicy
}

func normalizedSceneRestorationID(id, fallback string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return strings.TrimSpace(fallback)
}

func normalizeAuxiliaryWindowPolicy(policy AuxiliaryWindowPolicy) AuxiliaryWindowPolicy {
	switch policy {
	case AuxiliaryWindowOpenOnLaunch:
		return AuxiliaryWindowOpenOnLaunch
	case AuxiliaryWindowManual:
		return AuxiliaryWindowManual
	default:
		return AuxiliaryWindowManual
	}
}

func normalizeDocumentSession(session DocumentSession) DocumentSession {
	session.ID = strings.TrimSpace(session.ID)
	session.DisplayName = strings.TrimSpace(session.DisplayName)
	session.Path = strings.TrimSpace(session.Path)
	if session.DisplayName == "" && session.Path != "" {
		if base := filepath.Base(session.Path); base != "." && base != "/" && base != "" {
			session.DisplayName = base
		}
	}
	return session
}

func normalizeDocumentRecent(recent DocumentRecent) DocumentRecent {
	recent.DisplayName = strings.TrimSpace(recent.DisplayName)
	recent.Path = strings.TrimSpace(recent.Path)
	if recent.DisplayName == "" && recent.Path != "" {
		if base := filepath.Base(recent.Path); base != "." && base != "/" && base != "" {
			recent.DisplayName = base
		}
	}
	return recent
}

func normalizeDocumentRecents(recents []DocumentRecent) []DocumentRecent {
	if len(recents) == 0 {
		return nil
	}
	out := make([]DocumentRecent, 0, len(recents))
	seen := make(map[string]bool, len(recents))
	for _, recent := range recents {
		recent = normalizeDocumentRecent(recent)
		if recent.Path == "" || seen[recent.Path] {
			continue
		}
		seen[recent.Path] = true
		out = append(out, recent)
		if len(out) == 8 {
			break
		}
	}
	return out
}

var (
	documentRecentStoreMu     sync.Mutex
	documentRecentStoreLoaded bool
	documentRecentStore       map[string][]DocumentRecent
	documentRecentStorePath   = defaultDocumentRecentStorePath
)

func defaultDocumentRecentStorePath() string {
	dir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "swiftui", "recent-documents.json")
}

func currentDocumentRecentStoreAppID() string {
	name := strings.TrimSpace(filepath.Base(os.Args[0]))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "swiftui"
	}
	return name
}

func normalizeDocumentRecentStoreKey(sceneID string) string {
	sceneID = strings.TrimSpace(sceneID)
	if sceneID == "" {
		return ""
	}
	return currentDocumentRecentStoreAppID() + ":" + sceneID
}

func loadDocumentRecentStoreLocked() map[string][]DocumentRecent {
	if documentRecentStoreLoaded {
		if documentRecentStore == nil {
			documentRecentStore = make(map[string][]DocumentRecent)
		}
		return documentRecentStore
	}
	store := make(map[string][]DocumentRecent)
	path := strings.TrimSpace(documentRecentStorePath())
	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &store)
		}
	}
	for key, recents := range store {
		store[key] = normalizeDocumentRecents(recents)
	}
	documentRecentStore = store
	documentRecentStoreLoaded = true
	return documentRecentStore
}

func writeDocumentRecentStoreLocked() {
	path := strings.TrimSpace(documentRecentStorePath())
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(documentRecentStore, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "recent-documents-*.json")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return
	}
}

func restoreDocumentRecentDocuments(sceneID string, recents []DocumentRecent) []DocumentRecent {
	sceneID = normalizeDocumentRecentStoreKey(sceneID)
	recents = normalizeDocumentRecents(recents)
	if sceneID == "" {
		return recents
	}
	documentRecentStoreMu.Lock()
	store := loadDocumentRecentStoreLocked()
	persisted := append([]DocumentRecent(nil), store[sceneID]...)
	documentRecentStoreMu.Unlock()
	if len(persisted) == 0 {
		return recents
	}
	return normalizeDocumentRecents(append(recents, persisted...))
}

func persistDocumentRecentDocuments(sceneID string, recents []DocumentRecent) {
	sceneID = normalizeDocumentRecentStoreKey(sceneID)
	if sceneID == "" {
		return
	}
	recents = normalizeDocumentRecents(recents)
	documentRecentStoreMu.Lock()
	store := loadDocumentRecentStoreLocked()
	if len(recents) == 0 {
		delete(store, sceneID)
	} else {
		store[sceneID] = recents
	}
	writeDocumentRecentStoreLocked()
	documentRecentStoreMu.Unlock()
}

func hydrateDocumentRecentDocuments(sceneID string, handle DocumentHandle) DocumentHandle {
	handle = normalizeDocumentHandle(handle)
	handle.RecentDocuments = restoreDocumentRecentDocuments(sceneID, handle.RecentDocuments)
	return handle
}

func prependDocumentRecent(recents []DocumentRecent, recent DocumentRecent) []DocumentRecent {
	recent = normalizeDocumentRecent(recent)
	if recent.Path == "" {
		return normalizeDocumentRecents(recents)
	}
	out := make([]DocumentRecent, 0, len(recents)+1)
	out = append(out, recent)
	for _, item := range normalizeDocumentRecents(recents) {
		if item.Path == recent.Path {
			continue
		}
		out = append(out, item)
		if len(out) == 8 {
			break
		}
	}
	return out
}

func normalizeDocumentHandle(handle DocumentHandle) DocumentHandle {
	session := normalizeDocumentSession(handle.Session)
	if session.ID == "" {
		session.ID = strings.TrimSpace(handle.ID)
	}
	if session.DisplayName == "" {
		session.DisplayName = strings.TrimSpace(handle.DisplayName)
	}
	if session.Path == "" {
		session.Path = strings.TrimSpace(handle.Path)
	}
	session = normalizeDocumentSession(session)
	handle.Session = session
	handle.ID = session.ID
	handle.DisplayName = session.DisplayName
	handle.Path = session.Path
	handle.RecentDocuments = normalizeDocumentRecents(handle.RecentDocuments)
	return handle
}

type sceneSpec struct {
	kind sceneKind

	id                string
	actionCallbackID  uintptr
	restorationID     string
	restoreVisibility *bool
	multipleInstances bool

	appConfig             AppConfig
	appView               View
	auxiliaryWindowPolicy AuxiliaryWindowPolicy

	documentConfig   DocumentConfig
	documentHandle   DocumentHandle
	documentActions  SceneActions
	documentWorkflow DocumentWorkflow
	documentOpenID   uintptr
	documentSaveID   uintptr
	documentExportID uintptr
	documentImportID uintptr
	documentCloseID  uintptr
	documentDirtyID  uintptr

	menuConfig MenuBarConfig
	menuView   View
}

// WindowGroupScene describes one window group.
//
// Runtime surface.
type WindowGroupScene struct {
	id                string
	config            AppConfig
	content           View
	builder           func(SceneStatus) View
	actions           SceneActions
	runtime           *windowSceneRuntime
	restorationID     string
	multipleInstances bool
	auxiliaryPolicy   AuxiliaryWindowPolicy
	restoreVisibility *bool
}

// WindowGroup declares a window group scene.
func WindowGroup(id string, config AppConfig, content View) WindowGroupScene {
	id = strings.TrimSpace(id)
	return WindowGroupScene{
		id:                id,
		config:            config,
		content:           content,
		runtime:           newWindowSceneRuntime(id),
		multipleInstances: true,
	}
}

// WindowGroupWithStatus declares a window group scene from a status-aware builder.
func WindowGroupWithStatus(id string, config AppConfig, builder func(SceneStatus) View) WindowGroupScene {
	id = strings.TrimSpace(id)
	scene := WindowGroupScene{
		id:                id,
		config:            config,
		builder:           builder,
		runtime:           newWindowSceneRuntime(id),
		multipleInstances: true,
	}
	if builder != nil {
		scene.content = builder(scene.Status())
	}
	return scene
}

func (s WindowGroupScene) sceneSpec() sceneSpec {
	actionCallbackID := uintptr(0)
	content := s.content
	if s.runtime != nil {
		actionCallbackID = s.runtime.sceneActionCallbackID
		s.runtime.setActions(s.actions)
	}
	if s.builder != nil {
		if s.runtime != nil && s.runtime.revision != nil {
			content = DynamicView(s.runtime.revision, func(int) View {
				return s.builder(s.Status())
			})
		} else {
			content = s.builder(s.Status())
		}
	}
	return sceneSpec{
		kind:                  sceneWindow,
		id:                    s.id,
		actionCallbackID:      actionCallbackID,
		restorationID:         normalizedSceneRestorationID(s.restorationID, s.id),
		restoreVisibility:     s.restoreVisibility,
		multipleInstances:     s.multipleInstances,
		appConfig:             s.config,
		appView:               content,
		auxiliaryWindowPolicy: normalizeAuxiliaryWindowPolicy(s.auxiliaryPolicy),
	}
}

// ID reports the stable scene identifier.
func (s WindowGroupScene) ID() string { return s.id }

// Config reports the immutable window config.
func (s WindowGroupScene) Config() AppConfig { return s.config }

// RestorationID reports the stable restoration identity for the current runner.
func (s WindowGroupScene) RestorationID() string {
	return normalizedSceneRestorationID(s.restorationID, s.id)
}

// Actions reports the borrowed scene-scoped capabilities for the window scene.
func (s WindowGroupScene) Actions() SceneActions {
	if s.runtime != nil {
		return s.runtime.actions()
	}
	return s.actions
}

// AuxiliaryWindowPolicy reports how the current runner should launch the
// window when it is not the primary window/document scene.
func (s WindowGroupScene) AuxiliaryWindowPolicy() AuxiliaryWindowPolicy {
	return normalizeAuxiliaryWindowPolicy(s.auxiliaryPolicy)
}

// WithVisibilityRestore returns a copy of s with explicit visibility
// restoration policy for the current runner.
func (s WindowGroupScene) WithVisibilityRestore(restore bool) WindowGroupScene {
	s.restoreVisibility = boolPtr(restore)
	return s
}

// WithRestorationID returns a copy of s with an explicit restoration identity.
func (s WindowGroupScene) WithRestorationID(id string) WindowGroupScene {
	s.restorationID = strings.TrimSpace(id)
	return s
}

// WithAuxiliaryWindowPolicy returns a copy of s with an explicit auxiliary
// launch policy for the current scene-plan runner.
func (s WindowGroupScene) WithAuxiliaryWindowPolicy(policy AuxiliaryWindowPolicy) WindowGroupScene {
	s.auxiliaryPolicy = normalizeAuxiliaryWindowPolicy(policy)
	return s
}

// RuntimeState reports the current runner-owned scene state.
func (s WindowGroupScene) RuntimeState() SceneRuntimeState {
	lifecycle := SceneLifecycleInactive
	live := false
	instanceCount := 0
	focusedWindowInstanceID := ""
	var windowInstances []SceneWindowInstanceState
	actions := s.actions
	if s.runtime == nil {
		lifecycle = SceneLifecycleUnknown
	} else {
		var active bool
		live, active, instanceCount, focusedWindowInstanceID, windowInstances = s.runtime.snapshotState()
		if active {
			lifecycle = SceneLifecycleActive
		}
		actions = s.runtime.actions()
	}
	return SceneRuntimeState{
		Kind:                    sceneKindLabel(sceneWindow),
		ID:                      s.id,
		RestorationID:           normalizedSceneRestorationID(s.restorationID, s.id),
		Lifecycle:               lifecycle,
		Live:                    live,
		WindowInstanceCount:     instanceCount,
		FocusedWindowInstanceID: focusedWindowInstanceID,
		WindowInstances:         windowInstances,
		RestoreVisibility:       s.restoreVisibility,
		AuxiliaryWindowPolicy:   normalizeAuxiliaryWindowPolicy(s.auxiliaryPolicy),
		Actions:                 actions,
	}
}

// Status reports the current runner-backed status for the window scene.
func (s WindowGroupScene) Status() SceneStatus {
	active := false
	if s.runtime != nil {
		active = s.runtime.activeScene()
	}
	return SceneStatus{
		Kind:               sceneKindLabel(sceneWindow),
		ID:                 s.id,
		Active:             active,
		RestorationID:      normalizedSceneRestorationID(s.restorationID, s.id),
		RestoresVisibility: s.restoreVisibility,
		AuxiliaryPolicy:    normalizeAuxiliaryWindowPolicy(s.auxiliaryPolicy),
	}
}

// WithActions returns a copy of s with borrowed scene-scoped capabilities.
//
// Runtime surface.
func (s WindowGroupScene) WithActions(actions SceneActions) WindowGroupScene {
	s.actions = actions
	if s.runtime != nil {
		s.runtime.setActions(actions)
	}
	return s
}

// DocumentGroupScene describes one document group.
//
// Runtime surface.
type DocumentGroupScene struct {
	id                string
	config            DocumentConfig
	handle            DocumentHandle
	workflow          DocumentWorkflow
	actions           SceneActions
	content           View
	builder           func(DocumentHandle, SceneActions) View
	runtime           *documentSceneRuntime
	restorationID     string
	restoreVisibility *bool
}

type documentSceneRuntime struct {
	mu                     sync.RWMutex
	handle                 DocumentHandle
	workflow               DocumentWorkflow
	live                   bool
	focused                bool
	manualActions          SceneActions
	injectedActions        SceneActions
	revision               *IntState
	sceneID                string
	sceneActionCallbackID  uintptr
	documentOpenCallback   uintptr
	documentSaveCallback   uintptr
	documentExportCallback uintptr
	documentImportCallback uintptr
	documentCloseCallback  uintptr
	documentDirtyCallback  uintptr
}

type windowSceneRuntime struct {
	mu                    sync.RWMutex
	live                  bool
	focused               bool
	instanceCount         int
	focusedInstanceID     string
	instances             map[string]SceneWindowInstanceState
	manualActions         SceneActions
	injectedActions       SceneActions
	revision              *IntState
	sceneID               string
	sceneActionCallbackID uintptr
}

type sceneActionEventKind int

const (
	sceneActionEventUnknown sceneActionEventKind = iota
	sceneActionEventAvailable
	sceneActionEventUnavailable
	sceneActionEventFocused
	sceneActionEventBlurred
)

type sceneActionCapabilities struct {
	Window    bool
	Document  bool
	Refresh   bool
	Immersive bool
	Count     int
	Instance  string
	Visible   *bool
}

func defaultSceneActionCapabilities() sceneActionCapabilities {
	return sceneActionCapabilities{
		Window:   true,
		Document: true,
		Refresh:  true,
		Count:    1,
	}
}

func parseSceneActionCapabilityPayload(payload string) (sceneActionCapabilities, bool) {
	caps := sceneActionCapabilities{}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return caps, true
	}
	for _, part := range strings.Split(payload, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if key, value, ok := strings.Cut(part, "="); ok {
			switch strings.TrimSpace(key) {
			case "count", "instances":
				value = strings.TrimSpace(value)
				if value == "" {
					return sceneActionCapabilities{}, false
				}
				n, err := strconv.Atoi(value)
				if err != nil || n < 0 {
					return sceneActionCapabilities{}, false
				}
				caps.Count = n
			case "instance":
				value = strings.TrimSpace(value)
				if value == "" {
					return sceneActionCapabilities{}, false
				}
				caps.Instance = value
			case "visible":
				visible, ok := parseSceneActionVisibility(value)
				if !ok {
					return sceneActionCapabilities{}, false
				}
				caps.Visible = &visible
			default:
				return sceneActionCapabilities{}, false
			}
			continue
		}
		for _, capability := range strings.Split(part, ",") {
			switch strings.TrimSpace(capability) {
			case "", "available":
				continue
			case "window":
				caps.Window = true
			case "document":
				caps.Document = true
			case "refresh":
				caps.Refresh = true
			case "immersive":
				caps.Immersive = true
			default:
				return sceneActionCapabilities{}, false
			}
		}
	}
	return caps, true
}

func parseSceneActionVisibility(value string) (bool, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "true", "yes":
		return true, true
	case "0", "false", "no":
		return false, true
	default:
		return false, false
	}
}

func parseSceneActionEvent(event string) (sceneActionEventKind, sceneActionCapabilities, bool) {
	event = strings.TrimSpace(event)
	if event == "" {
		return sceneActionEventUnknown, sceneActionCapabilities{}, false
	}
	sep := strings.IndexAny(event, ":;")
	if sep < 0 {
		switch event {
		case "available":
			return sceneActionEventAvailable, defaultSceneActionCapabilities(), true
		case "unavailable":
			return sceneActionEventUnavailable, sceneActionCapabilities{Count: 0}, true
		case "focused":
			return sceneActionEventFocused, sceneActionCapabilities{Count: 1}, true
		case "blurred":
			return sceneActionEventBlurred, sceneActionCapabilities{Count: 1}, true
		default:
			return sceneActionEventUnknown, sceneActionCapabilities{}, false
		}
	}
	head := event[:sep]
	tail := event[sep+1:]
	switch strings.TrimSpace(head) {
	case "available":
		caps, ok := parseSceneActionCapabilityPayload(tail)
		if !ok {
			return sceneActionEventUnknown, sceneActionCapabilities{}, false
		}
		if caps.Count == 0 {
			caps.Count = 1
		}
		return sceneActionEventAvailable, caps, true
	case "unavailable":
		caps, ok := parseSceneActionCapabilityPayload(tail)
		if !ok {
			return sceneActionEventUnknown, sceneActionCapabilities{}, false
		}
		if caps.Count == 0 {
			caps.Count = 0
		}
		return sceneActionEventUnavailable, caps, true
	case "focused":
		caps, ok := parseSceneActionCapabilityPayload(tail)
		if !ok {
			return sceneActionEventUnknown, sceneActionCapabilities{}, false
		}
		if caps.Count == 0 {
			caps.Count = 1
		}
		return sceneActionEventFocused, caps, true
	case "blurred":
		caps, ok := parseSceneActionCapabilityPayload(tail)
		if !ok {
			return sceneActionEventUnknown, sceneActionCapabilities{}, false
		}
		if caps.Count == 0 {
			caps.Count = 1
		}
		return sceneActionEventBlurred, caps, true
	default:
		return sceneActionEventUnknown, sceneActionCapabilities{}, false
	}
}

func mergeSceneActions(manual, injected SceneActions) SceneActions {
	merged := injected
	if manual.Window != nil {
		merged.Window = manual.Window
	}
	if manual.Document != nil {
		merged.Document = manual.Document
	}
	if manual.RefreshScene != nil {
		merged.RefreshScene = manual.RefreshScene
	}
	if manual.ImmersiveSpace != nil {
		merged.ImmersiveSpace = manual.ImmersiveSpace
	}
	return merged
}

// DocumentGroup declares a document group scene from a concrete scene view.
func DocumentGroup(id string, config DocumentConfig, content View) DocumentGroupScene {
	id = strings.TrimSpace(id)
	config.Title = strings.TrimSpace(config.Title)
	handle := hydrateDocumentRecentDocuments(id, DocumentHandle{
		Session: DocumentSession{
			ID:          id,
			DisplayName: config.Title,
		},
	})
	scene := DocumentGroupScene{id: id, config: config, handle: handle, content: content}
	scene.runtime = newDocumentSceneRuntime(handle)
	return scene
}

// DocumentGroupWithHandle declares a document group scene from a handle-aware builder.
func DocumentGroupWithHandle(id string, config DocumentConfig, builder func(DocumentHandle, SceneActions) View) DocumentGroupScene {
	id = strings.TrimSpace(id)
	config.Title = strings.TrimSpace(config.Title)
	handle := hydrateDocumentRecentDocuments(id, DocumentHandle{
		Session: DocumentSession{
			ID:          id,
			DisplayName: config.Title,
		},
	})
	scene := DocumentGroupScene{id: id, config: config, handle: handle, builder: builder}
	scene.runtime = newDocumentSceneRuntime(handle)
	var content View
	if builder != nil {
		content = builder(handle, SceneActions{})
	}
	scene.content = content
	return scene
}

func (s DocumentGroupScene) sceneSpec() sceneSpec {
	content := s.content
	actions := s.actions
	handle := normalizeDocumentHandle(s.handle)
	workflow := s.workflow
	var openID, saveID, exportID, importID, closeID, dirtyID uintptr
	if s.runtime != nil {
		s.runtime.setActions(actions)
		s.runtime.setWorkflow(workflow)
		handle, actions = s.runtime.snapshot()
		openID, saveID, exportID, importID, closeID, dirtyID = s.runtime.workflowCallbackIDs()
	}
	if s.builder != nil {
		if s.runtime != nil && s.runtime.revision != nil {
			content = DynamicView(s.runtime.revision, func(int) View {
				handle, actions := s.runtime.snapshot()
				return s.builder(handle, actions)
			})
		} else {
			content = s.builder(handle, actions)
		}
	}
	return sceneSpec{
		kind:              sceneDocument,
		id:                s.id,
		actionCallbackID:  s.runtime.sceneActionCallbackID,
		restorationID:     normalizedSceneRestorationID(s.restorationID, s.id),
		restoreVisibility: s.restoreVisibility,
		appConfig:         s.config.AppConfig(),
		appView:           content,
		documentConfig:    s.config,
		documentHandle:    handle,
		documentActions:   actions,
		documentWorkflow:  workflow,
		documentOpenID:    openID,
		documentSaveID:    saveID,
		documentExportID:  exportID,
		documentImportID:  importID,
		documentCloseID:   closeID,
		documentDirtyID:   dirtyID,
	}
}

// ID reports the stable scene identifier.
func (s DocumentGroupScene) ID() string { return s.id }

// Config reports the immutable document config.
func (s DocumentGroupScene) Config() DocumentConfig { return s.config }

// RestorationID reports the stable restoration identity for the current runner.
func (s DocumentGroupScene) RestorationID() string {
	return normalizedSceneRestorationID(s.restorationID, s.id)
}

// Handle reports the current document handle.
func (s DocumentGroupScene) Handle() DocumentHandle {
	if s.runtime != nil {
		handle, _ := s.runtime.snapshot()
		return handle
	}
	return normalizeDocumentHandle(s.handle)
}

// Actions reports the borrowed scene-scoped capabilities.
func (s DocumentGroupScene) Actions() SceneActions {
	if s.runtime != nil {
		_, actions := s.runtime.snapshot()
		return actions
	}
	return s.actions
}

// Active reports whether the current runner considers the scene live.
func (s DocumentGroupScene) Active() bool {
	if s.runtime == nil {
		return false
	}
	return s.runtime.activeScene()
}

// RuntimeState reports the current runner-owned document scene state.
func (s DocumentGroupScene) RuntimeState() SceneRuntimeState {
	handle, actions := s.handle, s.actions
	if s.runtime != nil {
		handle, actions = s.runtime.snapshot()
	}
	lifecycle := SceneLifecycleInactive
	live := false
	if s.runtime != nil {
		live = s.runtime.liveScene()
	}
	if s.Active() {
		lifecycle = SceneLifecycleActive
	}
	return SceneRuntimeState{
		Kind:              sceneKindLabel(sceneDocument),
		ID:                s.id,
		RestorationID:     normalizedSceneRestorationID(s.restorationID, s.id),
		Lifecycle:         lifecycle,
		Live:              live,
		RestoreVisibility: s.restoreVisibility,
		Handle:            handle,
		Actions:           actions,
	}
}

// Status reports the current runner-backed status for the document scene.
//
// This is a concrete query surface over the current scene-plan runner state,
// not a claim of native SwiftUI Scene lifecycle parity.
func (s DocumentGroupScene) Status() SceneStatus {
	return SceneStatus{
		Kind:               sceneKindLabel(sceneDocument),
		ID:                 s.id,
		Active:             s.Active(),
		RestorationID:      normalizedSceneRestorationID(s.restorationID, s.id),
		RestoresVisibility: s.restoreVisibility,
		AuxiliaryPolicy:    AuxiliaryWindowManual,
	}
}

// WithHandle returns a copy of s with a concrete document handle override.
func (s DocumentGroupScene) WithHandle(handle DocumentHandle) DocumentGroupScene {
	handle = normalizeDocumentHandle(handle)
	if handle.Session.ID == "" {
		handle.Session.ID = s.id
	}
	if handle.Session.DisplayName == "" {
		handle.Session.DisplayName = s.config.Title
	}
	handle = hydrateDocumentRecentDocuments(s.id, handle)
	s.handle = handle
	if s.runtime != nil {
		s.runtime.setHandle(handle)
	}
	if s.builder != nil {
		s.content = s.builder(s.handle, s.actions)
	}
	return s
}

// SetHandle replaces the current runtime-owned document handle.
//
// Runtime surface.
func (s DocumentGroupScene) SetHandle(handle DocumentHandle) {
	handle = normalizeDocumentHandle(handle)
	if handle.Session.ID == "" {
		handle.Session.ID = s.id
	}
	if handle.Session.DisplayName == "" {
		handle.Session.DisplayName = s.config.Title
	}
	handle = hydrateDocumentRecentDocuments(s.id, handle)
	if s.runtime != nil {
		s.runtime.setHandle(handle)
	}
}

// SetDirty updates the runtime-owned dirty flag for the current document
// session.
//
// Runtime surface.
func (s DocumentGroupScene) SetDirty(dirty bool) {
	if s.runtime == nil {
		return
	}
	s.runtime.setDirty(dirty)
}

// WithDocumentWorkflow returns a copy of s with runner-owned file workflow
// callbacks.
//
// Runtime surface.
func (s DocumentGroupScene) WithDocumentWorkflow(workflow DocumentWorkflow) DocumentGroupScene {
	s.workflow = workflow
	if s.runtime != nil {
		s.runtime.setWorkflow(workflow)
	}
	if s.builder != nil && s.runtime != nil {
		handle, actions := s.runtime.snapshot()
		s.content = s.builder(handle, actions)
	}
	return s
}

// WithActions returns a copy of s with borrowed scene-scoped capabilities.
func (s DocumentGroupScene) WithActions(actions SceneActions) DocumentGroupScene {
	s.actions = actions
	if s.runtime != nil {
		s.runtime.setActions(actions)
	}
	if s.builder != nil {
		s.content = s.builder(s.handle, s.actions)
	}
	return s
}

// WithWindowAction binds a borrowed OpenWindowAction to the scene.
func (s DocumentGroupScene) WithWindowAction(action OpenWindowAction) DocumentGroupScene {
	s.actions.Window = action
	if s.runtime != nil {
		s.runtime.setActions(s.actions)
	}
	if s.builder != nil {
		s.content = s.builder(s.handle, s.actions)
	}
	return s
}

// WithDocumentAction binds a borrowed OpenDocumentAction to the scene.
func (s DocumentGroupScene) WithDocumentAction(action OpenDocumentAction) DocumentGroupScene {
	s.actions.Document = action
	if s.runtime != nil {
		s.runtime.setActions(s.actions)
	}
	if s.builder != nil {
		s.content = s.builder(s.handle, s.actions)
	}
	return s
}

// WithRefreshAction binds a borrowed RefreshAction to the scene.
func (s DocumentGroupScene) WithRefreshAction(action RefreshAction) DocumentGroupScene {
	s.actions.RefreshScene = action
	if s.runtime != nil {
		s.runtime.setActions(s.actions)
	}
	if s.builder != nil {
		s.content = s.builder(s.handle, s.actions)
	}
	return s
}

// WithRestorationID returns a copy of s with an explicit restoration identity.
func (s DocumentGroupScene) WithRestorationID(id string) DocumentGroupScene {
	s.restorationID = strings.TrimSpace(id)
	return s
}

// WithVisibilityRestore returns a copy of s with explicit visibility
// restoration policy for the current runner.
func (s DocumentGroupScene) WithVisibilityRestore(restore bool) DocumentGroupScene {
	s.restoreVisibility = boolPtr(restore)
	return s
}

// Window declares a single-window scene.
func Window(id string, config AppConfig, content View) WindowGroupScene {
	id = strings.TrimSpace(id)
	scene := WindowGroup(id, config, content)
	scene.multipleInstances = false
	return scene
}

// MenuBarExtraScene describes one menu bar extra scene.
//
// Runtime surface.
type MenuBarExtraScene struct {
	config  MenuBarConfig
	content View
}

// MenuBarExtra declares a menu bar extra scene.
func MenuBarExtra(config MenuBarConfig, content View) MenuBarExtraScene {
	return MenuBarExtraScene{config: config, content: content}
}

func (s MenuBarExtraScene) sceneSpec() sceneSpec {
	return sceneSpec{
		kind:       sceneMenuBar,
		menuConfig: s.config,
		menuView:   s.content,
	}
}

// SettingsScene describes one app settings scene.
//
// Runtime surface.
//
// The current runner lowers Settings onto an AppKit-owned preferences window
// and installs a standard app-menu Settings item that reveals it.
type SettingsScene struct {
	config            AppConfig
	content           View
	restorationID     string
	restoreVisibility *bool
}

const settingsSceneID = "settings"

// Settings declares the application's settings scene.
//
// Settings is intentionally concrete. It lowers onto the current scene-plan
// runner as a dedicated settings window rather than claiming native SwiftUI
// Settings scene parity.
func Settings(config AppConfig, content View) SettingsScene {
	config.Title = strings.TrimSpace(config.Title)
	if config.Title == "" {
		config.Title = "Settings"
	}
	if config.Width == 0 {
		config.Width = 660
	}
	if config.Height == 0 {
		config.Height = 620
	}
	return SettingsScene{
		config:            config,
		content:           content,
		restoreVisibility: boolPtr(false),
	}
}

func (s SettingsScene) sceneSpec() sceneSpec {
	return sceneSpec{
		kind:              sceneSettings,
		id:                settingsSceneID,
		restorationID:     normalizedSceneRestorationID(s.restorationID, settingsSceneID),
		restoreVisibility: s.restoreVisibility,
		appConfig:         s.config,
		appView:           s.content,
	}
}

// ID reports the stable runner-owned settings scene identifier.
func (s SettingsScene) ID() string { return settingsSceneID }

// Config reports the immutable settings window config.
func (s SettingsScene) Config() AppConfig { return s.config }

// RestorationID reports the stable restoration identity for the current runner.
func (s SettingsScene) RestorationID() string {
	return normalizedSceneRestorationID(s.restorationID, settingsSceneID)
}

// WithRestorationID returns a copy of s with an explicit restoration identity.
func (s SettingsScene) WithRestorationID(id string) SettingsScene {
	s.restorationID = strings.TrimSpace(id)
	return s
}

// WithVisibilityRestore returns a copy of s with explicit visibility
// restoration policy for the current runner.
func (s SettingsScene) WithVisibilityRestore(restore bool) SettingsScene {
	s.restoreVisibility = boolPtr(restore)
	return s
}

// RunScenes starts the app with declarative scenes.
//
// Runtime surface.
//
// On runtimes with the generated scene-plan bridge, RunScenes supports
// multiple window and document scenes plus one menu bar extra. Older bridges
// fall back to the legacy single-window runner.
//
// Options may be passed to attach commands or lifecycle callbacks:
//
//	swiftui.RunScenes(
//	    myWindow,
//	    swiftui.WithCommands(swiftui.Commands(swiftui.StandardEditMenu())),
//	    swiftui.WithLifecycle(lifecycle),
//	)
func RunScenes(scenes ...Scene) error {
	var opts []SceneOption
	var filtered []Scene
	for _, s := range scenes {
		if opt, ok := s.(SceneOption); ok {
			opts = append(opts, opt)
		} else {
			filtered = append(filtered, s)
		}
	}
	plan, err := planScenes(filtered)
	if err != nil {
		return err
	}
	for _, opt := range opts {
		opt.applySceneOption(&plan)
	}
	if _SUIRunScenePlan != nil {
		planJSON, views, err := plan.marshal()
		if err != nil {
			return err
		}
		runScenePlan(planJSON, views)
		return nil
	}
	return legacyRunScenePlan(plan)
}

// SceneOption configures the scene plan passed to RunScenes.
//
// Runtime surface.
type SceneOption interface {
	Scene
	applySceneOption(plan *scenePlan)
}

type commandsOption struct {
	commands AppCommands
}

func (o commandsOption) sceneSpec() sceneSpec { return sceneSpec{} }

func (o commandsOption) applySceneOption(plan *scenePlan) {
	plan.commands = &o.commands
}

// WithCommands attaches runner-owned app command menus to the scene plan.
//
// Runtime surface.
//
// The current runner recomputes Enabled predicates on menu open, app
// activation changes, focused-scene changes, and scene teardown. Command
// actions run on the main thread; long-running work should hop to a goroutine.
func WithCommands(cmds AppCommands) SceneOption {
	return commandsOption{commands: cmds}
}

type lifecycleOption struct {
	lifecycle AppLifecycle
}

func (o lifecycleOption) sceneSpec() sceneSpec { return sceneSpec{} }

func (o lifecycleOption) applySceneOption(plan *scenePlan) {
	plan.lifecycle = &o.lifecycle
}

// WithLifecycle attaches lifecycle callbacks to the scene plan.
func WithLifecycle(lc AppLifecycle) SceneOption {
	return lifecycleOption{lifecycle: lc}
}

type scenePlan struct {
	specs     []sceneSpec
	commands  *AppCommands
	lifecycle *AppLifecycle
}

type sceneRunPlan struct {
	Scenes    []sceneRunPlanScene    `json:"scenes"`
	Commands  []sceneRunPlanCommand  `json:"commands,omitempty"`
	Lifecycle *sceneRunPlanLifecycle `json:"lifecycle,omitempty"`
}

type sceneRunPlanCommand struct {
	Title string                    `json:"title"`
	Items []sceneRunPlanCommandItem `json:"items"`
}

type sceneRunPlanCommandItem struct {
	Kind              string                    `json:"kind"`
	Title             string                    `json:"title,omitempty"`
	ShortcutKey       string                    `json:"shortcutKey,omitempty"`
	ShortcutModifiers uint64                    `json:"shortcutModifiers,omitempty"`
	SystemAction      string                    `json:"systemAction,omitempty"`
	ActionCallbackID  uint64                    `json:"actionCallbackID,omitempty"`
	EnabledCallbackID uint64                    `json:"enabledCallbackID,omitempty"`
	Children          []sceneRunPlanCommandItem `json:"children,omitempty"`
}

type sceneRunPlanLifecycle struct {
	DidFinishLaunchingCallbackID uint64 `json:"didFinishLaunchingCallbackID,omitempty"`
	DidBecomeActiveCallbackID    uint64 `json:"didBecomeActiveCallbackID,omitempty"`
	DidResignActiveCallbackID    uint64 `json:"didResignActiveCallbackID,omitempty"`
	ShouldTerminateCallbackID    uint64 `json:"shouldTerminateCallbackID,omitempty"`
	WillTerminateCallbackID      uint64 `json:"willTerminateCallbackID,omitempty"`
}

type sceneRunPlanScene struct {
	Kind                     string  `json:"kind"`
	ID                       string  `json:"id,omitempty"`
	RestorationID            string  `json:"restorationID,omitempty"`
	Title                    string  `json:"title,omitempty"`
	Width                    float64 `json:"width,omitempty"`
	Height                   float64 `json:"height,omitempty"`
	OpenOnLaunch             bool    `json:"openOnLaunch"`
	RestoreVisibility        *bool   `json:"restoreVisibility,omitempty"`
	MultipleInstances        bool    `json:"multipleInstances"`
	AuxiliaryWindowMode      string  `json:"auxiliaryWindowMode,omitempty"`
	ActionCallbackID         uint64  `json:"actionCallbackID,omitempty"`
	DocumentDisplayName      string  `json:"documentDisplayName,omitempty"`
	DocumentPath             string  `json:"documentPath,omitempty"`
	DocumentDirty            bool    `json:"documentDirty,omitempty"`
	DocumentOpenCallbackID   uint64  `json:"documentOpenCallbackID,omitempty"`
	DocumentSaveCallbackID   uint64  `json:"documentSaveCallbackID,omitempty"`
	DocumentExportCallbackID uint64  `json:"documentExportCallbackID,omitempty"`
	DocumentImportCallbackID uint64  `json:"documentImportCallbackID,omitempty"`
	DocumentCloseCallbackID  uint64  `json:"documentCloseCallbackID,omitempty"`
	DocumentDirtyCallbackID  uint64  `json:"documentDirtyCallbackID,omitempty"`
	Label                    string  `json:"label,omitempty"`
	SystemImage              string  `json:"systemImage,omitempty"`
	ViewIndex                int     `json:"viewIndex"`
}

func planScenes(scenes []Scene) (scenePlan, error) {
	if len(scenes) == 0 {
		return scenePlan{}, errNoScenes
	}
	var plan scenePlan
	seen := make(map[string]struct{}, len(scenes))
	hasMenuBar := false
	hasSettings := false
	for _, scene := range scenes {
		if scene == nil {
			return scenePlan{}, errNilScene
		}
		spec := scene.sceneSpec()
		switch spec.kind {
		case sceneWindow, sceneDocument:
			if strings.TrimSpace(spec.id) == "" {
				return scenePlan{}, errEmptySceneID
			}
			if spec.kind == sceneDocument && strings.TrimSpace(spec.documentConfig.Title) == "" {
				return scenePlan{}, errEmptyDocumentTitle
			}
			if _, ok := seen[spec.id]; ok {
				return scenePlan{}, errDuplicateSceneID
			}
			seen[spec.id] = struct{}{}
			plan.specs = append(plan.specs, spec)
		case sceneMenuBar:
			if hasMenuBar {
				return scenePlan{}, errMultipleMenuExtras
			}
			hasMenuBar = true
			plan.specs = append(plan.specs, spec)
		case sceneSettings:
			if hasSettings {
				return scenePlan{}, errMultipleSettings
			}
			hasSettings = true
			plan.specs = append(plan.specs, spec)
		default:
			return scenePlan{}, errUnsupportedScene
		}
	}
	if len(plan.specs) == 0 {
		return scenePlan{}, errNoScenes
	}
	return plan, nil
}

func marshalCommandItems(items []CommandItem) []sceneRunPlanCommandItem {
	out := make([]sceneRunPlanCommandItem, 0, len(items))
	for _, item := range items {
		if item.IsSeparator() {
			out = append(out, sceneRunPlanCommandItem{Kind: "separator"})
			continue
		}
		planItem := sceneRunPlanCommandItem{
			Kind:              "item",
			Title:             item.Title,
			ShortcutKey:       item.Shortcut.Key,
			ShortcutModifiers: uint64(item.Shortcut.Modifiers),
			SystemAction:      string(item.SystemAction),
		}
		if item.Action != nil {
			id := registerCommandCallback(func() int32 {
				item.Action(CommandContext{})
				return 1
			})
			planItem.ActionCallbackID = uint64(id)
		}
		if item.Enabled != nil {
			id := registerCommandCallback(func() int32 {
				if item.Enabled() {
					return 1
				}
				return 0
			})
			planItem.EnabledCallbackID = uint64(id)
		}
		if item.IsSubmenu() {
			planItem.Children = marshalCommandItems(item.Children)
		}
		out = append(out, planItem)
	}
	return out
}

func (p scenePlan) marshal() (string, []uintptr, error) {
	run := sceneRunPlan{
		Scenes: make([]sceneRunPlanScene, 0, len(p.specs)),
	}
	if p.commands != nil {
		for _, menu := range p.commands.Menus {
			run.Commands = append(run.Commands, sceneRunPlanCommand{
				Title: menu.Title,
				Items: marshalCommandItems(menu.Items),
			})
		}
	}
	if p.lifecycle != nil {
		lc := &sceneRunPlanLifecycle{}
		if p.lifecycle.OnLaunched != nil {
			lc.DidFinishLaunchingCallbackID = uint64(registerCallback(p.lifecycle.OnLaunched))
		}
		if p.lifecycle.OnActivate != nil {
			lc.DidBecomeActiveCallbackID = uint64(registerCallback(p.lifecycle.OnActivate))
		}
		if p.lifecycle.OnResignActive != nil {
			lc.DidResignActiveCallbackID = uint64(registerCallback(p.lifecycle.OnResignActive))
		}
		if p.lifecycle.ShouldTerminate != nil {
			fn := p.lifecycle.ShouldTerminate
			lc.ShouldTerminateCallbackID = uint64(registerCommandCallback(func() int32 {
				if fn() {
					return 1
				}
				return 0
			}))
		}
		if p.lifecycle.OnTerminate != nil {
			lc.WillTerminateCallbackID = uint64(registerCallback(p.lifecycle.OnTerminate))
		}
		run.Lifecycle = lc
	}
	views := make([]uintptr, 0, len(p.specs))
	windowIndex := 0
	launchPrimaryWindow := false
	for _, spec := range p.specs {
		if spec.kind == sceneWindow || spec.kind == sceneDocument {
			launchPrimaryWindow = true
			break
		}
	}
	for _, spec := range p.specs {
		switch spec.kind {
		case sceneWindow, sceneDocument, sceneSettings:
			viewIndex := len(views)
			views = append(views, spec.appView.ptr)
			openOnLaunch := windowIndex == 0
			auxiliaryWindowMode := ""
			if spec.kind == sceneSettings {
				openOnLaunch = !launchPrimaryWindow
			} else if windowIndex > 0 {
				auxiliaryWindowMode = string(normalizeAuxiliaryWindowPolicy(spec.auxiliaryWindowPolicy))
				openOnLaunch = auxiliaryWindowMode == string(AuxiliaryWindowOpenOnLaunch)
			}
			if spec.kind != sceneSettings {
				windowIndex++
			}
			scenePlanScene := sceneRunPlanScene{
				Kind:                sceneKindLabel(spec.kind),
				ID:                  spec.id,
				RestorationID:       normalizedSceneRestorationID(spec.restorationID, spec.id),
				Title:               spec.appConfig.Title,
				Width:               spec.appConfig.Width,
				Height:              spec.appConfig.Height,
				OpenOnLaunch:        openOnLaunch,
				RestoreVisibility:   spec.restoreVisibility,
				MultipleInstances:   spec.multipleInstances,
				AuxiliaryWindowMode: auxiliaryWindowMode,
				ActionCallbackID:    uint64(spec.actionCallbackID),
				ViewIndex:           viewIndex,
			}
			if spec.kind == sceneDocument {
				handle := normalizeDocumentHandle(spec.documentHandle)
				scenePlanScene.DocumentDisplayName = handle.Session.DisplayName
				scenePlanScene.DocumentPath = handle.Session.Path
				scenePlanScene.DocumentDirty = handle.Dirty
				scenePlanScene.DocumentOpenCallbackID = uint64(spec.documentOpenID)
				scenePlanScene.DocumentSaveCallbackID = uint64(spec.documentSaveID)
				scenePlanScene.DocumentExportCallbackID = uint64(spec.documentExportID)
				scenePlanScene.DocumentImportCallbackID = uint64(spec.documentImportID)
				scenePlanScene.DocumentCloseCallbackID = uint64(spec.documentCloseID)
				scenePlanScene.DocumentDirtyCallbackID = uint64(spec.documentDirtyID)
			}
			run.Scenes = append(run.Scenes, scenePlanScene)
		case sceneMenuBar:
			viewIndex := len(views)
			views = append(views, spec.menuView.ptr)
			run.Scenes = append(run.Scenes, sceneRunPlanScene{
				Kind:         sceneKindLabel(spec.kind),
				Label:        spec.menuConfig.Label,
				SystemImage:  spec.menuConfig.SystemImage,
				Width:        spec.menuConfig.Width,
				Height:       spec.menuConfig.Height,
				OpenOnLaunch: spec.menuConfig.OpenOnLaunch,
				ViewIndex:    viewIndex,
			})
		default:
			return "", nil, errUnsupportedScene
		}
	}
	data, err := json.Marshal(run)
	if err != nil {
		return "", nil, err
	}
	return string(data), views, nil
}

func sceneKindLabel(kind sceneKind) string {
	switch kind {
	case sceneWindow:
		return "window"
	case sceneDocument:
		return "document"
	case sceneMenuBar:
		return "menuBar"
	case sceneSettings:
		return "settings"
	default:
		return ""
	}
}

func legacyRunScenePlan(plan scenePlan) error {
	var (
		windowSpec sceneSpec
		menuSpec   sceneSpec
		hasWindow  bool
		hasMenuBar bool
	)
	for i := range plan.specs {
		spec := plan.specs[i]
		switch spec.kind {
		case sceneWindow, sceneDocument, sceneSettings:
			if hasWindow {
				return errDuplicateSceneID
			}
			hasWindow = true
			windowSpec = spec
		case sceneMenuBar:
			if hasMenuBar {
				return errMultipleMenuExtras
			}
			hasMenuBar = true
			menuSpec = spec
		}
	}
	switch {
	case hasWindow && hasMenuBar:
		RunWithMenuBar(windowSpec.appConfig, windowSpec.appView, menuSpec.menuConfig, menuSpec.menuView)
	case hasWindow:
		Run(windowSpec.appConfig, windowSpec.appView)
	case hasMenuBar:
		RunMenuBar(menuSpec.menuConfig, menuSpec.menuView)
	default:
		return errNoScenes
	}
	return nil
}

func newWindowSceneRuntime(id string) *windowSceneRuntime {
	r := &windowSceneRuntime{
		revision: newIntStateIfReady(0),
		sceneID:  strings.TrimSpace(id),
	}
	r.sceneActionCallbackID = registerStringCallback(func(event string) bool {
		return r.handleSceneActionEvent(event)
	})
	return r
}

func (r *windowSceneRuntime) handleSceneActionEvent(event string) bool {
	kind, caps, ok := parseSceneActionEvent(event)
	if !ok {
		return false
	}
	switch kind {
	case sceneActionEventAvailable:
		r.setAvailability(true, caps)
		r.bindScenePlanActions(caps)
		return true
	case sceneActionEventUnavailable:
		r.setAvailability(false, caps)
		r.clearScenePlanActions()
		return true
	case sceneActionEventFocused:
		r.setFocused(true, caps)
		return true
	case sceneActionEventBlurred:
		r.setFocused(false, caps)
		return true
	default:
		return false
	}
}

func (r *windowSceneRuntime) activeScene() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.focused
}

func (r *windowSceneRuntime) liveScene() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.live
}

func (r *windowSceneRuntime) windowCount() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.instanceCount
}

func (r *windowSceneRuntime) snapshotState() (bool, bool, int, string, []SceneWindowInstanceState) {
	if r == nil {
		return false, false, 0, "", nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	instances := make([]SceneWindowInstanceState, 0, len(r.instances))
	for _, instance := range r.instances {
		instances = append(instances, instance)
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].ID < instances[j].ID
	})
	return r.live, r.focused, r.instanceCount, r.focusedInstanceID, instances
}

func (r *windowSceneRuntime) setAvailability(live bool, caps sceneActionCapabilities) {
	if r == nil {
		return
	}
	r.mu.Lock()
	changed := r.applyWindowSceneAvailabilityLocked(live, caps)
	revision := r.revision
	r.mu.Unlock()
	if changed && revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *windowSceneRuntime) setFocused(focused bool, caps sceneActionCapabilities) {
	if r == nil {
		return
	}
	r.mu.Lock()
	changed := r.applyWindowSceneFocusLocked(focused, caps)
	revision := r.revision
	r.mu.Unlock()
	if changed && revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *windowSceneRuntime) applyWindowSceneAvailabilityLocked(live bool, caps sceneActionCapabilities) bool {
	changed := false
	if caps.Instance != "" {
		visible := true
		if caps.Visible != nil {
			visible = *caps.Visible
		}
		changed = r.upsertWindowInstanceLocked(SceneWindowInstanceState{
			ID:      caps.Instance,
			Visible: visible,
		}, false) || changed
		if !visible {
			changed = r.clearFocusedWindowInstanceLocked(caps.Instance) || changed
		}
	} else if !live {
		changed = r.clearAllWindowInstancesLocked() || changed
	}
	if !live && caps.Instance != "" {
		changed = r.removeWindowInstanceLocked(caps.Instance) || changed
	}
	changed = r.syncWindowRuntimeStateLocked(live, caps.Count, caps.Instance == "" && live) || changed
	return changed
}

func (r *windowSceneRuntime) applyWindowSceneFocusLocked(focused bool, caps sceneActionCapabilities) bool {
	changed := false
	if caps.Instance != "" {
		if caps.Visible != nil && !*caps.Visible {
			changed = r.removeWindowInstanceLocked(caps.Instance) || changed
			changed = r.clearFocusedWindowInstanceLocked(caps.Instance) || changed
		} else {
			visible := true
			if caps.Visible != nil {
				visible = *caps.Visible
			}
			changed = r.upsertWindowInstanceLocked(SceneWindowInstanceState{
				ID:      caps.Instance,
				Visible: visible,
				Focused: focused,
			}, true) || changed
			if focused {
				for id, instance := range r.instances {
					if id == caps.Instance || !instance.Focused {
						continue
					}
					instance.Focused = false
					r.instances[id] = instance
					changed = true
				}
				if r.focusedInstanceID != caps.Instance {
					r.focusedInstanceID = caps.Instance
					changed = true
				}
			} else {
				changed = r.clearFocusedWindowInstanceLocked(caps.Instance) || changed
			}
		}
	}
	changed = r.syncWindowRuntimeStateLocked(true, caps.Count, caps.Instance == "" && focused) || changed
	return changed
}

func (r *windowSceneRuntime) upsertWindowInstanceLocked(instance SceneWindowInstanceState, replaceFocus bool) bool {
	instance.ID = strings.TrimSpace(instance.ID)
	if instance.ID == "" {
		return false
	}
	if r.instances == nil {
		r.instances = make(map[string]SceneWindowInstanceState)
	}
	current, ok := r.instances[instance.ID]
	if ok && !replaceFocus {
		instance.Focused = current.Focused
	}
	if ok && current == instance {
		return false
	}
	r.instances[instance.ID] = instance
	return true
}

func (r *windowSceneRuntime) removeWindowInstanceLocked(id string) bool {
	if r.instances == nil {
		return false
	}
	if _, ok := r.instances[id]; !ok {
		return false
	}
	delete(r.instances, id)
	if r.focusedInstanceID == id {
		r.focusedInstanceID = ""
	}
	return true
}

func (r *windowSceneRuntime) clearAllWindowInstancesLocked() bool {
	changed := false
	if len(r.instances) > 0 {
		r.instances = nil
		changed = true
	}
	if r.focusedInstanceID != "" {
		r.focusedInstanceID = ""
		changed = true
	}
	return changed
}

func (r *windowSceneRuntime) clearFocusedWindowInstanceLocked(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	changed := false
	if instance, ok := r.instances[id]; ok && instance.Focused {
		instance.Focused = false
		r.instances[id] = instance
		changed = true
	}
	if r.focusedInstanceID == id {
		r.focusedInstanceID = ""
		changed = true
	}
	return changed
}

func (r *windowSceneRuntime) syncWindowRuntimeStateLocked(live bool, instanceCount int, legacyFocused bool) bool {
	if instanceCount < 0 {
		instanceCount = 0
	}
	if n := len(r.instances); n > instanceCount {
		instanceCount = n
	}
	changed := false
	focusedInstanceID := strings.TrimSpace(r.focusedInstanceID)
	if focusedInstanceID != "" {
		instance, ok := r.instances[focusedInstanceID]
		if !ok || !instance.Focused {
			focusedInstanceID = ""
		}
	}
	if focusedInstanceID == "" {
		for _, instance := range r.instances {
			if !instance.Focused {
				continue
			}
			focusedInstanceID = instance.ID
			break
		}
	}
	focused := legacyFocused
	if focusedInstanceID != "" {
		focused = true
	}
	if !live && instanceCount > 0 {
		live = true
	}
	if !live {
		focused = false
		focusedInstanceID = ""
	}
	if r.live != live {
		r.live = live
		changed = true
	}
	if r.focused != focused {
		r.focused = focused
		changed = true
	}
	if r.instanceCount != instanceCount {
		r.instanceCount = instanceCount
		changed = true
	}
	if r.focusedInstanceID != focusedInstanceID {
		r.focusedInstanceID = focusedInstanceID
		changed = true
	}
	return changed
}

func (r *windowSceneRuntime) bump() {
	if r == nil {
		return
	}
	r.mu.RLock()
	revision := r.revision
	r.mu.RUnlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *windowSceneRuntime) setActions(actions SceneActions) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.manualActions = actions
	r.mu.Unlock()
}

func (r *windowSceneRuntime) actions() SceneActions {
	if r == nil {
		return SceneActions{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mergeSceneActions(r.manualActions, r.injectedActions)
}

func (r *windowSceneRuntime) bindScenePlanActions(caps sceneActionCapabilities) {
	if r == nil {
		return
	}
	r.mu.Lock()
	var injected SceneActions
	if caps.Window {
		injected.Window = OpenWindowAction(func(id string) error {
			if strings.TrimSpace(id) == "" {
				return errEmptySceneID
			}
			if !openSceneWindow(id) {
				return ErrActionUnavailable
			}
			return nil
		})
	}
	if caps.Refresh {
		injected.RefreshScene = RefreshAction(func() error {
			r.bump()
			return nil
		})
	}
	if caps.Immersive {
		injected.ImmersiveSpace = OpenImmersiveSpaceAction(func(id string) error {
			if strings.TrimSpace(id) == "" {
				return errEmptyImmersiveSpaceID
			}
			if !openSceneWindow(id) {
				return ErrActionUnavailable
			}
			return nil
		})
	}
	r.injectedActions = injected
	revision := r.revision
	r.mu.Unlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *windowSceneRuntime) clearScenePlanActions() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.injectedActions = SceneActions{}
	revision := r.revision
	r.mu.Unlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func newDocumentSceneRuntime(handle DocumentHandle) *documentSceneRuntime {
	handle = normalizeDocumentHandle(handle)
	if handle.Session.Path != "" {
		handle.RecentDocuments = prependDocumentRecent(handle.RecentDocuments, DocumentRecent{
			DisplayName: handle.Session.DisplayName,
			Path:        handle.Session.Path,
		})
	}
	r := &documentSceneRuntime{
		handle:   handle,
		revision: newIntStateIfReady(0),
	}
	r.sceneID = strings.TrimSpace(handle.Session.ID)
	r.sceneActionCallbackID = registerStringCallback(func(event string) bool {
		return r.handleSceneActionEvent(event)
	})
	r.documentOpenCallback = registerStringCallback(func(path string) bool {
		return r.handleDocumentOpen(path)
	})
	r.documentSaveCallback = registerStringCallback(func(path string) bool {
		return r.handleDocumentSave(path)
	})
	r.documentExportCallback = registerStringCallback(func(path string) bool {
		return r.handleDocumentExport(path)
	})
	r.documentImportCallback = registerStringCallback(func(path string) bool {
		return r.handleDocumentImport(path)
	})
	r.documentCloseCallback = registerCommandCallback(func() int32 {
		if r.handleDocumentClose() {
			return 1
		}
		return 0
	})
	r.documentDirtyCallback = registerCommandCallback(func() int32 {
		if r.documentDirty() {
			return 1
		}
		return 0
	})
	persistDocumentRecentDocuments(r.sceneID, r.handle.RecentDocuments)
	return r
}

func (r *documentSceneRuntime) handleSceneActionEvent(event string) bool {
	kind, caps, ok := parseSceneActionEvent(event)
	if !ok {
		return false
	}
	switch kind {
	case sceneActionEventAvailable:
		r.setAvailability(true, true)
		r.bindScenePlanActions(caps)
		return true
	case sceneActionEventUnavailable:
		r.setAvailability(false, false)
		r.clearScenePlanActions()
		return true
	case sceneActionEventFocused:
		r.setFocused(true)
		return true
	case sceneActionEventBlurred:
		r.setFocused(false)
		return true
	default:
		return false
	}
}

func (r *documentSceneRuntime) snapshot() (DocumentHandle, SceneActions) {
	r.mu.RLock()
	handle := normalizeDocumentHandle(r.handle)
	workflow := r.workflow
	sceneID := r.sceneID
	actions := mergeSceneActions(r.manualActions, r.injectedActions)
	r.mu.RUnlock()
	if sceneID != "" {
		if workflow.Open != nil {
			handle.Open = func() error {
				return runSceneDocumentOperation(sceneID, "open")
			}
			handle.OpenPath = func(path string) error {
				return runSceneDocumentPathOperation(sceneID, "openPath", path)
			}
			if handle.Session.Path != "" {
				handle.Revert = func() error {
					return runSceneDocumentOperation(sceneID, "revert")
				}
			}
		}
		if workflow.Save != nil {
			handle.Save = func() error {
				return runSceneDocumentOperation(sceneID, "save")
			}
			handle.SaveAs = func() error {
				return runSceneDocumentOperation(sceneID, "saveAs")
			}
		}
		if workflow.Export != nil {
			handle.Export = func() error {
				return runSceneDocumentOperation(sceneID, "export")
			}
		}
		if workflow.Import != nil {
			handle.Import = func() error {
				return runSceneDocumentOperation(sceneID, "import")
			}
		}
		handle.Close = func() error {
			return runSceneDocumentOperation(sceneID, "close")
		}
	}
	return handle, actions
}

func (r *documentSceneRuntime) activeScene() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.focused
}

func (r *documentSceneRuntime) liveScene() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.live
}

func (r *documentSceneRuntime) setAvailability(live, focused bool) {
	r.mu.Lock()
	if r.live == live && r.focused == focused {
		r.mu.Unlock()
		return
	}
	r.live = live
	r.focused = focused
	revision := r.revision
	r.mu.Unlock()
	if live {
		r.syncBridgeState()
	}
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) setFocused(focused bool) {
	r.mu.Lock()
	if r.focused == focused {
		r.mu.Unlock()
		return
	}
	r.focused = focused
	revision := r.revision
	r.mu.Unlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) setHandle(handle DocumentHandle) {
	handle = normalizeDocumentHandle(handle)
	if handle.Session.Path != "" {
		handle.RecentDocuments = prependDocumentRecent(handle.RecentDocuments, DocumentRecent{
			DisplayName: handle.Session.DisplayName,
			Path:        handle.Session.Path,
		})
	}
	r.mu.Lock()
	r.handle = handle
	if id := strings.TrimSpace(handle.Session.ID); id != "" {
		r.sceneID = id
	}
	sceneID := r.sceneID
	live := r.live
	revision := r.revision
	r.mu.Unlock()
	persistDocumentRecentDocuments(sceneID, handle.RecentDocuments)
	if live {
		r.syncBridgeState()
	}
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) setDirty(dirty bool) {
	r.mu.Lock()
	handle := normalizeDocumentHandle(r.handle)
	if handle.Dirty == dirty {
		r.mu.Unlock()
		return
	}
	handle.Dirty = dirty
	r.handle = handle
	live := r.live
	revision := r.revision
	r.mu.Unlock()
	if live {
		r.syncBridgeState()
	}
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) setActions(actions SceneActions) {
	r.mu.Lock()
	r.manualActions = actions
	r.mu.Unlock()
}

func (r *documentSceneRuntime) setWorkflow(workflow DocumentWorkflow) {
	r.mu.Lock()
	r.workflow = workflow
	revision := r.revision
	r.mu.Unlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) workflowCallbackIDs() (openID, saveID, exportID, importID, closeID, dirtyID uintptr) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.workflow.Open != nil {
		openID = r.documentOpenCallback
	}
	if r.workflow.Save != nil {
		saveID = r.documentSaveCallback
	}
	if r.workflow.Export != nil {
		exportID = r.documentExportCallback
	}
	if r.workflow.Import != nil {
		importID = r.documentImportCallback
	}
	if r.workflow.Close != nil {
		closeID = r.documentCloseCallback
	}
	return openID, saveID, exportID, importID, closeID, r.documentDirtyCallback
}

func (r *documentSceneRuntime) bindScenePlanActions(caps sceneActionCapabilities) {
	r.mu.Lock()
	if r.sceneID == "" {
		r.sceneID = strings.TrimSpace(r.handle.ID)
	}
	var injected SceneActions
	if caps.Window {
		injected.Window = OpenWindowAction(func(id string) error {
			if strings.TrimSpace(id) == "" {
				return errEmptySceneID
			}
			if !openSceneWindow(id) {
				return ErrActionUnavailable
			}
			return nil
		})
	}
	if caps.Document {
		injected.Document = OpenDocumentAction(func(path string) error {
			if strings.TrimSpace(path) == "" {
				return errEmptyDocumentPath
			}
			if r.handleDocumentOpen(path) {
				return nil
			}
			r.updateDocumentPath(path)
			return nil
		})
	}
	if caps.Refresh {
		injected.RefreshScene = RefreshAction(func() error {
			r.bump()
			return nil
		})
	}
	if caps.Immersive {
		injected.ImmersiveSpace = OpenImmersiveSpaceAction(func(id string) error {
			if strings.TrimSpace(id) == "" {
				return errEmptyImmersiveSpaceID
			}
			if !openSceneWindow(id) {
				return ErrActionUnavailable
			}
			return nil
		})
	}
	r.injectedActions = injected
	revision := r.revision
	r.mu.Unlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) clearScenePlanActions() {
	r.mu.Lock()
	r.injectedActions = SceneActions{}
	revision := r.revision
	r.mu.Unlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) updateDocumentPath(path string) {
	path = strings.TrimSpace(path)
	r.mu.Lock()
	handle := normalizeDocumentHandle(r.handle)
	handle.Session.Path = path
	if base := filepath.Base(path); base != "." && base != "/" && base != "" {
		handle.Session.DisplayName = base
	}
	handle.Dirty = false
	handle = normalizeDocumentHandle(handle)
	handle.RecentDocuments = prependDocumentRecent(handle.RecentDocuments, DocumentRecent{
		DisplayName: handle.Session.DisplayName,
		Path:        handle.Session.Path,
	})
	r.handle = handle
	sceneID := r.sceneID
	live := r.live
	revision := r.revision
	r.mu.Unlock()
	persistDocumentRecentDocuments(sceneID, handle.RecentDocuments)
	if live {
		r.syncBridgeState()
	}
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func (r *documentSceneRuntime) documentDirty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return normalizeDocumentHandle(r.handle).Dirty
}

func (r *documentSceneRuntime) handleDocumentOpen(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	r.mu.RLock()
	fn := r.workflow.Open
	r.mu.RUnlock()
	if fn == nil {
		return false
	}
	if err := fn(path); err != nil {
		return false
	}
	r.updateDocumentPath(path)
	return true
}

func (r *documentSceneRuntime) handleDocumentSave(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	r.mu.RLock()
	fn := r.workflow.Save
	handle := normalizeDocumentHandle(r.handle)
	r.mu.RUnlock()
	if fn == nil {
		return false
	}
	if err := fn(handle.Session, path); err != nil {
		return false
	}
	r.updateDocumentPath(path)
	return true
}

func (r *documentSceneRuntime) handleDocumentExport(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	r.mu.RLock()
	fn := r.workflow.Export
	handle := normalizeDocumentHandle(r.handle)
	r.mu.RUnlock()
	if fn == nil {
		return false
	}
	return fn(handle.Session, path) == nil
}

func (r *documentSceneRuntime) handleDocumentImport(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	r.mu.RLock()
	fn := r.workflow.Import
	r.mu.RUnlock()
	if fn == nil {
		return false
	}
	return fn(path) == nil
}

func (r *documentSceneRuntime) handleDocumentClose() bool {
	r.mu.RLock()
	fn := r.workflow.Close
	handle := normalizeDocumentHandle(r.handle)
	r.mu.RUnlock()
	if fn == nil {
		return true
	}
	return fn(handle.Session) == nil
}

func (r *documentSceneRuntime) syncBridgeState() {
	r.mu.RLock()
	sceneID := strings.TrimSpace(r.sceneID)
	handle := normalizeDocumentHandle(r.handle)
	live := r.live
	r.mu.RUnlock()
	if !live || sceneID == "" {
		return
	}
	updateSceneDocumentState(sceneID, handle)
}

func (r *documentSceneRuntime) bump() {
	r.mu.RLock()
	revision := r.revision
	r.mu.RUnlock()
	if revision != nil {
		revision.Set(revision.Get() + 1)
	}
}

func boolPtr(v bool) *bool { return &v }
