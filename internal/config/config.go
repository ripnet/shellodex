package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/ripnet/shellodex/internal/model"
)

const (
	appDir   = "shellodex"
	filename = "shellodex.json"

	credWarning = "Credentials in this file are stored in plaintext. " +
		"Ensure appropriate filesystem permissions (600) and do not commit this file to version control."
)

// DefaultPath returns the platform-appropriate config file path.
// Uses $XDG_CONFIG_HOME on Linux, ~/Library/Application Support on macOS.
// Overridable via --config flag in main.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDir, filename), nil
}

// Load reads the config file at path. Returns a default config if the file
// does not exist (first run).
func Load(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}
	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to path, creating parent directories as needed.
// File is written with mode 0600 to limit credential exposure.
func Save(path string, cfg *model.Config) error {
	cfg.Warning = credWarning
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func defaultConfig() *model.Config {
	return &model.Config{
		Version: 1,
		Warning: credWarning,
	}
}
