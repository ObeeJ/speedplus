package middleware

// WebSocket credential-carrier tests.
//
// Browsers cannot set an Authorization header on a WS handshake, so the token
// travels in Sec-WebSocket-Protocol ("bearer, <token>") — keeping it out of
// the URL, where upstream proxy/CDN access logs would capture it.

import (
	"net/http"
	"testing"
)

func TestWSTokenFromSubprotocol(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		want   string
	}{
		{"browser form with space", "bearer, tok-abc123", "tok-abc123"},
		{"no space after comma", "bearer,tok-abc123", "tok-abc123"},
		{"case-insensitive sentinel", "Bearer, tok-abc123", "tok-abc123"},
		{"extra subprotocol first", "json, bearer, tok-abc123", "tok-abc123"},
		{"absent header", "", ""},
		{"sentinel with no token", "bearer", ""},
		{"sentinel trailing comma", "bearer, ", ""},
		{"unrelated subprotocol only", "graphql-ws", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "/ws", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if tc.header != "" {
				r.Header.Set("Sec-WebSocket-Protocol", tc.header)
			}
			if got := wsTokenFromSubprotocol(r); got != tc.want {
				t.Errorf("wsTokenFromSubprotocol(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

// The query-parameter fallback must stay confined to WebSocket handshakes —
// a plain REST request may never authenticate via ?token=.
func TestIsWSUpgrade(t *testing.T) {
	for _, tc := range []struct {
		name    string
		upgrade string
		want    bool
	}{
		{"websocket lowercase", "websocket", true},
		{"websocket mixed case", "WebSocket", true},
		{"no upgrade header", "", false},
		{"other protocol", "h2c", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "/orders", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if tc.upgrade != "" {
				r.Header.Set("Upgrade", tc.upgrade)
			}
			if got := isWSUpgrade(r); got != tc.want {
				t.Errorf("isWSUpgrade(Upgrade: %q) = %v, want %v", tc.upgrade, got, tc.want)
			}
		})
	}
}
