package config

import (
	"os"
	"path/filepath"

	"github.com/pzep1/lazycont/internal/appmeta"
)

func configPath(dir, name string) string {
	return filepath.Join(dir, name, "config.json")
}

// DefaultPath returns the config file path for this install. Existing configs
// under the legacy lazycont directory are used until a lazycontainer config
// exists; new installs write to the lazycontainer directory.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return defaultPathFrom(dir)
}

func defaultPathFrom(configDir string) (string, error) {
	newPath := configPath(configDir, appmeta.ConfigDir)
	legacyPath := configPath(configDir, appmeta.LegacyConfigDir)

	if _, err := os.Stat(newPath); err == nil {
		return newPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	return newPath, nil
}
