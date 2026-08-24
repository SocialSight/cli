package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// DefaultBaseURL is the production SocialSight API.
const DefaultBaseURL = "https://api.socialsight.ai"

// DefaultMCPBaseURL and stagingMCPBaseURL are the OAuth-capable MCP service
// hosts, which sit at a different host than the plain API (see ENG-285).
const (
	DefaultMCPBaseURL = "https://mcp.socialsight.ai"
	stagingMCPBaseURL = "https://staging-mcp.socialsight.ai"
	stagingAPIBaseURL = "https://staging-api.socialsight.ai"
)

const (
	envBaseURL    = "SOCIALSIGHT_API_BASE_URL"
	envMCPBaseURL = "SOCIALSIGHT_MCP_BASE_URL"
)

// BaseURL returns SOCIALSIGHT_API_BASE_URL if set, otherwise DefaultBaseURL.
// The env var override exists so the CLI can be pointed at staging without a
// dedicated flag on every command.
func BaseURL() string {
	if v := os.Getenv(envBaseURL); v != "" {
		return v
	}
	return DefaultBaseURL
}

// MCPBaseURL returns the OAuth-capable MCP service host to use for the
// browser login flow. SOCIALSIGHT_MCP_BASE_URL always wins if set;
// otherwise it's derived from the active API base URL (prod -> prod MCP
// host, staging -> staging MCP host). Any other custom API base URL (e.g. a
// local test server) can't be mapped automatically, so that's an error
// asking for the env var explicitly.
func MCPBaseURL() (string, error) {
	if v := os.Getenv(envMCPBaseURL); v != "" {
		return v, nil
	}

	switch base := BaseURL(); {
	case base == DefaultBaseURL:
		return DefaultMCPBaseURL, nil
	case base == stagingAPIBaseURL || strings.Contains(base, "staging"):
		return stagingMCPBaseURL, nil
	default:
		return "", fmt.Errorf("can't determine the MCP host for API base URL %q; set %s explicitly", base, envMCPBaseURL)
	}
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
