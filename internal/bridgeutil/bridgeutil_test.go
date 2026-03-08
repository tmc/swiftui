package bridgeutil

import (
	"os"
	"path/filepath"
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
