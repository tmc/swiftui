package a2ui

import (
	"encoding/json"
	"testing"
)

func TestServerMessageRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  ServerMessage
	}{
		{
			name: "create surface",
			msg: ServerMessage{
				Version: Version,
				CreateSurface: &CreateSurface{
					SurfaceID: "s1",
					CatalogID: "demo",
					Theme: &Theme{
						PrimaryColor: "#007AFF",
					},
				},
			},
		},
		{
			name: "update components",
			msg: ServerMessage{
				Version: Version,
				UpdateComponents: &UpdateComponents{
					SurfaceID: "s1",
					Components: []Component{
						{
							ID: "title",
							Text: &TextComponent{
								Text:    StringLiteral("hello"),
								Variant: TextVariantH3,
							},
						},
						ProgressBar("progress", NumberBinding("/progress"), 100),
					},
				},
			},
		},
		{
			name: "update data model",
			msg: ServerMessage{
				Version: Version,
				UpdateDataModel: &UpdateDataModel{
					SurfaceID: "s1",
					Path:      "/count",
					Value:     float64(1),
				},
			},
		},
		{
			name: "delete surface",
			msg: ServerMessage{
				Version: Version,
				DeleteSurface: &DeleteSurface{
					SurfaceID: "s1",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got ServerMessage
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			gotJSON, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal round-trip: %v", err)
			}
			if string(gotJSON) != string(data) {
				t.Fatalf("round-trip mismatch:\ngot  %s\nwant %s", gotJSON, data)
			}
		})
	}
}

func TestComponentRoundTrip(t *testing.T) {
	spacing := 8.0
	strike := true
	action := Action{
		Event: &EventAction{
			Name: "submit",
			Context: map[string]DynamicValue{
				"taskID": ValueNumber(42),
			},
		},
	}

	want := Component{
		ID: "input",
		TextField: &TextFieldComponent{
			Label:   StringLiteral("Task"),
			Variant: TextFieldVariantShortText,
		},
		Action:        &action,
		Spacing:       &spacing,
		Strikethrough: &strike,
	}
	value := StringBinding("/task")
	want.TextField.Value = &value

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Component
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ComponentType() != ComponentTextField {
		t.Fatalf("ComponentType() = %q, want %q", got.ComponentType(), ComponentTextField)
	}
	if got.Action == nil || got.Action.Event == nil || got.Action.Event.Name != "submit" {
		t.Fatalf("Action = %+v, want submit event", got.Action)
	}
	if got.Spacing == nil || *got.Spacing != spacing {
		t.Fatalf("Spacing = %v, want %v", got.Spacing, spacing)
	}
	if got.Strikethrough == nil || *got.Strikethrough != strike {
		t.Fatalf("Strikethrough = %v, want %v", got.Strikethrough, strike)
	}
	if got.TextField == nil || got.TextField.Value == nil || got.TextField.Value.Binding == nil || got.TextField.Value.Binding.Path != "/task" {
		t.Fatalf("TextField value = %+v, want /task binding", got.TextField)
	}
}
