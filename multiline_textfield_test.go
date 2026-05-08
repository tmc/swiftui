package swiftui

import "testing"

func TestMultilineTextFieldCallbacksRegistered(t *testing.T) {
	if loadErr != nil {
		t.Fatalf("load bridge: %v", loadErr)
	}
	if libHandle == 0 {
		t.Fatal("bridge dylib not loaded")
	}

	ensureMultilineTextFieldLibFuncs()
	if _SUIMultilineTextFieldCallbacks == nil {
		t.Fatal("SUIMultilineTextFieldCallbacks not registered")
	}
}
