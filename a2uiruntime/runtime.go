package a2uiruntime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/a2ui"
)

const BasicCatalogID = a2ui.BasicCatalogID

type Status string

const (
	StatusDisconnected Status = "disconnected"
	StatusConnecting   Status = "connecting"
	StatusConnected    Status = "connected"
	StatusFile         Status = "file"
)

type Surface struct {
	ID            string
	CatalogID     string
	Theme         *a2ui.Theme
	Components     map[string]a2ui.Component
	RootID        string
	DataModel     *a2ui.DataModel
	SendDataModel bool
}

type Snapshot struct {
	SurfaceID      string
	CatalogID      string
	Theme          *a2ui.Theme
	Components     map[string]a2ui.Component
	RootID         string
	DataModel      *a2ui.DataModel
	Status         Status
	Revision       int
	StatusRevision int
	LastError      string
}

type Runtime struct {
	mu          sync.Mutex
	surfaces    map[string]*Surface
	activeID    string
	status      Status
	revision    int
	statusRev   int
	lastError   string
	cache       *stateCache
	transport   Transport
	strict      bool
	logger      *log.Logger
	catalogs    map[string]struct{}
	extensions  map[string]struct{}
}

type Option func(*Runtime)

func WithTransport(t Transport) Option {
	return func(rt *Runtime) { rt.transport = t }
}

func (rt *Runtime) SetTransport(t Transport) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if t == nil {
		rt.transport = NopTransport{}
		return
	}
	rt.transport = t
}

func WithStrict(strict bool) Option {
	return func(rt *Runtime) { rt.strict = strict }
}

func WithLogger(logger *log.Logger) Option {
	return func(rt *Runtime) {
		if logger != nil {
			rt.logger = logger
		}
	}
}

func New(opts ...Option) *Runtime {
	rt := &Runtime{
		surfaces:   make(map[string]*Surface),
		status:     StatusDisconnected,
		cache:      newStateCache(),
		transport:  NopTransport{},
		logger:     log.Default(),
		catalogs:   map[string]struct{}{BasicCatalogID: {}},
		extensions: map[string]struct{}{a2ui.ComponentProgress: {}, "Padding": {}, "Spacing": {}, "Strikethrough": {}},
	}
	for _, opt := range opts {
		opt(rt)
	}
	return rt
}

func (rt *Runtime) ClientCapabilities() a2ui.ClientCapabilities {
	ids := make([]string, 0, len(rt.catalogs))
	for id := range rt.catalogs {
		ids = append(ids, id)
	}
	return a2ui.ClientCapabilities{
		V09: &a2ui.ClientCapabilitiesV09{
			SupportedCatalogIDs: ids,
		},
	}
}

func (rt *Runtime) SetStatus(status Status) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.status == status {
		return
	}
	rt.status = status
	rt.statusRev++
}

func (rt *Runtime) Snapshot() Snapshot {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.snapshotLocked()
}

func (rt *Runtime) snapshotLocked() Snapshot {
	snap := Snapshot{
		Status:         rt.status,
		Revision:       rt.revision,
		StatusRevision: rt.statusRev,
		LastError:      rt.lastError,
	}
	surface := rt.surfaces[rt.activeID]
	if surface == nil {
		return snap
	}
	comps := make(map[string]a2ui.Component, len(surface.Components))
	maps.Copy(comps, surface.Components)
	snap.SurfaceID = surface.ID
	snap.CatalogID = surface.CatalogID
	snap.Theme = surface.Theme
	snap.Components = comps
	snap.RootID = surface.RootID
	snap.DataModel = surface.DataModel
	return snap
}

func (rt *Runtime) RenderActiveSurface() swiftui.View {
	snap := rt.Snapshot()
	if snap.RootID == "" {
		return EmptyView()
	}
	return renderComponent(rt, snap.Components, snap.DataModel, snap.SurfaceID, snap.RootID, snap.Theme)
}

func (rt *Runtime) ApplyServerMessage(msg a2ui.ServerMessage) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	switch {
	case msg.CreateSurface != nil:
		rt.applyCreateLocked(*msg.CreateSurface)
	case msg.UpdateComponents != nil:
		rt.applyComponentsLocked(*msg.UpdateComponents)
	case msg.UpdateDataModel != nil:
		rt.applyDataModelLocked(*msg.UpdateDataModel)
	case msg.DeleteSurface != nil:
		delete(rt.surfaces, msg.DeleteSurface.SurfaceID)
		if rt.activeID == msg.DeleteSurface.SurfaceID {
			rt.activeID = ""
		}
		rt.revision++
	}
}

func (rt *Runtime) applyCreateLocked(msg a2ui.CreateSurface) {
	if _, ok := rt.catalogs[msg.CatalogID]; !ok {
		rt.reportLocked(RuntimeError{
			Code:      "unsupported_catalog",
			SurfaceID: msg.SurfaceID,
			Message:   fmt.Sprintf("unsupported catalog %q", msg.CatalogID),
		})
		if rt.strict {
			return
		}
	}
	surface := &Surface{
		ID:            msg.SurfaceID,
		CatalogID:     msg.CatalogID,
		Theme:         msg.Theme,
		Components:    make(map[string]a2ui.Component),
		DataModel:     a2ui.NewDataModel(),
		SendDataModel: msg.SendDataModel,
	}
	rt.surfaces[msg.SurfaceID] = surface
	rt.activeID = msg.SurfaceID
	rt.revision++
}

func (rt *Runtime) applyComponentsLocked(msg a2ui.UpdateComponents) {
	surface := rt.ensureSurfaceLocked(msg.SurfaceID)
	for _, c := range msg.Components {
		surface.Components[c.ID] = c
	}
	if surface.RootID == "" {
		if _, ok := surface.Components["root"]; ok {
			surface.RootID = "root"
		} else {
			for _, c := range msg.Components {
				surface.RootID = c.ID
				break
			}
		}
	}
	rt.activeID = msg.SurfaceID
	rt.revision++
}

func (rt *Runtime) applyDataModelLocked(msg a2ui.UpdateDataModel) {
	surface := rt.ensureSurfaceLocked(msg.SurfaceID)
	if msg.Value != nil {
		if err := surface.DataModel.Set(msg.Path, msg.Value); err != nil {
			rt.reportLocked(RuntimeError{
				Code:      "invalid_binding",
				SurfaceID: msg.SurfaceID,
				Path:      msg.Path,
				Message:   err.Error(),
			})
			return
		}
	} else if msg.Path != "" {
		if err := surface.DataModel.Remove(msg.Path); err != nil {
			rt.reportLocked(RuntimeError{
				Code:      "invalid_binding",
				SurfaceID: msg.SurfaceID,
				Path:      msg.Path,
				Message:   err.Error(),
			})
			return
		}
	}
	rt.cache.syncFromDataModel(surface.DataModel)
	rt.activeID = msg.SurfaceID
}

func (rt *Runtime) ensureSurfaceLocked(id string) *Surface {
	if surface := rt.surfaces[id]; surface != nil {
		return surface
	}
	surface := &Surface{
		ID:         id,
		Components: make(map[string]a2ui.Component),
		DataModel:  a2ui.NewDataModel(),
	}
	rt.surfaces[id] = surface
	return surface
}

func (rt *Runtime) HandleAction(surfaceID, componentID string, action *a2ui.Action) {
	if action == nil {
		return
	}
	rt.mu.Lock()
	surface := rt.surfaces[surfaceID]
	var dm *a2ui.DataModel
	if surface != nil {
		dm = surface.DataModel
	}
	rt.mu.Unlock()
	if dm == nil {
		rt.report(RuntimeError{
			Code:        "missing_surface",
			SurfaceID:   surfaceID,
			ComponentID: componentID,
			Message:     "surface not found",
		})
		return
	}

	if action.Event != nil {
		ctx := make(map[string]any, len(action.Event.Context))
		for key, value := range action.Event.Context {
			resolved, err := a2ui.Resolve(value, dm)
			if err != nil {
				rt.report(RuntimeError{
					Code:        "invalid_binding",
					SurfaceID:   surfaceID,
					ComponentID: componentID,
					Message:     err.Error(),
				})
				return
			}
			ctx[key] = resolved
		}
		for key, value := range rt.cache.snapshotValues() {
			ctx[key] = value
		}
		rt.send(a2ui.ClientMessage{
			Version: a2ui.Version,
			Action: &a2ui.ClientAction{
				Name:              action.Event.Name,
				SurfaceID:         surfaceID,
				SourceComponentID: componentID,
				Context:           ctx,
			},
		})
		return
	}

	if action.FunctionCall == nil {
		return
	}
	if err := executeClientFunction(action.FunctionCall, dm); err != nil {
		rt.report(RuntimeError{
			Code:        "unsupported_function",
			SurfaceID:   surfaceID,
			ComponentID: componentID,
			Message:     err.Error(),
		})
	}
}

func (rt *Runtime) send(msg a2ui.ClientMessage) {
	if err := rt.transport.Send(context.Background(), msg); err != nil {
		rt.report(RuntimeError{Code: "transport_error", Message: err.Error()})
	}
}

func (rt *Runtime) report(err RuntimeError) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.reportLocked(err)
}

func (rt *Runtime) reportLocked(err RuntimeError) {
	rt.lastError = err.Message
	rt.statusRev++
	if rt.logger != nil {
		rt.logger.Printf("%s: %s", err.Code, err.Message)
	}
	rt.send(a2ui.ClientMessage{
		Version: a2ui.Version,
		Error: &a2ui.ClientError{
			Code:      err.Code,
			SurfaceID: err.SurfaceID,
			Message:   err.Message,
			Path:      err.Path,
		},
	})
}

func (rt *Runtime) LoadFiles(componentsPath, dataPath string) error {
	compData, err := os.ReadFile(componentsPath)
	if err != nil {
		return fmt.Errorf("read components: %w", err)
	}

	rt.mu.Lock()
	rt.surfaces = make(map[string]*Surface)
	rt.activeID = "file"
	rt.status = StatusFile
	rt.statusRev++
	rt.mu.Unlock()

	trimmed := bytes.TrimSpace(compData)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if err := rt.LoadJSONL(compData); err != nil {
			return err
		}
	} else {
		var comps []a2ui.Component
		if err := json.Unmarshal(compData, &comps); err != nil {
			return fmt.Errorf("parse components: %w", err)
		}
		rt.mu.Lock()
		surface := rt.ensureSurfaceLocked("file")
		surface.CatalogID = BasicCatalogID
		for _, c := range comps {
			surface.Components[c.ID] = c
		}
		surface.RootID = findRootID(surface.Components)
		rt.revision++
		rt.mu.Unlock()
	}

	if dataPath == "" {
		return nil
	}
	dataBytes, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("read data: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return fmt.Errorf("parse data: %w", err)
	}
	rt.mu.Lock()
	surface := rt.ensureSurfaceLocked("file")
	surface.DataModel.Data = data
	rt.cache.syncFromDataModel(surface.DataModel)
	rt.mu.Unlock()
	return nil
}

func (rt *Runtime) LoadJSONL(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var msg a2ui.ServerMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			return fmt.Errorf("jsonl unmarshal: %w", err)
		}
		rt.ApplyServerMessage(msg)
	}
	return scanner.Err()
}

func (rt *Runtime) ConsumeSSE(ctx context.Context, client *http.Client, url string) error {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rt.SetStatus(StatusConnected)
	return readSSE(resp.Body, func(data []byte) {
		var msg a2ui.ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			rt.report(RuntimeError{Code: "bad_message", Message: err.Error()})
			return
		}
		rt.ApplyServerMessage(msg)
	})
}

func readSSE(r io.Reader, handle func([]byte)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	var dataBuf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if data, ok := strings.CutPrefix(line, "data: "); ok {
			dataBuf.WriteString(data)
			continue
		}
		if line == "" && dataBuf.Len() > 0 {
			handle(dataBuf.Bytes())
			dataBuf.Reset()
		}
	}
	return scanner.Err()
}

func findRootID(comps map[string]a2ui.Component) string {
	if _, ok := comps["root"]; ok {
		return "root"
	}
	for id := range comps {
		return id
	}
	return ""
}

func EmptyView() swiftui.View {
	return swiftui.VStackSpaced(8,
		swiftui.Spacer(),
		swiftui.Image("antenna.radiowaves.left.and.right").
			ForegroundStyleNamed("secondary").
			ImageScale(swiftui.ImageScaleLarge),
		swiftui.Text("Waiting for A2UI surface...").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary"),
		swiftui.Text("Connect to an A2UI server to render its UI.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("tertiary"),
		swiftui.Spacer(),
	).Padding(36)
}
