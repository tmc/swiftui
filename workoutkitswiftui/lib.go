package workoutkitswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libWorkoutKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libWorkoutKitSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("workoutkitswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("workoutkitswiftui: swift build failed: %w", err)
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
	_WKS_WorkoutPreview func(uintptr) uintptr
	_WKS_Retain         func(uintptr) uintptr
	_WKS_Release        func(uintptr)
	_WKS_FreeString     func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("WorkoutKitSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libWorkoutKitSwiftUIBridge.dylib",
		"/usr/local/lib/libWorkoutKitSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_WKS_WorkoutPreview, libHandle, "WKS_WorkoutPreview")
	tryRegisterLibFunc(&_WKS_Retain, libHandle, "WKS_Retain")
	tryRegisterLibFunc(&_WKS_Release, libHandle, "WKS_Release")
	tryRegisterLibFunc(&_WKS_FreeString, libHandle, "WKS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("workoutkitswiftui: " + name + ": dylib not loaded") }
	if _WKS_WorkoutPreview == nil {
		_WKS_WorkoutPreview = func(uintptr) uintptr { stub("WKS_WorkoutPreview"); return 0 }
	}
	if _WKS_Retain == nil {
		_WKS_Retain = func(uintptr) uintptr { stub("WKS_Retain"); return 0 }
	}
	if _WKS_Release == nil {
		_WKS_Release = func(uintptr) { stub("WKS_Release") }
	}
	if _WKS_FreeString == nil {
		_WKS_FreeString = func(*byte) { stub("WKS_FreeString") }
	}
}
