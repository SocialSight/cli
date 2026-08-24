package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestCodeChallengeMatchesRFC7636Vector(t *testing.T) {
	// RFC 7636 Appendix B example.
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got := CodeChallenge(verifier); got != want {
		t.Fatalf("CodeChallenge(%q) = %q, want %q", verifier, got, want)
	}
}

func TestCodeChallengeIsSHA256Base64URL(t *testing.T) {
	verifier, err := NewCodeVerifier()
	if err != nil {
		t.Fatalf("NewCodeVerifier: %v", err)
	}
	if len(verifier) < 43 {
		t.Fatalf("verifier %q shorter than RFC 7636's 43-char minimum", verifier)
	}

	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if got := CodeChallenge(verifier); got != want {
		t.Fatalf("CodeChallenge = %q, want %q", got, want)
	}
}

func TestNewVerifierAndStateAreRandomAndDistinct(t *testing.T) {
	v1, _ := NewCodeVerifier()
	v2, _ := NewCodeVerifier()
	if v1 == v2 {
		t.Fatal("two calls to NewCodeVerifier produced the same value")
	}

	s1, _ := NewState()
	s2, _ := NewState()
	if s1 == s2 {
		t.Fatal("two calls to NewState produced the same value")
	}
	if s1 == v1 {
		t.Fatal("NewState and NewCodeVerifier collided (shouldn't happen, but shouldn't quietly reuse either)")
	}
}
