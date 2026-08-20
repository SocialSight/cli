package client

import (
	"context"
	"net/http"
	"os"
)

// DefaultBaseURL is the production SocialSight API.
const DefaultBaseURL = "https://api.socialsight.ai"

const envBaseURL = "SOCIALSIGHT_API_BASE_URL"

// BaseURL returns SOCIALSIGHT_API_BASE_URL if set, otherwise DefaultBaseURL.
// The env var override exists so the CLI can be pointed at staging without a
// dedicated flag on every command.
func BaseURL() string {
	if v := os.Getenv(envBaseURL); v != "" {
		return v
	}
	return DefaultBaseURL
}

// NewAuthenticated returns a client that sends apiKey as a bearer token on
// every request.
func NewAuthenticated(baseURL, apiKey string) (*ClientWithResponses, error) {
	return NewClientWithResponses(baseURL, WithRequestEditorFn(
		func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return nil
		},
	))
}
