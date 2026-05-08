package swiftui

import (
	"os"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	runtime.LockOSThread()
	_SUIUpdateSceneDocumentState = func(*byte, *byte, *byte, int32) {}
	os.Exit(m.Run())
}
