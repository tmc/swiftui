package a2ui

// ClientAction represents an action sent from the client back to the agent.
type ClientAction struct {
	Name              string         `json:"name"`
	SurfaceID         string         `json:"surfaceID"`
	SourceComponentID string         `json:"sourceComponentID"`
	Context           map[string]any `json:"context,omitempty"`
}
