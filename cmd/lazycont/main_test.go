package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	appconfig "github.com/pzep1/lazycont/internal/config"
	"github.com/pzep1/lazycont/internal/tui"
)

func TestRunPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if got, want := strings.TrimSpace(stdout.String()), "lazycont dev"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") || !strings.Contains(got, "--version") {
		t.Fatalf("stdout = %q, want usage with --version", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsUnexpectedArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"containers"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run returned %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, `unexpected argument "containers"`) || !strings.Contains(got, "Usage:") {
		t.Fatalf("stderr = %q, want error and usage", got)
	}
}

func TestRunRejectsTrailingArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version", "extra"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run returned %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, `unexpected argument "extra"`) || !strings.Contains(got, "Usage:") {
		t.Fatalf("stderr = %q, want error and usage", got)
	}
}

func TestConfigWarningIncludesPathWhenAvailable(t *testing.T) {
	got := configWarning("/tmp/lazycont/config.json", errors.New("bad json"))
	want := "config /tmp/lazycont/config.json: bad json"
	if got != want {
		t.Fatalf("configWarning = %q, want %q", got, want)
	}
}

func TestCustomCommandsFlattensFlatAndPerContextCommands(t *testing.T) {
	cfg := appconfig.Config{
		Commands: []appconfig.Command{{
			Name: "Images",
			Args: []string{"image", "list"},
		}},
		CustomCommands: map[string][]appconfig.Command{
			"containers": {{
				Name:   "Shell",
				Args:   []string{"exec", "-it", "{container}", "sh"},
				Attach: true,
			}},
		},
	}

	got := customCommands(cfg)
	want := []tui.CustomCommand{
		{Name: "Images", Args: []string{"image", "list"}},
		{Name: "Shell", Args: []string{"exec", "-it", "{container}", "sh"}, Attach: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("customCommands mismatch\nwant: %#v\n got: %#v", want, got)
	}

	cfg.Commands[0].Args[0] = "mutated"
	if got[0].Args[0] != "image" {
		t.Fatalf("customCommands did not copy args: %#v", got)
	}
}

func TestApplyConfigToOptionsMapsGUIAndLogs(t *testing.T) {
	cfg := appconfig.Config{
		GUI: appconfig.GUI{
			SidePanelWidth: 0.4,
			ScreenMode:     "half",
			Border:         "double",
			Theme:          appconfig.Theme{ActiveBorderColor: "201", SelectedLineBgColor: "57"},
		},
		Logs:              appconfig.Logs{Tail: 500, Since: "10m"},
		RefreshIntervalMs: 2000,
	}
	var opts tui.Options
	applyConfigToOptions(&opts, cfg)
	if opts.ScreenMode != "half" || opts.SidePanelWidth != 0.4 || opts.BorderStyle != "double" {
		t.Fatalf("gui mapping mismatch: %#v", opts)
	}
	if opts.ActiveColor != "201" || opts.SelectedBgColor != "57" {
		t.Fatalf("theme mapping mismatch: %#v", opts)
	}
	if opts.LogsTail != 500 || opts.LogsSince != "10m" {
		t.Fatalf("logs mapping mismatch: %#v", opts)
	}
	if opts.RefreshInterval != 2*time.Second {
		t.Fatalf("refresh mapping mismatch: %v", opts.RefreshInterval)
	}
}

func TestEditorCommandAppendsConfigPath(t *testing.T) {
	cmd, err := editorCommand("code --wait", "/tmp/lazycont/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "code" {
		t.Fatalf("Path = %q, want code", cmd.Path)
	}
	wantArgs := []string{"code", "--wait", "/tmp/lazycont/config.json"}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Fatalf("Args mismatch\nwant: %#v\n got: %#v", wantArgs, cmd.Args)
	}
}

func TestEditorCommandRequiresEditor(t *testing.T) {
	if _, err := editorCommand(" ", "/tmp/lazycont/config.json"); err == nil || !strings.Contains(err.Error(), "editor is required") {
		t.Fatalf("err = %v, want editor validation error", err)
	}
}
