package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingConfigReturnsEmptyConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Commands) != 0 {
		t.Fatalf("commands = %#v, want empty", cfg.Commands)
	}
}

func TestLoadCustomCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, `{
		"commands": [
			{"name": " Images ", "args": [" image ", " list ", "--format", " json "]},
			{"name": "Disk usage", "args": ["system", "df"]},
			{"name": "Empty env", "args": ["run", "--env", ""]}
		]
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []Command{
		{Name: "Images", Args: []string{"image", "list", "--format", "json"}},
		{Name: "Disk usage", Args: []string{"system", "df"}},
		{Name: "Empty env", Args: []string{"run", "--env", ""}},
	}
	if !reflect.DeepEqual(cfg.Commands, want) {
		t.Fatalf("commands mismatch\nwant: %#v\n got: %#v", want, cfg.Commands)
	}
}

func TestLoadRejectsInvalidCustomCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, `{"commands": [{"name": "Broken", "args": []}]}`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "commands[0].args") {
		t.Fatalf("err = %v, want args validation error", err)
	}
}

func TestLoadGUILogsAndPerContextCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, `{
		"gui": {"sidePanelWidth": 0.4, "screenMode": "half", "border": "double", "theme": {"activeBorderColor": "201", "selectedLineBgColor": "57"}},
		"logs": {"tail": 500, "since": "10m"},
		"refreshIntervalMs": 2000,
		"customCommands": {
			"containers": [{"name": "Shell", "args": ["exec", "-it", "{container}", "sh"], "attach": true}]
		}
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GUI.ScreenMode != "half" || cfg.GUI.Border != "double" || cfg.GUI.SidePanelWidth != 0.4 {
		t.Fatalf("gui mismatch: %#v", cfg.GUI)
	}
	if cfg.GUI.Theme.ActiveBorderColor != "201" || cfg.GUI.Theme.SelectedLineBgColor != "57" {
		t.Fatalf("theme mismatch: %#v", cfg.GUI.Theme)
	}
	if cfg.Logs.Tail != 500 || cfg.Logs.Since != "10m" || cfg.RefreshIntervalMs != 2000 {
		t.Fatalf("logs/refresh mismatch: %#v %d", cfg.Logs, cfg.RefreshIntervalMs)
	}
	cc := cfg.CustomCommands["containers"]
	if len(cc) != 1 || cc[0].Name != "Shell" || !cc[0].Attach {
		t.Fatalf("per-context command mismatch: %#v", cc)
	}
}

func TestLoadRejectsInvalidPerContextCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, `{"customCommands": {"images": [{"name": "", "args": ["x"]}]}}`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), `customCommands["images"][0].name`) {
		t.Fatalf("err = %v, want per-context name validation error", err)
	}
}

func TestLoadRejectsTrailingJSONData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, `{"commands": []} {"commands": []}`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "trailing JSON data") {
		t.Fatalf("err = %v, want trailing data error", err)
	}
}

func TestEnsureCreatesStarterConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lazycontainer", "config.json")

	if err := Ensure(path); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != Starter {
		t.Fatalf("starter config mismatch\nwant: %q\n got: %q", Starter, string(body))
	}
}

func TestEnsureLeavesExistingConfigAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, `{"commands":[{"name":"Images","args":["image","list"]}]}`)

	if err := Ensure(path); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(body); !strings.Contains(got, `"Images"`) {
		t.Fatalf("existing config was replaced: %s", got)
	}
}

func TestEnsureRequiresPath(t *testing.T) {
	if err := Ensure(""); err == nil || !strings.Contains(err.Error(), "config path is required") {
		t.Fatalf("err = %v, want path validation error", err)
	}
}

func writeConfig(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
