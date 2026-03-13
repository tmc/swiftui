package spritekitswiftui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

// libHandle is the handle to the loaded libSpriteKitSwiftUIBridge dylib.
var libHandle uintptr

// swiftBridgeDir returns the path to the vendored Swift bridge source directory.
func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "swift")
}

// builtDylibPath returns the expected path of the built dylib.
func builtDylibPath() string {
	return filepath.Join(swiftBridgeDir(), ".build", "arm64-apple-macosx", "release", "libSpriteKitSwiftUIBridge.dylib")
}

// buildSwiftBridge builds the vendored Swift bridge dylib if needed.
func buildSwiftBridge() (string, error) {
	path := builtDylibPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	dir := swiftBridgeDir()
	if _, err := os.Stat(filepath.Join(dir, "Package.swift")); err != nil {
		return "", fmt.Errorf("spritekitswiftui: swift bridge source not found at %s", dir)
	}
	cmd := exec.Command("swift", "build", "-c", "release", "--quiet")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("spritekitswiftui: swift build failed: %w", err)
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
	_SPS_SpriteViewCreate               func(uintptr) uintptr
	_SPS_SpriteViewCreateWithTransition func(uintptr, uintptr) uintptr
	_SPS_SpriteViewCreateWithOptions    func(uintptr, bool, int32) uintptr
	_SPS_Retain                         func(uintptr) uintptr
	_SPS_Release                        func(uintptr)
	_SPS_FreeString                     func(*byte)
)

func init() {
	envKey := "LIB" + strings.ToUpper("SpriteKitSwiftUIBridge") + "_PATH"
	paths := []string{
		os.Getenv(envKey),
		"libSpriteKitSwiftUIBridge.dylib",
		"/usr/local/lib/libSpriteKitSwiftUIBridge.dylib",
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
	tryRegisterLibFunc(&_SPS_SpriteViewCreate, libHandle, "SPS_SpriteViewCreate")
	tryRegisterLibFunc(&_SPS_SpriteViewCreateWithTransition, libHandle, "SPS_SpriteViewCreateWithTransition")
	tryRegisterLibFunc(&_SPS_SpriteViewCreateWithOptions, libHandle, "SPS_SpriteViewCreateWithOptions")
	tryRegisterLibFunc(&_SPS_Retain, libHandle, "SPS_Retain")
	tryRegisterLibFunc(&_SPS_Release, libHandle, "SPS_Release")
	tryRegisterLibFunc(&_SPS_FreeString, libHandle, "SPS_FreeString")

	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) { panic("spritekitswiftui: " + name + ": dylib not loaded") }
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
