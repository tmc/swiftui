//go:build swiftui_embed

package embeddedbridge

import "testing"

// TestPayloadEmbed verifies the swiftui_embed build returns the embedded
// dylib payload with its expected name.
func TestPayloadEmbed(t *testing.T) {
	data, name := Payload()
	if name != "libSwiftUIBridge.dylib" {
		t.Fatalf("Payload name = %q, want libSwiftUIBridge.dylib", name)
	}
	if len(data) == 0 {
		t.Fatal("Payload returned empty data")
	}
}
