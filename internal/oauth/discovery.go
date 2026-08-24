package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Metadata is the subset of RFC 8414 authorization server metadata this
// package needs.
type Metadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
}

// Discover fetches the authorization server metadata document from
// mcpBaseURL. This confirms the server is reachable and avoids hardcoding
// its endpoint paths.
func Discover(ctx context.Context, mcpBaseURL string) (*Metadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mcpBaseURL+"/.well-known/oauth-authorization-server", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching OAuth metadata from %s: HTTP %d", mcpBaseURL, resp.StatusCode)
	}

	var meta Metadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decoding OAuth metadata: %w", err)
	}
	if meta.AuthorizationEndpoint == "" || meta.TokenEndpoint == "" || meta.RegistrationEndpoint == "" {
		return nil, fmt.Errorf("OAuth metadata from %s is missing required endpoints", mcpBaseURL)
	}
	return &meta, nil
}
