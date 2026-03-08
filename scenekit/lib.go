package scenekit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tmc/swiftui/internal/bridgeutil"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libSceneKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "internal/swift")
}

// discoverBuiltDylib locates a built bridge dylib across SwiftPM layouts.
func discoverBuiltDylib() (string, error) {
	path, err := bridgeutil.DiscoverBuiltDylib(swiftBridgeDir(), "libSceneKitSwiftUIBridge.dylib")
	if err != nil {
		return "", fmt.Errorf("scenekit: %w", err)
	}
	return path, nil
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	return bridgeutil.BuildSwiftBridge(swiftBridgeDir(), "scenekit", "libSceneKitSwiftUIBridge.dylib", "scenekit")
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

	if path := os.Getenv(envKey); path != "" {
		libHandle, _ = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	}

	if libHandle == 0 {
		if path, err := discoverBuiltDylib(); err == nil {
			libHandle, _ = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		}
	}

	if libHandle == 0 {
		if path, err := buildSwiftBridge(); err == nil {
			libHandle, _ = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		}
	}

	if libHandle == 0 {
		for _, path := range []string{
			"libSceneKitSwiftUIBridge.dylib",
			"/usr/local/lib/libSceneKitSwiftUIBridge.dylib",
		} {
			var err error
			libHandle, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err == nil {
				break
			}
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
	stub := func(name string) { panic("scenekit: " + name + ": dylib not loaded") }
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
