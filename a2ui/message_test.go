package a2ui

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  any
		typ  string
	}{
		{
			name: "create surface",
			msg: CreateSurface{
				SurfaceID: "s1",
				Title:     "Hello",
				InitialComponents: []Component{
					{ID: "c1", Type: ComponentText, Properties: map[string]any{"text": "hi"}},
				},
				InitialDataModel: map[string]any{"count": float64(0)},
			},
			typ: TypeCreateSurface,
		},
		{
			name: "update components",
			msg: UpdateComponents{
				SurfaceID: "s1",
				Components: []Component{
					{ID: "c1", Type: ComponentButton, Children: []string{"c2"}},
				},
			},
			typ: TypeUpdateComponents,
		},
		{
			name: "update data model",
			msg: UpdateDataModel{
				SurfaceID: "s1",
				Operations: []DataModelOperation{
					{Op: "set", Path: "/count", Value: float64(1)},
					{Op: "remove", Path: "/old"},
				},
			},
			typ: TypeUpdateDataModel,
		},
		{
			name: "delete surface",
			msg:  DeleteSurface{SurfaceID: "s1"},
			typ:  TypeDeleteSurface,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := NewEnvelope(tt.msg)
			if err != nil {
				t.Fatalf("NewEnvelope: %v", err)
			}
			if env.Type != tt.typ {
				t.Errorf("type = %q, want %q", env.Type, tt.typ)
			}

			// Marshal and unmarshal the envelope.
			data, err := json.Marshal(env)
			if err != nil {
				t.Fatalf("marshal envelope: %v", err)
			}
			var env2 Envelope
			if err := json.Unmarshal(data, &env2); err != nil {
				t.Fatalf("unmarshal envelope: %v", err)
			}
			if env2.Type != tt.typ {
				t.Errorf("round-trip type = %q, want %q", env2.Type, tt.typ)
			}

			// Decode payload.
			decoded, err := env2.DecodePayload()
			if err != nil {
				t.Fatalf("DecodePayload: %v", err)
			}

			// Re-marshal decoded and original, compare JSON.
			got, _ := json.Marshal(decoded)
			want, _ := json.Marshal(tt.msg)
			if string(got) != string(want) {
				t.Errorf("payload mismatch:\ngot  %s\nwant %s", got, want)
			}
		})
	}
}

func TestComponentRoundTrip(t *testing.T) {
	c := Component{
		ID:       "btn-1",
		Type:     ComponentButton,
		Children: []string{"icon-1", "label-1"},
		Properties: map[string]any{
			"label":   "Click me",
			"enabled": true,
			"count":   float64(42),
		},
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Component
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != c.ID || got.Type != c.Type {
		t.Errorf("component mismatch: got %+v", got)
	}
	if len(got.Children) != len(c.Children) {
		t.Errorf("children len = %d, want %d", len(got.Children), len(c.Children))
	}
	if got.Properties["label"] != "Click me" {
		t.Errorf("properties[label] = %v, want %q", got.Properties["label"], "Click me")
	}
}
