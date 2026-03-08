package spritekit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tmc/swiftui/internal/bridgeutil"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libSpriteKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "internal/swift")
}

// discoverBuiltDylib locates a built bridge dylib across SwiftPM layouts.
func discoverBuiltDylib() (string, error) {
	path, err := bridgeutil.DiscoverBuiltDylib(swiftBridgeDir(), "libSpriteKitSwiftUIBridge.dylib")
	if err != nil {
		return "", fmt.Errorf("spritekit: %w", err)
	}
	return path, nil
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	return bridgeutil.BuildSwiftBridge(swiftBridgeDir(), "spritekit", "libSpriteKitSwiftUIBridge.dylib", "spritekit")
}

// tryRegisterLibFunc attempts to register a C function from the dylib.
func tryRegisterLibFunc(fptr any, handle uintptr, name string) bool {
	defer func() { recover() }()
	purego.RegisterLibFunc(fptr, handle, name)
	return true
}

// C function variables registered via purego.RegisterLibFunc.
var (
	_SPS_SpriteViewCreate               func(uintptr) uintptr
	_SPS_SpriteViewCreateWithTransition func(uintptr, uintptr) uintptr
	_SPS_SpriteViewCreateWithOptions    func(uintptr, bool, int32) uintptr
	_SPS_Retain                         func(uintptr) uintptr
	_SPS_Release                        func(uintptr)
	_SPS_FreeString                     func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("SpriteKitSwiftUIBridge") + "_PATH"

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
			"libSpriteKitSwiftUIBridge.dylib",
			"/usr/local/lib/libSpriteKitSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_SPS_SpriteViewCreate, libHandle, "SPS_SpriteViewCreate")
	tryRegisterLibFunc(&_SPS_SpriteViewCreateWithTransition, libHandle, "SPS_SpriteViewCreateWithTransition")
	tryRegisterLibFunc(&_SPS_SpriteViewCreateWithOptions, libHandle, "SPS_SpriteViewCreateWithOptions")
	tryRegisterLibFunc(&_SPS_Retain, libHandle, "SPS_Retain")
	tryRegisterLibFunc(&_SPS_Release, libHandle, "SPS_Release")
	tryRegisterLibFunc(&_SPS_FreeString, libHandle, "SPS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("spritekit: " + name + ": dylib not loaded") }
	if _SPS_SpriteViewCreate == nil {
		_SPS_SpriteViewCreate = func(uintptr) uintptr { stub("SPS_SpriteViewCreate"); return 0 }
	}
	if _SPS_SpriteViewCreateWithTransition == nil {
		_SPS_SpriteViewCreateWithTransition = func(uintptr, uintptr) uintptr { stub("SPS_SpriteViewCreateWithTransition"); return 0 }
	}
	if _SPS_SpriteViewCreateWithOptions == nil {
		_SPS_SpriteViewCreateWithOptions = func(uintptr, bool, int32) uintptr { stub("SPS_SpriteViewCreateWithOptions"); return 0 }
	}
	if _SPS_Retain == nil {
		_SPS_Retain = func(uintptr) uintptr { stub("SPS_Retain"); return 0 }
	}
	if _SPS_Release == nil {
		_SPS_Release = func(uintptr) { stub("SPS_Release") }
	}
	if _SPS_FreeString == nil {
		_SPS_FreeString = func(*byte) { stub("SPS_FreeString") }
	}
}
