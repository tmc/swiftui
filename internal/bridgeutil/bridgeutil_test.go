package bridgeutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsurePrivateDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "cache")

	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatalf("EnsurePrivateDir create: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat created dir: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("mode after create = %o, want %o", got, want)
	}

	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod dir to 0755: %v", err)
	}
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatalf("EnsurePrivateDir tighten: %v", err)
	}
	info, err = os.Stat(dir)
	if err != nil {
		t.Fatalf("stat tightened dir: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("mode after tighten = %o, want %o", got, want)
	}
}

func TestDiscoverBuiltDylib(t *testing.T) {
	base := t.TempDir()
	lib := "libExampleBridge.dylib"
	path := filepath.Join(base, ".build", "universal", "release", lib)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir dylib dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("dylib"), 0o644); err != nil {
		t.Fatalf("write dylib file: %v", err)
	}

	got, err := DiscoverBuiltDylib(base, lib)
	if err != nil {
		t.Fatalf("DiscoverBuiltDylib: %v", err)
	}
	if got != path {
		t.Fatalf("DiscoverBuiltDylib path = %q, want %q", got, path)
	}
}

func TestSwiftBuildCacheKeyChangesWithSources(t *testing.T) {
	swiftDir := writeTestSwiftPackage(t, "struct Example {}\n")
	installFakeSwift(t, "Apple Swift version 6.2")

	first, err := swiftBuildCacheKey(swiftDir, "libExampleBridge.dylib")
	if err != nil {
		t.Fatalf("swiftBuildCacheKey first: %v", err)
	}

	source := filepath.Join(swiftDir, "Sources", "bridge.swift")
	if err := os.WriteFile(source, []byte("struct Example { let value = 1 }\n"), 0o644); err != nil {
		t.Fatalf("rewrite source file: %v", err)
	}

	second, err := swiftBuildCacheKey(swiftDir, "libExampleBridge.dylib")
	if err != nil {
		t.Fatalf("swiftBuildCacheKey second: %v", err)
	}

	if first == second {
		t.Fatalf("swiftBuildCacheKey = %q after source change, want different key", second)
	}
	if !strings.HasSuffix(second, "-"+runtime.GOARCH) {
		t.Fatalf("swiftBuildCacheKey = %q, want GOARCH suffix %q", second, runtime.GOARCH)
	}
}

func TestSwiftBuildCacheKeyChangesWithToolchainVersion(t *testing.T) {
	swiftDir := writeTestSwiftPackage(t, "struct Example {}\n")
	installFakeSwift(t, "Apple Swift version 6.1")

	first, err := swiftBuildCacheKey(swiftDir, "libExampleBridge.dylib")
	if err != nil {
		t.Fatalf("swiftBuildCacheKey first: %v", err)
	}

	installFakeSwift(t, "Apple Swift version 6.2")

	second, err := swiftBuildCacheKey(swiftDir, "libExampleBridge.dylib")
	if err != nil {
		t.Fatalf("swiftBuildCacheKey second: %v", err)
	}

	if first == second {
		t.Fatalf("swiftBuildCacheKey = %q after swift version change, want different key", second)
	}
}

func writeTestSwiftPackage(t *testing.T, source string) string {
	t.Helper()

	root := t.TempDir()
	pkg := filepath.Join(root, "swift")
	if err := os.MkdirAll(filepath.Join(pkg, "Sources"), 0o755); err != nil {
		t.Fatalf("mkdir Sources: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "Package.swift"), []byte("// swift-tools-version: 6.2\n"), 0o644); err != nil {
		t.Fatalf("write Package.swift: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "Sources", "bridge.swift"), []byte(source), 0o644); err != nil {
		t.Fatalf("write bridge.swift: %v", err)
	}
	return pkg
}

func installFakeSwift(t *testing.T, version string) {
	t.Helper()

	dir := t.TempDir()
	swift := filepath.Join(dir, "swift")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  printf '%s\\n' \"" + version + "\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo unexpected args: \"$@\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(swift, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake swift: %v", err)
	}
	t.Setenv("PATH", dir)
}
