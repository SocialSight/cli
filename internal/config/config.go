// Package config manages the CLI's local credential file (~/.socialsight/config).
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const envAPIKey = "SOCIALSIGHT_API_KEY"

type fileConfig struct {
	APIKey string `json:"api_key"`
}

// Path returns the on-disk location of the config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".socialsight", "config"), nil
}

// APIKey resolves the effective API key and where it came from ("env" or
// "file"). It returns ("", "", nil) if no key is configured anywhere.
func APIKey() (key string, source string, err error) {
	if v := os.Getenv(envAPIKey); v != "" {
		return v, "env", nil
	}

	path, err := Path()
	if err != nil {
		return "", "", err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}
	if cfg.APIKey == "" {
		return "", "", nil
	}
	return cfg.APIKey, "file", nil
}

// SaveAPIKey persists the API key to the config file, creating its parent
// directory if needed. The file is written with owner-only permissions since
// it holds a credential.
func SaveAPIKey(key string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(fileConfig{APIKey: key})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// DeleteAPIKey removes the config file, if present.
func DeleteAPIKey() error {
	path, err := Path()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Mask renders a key for display, e.g. "ss_live_ab12...".
func Mask(key string) string {
	const prefixLen = 12 // matches the backend's own KeyPrefix convention
	if len(key) <= prefixLen {
		return "***"
	}
	return key[:prefixLen] + "..."
}
