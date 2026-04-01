package a2ui

import (
	"encoding/json"
	"fmt"
)

// Envelope wraps all A2UI messages with a Type field for dispatch.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// CreateSurface requests creation of a new UI surface.
type CreateSurface struct {
	SurfaceID         string         `json:"surfaceID"`
	Title             string         `json:"title,omitempty"`
	InitialComponents []Component    `json:"initialComponents,omitempty"`
	InitialDataModel  map[string]any `json:"initialDataModel,omitempty"`
}

// UpdateComponents sends a partial component update to an existing surface.
type UpdateComponents struct {
	SurfaceID  string      `json:"surfaceID"`
	Components []Component `json:"components"`
}

// DataModelOperation represents a single set or remove operation on the data model.
type DataModelOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// UpdateDataModel sends data model mutations to an existing surface.
type UpdateDataModel struct {
	SurfaceID  string               `json:"surfaceID"`
	Operations []DataModelOperation `json:"operations"`
}

// DeleteSurface requests deletion of an existing surface.
type DeleteSurface struct {
	SurfaceID string `json:"surfaceID"`
}

// Message type constants used in Envelope.Type.
const (
	TypeCreateSurface  = "createSurface"
	TypeUpdateComponents = "updateComponents"
	TypeUpdateDataModel  = "updateDataModel"
	TypeDeleteSurface    = "deleteSurface"
)

// NewEnvelope wraps a message in an Envelope with the appropriate type tag.
func NewEnvelope(msg any) (Envelope, error) {
	var typ string
	switch msg.(type) {
	case CreateSurface:
		typ = TypeCreateSurface
	case UpdateComponents:
		typ = TypeUpdateComponents
	case UpdateDataModel:
		typ = TypeUpdateDataModel
	case DeleteSurface:
		typ = TypeDeleteSurface
	default:
		return Envelope{}, fmt.Errorf("unknown message type %T", msg)
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return Envelope{}, fmt.Errorf("marshal payload: %w", err)
	}
	return Envelope{Type: typ, Payload: raw}, nil
}

// DecodePayload unmarshals the Envelope payload into the concrete message type
// indicated by Envelope.Type.
func (e Envelope) DecodePayload() (any, error) {
	switch e.Type {
	case TypeCreateSurface:
		var m CreateSurface
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil, fmt.Errorf("decode %s: %w", e.Type, err)
		}
		return m, nil
	case TypeUpdateComponents:
		var m UpdateComponents
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil, fmt.Errorf("decode %s: %w", e.Type, err)
		}
		return m, nil
	case TypeUpdateDataModel:
		var m UpdateDataModel
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil, fmt.Errorf("decode %s: %w", e.Type, err)
		}
		return m, nil
	case TypeDeleteSurface:
		var m DeleteSurface
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil, fmt.Errorf("decode %s: %w", e.Type, err)
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unknown envelope type %q", e.Type)
	}
}
