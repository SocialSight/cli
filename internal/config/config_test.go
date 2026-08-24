package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadDeleteRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SOCIALSIGHT_API_KEY", "")

	if key, source, err := APIKey(); err != nil || key != "" || source != "" {
		t.Fatalf("expected no key before save, got key=%q source=%q err=%v", key, source, err)
	}

	if err := SaveAPIKey("ss_live_abc123"); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}

	key, source, err := APIKey()
	if err != nil {
		t.Fatalf("APIKey: %v", err)
	}
	if key != "ss_live_abc123" || source != "file" {
		t.Fatalf("got key=%q source=%q, want ss_live_abc123/file", key, source)
	}

	if err := DeleteAPIKey(); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	if key, source, err := APIKey(); err != nil || key != "" || source != "" {
		t.Fatalf("expected no key after delete, got key=%q source=%q err=%v", key, source, err)
	}

	// Deleting again should be a no-op, not an error.
	if err := DeleteAPIKey(); err != nil {
		t.Fatalf("DeleteAPIKey (again): %v", err)
	}
}

func TestEnvVarOverridesFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveAPIKey("from-file"); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	t.Setenv("SOCIALSIGHT_API_KEY", "from-env")

	key, source, err := APIKey()
	if err != nil {
		t.Fatalf("APIKey: %v", err)
	}
	if key != "from-env" || source != "env" {
		t.Fatalf("got key=%q source=%q, want from-env/env", key, source)
	}
}

func TestPathUnderHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := filepath.Join(home, ".socialsight", "config"); path != want {
		t.Fatalf("got path %q, want %q", path, want)
	}
}

func TestSaveOAuthRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SOCIALSIGHT_API_KEY", "")

	expiresAt := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	if err := SaveOAuth("access-tok", "refresh-tok", expiresAt, "https://staging-mcp.socialsight.ai"); err != nil {
		t.Fatalf("SaveOAuth: %v", err)
	}

	// APIKey() (the generic "give me the bearer token" accessor) should
	// transparently return the OAuth access token.
	key, source, err := APIKey()
	if err != nil {
		t.Fatalf("APIKey: %v", err)
	}
	if key != "access-tok" || source != "file" {
		t.Fatalf("got key=%q source=%q, want access-tok/file", key, source)
	}

	cred, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cred.Method != MethodOAuth {
		t.Fatalf("got Method=%q, want %q", cred.Method, MethodOAuth)
	}
	if cred.RefreshToken != "refresh-tok" {
		t.Fatalf("got RefreshToken=%q", cred.RefreshToken)
	}
	if !cred.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("got ExpiresAt=%v, want %v", cred.ExpiresAt, expiresAt)
	}
	if cred.MCPBaseURL != "https://staging-mcp.socialsight.ai" {
		t.Fatalf("got MCPBaseURL=%q", cred.MCPBaseURL)
	}

	if err := DeleteAPIKey(); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	cred, err = Load()
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}
	if cred.Token != "" || cred.Method != "" {
		t.Fatalf("expected an empty Credential after delete, got %+v", cred)
	}
}

func TestSaveOAuthOverwritesAPIKeyAndViceVersa(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SOCIALSIGHT_API_KEY", "")

	if err := SaveAPIKey("ss_live_abc"); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	if err := SaveOAuth("access-tok", "refresh-tok", time.Time{}, "https://mcp.socialsight.ai"); err != nil {
		t.Fatalf("SaveOAuth: %v", err)
	}
	if cred, err := Load(); err != nil || cred.Method != MethodOAuth {
		t.Fatalf("expected OAuth to have overwritten the API key, got %+v (err=%v)", cred, err)
	}

	if err := SaveAPIKey("ss_live_def"); err != nil {
		t.Fatalf("SaveAPIKey: %v", err)
	}
	cred, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cred.Method != MethodAPIKey || cred.Token != "ss_live_def" {
		t.Fatalf("expected the API key to have overwritten OAuth, got %+v", cred)
	}
	if cred.RefreshToken != "" {
		t.Fatalf("expected stale OAuth fields to be gone, got RefreshToken=%q", cred.RefreshToken)
	}
}

func TestLoadZeroValueWhenNothingConfigured(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SOCIALSIGHT_API_KEY", "")

	cred, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cred.Token != "" || cred.Source != "" || cred.Method != "" {
		t.Fatalf("expected a zero-value Credential, got %+v", cred)
	}
}

func TestMask(t *testing.T) {
	cases := map[string]string{
		"ss_live_ab12cdefgh34ij": "ss_live_ab12...",
		"short":                  "***",
	}
	for in, want := range cases {
		if got := Mask(in); got != want {
			t.Errorf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
}
