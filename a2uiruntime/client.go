package a2uiruntime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/tmc/swiftui/a2ui"
)

type Client struct {
	Runtime     *Runtime
	HTTPClient  *http.Client
	MaxAttempts int
	Logger      *log.Logger
}

func NewClient(rt *Runtime) *Client {
	return &Client{
		Runtime:     rt,
		HTTPClient:  http.DefaultClient,
		MaxAttempts: 5,
	}
}

func (c *Client) ConnectSSE(ctx context.Context, url string, onUpdate func(Snapshot)) error {
	if c.Runtime == nil {
		return nil
	}
	if onUpdate == nil {
		onUpdate = func(Snapshot) {}
	}
	c.Runtime.SetTransport(HTTPTransport{ActionURL: ActionURLFromSSE(url), Client: c.HTTPClient})
	c.Runtime.SetStatus(StatusConnecting)
	onUpdate(c.Runtime.Snapshot())

	attempts := c.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				c.Runtime.SetStatus(StatusDisconnected)
				onUpdate(c.Runtime.Snapshot())
				return ctx.Err()
			case <-timer.C:
			}
			c.Runtime.SetStatus(StatusConnecting)
			onUpdate(c.Runtime.Snapshot())
		}

		lastErr = c.connectOnce(ctx, url, onUpdate)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if lastErr == nil {
			return nil
		}
		if c.Logger != nil {
			c.Logger.Printf("a2uiruntime reconnect: %v", lastErr)
		}
	}
	c.Runtime.SetStatus(StatusDisconnected)
	onUpdate(c.Runtime.Snapshot())
	return lastErr
}

func (c *Client) connectOnce(ctx context.Context, url string, onUpdate func(Snapshot)) error {
	client := c.HTTPClient
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

	c.Runtime.SetStatus(StatusConnected)
	onUpdate(c.Runtime.Snapshot())

	err = readSSE(resp.Body, func(data []byte) {
		var msg a2ui.ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.Runtime.report(RuntimeError{Code: "bad_message", Message: err.Error()})
			onUpdate(c.Runtime.Snapshot())
			return
		}
		c.Runtime.ApplyServerMessage(msg)
		onUpdate(c.Runtime.Snapshot())
	})
	c.Runtime.SetStatus(StatusDisconnected)
	onUpdate(c.Runtime.Snapshot())
	return err
}
