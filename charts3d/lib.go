package charts3d

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tmc/swiftui/internal/bridgeutil"

	"github.com/ebitengine/purego"
)

var libHandle uintptr
var loadErr error

func swiftBridgeDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "internal/swift")
}

func discoverBuiltDylib() (string, error) {
	path, err := bridgeutil.DiscoverBuiltDylib(swiftBridgeDir(), "libCharts3DBridge.dylib")
	if err != nil {
		return "", fmt.Errorf("charts3d: %w", err)
	}
	return path, nil
}

func buildSwiftBridge() (string, error) {
	return bridgeutil.BuildSwiftBridge(swiftBridgeDir(), "libCharts3DBridge.dylib", "charts3d")
}

func tryRegisterLibFunc(fptr any, handle uintptr, name string) bool {
	defer func() { recover() }()
	purego.RegisterLibFunc(fptr, handle, name)
	return true
}

var (
	_CHBuildChart3D       func(*byte, int32) uintptr
	_CHRetain             func(uintptr) uintptr
	_CHRelease            func(uintptr)
	_CHSetSurfaceCallback func(uintptr)
)

func init() {
	envKey := "LIB" + strings.ToUpper("Charts3DBridge") + "_PATH"
	var lastErr error

	if path := os.Getenv(envKey); path != "" {
		var err error
		libHandle, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			lastErr = err
		}
	}

	if libHandle == 0 {
		if path, err := discoverBuiltDylib(); err == nil {
			libHandle, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err != nil {
				lastErr = err
			}
		} else {
			lastErr = err
		}
	}

	if libHandle == 0 {
		if path, err := buildSwiftBridge(); err == nil {
			libHandle, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err != nil {
				lastErr = err
			}
		} else {
			lastErr = err
		}
	}

	if libHandle == 0 {
		for _, path := range []string{
			"libCharts3DBridge.dylib",
			"/usr/local/lib/libCharts3DBridge.dylib",
		} {
			var err error
			libHandle, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err == nil {
				break
			}
			lastErr = err
		}
	}

	if libHandle == 0 {
		if _, err := exec.LookPath("swift"); err != nil {
			loadErr = fmt.Errorf("charts3d: swift toolchain not found; install Xcode or Command Line Tools")
		} else if lastErr != nil {
			loadErr = fmt.Errorf("charts3d: failed to load bridge dylib: %w", lastErr)
		} else {
			loadErr = fmt.Errorf("charts3d: bridge dylib not loaded")
		}
		setUnavailableStubs()
		return
	}

	tryRegisterLibFunc(&_CHBuildChart3D, libHandle, "CHBuildChart3D")
	tryRegisterLibFunc(&_CHRetain, libHandle, "CHRetain")
	tryRegisterLibFunc(&_CHRelease, libHandle, "CHRelease")
	tryRegisterLibFunc(&_CHSetSurfaceCallback, libHandle, "CHSetSurfaceCallback")
	setUnavailableStubs()
}

func setUnavailableStubs() {
	stub := func(name string) {
		if loadErr != nil {
			panic("charts3d: " + name + ": " + loadErr.Error())
		}
		panic("charts3d: " + name + ": dylib not loaded")
	}
	if _CHBuildChart3D == nil {
		_CHBuildChart3D = func(*byte, int32) uintptr { stub("CHBuildChart3D"); return 0 }
	}
	if _CHRetain == nil {
		_CHRetain = func(uintptr) uintptr { stub("CHRetain"); return 0 }
	}
	if _CHRelease == nil {
		_CHRelease = func(uintptr) { stub("CHRelease") }
	}
	if _CHSetSurfaceCallback == nil {
		_CHSetSurfaceCallback = func(uintptr) { stub("CHSetSurfaceCallback") }
	}
}
