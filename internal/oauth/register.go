package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// clientRegistrationRequest is a minimal RFC 7591 Dynamic Client
// Registration request for a public (no-secret) native-app client.
type clientRegistrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

type clientRegistrationResponse struct {
	ClientID string `json:"client_id"`
}

// RegisterClient dynamically registers a new OAuth client scoped to
// redirectURI. Registration is per-login rather than cached: the CLI binds
// to a fresh ephemeral loopback port each run, and RFC 8252 native-app
// loopback redirects aren't guaranteed to match on port alone across
// authorization server implementations, so matching exactly what was just
// registered avoids that ambiguity entirely. Clerk's DCR endpoint is cheap
// and designed for this per-session usage pattern (it's how MCP clients
// like Claude Desktop already use it).
func RegisterClient(ctx context.Context, registrationEndpoint, redirectURI string) (string, error) {
	reqBody := clientRegistrationRequest{
		ClientName:              "socialsight-cli",
		RedirectURIs:            []string{redirectURI},
		TokenEndpointAuthMethod: "none",
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("client registration failed: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var out clientRegistrationResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("decoding client registration response: %w", err)
	}
	if out.ClientID == "" {
		return "", fmt.Errorf("client registration response had no client_id: %s", string(respBody))
	}
	return out.ClientID, nil
}
