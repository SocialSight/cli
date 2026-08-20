package config

import (
	"path/filepath"
	"testing"
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
