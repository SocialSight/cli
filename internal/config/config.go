// Package config manages the CLI's local credential file (~/.socialsight/config).
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const envAPIKey = "SOCIALSIGHT_API_KEY"

// Auth method discriminators stored in the config file.
const (
	MethodAPIKey = "api_key"
	MethodOAuth  = "oauth"
)

type fileConfig struct {
	AuthMethod   string `json:"auth_method,omitempty"` // MethodOAuth if set; empty/absent means a legacy api_key-only file
	APIKey       string `json:"api_key,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`         // unix seconds; 0 means unknown/no expiry
	MCPBaseURL   string `json:"oauth_mcp_base_url,omitempty"` // needed to revoke at logout
}

// Credential is the resolved bearer credential the CLI should send as
// "Authorization: Bearer <Token>", plus metadata about where it came from.
type Credential struct {
	Token  string
	Source string // "env" | "file" | "" (nothing configured)
	Method string // MethodAPIKey | MethodOAuth | "" (nothing configured)

	// OAuth-only; zero values otherwise.
	RefreshToken string
	ExpiresAt    time.Time
	MCPBaseURL   string
}

// Path returns the on-disk location of the config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".socialsight", "config"), nil
}

// Load resolves the effective credential: SOCIALSIGHT_API_KEY takes
// priority over the config file. Returns a zero-value Credential (no error)
// if nothing is configured.
func Load() (*Credential, error) {
	if v := os.Getenv(envAPIKey); v != "" {
		return &Credential{Token: v, Source: "env", Method: MethodAPIKey}, nil
	}

	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Credential{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	switch {
	case cfg.AuthMethod == MethodOAuth && cfg.AccessToken != "":
		cred := &Credential{
			Token:        cfg.AccessToken,
			Source:       "file",
			Method:       MethodOAuth,
			RefreshToken: cfg.RefreshToken,
			MCPBaseURL:   cfg.MCPBaseURL,
		}
		if cfg.ExpiresAt > 0 {
			cred.ExpiresAt = time.Unix(cfg.ExpiresAt, 0)
		}
		return cred, nil
	case cfg.APIKey != "":
		return &Credential{Token: cfg.APIKey, Source: "file", Method: MethodAPIKey}, nil
	default:
		return &Credential{}, nil
	}
}

// APIKey resolves the effective bearer credential and where it came from
// ("env" or "file"). Despite the name, this also returns an OAuth access
// token when that's what's stored -- from every caller's point of view it's
// just the bearer token to send, api key or not. Returns ("", "", nil) if
// nothing is configured.
func APIKey() (key string, source string, err error) {
	cred, err := Load()
	if err != nil {
		return "", "", err
	}
	return cred.Token, cred.Source, nil
}

// SaveAPIKey persists a plain API key to the config file, creating its
// parent directory if needed. The file is written with owner-only
// permissions since it holds a credential. Overwrites any previously
// stored OAuth credential.
func SaveAPIKey(key string) error {
	return writeConfig(fileConfig{APIKey: key})
}

// SaveOAuth persists an OAuth credential obtained via internal/oauth.Login.
// mcpBaseURL is recorded so a future `auth logout` can revoke the token
// against the right host. expiresAt may be zero if the token has no known
// expiry. Overwrites any previously stored API key.
func SaveOAuth(accessToken, refreshToken string, expiresAt time.Time, mcpBaseURL string) error {
	cfg := fileConfig{
		AuthMethod:   MethodOAuth,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		MCPBaseURL:   mcpBaseURL,
	}
	if !expiresAt.IsZero() {
		cfg.ExpiresAt = expiresAt.Unix()
	}
	return writeConfig(cfg)
}

func writeConfig(cfg fileConfig) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// DeleteAPIKey removes the config file, if present -- whichever credential
// type it holds.
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

// Mask renders a credential for display, e.g. "ss_live_ab12..." for an API
// key, or a truncated prefix for an OAuth token.
func Mask(key string) string {
	const prefixLen = 12 // matches the backend's own KeyPrefix convention
	if len(key) <= prefixLen {
		return "***"
	}
	return key[:prefixLen] + "..."
}
