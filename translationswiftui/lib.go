package translationswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libTranslationSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libTranslationSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("translationswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("translationswiftui: swift build failed: %w", err)
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
	_TRS_TranslationPresentation func(uintptr, *byte) uintptr
	_TRS_Retain                  func(uintptr) uintptr
	_TRS_Release                 func(uintptr)
	_TRS_FreeString              func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("TranslationSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libTranslationSwiftUIBridge.dylib",
		"/usr/local/lib/libTranslationSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_TRS_TranslationPresentation, libHandle, "TRS_TranslationPresentation")
	tryRegisterLibFunc(&_TRS_Retain, libHandle, "TRS_Retain")
	tryRegisterLibFunc(&_TRS_Release, libHandle, "TRS_Release")
	tryRegisterLibFunc(&_TRS_FreeString, libHandle, "TRS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("translationswiftui: " + name + ": dylib not loaded") }
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
