package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	appconfig "github.com/pzep1/lazycont/internal/config"
	"github.com/pzep1/lazycont/internal/containercli"
	"github.com/pzep1/lazycont/internal/tui"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		if len(args) > 1 {
			fmt.Fprintf(stderr, "lazycont: unexpected argument %q\n", args[1])
			printUsage(stderr)
			return 2
		}

		switch args[0] {
		case "--help", "-h", "help":
			printUsage(stdout)
			return 0
		case "--version", "-v", "version":
			fmt.Fprintf(stdout, "lazycont %s\n", version)
			return 0
		default:
			fmt.Fprintf(stderr, "lazycont: unexpected argument %q\n", args[0])
			printUsage(stderr)
			return 2
		}
	}

	return runTUI(stderr)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `lazycont - terminal UI for Apple's container CLI

Usage:
  lazycont [--help] [--version]

Options:
  --help     Show this help.
  --version  Print the lazycont version.
`)
}

func runTUI(stderr io.Writer) int {
	client := containercli.New("container")
	opts := tui.Options{}
	cfg, path, err := appconfig.LoadDefault()
	if err != nil {
		opts.StartupWarning = configWarning(path, err)
	} else {
		opts.CustomCommands = customCommands(cfg)
		applyConfigToOptions(&opts, cfg)
	}
	opts.ConfigPath = path
	opts.OpenConfigCommand = openConfigCommand
	opts.LoadConfigCommands = loadConfigCommands
	opts.ReloadConfig = reloadConfig
	opts.OpenLinkCommand = openLinkCommand
	program := tea.NewProgram(tui.NewWithOptions(client, opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(stderr, "lazycont: %v\n", err)
		return 1
	}
	return 0
}

func openConfigCommand(path string) (*exec.Cmd, error) {
	if err := appconfig.Ensure(path); err != nil {
		return nil, err
	}
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		editor = "vi"
	}
	return editorCommand(editor, path)
}

// openLinkCommand opens a URL in the default browser on macOS.
func openLinkCommand(url string) (*exec.Cmd, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("url is required")
	}
	return exec.Command("open", url), nil
}

func editorCommand(editor string, path string) (*exec.Cmd, error) {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, errors.New("editor is required")
	}
	args := append(append([]string(nil), parts[1:]...), path)
	return exec.Command(parts[0], args...), nil
}

func loadConfigCommands() ([]tui.CustomCommand, error) {
	cfg, _, err := appconfig.LoadDefault()
	if err != nil {
		return nil, err
	}
	return customCommands(cfg), nil
}

// reloadConfig re-reads the config file and returns a fresh Options so the TUI
// can reapply commands, theme, and gui/log settings after an in-session edit.
func reloadConfig() (tui.Options, error) {
	cfg, _, err := appconfig.LoadDefault()
	if err != nil {
		return tui.Options{}, err
	}
	opts := tui.Options{CustomCommands: customCommands(cfg)}
	applyConfigToOptions(&opts, cfg)
	return opts, nil
}

func configWarning(path string, err error) string {
	if path == "" {
		return fmt.Sprintf("config: %v", err)
	}
	return fmt.Sprintf("config %s: %v", path, err)
}

// customCommands flattens the legacy flat command list and every per-context
// customCommands group into a single list. Placeholder expansion ({container},
// {image}, …) naturally scopes each command to the relevant resource.
func customCommands(cfg appconfig.Config) []tui.CustomCommand {
	var out []tui.CustomCommand
	add := func(commands []appconfig.Command) {
		for _, command := range commands {
			out = append(out, tui.CustomCommand{
				Name:   command.Name,
				Args:   append([]string(nil), command.Args...),
				Attach: command.Attach,
			})
		}
	}
	add(cfg.Commands)
	for _, context := range appconfig.ContainerContexts {
		add(cfg.CustomCommands[context])
	}
	return out
}

func applyConfigToOptions(opts *tui.Options, cfg appconfig.Config) {
	opts.ScreenMode = cfg.GUI.ScreenMode
	opts.SidePanelWidth = cfg.GUI.SidePanelWidth
	opts.BorderStyle = cfg.GUI.Border
	opts.ActiveColor = cfg.GUI.Theme.ActiveBorderColor
	opts.SelectedBgColor = cfg.GUI.Theme.SelectedLineBgColor
	opts.LogsTail = cfg.Logs.Tail
	opts.LogsSince = cfg.Logs.Since
	if cfg.RefreshIntervalMs > 0 {
		opts.RefreshInterval = time.Duration(cfg.RefreshIntervalMs) * time.Millisecond
	}
}
