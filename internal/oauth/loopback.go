package oauth

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
)

// CallbackResult is what the loopback listener captures from the OAuth
// redirect.
type CallbackResult struct {
	Code    string
	State   string
	Err     string // the OAuth "error" param, if the user denied access etc.
	ErrDesc string
}

// Callback listens on a loopback port for exactly one OAuth redirect,
// serving a human-readable response page either way. RedirectURI is the
// exact value to register and use in the authorize request. Wait blocks
// until a callback is received or ctx is done.
type Callback struct {
	RedirectURI string

	listener net.Listener
	server   *http.Server
	resultCh chan CallbackResult
}

// NewCallback binds an ephemeral loopback port and starts serving. Call
// Wait to block for the redirect, and Close when done (Wait calls Close
// itself, so an explicit Close is only needed if the caller gives up before
// a callback arrives).
func NewCallback() (*Callback, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting local callback listener: %w", err)
	}

	cb := &Callback{
		RedirectURI: fmt.Sprintf("http://127.0.0.1:%d/callback", listener.Addr().(*net.TCPAddr).Port),
		listener:    listener,
		resultCh:    make(chan CallbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", cb.handle)
	cb.server = &http.Server{Handler: mux}

	go func() { _ = cb.server.Serve(listener) }()

	return cb, nil
}

func (cb *Callback) handle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	result := CallbackResult{
		Code:    q.Get("code"),
		State:   q.Get("state"),
		Err:     q.Get("error"),
		ErrDesc: q.Get("error_description"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if result.Err != "" {
		fmt.Fprintf(w, "<html><body><h3>Login failed: %s</h3><p>You can close this tab and return to your terminal.</p></body></html>", html.EscapeString(result.Err))
	} else {
		fmt.Fprint(w, "<html><body><h3>Signed in to SocialSight</h3><p>You can close this tab and return to your terminal.</p></body></html>")
	}

	select {
	case cb.resultCh <- result:
	default:
	}
}

// Wait blocks until a redirect is received or ctx is done, then shuts down
// the listener.
func (cb *Callback) Wait(ctx context.Context) (CallbackResult, error) {
	defer cb.Close()
	select {
	case result := <-cb.resultCh:
		return result, nil
	case <-ctx.Done():
		return CallbackResult{}, ctx.Err()
	}
}

// Close shuts down the listener. Safe to call multiple times.
func (cb *Callback) Close() {
	_ = cb.server.Close()
}
