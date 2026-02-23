package eero

import (
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
		got := c.originURL()
		if got != tt.expected {
			t.Errorf("originURL(%q) = %q; want %q", tt.baseURL, got, tt.expected)
		}
	}
}
