package quicklookswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libQuickLookSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libQuickLookSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("quicklookswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("quicklookswiftui: swift build failed: %w", err)
	}
	return path, nil
}

// tryRegisterLibFunc attempts to register a C function from the dylib.
func tryRegisterLibFunc(fptr any, handle uintptr, name string) bool {
	defer func() { recover() }()
	purego.RegisterLibFunc(fptr, handle, name)
	return true
}

// C function variables registered via purego.RegisterLibFunc.
var (
	_QLS_QuickLookPreview func(uintptr, *byte) uintptr
	_QLS_Retain           func(uintptr) uintptr
	_QLS_Release          func(uintptr)
	_QLS_FreeString       func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("QuickLookSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libQuickLookSwiftUIBridge.dylib",
		"/usr/local/lib/libQuickLookSwiftUIBridge.dylib",
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		var err error
		libHandle, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			break
		}
	}

	if libHandle == 0 {
		if path, err := buildSwiftBridge(); err == nil {
			libHandle, _ = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		}
	}

	if libHandle == 0 {
		setUnavailableStubs()
		return
	}
	tryRegisterLibFunc(&_QLS_QuickLookPreview, libHandle, "QLS_QuickLookPreview")
	tryRegisterLibFunc(&_QLS_Retain, libHandle, "QLS_Retain")
	tryRegisterLibFunc(&_QLS_Release, libHandle, "QLS_Release")
	tryRegisterLibFunc(&_QLS_FreeString, libHandle, "QLS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("quicklookswiftui: " + name + ": dylib not loaded") }
	if _QLS_QuickLookPreview == nil {
		_QLS_QuickLookPreview = func(uintptr, *byte) uintptr { stub("QLS_QuickLookPreview"); return 0 }
	}
	if _QLS_Retain == nil {
		_QLS_Retain = func(uintptr) uintptr { stub("QLS_Retain"); return 0 }
	}
	if _QLS_Release == nil {
		_QLS_Release = func(uintptr) { stub("QLS_Release") }
	}
	if _QLS_FreeString == nil {
		_QLS_FreeString = func(*byte) { stub("QLS_FreeString") }
	}
}
