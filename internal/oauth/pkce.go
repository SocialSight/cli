// Package oauth drives the browser-based OAuth 2.0 Authorization Code +
// PKCE login flow against SocialSight's MCP service (which proxies to
// Clerk). See ENG-285 for the design.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func randomURLSafe(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// NewCodeVerifier returns a PKCE code_verifier (RFC 7636 requires 43-128
// chars of the unreserved charset; 32 random bytes base64url-encode to 43).
func NewCodeVerifier() (string, error) {
	return randomURLSafe(32)
}

// CodeChallenge derives the S256 PKCE code_challenge from a code_verifier.
func CodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewState returns a random value for the OAuth state parameter (CSRF
// protection for the redirect).
func NewState() (string, error) {
	return randomURLSafe(16)
}
