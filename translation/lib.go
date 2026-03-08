package translation

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tmc/swiftui/internal/bridgeutil"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libTranslationSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "internal/swift")
}

// discoverBuiltDylib locates a built bridge dylib across SwiftPM layouts.
func discoverBuiltDylib() (string, error) {
	path, err := bridgeutil.DiscoverBuiltDylib(swiftBridgeDir(), "libTranslationSwiftUIBridge.dylib")
	if err != nil {
		return "", fmt.Errorf("translation: %w", err)
	}
	return path, nil
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	return bridgeutil.BuildSwiftBridge(swiftBridgeDir(), "translation", "libTranslationSwiftUIBridge.dylib", "translation")
}

// tryRegisterLibFunc attempts to register a C function from the dylib.
func tryRegisterLibFunc(fptr any, handle uintptr, name string) bool {
	defer func() { recover() }()
	purego.RegisterLibFunc(fptr, handle, name)
	return true
}

// C function variables registered via purego.RegisterLibFunc.
var (
	_TRS_TranslationPresentation func(uintptr, *byte) uintptr
	_TRS_Retain                  func(uintptr) uintptr
	_TRS_Release                 func(uintptr)
	_TRS_FreeString              func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("TranslationSwiftUIBridge") + "_PATH"

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
			"libTranslationSwiftUIBridge.dylib",
			"/usr/local/lib/libTranslationSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_TRS_TranslationPresentation, libHandle, "TRS_TranslationPresentation")
	tryRegisterLibFunc(&_TRS_Retain, libHandle, "TRS_Retain")
	tryRegisterLibFunc(&_TRS_Release, libHandle, "TRS_Release")
	tryRegisterLibFunc(&_TRS_FreeString, libHandle, "TRS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("translation: " + name + ": dylib not loaded") }
	if _TRS_TranslationPresentation == nil {
		_TRS_TranslationPresentation = func(uintptr, *byte) uintptr { stub("TRS_TranslationPresentation"); return 0 }
	}
	if _TRS_Retain == nil {
		_TRS_Retain = func(uintptr) uintptr { stub("TRS_Retain"); return 0 }
	}
	if _TRS_Release == nil {
		_TRS_Release = func(uintptr) { stub("TRS_Release") }
	}
	if _TRS_FreeString == nil {
		_TRS_FreeString = func(*byte) { stub("TRS_FreeString") }
	}
}
