package embeddedbridge

import _ "embed"

//go:embed libAVKitSwiftUIBridge.dylib
var bridge []byte

// Payload returns the embedded bridge dylib payload.
func Payload() ([]byte, string) {
	return bridge, "libAVKitSwiftUIBridge.dylib"
}
