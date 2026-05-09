package a2ui

import googlea2ui "github.com/google/A2UI/agent_sdks/go/a2ui"

// ClientAction is the client-to-server action event defined by the Google SDK.
type ClientAction = googlea2ui.ActionEvent

// ClientError reports a client-side error to the server.
type ClientError = googlea2ui.ClientError

// ClientMessage is the structured client-to-server message envelope.
type ClientMessage struct {
	Version string        `json:"version"`
	Action  *ClientAction `json:"action,omitempty"`
	Error   *ClientError  `json:"error,omitempty"`
}
