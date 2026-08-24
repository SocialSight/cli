package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterClient(t *testing.T) {
	const redirectURI = "http://127.0.0.1:12345/callback"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var body clientRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if len(body.RedirectURIs) != 1 || body.RedirectURIs[0] != redirectURI {
			t.Fatalf("unexpected redirect_uris: %+v", body.RedirectURIs)
		}
		if body.TokenEndpointAuthMethod != "none" {
			t.Fatalf("expected a public client (token_endpoint_auth_method=none), got %q", body.TokenEndpointAuthMethod)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id": "test-client-id"}`))
	}))
	defer srv.Close()

	clientID, err := RegisterClient(context.Background(), srv.URL, redirectURI)
	if err != nil {
		t.Fatalf("RegisterClient: %v", err)
	}
	if clientID != "test-client-id" {
		t.Fatalf("got client_id %q, want %q", clientID, "test-client-id")
	}
}

func TestRegisterClientServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server_error"}`))
	}))
	defer srv.Close()

	if _, err := RegisterClient(context.Background(), srv.URL, "http://127.0.0.1:1/callback"); err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}

func TestRegisterClientMissingClientID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	if _, err := RegisterClient(context.Background(), srv.URL, "http://127.0.0.1:1/callback"); err == nil {
		t.Fatal("expected an error when the response has no client_id")
	}
}
