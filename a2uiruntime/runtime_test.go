package a2uiruntime

import (
	"fmt"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tmc/swiftui/a2ui"
)

type recordTransport struct {
	msgs []a2ui.ClientMessage
}

func (r *recordTransport) Send(_ context.Context, msg a2ui.ClientMessage) error {
	r.msgs = append(r.msgs, msg)
	return nil
}

func TestApplyServerMessageLifecycle(t *testing.T) {
	rt := New()
	rt.ApplyServerMessage(a2ui.ServerMessage{
		Version: a2ui.Version,
		CreateSurface: &a2ui.CreateSurface{
			SurfaceID: "demo",
			CatalogID: a2ui.BasicCatalogID,
		},
	})
	rt.ApplyServerMessage(a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "demo",
			Components: []a2ui.Component{
				{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("hello")}},
			},
		},
	})
	rt.ApplyServerMessage(a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateDataModel: &a2ui.UpdateDataModel{
			SurfaceID: "demo",
			Path:      "/name",
			Value:     "gopher",
		},
	})

	snap := rt.Snapshot()
	if snap.SurfaceID != "demo" {
		t.Fatalf("SurfaceID = %q, want demo", snap.SurfaceID)
	}
	if snap.RootID != "root" {
		t.Fatalf("RootID = %q, want root", snap.RootID)
	}
	if got, err := snap.DataModel.Get("/name"); err != nil || got != "gopher" {
		t.Fatalf("Get(/name) = %v, %v, want gopher", got, err)
	}

	rt.ApplyServerMessage(a2ui.ServerMessage{
		Version: a2ui.Version,
		DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "demo"},
	})
	if snap := rt.Snapshot(); snap.SurfaceID != "" {
		t.Fatalf("SurfaceID after delete = %q, want empty", snap.SurfaceID)
	}
}

func TestLoadJSONL(t *testing.T) {
	rt := New()
	lines := []a2ui.ServerMessage{
		{
			Version: a2ui.Version,
			CreateSurface: &a2ui.CreateSurface{
				SurfaceID: "demo",
				CatalogID: a2ui.BasicCatalogID,
			},
		},
		{
			Version: a2ui.Version,
			UpdateComponents: &a2ui.UpdateComponents{
				SurfaceID: "demo",
				Components: []a2ui.Component{
					{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("hello")}},
				},
			},
		},
	}
	var payload []byte
	for _, line := range lines {
		data, err := json.Marshal(line)
		if err != nil {
			t.Fatal(err)
		}
		payload = append(payload, data...)
		payload = append(payload, '\n')
	}
	if err := rt.LoadJSONL(payload); err != nil {
		t.Fatalf("LoadJSONL: %v", err)
	}
	if snap := rt.Snapshot(); snap.RootID != "root" {
		t.Fatalf("RootID = %q, want root", snap.RootID)
	}
}

func TestHandleActionEventPostsClientMessage(t *testing.T) {
	transport := &recordTransport{}
	rt := New(WithTransport(transport))
	rt.ApplyServerMessage(a2ui.ServerMessage{
		Version: a2ui.Version,
		CreateSurface: &a2ui.CreateSurface{
			SurfaceID: "demo",
			CatalogID: a2ui.BasicCatalogID,
		},
	})
	rt.ApplyServerMessage(a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateDataModel: &a2ui.UpdateDataModel{
			SurfaceID: "demo",
			Path:      "/value",
			Value:     "hello",
		},
	})
	rt.HandleAction("demo", "button", &a2ui.Action{
		Event: &a2ui.EventAction{
			Name: "submit",
			Context: map[string]a2ui.DynamicValue{
				"value": a2ui.ValueBinding("/value"),
			},
		},
	})
	if len(transport.msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(transport.msgs))
	}
	if got := transport.msgs[0].Action; got == nil || got.Name != "submit" || got.Context["value"] != "hello" {
		t.Fatalf("Action = %+v, want submit with hello", got)
	}
}

func TestInstantiateTemplate(t *testing.T) {
	base := a2ui.Component{
		ID: "template",
		Text: &a2ui.TextComponent{
			Text:    a2ui.StringLiteral("base"),
			Variant: a2ui.TextVariantBody,
		},
	}
	got, err := instantiateTemplate(base, map[string]any{
		"text":    "overridden",
		"padding": 6.0,
	}, 2)
	if err != nil {
		t.Fatalf("instantiateTemplate: %v", err)
	}
	if got.ID != "template[2]" {
		t.Fatalf("ID = %q, want template[2]", got.ID)
	}
	if got.Text == nil || got.Text.Text.Literal == nil || *got.Text.Text.Literal != "overridden" {
		t.Fatalf("Text = %+v, want overridden", got.Text)
	}
	if got.Padding == nil || *got.Padding != 6 {
		t.Fatalf("Padding = %v, want 6", got.Padding)
	}
}

func TestClientCapabilities(t *testing.T) {
	rt := New()
	caps := rt.ClientCapabilities()
	if caps.V09 == nil || len(caps.V09.SupportedCatalogIDs) == 0 {
		t.Fatalf("ClientCapabilities = %+v, want supported catalogs", caps)
	}
}

func TestClientConnectSSE(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		msgs := []a2ui.ServerMessage{
			{
				Version: a2ui.Version,
				CreateSurface: &a2ui.CreateSurface{
					SurfaceID: "demo",
					CatalogID: a2ui.BasicCatalogID,
				},
			},
			{
				Version: a2ui.Version,
				UpdateComponents: &a2ui.UpdateComponents{
					SurfaceID: "demo",
					Components: []a2ui.Component{
						{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("hello")}},
					},
				},
			},
		}
		for _, msg := range msgs {
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	rt := New()
	client := NewClient(rt)
	err := client.ConnectSSE(context.Background(), srv.URL, func(Snapshot) {})
	if err != nil {
		t.Fatalf("ConnectSSE: %v", err)
	}
	if snap := rt.Snapshot(); snap.RootID != "root" {
		t.Fatalf("RootID = %q, want root", snap.RootID)
	}
}
