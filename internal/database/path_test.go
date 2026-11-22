package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirHonoursOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("POSTWOMAN_DATA_DIR", dir)

	if got := DataDir(); got != dir {
		t.Fatalf("DataDir = %q, want %q", got, dir)
	}
	if got, want := GetDBPath(), filepath.Join(dir, "postwoman.db"); got != want {
		t.Fatalf("GetDBPath = %q, want %q", got, want)
	}
	if !IsPortable() {
		t.Fatal("POSTWOMAN_DATA_DIR should enable portable mode")
	}
	if got, want := PortableWebviewDataPath(), filepath.Join(dir, "webview2"); got != want {
		t.Fatalf("PortableWebviewDataPath = %q, want %q", got, want)
	}
}

func TestPortableDataDirFromExecutable(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "Postwoman.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := portableDataDirFrom(exe); ok {
		t.Fatal("portable mode must require the marker file")
	}
	if err := os.WriteFile(filepath.Join(dir, portableMarker), []byte("portable"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := portableDataDirFrom(exe)
	if !ok {
		t.Fatal("expected portable mode when portable.txt is present")
	}
	if want := filepath.Join(dir, "data"); got != want {
		t.Fatalf("portable data dir = %q, want %q", got, want)
	}
}
