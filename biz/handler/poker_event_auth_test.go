package handler

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// Fail at the real HTTP boundary without making external requests or replacing
// the business function under test.
type eventAuthFailureTransport struct {
	t     *testing.T
	calls atomic.Int32
}

func (transport *eventAuthFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	if req.URL.Path != "/api/auth/verify-token" || req.Header.Get("Authorization") != "Bearer session-token" {
		transport.t.Errorf("unexpected auth request: path=%q authorization forwarded=%t",
			req.URL.Path, req.Header.Get("Authorization") == "Bearer session-token")
	}
	return &http.Response{StatusCode: http.StatusServiceUnavailable,
		Body: io.NopCloser(strings.NewReader(`{"error":"temporary failure"}`)), Header: make(http.Header)}, nil
}

func failEventAuthentication(t *testing.T) *eventAuthFailureTransport {
	t.Helper()
	transport := &eventAuthFailureTransport{t: t}
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	return transport
}
