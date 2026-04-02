package a2uiruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tmc/swiftui/a2ui"
)

type Transport interface {
	Send(context.Context, a2ui.ClientMessage) error
}

type NopTransport struct{}

func (NopTransport) Send(context.Context, a2ui.ClientMessage) error { return nil }

type HTTPTransport struct {
	ActionURL string
	Client    *http.Client
}

func (t HTTPTransport) Send(ctx context.Context, msg a2ui.ClientMessage) error {
	if t.ActionURL == "" {
		return fmt.Errorf("http transport: empty action URL")
	}
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal client message: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.ActionURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post client message: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("post client message: status %s", resp.Status)
	}
	return nil
}

func ActionURLFromSSE(sseURL string) string {
	return strings.TrimSuffix(sseURL, "/sse") + "/action"
}
