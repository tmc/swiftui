package embeddedbridge

import _ "embed"

//go:embed libTranslationSwiftUIBridge.dylib
var bridge []byte

// Payload returns the embedded bridge dylib payload.
func Payload() ([]byte, string) {
	return bridge, "libTranslationSwiftUIBridge.dylib"
}
