package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscover(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/oauth-authorization-server" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"issuer": "https://example.test",
			"authorization_endpoint": "https://example.test/oauth2/authorize",
			"token_endpoint": "https://example.test/oauth2/token",
			"registration_endpoint": "https://example.test/oauth2/register",
			"revocation_endpoint": "https://example.test/oauth2/revoke"
		}`))
	}))
	defer srv.Close()

	meta, err := Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if meta.AuthorizationEndpoint != "https://example.test/oauth2/authorize" {
		t.Fatalf("unexpected AuthorizationEndpoint: %q", meta.AuthorizationEndpoint)
	}
	if meta.TokenEndpoint != "https://example.test/oauth2/token" {
		t.Fatalf("unexpected TokenEndpoint: %q", meta.TokenEndpoint)
	}
	if meta.RegistrationEndpoint != "https://example.test/oauth2/register" {
		t.Fatalf("unexpected RegistrationEndpoint: %q", meta.RegistrationEndpoint)
	}
}

func TestDiscoverMissingEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer": "https://example.test"}`))
	}))
	defer srv.Close()

	if _, err := Discover(context.Background(), srv.URL); err == nil {
		t.Fatal("expected an error for metadata missing required endpoints")
	}
}

func TestDiscoverHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := Discover(context.Background(), srv.URL); err == nil {
		t.Fatal("expected an error for a 404 response")
	}
}
