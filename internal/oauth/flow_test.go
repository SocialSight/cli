package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

// fakeAuthServer stands in for the whole MCP OAuth proxy (discovery,
// registration, authorize, token) in one httptest.Server, so Login can be
// exercised end to end without a real Clerk/browser.
type fakeAuthServer struct {
	srv *httptest.Server

	mu             sync.Mutex
	registeredURIs []string
	authorizeCalls int
	codeChallenges map[string]string // code -> code_challenge seen at authorize time

	// denyLogin, if set, makes /oauth2/authorize redirect with an OAuth
	// error instead of a code.
	denyLogin bool
}

func newFakeAuthServer(t *testing.T) *fakeAuthServer {
	t.Helper()
	f := &fakeAuthServer{codeChallenges: map[string]string{}}
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"issuer": %q,
			"authorization_endpoint": "%s/oauth2/authorize",
			"token_endpoint": "%s/oauth2/token",
			"registration_endpoint": "%s/oauth2/register"
		}`, f.srv.URL, f.srv.URL, f.srv.URL, f.srv.URL)
	})

	mux.HandleFunc("/oauth2/register", func(w http.ResponseWriter, r *http.Request) {
		var body clientRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.registeredURIs = append(f.registeredURIs, body.RedirectURIs...)
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id": "fake-client-id"}`))
	})

	mux.HandleFunc("/oauth2/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		redirectURI := q.Get("redirect_uri")
		state := q.Get("state")

		f.mu.Lock()
		f.authorizeCalls++
		if f.denyLogin {
			f.mu.Unlock()
			http.Redirect(w, r, redirectURI+"?error=access_denied&error_description=user+declined&state="+state, http.StatusFound)
			return
		}
		code := fmt.Sprintf("fake-code-%d", f.authorizeCalls)
		f.codeChallenges[code] = q.Get("code_challenge")
		f.mu.Unlock()

		http.Redirect(w, r, redirectURI+"?code="+url.QueryEscape(code)+"&state="+url.QueryEscape(state), http.StatusFound)
	})

	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		code := r.Form.Get("code")
		verifier := r.Form.Get("code_verifier")

		f.mu.Lock()
		wantChallenge, ok := f.codeChallenges[code]
		f.mu.Unlock()
		if !ok {
			http.Error(w, "unknown code", http.StatusBadRequest)
			return
		}

		sum := sha256.Sum256([]byte(verifier))
		gotChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
		if gotChallenge != wantChallenge {
			http.Error(w, fmt.Sprintf("PKCE verification failed: code_verifier does not match the code_challenge from /authorize (got %s, want %s)", gotChallenge, wantChallenge), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "fake-access-token", "refresh_token": "fake-refresh-token", "expires_in": 86400}`))
	})

	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// fakeBrowser simulates a user completing login instantly: it just performs
// the HTTP GET a real browser would, following redirects, which lands on
// the CLI's own loopback listener exactly as a real browser redirect would.
func fakeBrowser(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func TestLoginEndToEnd(t *testing.T) {
	f := newFakeAuthServer(t)

	tok, err := Login(context.Background(), LoginOptions{
		MCPBaseURL:  f.srv.URL,
		OpenBrowser: fakeBrowser,
		Output:      io.Discard,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken != "fake-access-token" || tok.RefreshToken != "fake-refresh-token" {
		t.Fatalf("unexpected token: %+v", tok)
	}
	if tok.ExpiresAt.IsZero() {
		t.Fatal("expected a non-zero ExpiresAt")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.registeredURIs) != 1 {
		t.Fatalf("expected exactly one client registration, got %d", len(f.registeredURIs))
	}
	if f.authorizeCalls != 1 {
		t.Fatalf("expected exactly one authorize call, got %d", f.authorizeCalls)
	}
}

func TestLoginUserDeniesAccess(t *testing.T) {
	f := newFakeAuthServer(t)
	f.denyLogin = true

	_, err := Login(context.Background(), LoginOptions{
		MCPBaseURL:  f.srv.URL,
		OpenBrowser: fakeBrowser,
		Output:      io.Discard,
	})
	if err == nil {
		t.Fatal("expected an error when the user denies login")
	}
	if got := err.Error(); got != "login failed: access_denied: user declined" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestLoginBrowserOpenFailureStillWorks(t *testing.T) {
	// If the browser fails to open (e.g. headless), Login should still print
	// the fallback URL and let the flow proceed rather than aborting -- the
	// user can open it themselves. Here OpenBrowser both reports failure AND
	// (standing in for a human copy-pasting the printed URL) performs the
	// GET, and we assert Login still succeeds.
	f := newFakeAuthServer(t)

	_, err := Login(context.Background(), LoginOptions{
		MCPBaseURL: f.srv.URL,
		OpenBrowser: func(url string) error {
			_ = fakeBrowser(url)
			return fmt.Errorf("no DISPLAY")
		},
		Output: io.Discard,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
}

func TestLoginPrintsAuthorizeURLToOutput(t *testing.T) {
	f := newFakeAuthServer(t)

	buf := &recordingWriter{}
	_, err := Login(context.Background(), LoginOptions{
		MCPBaseURL:  f.srv.URL,
		OpenBrowser: fakeBrowser,
		Output:      buf,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !buf.wroteAnything {
		t.Fatal("expected Login to print the authorize URL to Output as a fallback for when the browser doesn't open automatically")
	}
}

type recordingWriter struct {
	wroteAnything bool
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		w.wroteAnything = true
	}
	return len(p), nil
}
