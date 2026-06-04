package embeddedbridge

import "testing"

func TestPayload(t *testing.T) {
	data, name := Payload()
	if len(data) == 0 && name == "" {
		t.Skip("vendored dylib not present")
	}
	if name != "libSwiftUIBridge.dylib" {
		t.Fatalf("Payload name = %q, want libSwiftUIBridge.dylib", name)
	}
	if len(data) == 0 {
		t.Fatal("Payload returned empty data")
	}
}
