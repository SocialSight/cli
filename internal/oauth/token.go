package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Token is the result of a successful authorization_code exchange.
type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time // zero if the server didn't return expires_in
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// ExchangeCode redeems an authorization code for tokens.
func ExchangeCode(ctx context.Context, tokenEndpoint, clientID, code, codeVerifier, redirectURI string) (*Token, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"code_verifier": {codeVerifier},
		"redirect_uri":  {redirectURI},
	}
	return postForm(ctx, tokenEndpoint, form)
}

func postForm(ctx context.Context, tokenEndpoint string, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var out tokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding token response (HTTP %d): %w: %s", resp.StatusCode, err, string(body))
	}
	if out.Error != "" {
		if out.ErrorDesc != "" {
			return nil, fmt.Errorf("%s: %s", out.Error, out.ErrorDesc)
		}
		return nil, fmt.Errorf("%s", out.Error)
	}
	if resp.StatusCode != http.StatusOK || out.AccessToken == "" {
		return nil, fmt.Errorf("token exchange failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	tok := &Token{AccessToken: out.AccessToken, RefreshToken: out.RefreshToken}
	if out.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	}
	return tok, nil
}
