package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pzep1/lazycont/internal/appmeta"
)

func writeConfigAt(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, path, body)
}

func TestDefaultPathPrefersNewConfigDirectory(t *testing.T) {
	root := t.TempDir()
	newPath := filepath.Join(root, appmeta.ConfigDir, "config.json")
	legacyPath := filepath.Join(root, appmeta.LegacyConfigDir, "config.json")

	writeConfigAt(t, newPath, `{"commands": []}`)
	writeConfigAt(t, legacyPath, `{"commands": []}`)

	got, err := defaultPathFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != newPath {
		t.Fatalf("defaultPathFrom = %q, want %q", got, newPath)
	}
}

func TestDefaultPathUsesLegacyConfigDirectory(t *testing.T) {
	root := t.TempDir()
	legacyPath := filepath.Join(root, appmeta.LegacyConfigDir, "config.json")
	writeConfigAt(t, legacyPath, `{"commands": []}`)

	got, err := defaultPathFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != legacyPath {
		t.Fatalf("defaultPathFrom = %q, want %q", got, legacyPath)
	}
}

func TestDefaultPathDefaultsToNewDirectory(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, appmeta.ConfigDir, "config.json")

	got, err := defaultPathFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("defaultPathFrom = %q, want %q", got, want)
	}
}

func TestDefaultPathUsesUserConfigDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, appmeta.ConfigDir, "config.json")
	if got != want {
		t.Fatalf("DefaultPath = %q, want %q", got, want)
	}
}
