package arkitswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libARKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libARKitSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("arkitswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("arkitswiftui: swift build failed: %w", err)
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
	_ARS_ARViewCreate func() uintptr
	_ARS_Retain       func(uintptr) uintptr
	_ARS_Release      func(uintptr)
	_ARS_FreeString   func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("ARKitSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libARKitSwiftUIBridge.dylib",
		"/usr/local/lib/libARKitSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_ARS_ARViewCreate, libHandle, "ARS_ARViewCreate")
	tryRegisterLibFunc(&_ARS_Retain, libHandle, "ARS_Retain")
	tryRegisterLibFunc(&_ARS_Release, libHandle, "ARS_Release")
	tryRegisterLibFunc(&_ARS_FreeString, libHandle, "ARS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("arkitswiftui: " + name + ": dylib not loaded") }
	if _ARS_ARViewCreate == nil {
		_ARS_ARViewCreate = func() uintptr { stub("ARS_ARViewCreate"); return 0 }
	}
	if _ARS_Retain == nil {
		_ARS_Retain = func(uintptr) uintptr { stub("ARS_Retain"); return 0 }
	}
	if _ARS_Release == nil {
		_ARS_Release = func(uintptr) { stub("ARS_Release") }
	}
	if _ARS_FreeString == nil {
		_ARS_FreeString = func(*byte) { stub("ARS_FreeString") }
	}
}
