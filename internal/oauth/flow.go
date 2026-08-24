package oauth

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"
)

const defaultTimeout = 5 * time.Minute

// LoginOptions configures Login. Zero-value OpenBrowser/Output/Timeout fall
// back to sensible defaults (browser.OpenURL, os.Stderr, 5 minutes).
type LoginOptions struct {
	MCPBaseURL  string
	OpenBrowser func(url string) error
	Output      io.Writer
	Timeout     time.Duration
}

// Login runs the full Authorization Code + PKCE flow: discover the
// authorization server, register a client scoped to a fresh loopback
// redirect URI, open the browser, wait for the redirect, and exchange the
// code for a token.
func Login(ctx context.Context, opts LoginOptions) (*Token, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	meta, err := Discover(ctx, opts.MCPBaseURL)
	if err != nil {
		return nil, fmt.Errorf("discovering OAuth endpoints: %w", err)
	}

	callback, err := NewCallback()
	if err != nil {
		return nil, err
	}
	// Wait() below closes it on every path once a result (or ctx
	// cancellation) arrives; this covers the registration/URL-building
	// steps in between in case they error out first.
	defer callback.Close()

	clientID, err := RegisterClient(ctx, meta.RegistrationEndpoint, callback.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("registering OAuth client: %w", err)
	}

	verifier, err := NewCodeVerifier()
	if err != nil {
		return nil, err
	}
	state, err := NewState()
	if err != nil {
		return nil, err
	}

	authorizeURL := buildAuthorizeURL(meta.AuthorizationEndpoint, clientID, callback.RedirectURI, verifier, state)

	fmt.Fprintf(opts.Output, "Opening your browser to sign in...\nIf it doesn't open automatically, visit:\n%s\n\n", authorizeURL)
	openBrowser := opts.OpenBrowser
	if openBrowser == nil {
		openBrowser = defaultOpenBrowser
	}
	if err := openBrowser(authorizeURL); err != nil {
		fmt.Fprintf(opts.Output, "(couldn't open a browser automatically: %v)\n", err)
	}
	fmt.Fprintln(opts.Output, "Waiting for you to finish signing in...")

	result, err := callback.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for browser login: %w", err)
	}
	if result.Err != "" {
		if result.ErrDesc != "" {
			return nil, fmt.Errorf("login failed: %s: %s", result.Err, result.ErrDesc)
		}
		return nil, fmt.Errorf("login failed: %s", result.Err)
	}
	if result.State != state {
		return nil, fmt.Errorf("OAuth state mismatch (possible CSRF); aborting")
	}
	if result.Code == "" {
		return nil, fmt.Errorf("no authorization code in callback")
	}

	tok, err := ExchangeCode(ctx, meta.TokenEndpoint, clientID, result.Code, verifier, callback.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}
	return tok, nil
}

func buildAuthorizeURL(authorizationEndpoint, clientID, redirectURI, codeVerifier, state string) string {
	q := url.Values{
		"response_type": {"code"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		// "openid" is listed in the discovery metadata's scopes_supported
		// but is actually rejected for dynamically registered clients
		// ("not allowed to request scope 'openid'") -- confirmed against
		// real staging. "email offline_access profile" is what a fresh
		// client registration is actually granted.
		"scope":                 {"email offline_access profile"},
		"state":                 {state},
		"code_challenge":        {CodeChallenge(codeVerifier)},
		"code_challenge_method": {"S256"},
	}
	return authorizationEndpoint + "?" + q.Encode()
}
