//go:build !swiftui_embed

package embeddedbridge

import "testing"

// TestPayloadStub verifies the default (non-embed) build returns an empty
// payload so the loader falls back to building the bridge from source.
func TestPayloadStub(t *testing.T) {
	data, name := Payload()
	if name != "" || len(data) != 0 {
		t.Fatalf("Payload() = (%d bytes, %q), want empty stub", len(data), name)
	}
}
