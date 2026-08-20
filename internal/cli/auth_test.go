package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SocialSight/cli/internal/config"
)

const testValidKey = "ss_live_validkeyvalidkeyvalidkey"

func fakeCreditsServer(t *testing.T, validKey string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/credits" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+validKey {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"detail": "Unauthorized"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total_credits": 42, "subscription_credits": 40, "other_credits": 2}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func run(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf.String(), err
}

func TestAuthLoginSavesValidKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := fakeCreditsServer(t, testValidKey)
	t.Setenv("SOCIALSIGHT_API_BASE_URL", srv.URL)

	out, err := run(t, "auth", "login", "--key", testValidKey)
	if err != nil {
		t.Fatalf("auth login: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "Credits remaining: 42") {
		t.Fatalf("expected credit balance in output, got: %s", out)
	}

	key, source, err := config.APIKey()
	if err != nil {
		t.Fatalf("config.APIKey: %v", err)
	}
	if key != testValidKey || source != "file" {
		t.Fatalf("got key=%q source=%q after login", key, source)
	}
}

func TestAuthLoginRejectsInvalidKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := fakeCreditsServer(t, testValidKey)
	t.Setenv("SOCIALSIGHT_API_BASE_URL", srv.URL)

	_, err := run(t, "auth", "login", "--key", "ss_live_wrong")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}

	if key, _, _ := config.APIKey(); key != "" {
		t.Fatalf("expected no key saved after failed login, got %q", key)
	}
}

func TestAuthWhoamiRequiresLogin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := run(t, "auth", "whoami")
	if err == nil {
		t.Fatal("expected error when not logged in")
	}
}

func TestAuthWhoamiAndLogout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := fakeCreditsServer(t, testValidKey)
	t.Setenv("SOCIALSIGHT_API_BASE_URL", srv.URL)

	if _, err := run(t, "auth", "login", "--key", testValidKey); err != nil {
		t.Fatalf("auth login: %v", err)
	}

	out, err := run(t, "auth", "whoami")
	if err != nil {
		t.Fatalf("auth whoami: %v", err)
	}
	if !strings.Contains(out, "Credits remaining: 42") {
		t.Fatalf("expected credit balance in whoami output, got: %s", out)
	}

	if _, err := run(t, "auth", "logout"); err != nil {
		t.Fatalf("auth logout: %v", err)
	}
	if _, err := run(t, "auth", "whoami"); err == nil {
		t.Fatal("expected whoami to fail after logout")
	}
}
