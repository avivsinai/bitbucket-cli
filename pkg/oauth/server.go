package oauth

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"time"
)

// CallbackServer runs a temporary localhost HTTP server to receive the OAuth
// authorization code redirect from the browser.
type CallbackServer struct {
	listener net.Listener
	server   *http.Server
	result   chan callbackResult
}

type callbackResult struct {
	code  string
	state string
	err   error
}

// NewCallbackServer starts a localhost HTTP server on a random available port.
// Call Close when done to release the port.
func NewCallbackServer() (*CallbackServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}

	s := &CallbackServer{
		listener: ln,
		result:   make(chan callbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		_ = s.server.Serve(ln)
	}()

	return s, nil
}

// Port returns the TCP port the server is listening on.
func (s *CallbackServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// RedirectURI returns the full callback URL to use as the OAuth redirect_uri.
func (s *CallbackServer) RedirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", s.Port())
}

// Wait blocks until an authorization code is received or the context is cancelled.
// Returns the authorization code and state parameter from the callback.
func (s *CallbackServer) Wait(ctx context.Context) (code, state string, err error) {
	select {
	case res := <-s.result:
		return res.code, res.state, res.err
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

// Close shuts down the callback server gracefully.
func (s *CallbackServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if errParam := q.Get("error"); errParam != "" {
		desc := q.Get("error_description")
		if desc == "" {
			desc = errParam
		}
		s.result <- callbackResult{err: errors.New(desc)}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w,
			`<!DOCTYPE html><html><body><p>Authorization failed: %s. You may close this tab.</p></body></html>`,
			html.EscapeString(desc),
		)
		return
	}

	code := q.Get("code")
	state := q.Get("state")

	switch {
	case code == "":
		s.result <- callbackResult{err: errors.New("missing authorization code in callback")}
		w.WriteHeader(http.StatusBadRequest)
		return
	case state == "":
		s.result <- callbackResult{err: errors.New("missing state in callback; request may not be from Bitbucket")}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.result <- callbackResult{code: code, state: state}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintln(w,
		`<!DOCTYPE html><html><body><p>Authorization successful! You may close this tab and return to the terminal.</p></body></html>`,
	)
}
