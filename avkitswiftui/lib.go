package avkitswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libAVKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libAVKitSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("avkitswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("avkitswiftui: swift build failed: %w", err)
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
	_AVS_VideoPlayerCreate            func(uintptr) uintptr
	_AVS_VideoPlayerWithOverlayCreate func(uintptr, uintptr) uintptr
	_AVS_Retain                       func(uintptr) uintptr
	_AVS_Release                      func(uintptr)
	_AVS_FreeString                   func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("AVKitSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libAVKitSwiftUIBridge.dylib",
		"/usr/local/lib/libAVKitSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_AVS_VideoPlayerCreate, libHandle, "AVS_VideoPlayerCreate")
	tryRegisterLibFunc(&_AVS_VideoPlayerWithOverlayCreate, libHandle, "AVS_VideoPlayerWithOverlayCreate")
	tryRegisterLibFunc(&_AVS_Retain, libHandle, "AVS_Retain")
	tryRegisterLibFunc(&_AVS_Release, libHandle, "AVS_Release")
	tryRegisterLibFunc(&_AVS_FreeString, libHandle, "AVS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("avkitswiftui: " + name + ": dylib not loaded") }
	if _AVS_VideoPlayerCreate == nil {
		_AVS_VideoPlayerCreate = func(uintptr) uintptr { stub("AVS_VideoPlayerCreate"); return 0 }
	}
	if _AVS_VideoPlayerWithOverlayCreate == nil {
		_AVS_VideoPlayerWithOverlayCreate = func(uintptr, uintptr) uintptr { stub("AVS_VideoPlayerWithOverlayCreate"); return 0 }
	}
	if _AVS_Retain == nil {
		_AVS_Retain = func(uintptr) uintptr { stub("AVS_Retain"); return 0 }
	}
	if _AVS_Release == nil {
		_AVS_Release = func(uintptr) { stub("AVS_Release") }
	}
	if _AVS_FreeString == nil {
		_AVS_FreeString = func(*byte) { stub("AVS_FreeString") }
	}
}
