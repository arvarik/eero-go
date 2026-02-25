package eero_test

import (
	"net/url"
	"testing"

	"github.com/arvarik/eero-go/eero"
)

func TestSetSessionCookie(t *testing.T) {
	// 1. Initialize client
	client, err := eero.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// 2. Define test data
	userToken := "test-session-token-123"

	// 3. Call the method
	err = client.SetSessionCookie(userToken)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 4. Verify cookie jar state for HTTPS (Secure)
	u, err := url.Parse(client.BaseURL)
	if err != nil {
		t.Fatalf("Failed to parse client BaseURL: %v", err)
	}

	cookies := client.HTTPClient.Jar.Cookies(u)
	var found bool
	for _, cookie := range cookies {
		if cookie.Name == "s" {
			found = true
			if cookie.Value != userToken {
				t.Errorf("Expected cookie value '%s', got '%s'", userToken, cookie.Value)
			}
			// Note: jar.Cookies() returns cookies for the 'Cookie' header, which
			// only includes Name and Value. Attributes like Secure are not returned.
			// To verify Secure, we check that it is NOT returned for an HTTP URL.
			break
		}
	}

	if !found {
		t.Error("Expected session cookie 's' to be present in the jar for HTTPS URL")
	}

	// 5. Verify cookie is NOT returned for HTTP (Insecure)
	insecureBaseURL := "http" + client.BaseURL[len("https"):]
	insecureURL, err := url.Parse(insecureBaseURL)
	if err != nil {
		t.Fatalf("Failed to parse insecure BaseURL: %v", err)
	}

	insecureCookies := client.HTTPClient.Jar.Cookies(insecureURL)
	for _, cookie := range insecureCookies {
		if cookie.Name == "s" {
			t.Error("Expected Secure cookie 's' to NOT be present for HTTP URL")
		}
	}
}

func TestSetSessionCookie_InvalidBaseURL(t *testing.T) {
	client, err := eero.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Set an invalid URL that fails url.Parse
	// Control characters in URL are invalid
	client.BaseURL = "http://api.eero.com/2.2\x7f"

	err = client.SetSessionCookie("token")
	if err == nil {
		t.Error("Expected error due to invalid BaseURL, got nil")
	}
}
