package a2ui

import googlea2ui "github.com/google/A2UI/agent_sdks/go/a2ui"

// CreateSurface, UpdateDataModel, and DeleteSurface follow the Google SDK
// protocol definitions directly.
type (
	CreateSurface   = googlea2ui.CreateSurface
	UpdateDataModel = googlea2ui.UpdateDataModel
	DeleteSurface   = googlea2ui.DeleteSurface
)

// UpdateComponents uses the local Component wrapper so local extensions remain
// serializable alongside SDK-defined components.
type UpdateComponents struct {
	SurfaceID  string      `json:"surfaceId"`
	Components []Component `json:"components"`
}

// ServerMessage is the agent-to-client A2UI message envelope.
type ServerMessage struct {
	Version          string            `json:"version"`
	CreateSurface    *CreateSurface    `json:"createSurface,omitempty"`
	UpdateComponents *UpdateComponents `json:"updateComponents,omitempty"`
	UpdateDataModel  *UpdateDataModel  `json:"updateDataModel,omitempty"`
	DeleteSurface    *DeleteSurface    `json:"deleteSurface,omitempty"`
}
