package localauthswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libLocalAuthSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libLocalAuthSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("localauthswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("localauthswiftui: swift build failed: %w", err)
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
	_LAS_LocalAuthViewCreate func(*byte) uintptr
	_LAS_Retain              func(uintptr) uintptr
	_LAS_Release             func(uintptr)
	_LAS_FreeString          func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("LocalAuthSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libLocalAuthSwiftUIBridge.dylib",
		"/usr/local/lib/libLocalAuthSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_LAS_LocalAuthViewCreate, libHandle, "LAS_LocalAuthViewCreate")
	tryRegisterLibFunc(&_LAS_Retain, libHandle, "LAS_Retain")
	tryRegisterLibFunc(&_LAS_Release, libHandle, "LAS_Release")
	tryRegisterLibFunc(&_LAS_FreeString, libHandle, "LAS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("localauthswiftui: " + name + ": dylib not loaded") }
	if _LAS_LocalAuthViewCreate == nil {
		_LAS_LocalAuthViewCreate = func(*byte) uintptr { stub("LAS_LocalAuthViewCreate"); return 0 }
	}
	if _LAS_Retain == nil {
		_LAS_Retain = func(uintptr) uintptr { stub("LAS_Retain"); return 0 }
	}
	if _LAS_Release == nil {
		_LAS_Release = func(uintptr) { stub("LAS_Release") }
	}
	if _LAS_FreeString == nil {
		_LAS_FreeString = func(*byte) { stub("LAS_FreeString") }
	}
}
