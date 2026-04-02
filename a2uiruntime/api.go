package a2uiruntime

import (
	"slices"

	"github.com/tmc/swiftui/a2ui"
)

// FunctionExecutor executes client-side function actions.
type FunctionExecutor interface {
	Execute(fn *a2ui.FunctionCall, dm *a2ui.DataModel) error
}

// FeatureStatus classifies runtime support for a capability.
type FeatureStatus string

const (
	FeatureSupported     FeatureStatus = "supported"
	FeatureExtensionOnly FeatureStatus = "extension-only"
)

// SupportEntry describes one runtime-supported capability.
type SupportEntry struct {
	Name   string
	Kind   string
	Status FeatureStatus
	Notes  string
}

// SupportMatrix summarizes runtime support boundaries.
type SupportMatrix struct {
	Catalogs   []string
	Extensions []string
	Entries    []SupportEntry
}

// MediaPolicy controls runtime media behavior.
type MediaPolicy struct {
	AllowRemote bool
	Autoplay    bool
}

// Presentation summarizes host-facing surface metadata.
type Presentation struct {
	AgentName string
	IconURL   string
	AccentHex string
}

// RegisterCatalog marks a catalog as supported by the runtime.
func (rt *Runtime) RegisterCatalog(id string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.catalogs[id] = struct{}{}
}

// RegisterExtension marks an extension as supported by the runtime.
func (rt *Runtime) RegisterExtension(name string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.extensions[name] = struct{}{}
}

// SupportMatrix returns the runtime's declared support surface.
func (rt *Runtime) SupportMatrix() SupportMatrix {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	catalogs := make([]string, 0, len(rt.catalogs))
	for id := range rt.catalogs {
		catalogs = append(catalogs, id)
	}
	slices.Sort(catalogs)

	extensions := make([]string, 0, len(rt.extensions))
	for name := range rt.extensions {
		extensions = append(extensions, name)
	}
	slices.Sort(extensions)

	return SupportMatrix{
		Catalogs:   catalogs,
		Extensions: extensions,
		Entries: []SupportEntry{
			{Name: "ChildTemplate", Kind: "composition", Status: FeatureSupported, Notes: "expanded against data-model object lists"},
			{Name: "Checks", Kind: "validation", Status: FeatureSupported, Notes: "aggregated client-side before action dispatch"},
			{Name: "openUrl", Kind: "action", Status: FeatureSupported, Notes: "executed through FunctionExecutor"},
			{Name: a2ui.ComponentProgress, Kind: "component", Status: FeatureExtensionOnly, Notes: "local extension component"},
			{Name: "Padding", Kind: "layout", Status: FeatureExtensionOnly, Notes: "local extension modifier"},
			{Name: "Spacing", Kind: "layout", Status: FeatureExtensionOnly, Notes: "local extension modifier"},
			{Name: "Strikethrough", Kind: "text", Status: FeatureExtensionOnly, Notes: "local extension modifier"},
		},
	}
}

// Presentation returns host-facing theme metadata for the active surface.
func (s Snapshot) Presentation() Presentation {
	p := Presentation{}
	if s.Theme == nil {
		return p
	}
	p.AgentName = s.Theme.AgentDisplayName
	p.IconURL = s.Theme.IconURL
	p.AccentHex = s.Theme.PrimaryColor
	return p
}
