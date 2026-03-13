package scenekitswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libSceneKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libSceneKitSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("scenekitswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("scenekitswiftui: swift build failed: %w", err)
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
	_SKS_SceneViewCreate            func(uintptr) uintptr
	_SKS_SceneViewCreateWithOptions func(uintptr, int32) uintptr
	_SKS_Retain                     func(uintptr) uintptr
	_SKS_Release                    func(uintptr)
	_SKS_FreeString                 func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("SceneKitSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libSceneKitSwiftUIBridge.dylib",
		"/usr/local/lib/libSceneKitSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_SKS_SceneViewCreate, libHandle, "SKS_SceneViewCreate")
	tryRegisterLibFunc(&_SKS_SceneViewCreateWithOptions, libHandle, "SKS_SceneViewCreateWithOptions")
	tryRegisterLibFunc(&_SKS_Retain, libHandle, "SKS_Retain")
	tryRegisterLibFunc(&_SKS_Release, libHandle, "SKS_Release")
	tryRegisterLibFunc(&_SKS_FreeString, libHandle, "SKS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("scenekitswiftui: " + name + ": dylib not loaded") }
	if _SKS_SceneViewCreate == nil {
		_SKS_SceneViewCreate = func(uintptr) uintptr { stub("SKS_SceneViewCreate"); return 0 }
	}
	if _SKS_SceneViewCreateWithOptions == nil {
		_SKS_SceneViewCreateWithOptions = func(uintptr, int32) uintptr { stub("SKS_SceneViewCreateWithOptions"); return 0 }
	}
	if _SKS_Retain == nil {
		_SKS_Retain = func(uintptr) uintptr { stub("SKS_Retain"); return 0 }
	}
	if _SKS_Release == nil {
		_SKS_Release = func(uintptr) { stub("SKS_Release") }
	}
	if _SKS_FreeString == nil {
		_SKS_FreeString = func(*byte) { stub("SKS_FreeString") }
	}
}
