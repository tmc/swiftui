package avkit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tmc/swiftui/internal/bridgeutil"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libAVKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "internal/swift")
}

// discoverBuiltDylib locates a built bridge dylib across SwiftPM layouts.
func discoverBuiltDylib() (string, error) {
	path, err := bridgeutil.DiscoverBuiltDylib(swiftBridgeDir(), "libAVKitSwiftUIBridge.dylib")
	if err != nil {
		return "", fmt.Errorf("avkit: %w", err)
	}
	return path, nil
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	return bridgeutil.BuildSwiftBridge(swiftBridgeDir(), "avkit", "libAVKitSwiftUIBridge.dylib", "avkit")
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
			"libAVKitSwiftUIBridge.dylib",
			"/usr/local/lib/libAVKitSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_AVS_VideoPlayerCreate, libHandle, "AVS_VideoPlayerCreate")
	tryRegisterLibFunc(&_AVS_VideoPlayerWithOverlayCreate, libHandle, "AVS_VideoPlayerWithOverlayCreate")
	tryRegisterLibFunc(&_AVS_Retain, libHandle, "AVS_Retain")
	tryRegisterLibFunc(&_AVS_Release, libHandle, "AVS_Release")
	tryRegisterLibFunc(&_AVS_FreeString, libHandle, "AVS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("avkit: " + name + ": dylib not loaded") }
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
