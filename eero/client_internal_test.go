package eero

import (
	"context"
	"testing"
)

func TestClient_originURL_Robustness(t *testing.T) {
	tests := []struct {
		baseURL  string
		expected string
	}{
		{"https://api-user.e2ro.com/2.2", "https://api-user.e2ro.com"},
		{"http://a", "http://a"},
		{"custom", "custom"},
		{"https://example.com/api/v1", "https://example.com"},
	}

	for _, tt := range tests {
		c := &Client{BaseURL: tt.baseURL}
		u, err := c.originURL()
		if err != nil {
			t.Errorf("originURL(%q) error: %v", tt.baseURL, err)
			continue
		}
		got := u.String()
		if got != tt.expected {
			t.Errorf("originURL(%q) = %q; want %q", tt.baseURL, got, tt.expected)
		}
	}
}

func TestClient_newRequest_Concat(t *testing.T) {
	// Tests simple string concatenation for newRequest
	tests := []struct {
		baseURL  string
		path     string
		expected string
	}{
		{"https://api.eero.com/2.2", "/account", "https://api.eero.com/2.2/account"},
		// Query params should work fine with string concat
		{"https://api.eero.com/2.2", "/networks?limit=10", "https://api.eero.com/2.2/networks?limit=10"},
	}

	for _, tt := range tests {
		c := &Client{BaseURL: tt.baseURL}
		c.UserAgent = "test-agent"

		req, err := c.newRequest(context.Background(), "GET", tt.path, nil)
		if err != nil {
			t.Errorf("newRequest(%q, %q) error: %v", tt.baseURL, tt.path, err)
			continue
		}

		if req.URL.String() != tt.expected {
			t.Errorf("newRequest(%q, %q) URL = %q; want %q", tt.baseURL, tt.path, req.URL.String(), tt.expected)
		}
	}
}

func TestClient_newRequestFromURL_Resolve(t *testing.T) {
	// Tests ResolveReference for newRequestFromURL
	tests := []struct {
		baseURL     string // Used to derive originURL
		relativeURL string
		expected    string
	}{
		{"https://api.eero.com/2.2", "/2.2/networks/123", "https://api.eero.com/2.2/networks/123"},
		// Ensure query params are preserved and not escaped
		{"https://api.eero.com/2.2", "/2.2/networks?active=true", "https://api.eero.com/2.2/networks?active=true"},
	}

	for _, tt := range tests {
		c := &Client{BaseURL: tt.baseURL}
		c.UserAgent = "test-agent"

		req, err := c.newRequestFromURL(context.Background(), "GET", tt.relativeURL, nil)
		if err != nil {
			t.Errorf("newRequestFromURL(%q, %q) error: %v", tt.baseURL, tt.relativeURL, err)
			continue
		}

		if req.URL.String() != tt.expected {
			t.Errorf("newRequestFromURL(%q, %q) URL = %q; want %q", tt.baseURL, tt.relativeURL, req.URL.String(), tt.expected)
		}
	}
}

func BenchmarkOriginURL(b *testing.B) {
	c := &Client{
		BaseURL: "https://api-user.e2ro.com/2.2",
	}
	// Warm up the cache by calling it once
	_, _ = c.originURL()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.originURL()
	}
}
