package oauth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCallbackReceivesCodeAndState(t *testing.T) {
	cb, err := NewCallback()
	if err != nil {
		t.Fatalf("NewCallback: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get(cb.RedirectURI + "?code=abc123&state=xyz")
		if err != nil {
			t.Errorf("GET callback: %v", err)
			return
		}
		defer resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := cb.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if result.Code != "abc123" || result.State != "xyz" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCallbackReceivesOAuthError(t *testing.T) {
	cb, err := NewCallback()
	if err != nil {
		t.Fatalf("NewCallback: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get(cb.RedirectURI + "?error=access_denied&error_description=user+said+no")
		if err != nil {
			t.Errorf("GET callback: %v", err)
			return
		}
		defer resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := cb.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if result.Err != "access_denied" || result.ErrDesc != "user said no" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCallbackTimesOut(t *testing.T) {
	cb, err := NewCallback()
	if err != nil {
		t.Fatalf("NewCallback: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := cb.Wait(ctx); err == nil {
		t.Fatal("expected a timeout error when nothing calls back")
	}
}

func TestCallbackRedirectURIUsesEphemeralPort(t *testing.T) {
	cb1, err := NewCallback()
	if err != nil {
		t.Fatalf("NewCallback: %v", err)
	}
	defer cb1.Close()
	cb2, err := NewCallback()
	if err != nil {
		t.Fatalf("NewCallback: %v", err)
	}
	defer cb2.Close()

	if cb1.RedirectURI == cb2.RedirectURI {
		t.Fatalf("two callbacks got the same redirect URI: %s", cb1.RedirectURI)
	}
}
