package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Commands          []Command            `json:"commands"`
	CustomCommands    map[string][]Command `json:"customCommands"`
	GUI               GUI                  `json:"gui"`
	Logs              Logs                 `json:"logs"`
	RefreshIntervalMs int                  `json:"refreshIntervalMs"`
}

type Command struct {
	Name   string   `json:"name"`
	Args   []string `json:"args"`
	Attach bool     `json:"attach"`
}

// GUI holds appearance and layout preferences.
type GUI struct {
	SidePanelWidth float64 `json:"sidePanelWidth"`
	ScreenMode     string  `json:"screenMode"`
	Border         string  `json:"border"`
	Theme          Theme   `json:"theme"`
}

// Theme holds colour overrides (256-colour codes or names understood by the
// terminal profile).
type Theme struct {
	ActiveBorderColor   string `json:"activeBorderColor"`
	SelectedLineBgColor string `json:"selectedLineBgColor"`
}

// Logs controls log retrieval defaults.
type Logs struct {
	Tail  int    `json:"tail"`
	Since string `json:"since"`
}

const Starter = `{
  "commands": []
}
`

// ContainerContexts enumerates the customCommands contexts that map to
// lazycont resource panes.
var ContainerContexts = []string{
	"containers", "images", "volumes", "networks", "machines", "registries", "builder", "system",
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazycont", "config.json"), nil
}

func LoadDefault() (Config, string, error) {
	path, err := DefaultPath()
	if err != nil {
		return Config{}, "", err
	}
	cfg, err := Load(path)
	return cfg, path, err
}

func Ensure(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("config path is required")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(Starter), 0o600)
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("config contains trailing JSON data")
	}
	if err := cfg.normalize(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) normalize() error {
	for idx := range c.Commands {
		if err := normalizeCommand(&c.Commands[idx], fmt.Sprintf("commands[%d]", idx)); err != nil {
			return err
		}
	}
	for context, commands := range c.CustomCommands {
		for idx := range commands {
			label := fmt.Sprintf("customCommands[%q][%d]", context, idx)
			if err := normalizeCommand(&commands[idx], label); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeCommand(command *Command, label string) error {
	command.Name = strings.TrimSpace(command.Name)
	if command.Name == "" {
		return fmt.Errorf("%s.name is required", label)
	}
	if len(command.Args) == 0 || strings.TrimSpace(command.Args[0]) == "" {
		return fmt.Errorf("%s.args must start with a container subcommand", label)
	}
	for argIndex := range command.Args {
		command.Args[argIndex] = strings.TrimSpace(command.Args[argIndex])
	}
	return nil
}
