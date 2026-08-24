package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Fatalf("unexpected grant_type: %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "test-code" {
			t.Fatalf("unexpected code: %q", r.Form.Get("code"))
		}
		if r.Form.Get("code_verifier") != "test-verifier" {
			t.Fatalf("unexpected code_verifier: %q", r.Form.Get("code_verifier"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "AT", "refresh_token": "RT", "expires_in": 86400, "token_type": "Bearer"}`))
	}))
	defer srv.Close()

	before := time.Now()
	tok, err := ExchangeCode(context.Background(), srv.URL, "client-id", "test-code", "test-verifier", "http://127.0.0.1:1/callback")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tok.AccessToken != "AT" || tok.RefreshToken != "RT" {
		t.Fatalf("unexpected token: %+v", tok)
	}
	wantExpiry := before.Add(86400 * time.Second)
	if tok.ExpiresAt.Before(wantExpiry.Add(-5*time.Second)) || tok.ExpiresAt.After(wantExpiry.Add(5*time.Second)) {
		t.Fatalf("ExpiresAt %v not close to expected %v", tok.ExpiresAt, wantExpiry)
	}
}

func TestExchangeCodeOAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid_grant", "error_description": "code expired"}`))
	}))
	defer srv.Close()

	_, err := ExchangeCode(context.Background(), srv.URL, "client-id", "bad-code", "verifier", "http://127.0.0.1:1/callback")
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := err.Error(); got != "invalid_grant: code expired" {
		t.Fatalf("unexpected error message: %q", got)
	}
}
