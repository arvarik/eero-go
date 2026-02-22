package eero_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

// setupMockServer spins up a local httptest.Server with a custom handler.
// The caller is responsible for calling defer server.Close() to clean up.
func setupMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// TestLogin ensures the AuthService properly parses the user_token
// from a mocked /login JSON payload matching the Eero envelope.
func TestLogin(t *testing.T) {
	// 1. Spin up the mock Eero server intercepting the login path
	mockServer := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}

		// Assert HTTP Method constraints
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		// Mock the exact Eero JSON wrapper containing the user_token in 'data'
		_, _ = w.Write([]byte(`{
			"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
			"data": {"user_token": "mock_token_abc123"}
		}`))
	})
	defer mockServer.Close()

	// 2. Point our client to the local mock server
	client, err := eero.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}
	client.BaseURL = mockServer.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 3. Execute and Assert
	resp, err := client.Auth.Login(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.UserToken != "mock_token_abc123" {
		t.Errorf("Expected user_token 'mock_token_abc123', got '%s'", resp.UserToken)
	}
}

// TestGetNetwork ensures the NetworkService properly unmarshals nested
// metrics from the "data" payload (like network status and measured speeds).
func TestGetNetwork(t *testing.T) {
	// 1. Spin up the mock Eero server intercepting the network details path
	mockServer := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		// Note: Network.Get uses newRequestFromURL which extracts the origin.
		// Since our test BaseURL is the server URL, the path mapping requires /2.2.
		if r.URL.Path != "/2.2/networks/12345" {
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}

		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		// Mock the exact Eero JSON wrapper with network status and nested speeds
		_, _ = w.Write([]byte(`{
			"meta": {"code": 200},
			"data": {
				"name": "Home Mesh",
				"status": "online",
				"speed": {
					"down": {"value": 850.5, "units": "Mbps"},
					"up": {"value": 940.2, "units": "Mbps"}
				}
			}
		}`))
	})
	defer mockServer.Close()

	// 2. Point our client to the local mock server
	client, err := eero.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Because NetworkService.Get uses newRequestFromURL which extracts the origin and
	// appends the raw URL directly, we map BaseURL to include the API version prefix.
	client.BaseURL = mockServer.URL + "/2.2"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 3. Execute and Assert targeting the full relative path
	net, err := client.Network.Get(ctx, "/2.2/networks/12345")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if net.Name != "Home Mesh" {
		t.Errorf("Expected network name 'Home Mesh', got '%s'", net.Name)
	}
	if net.Status != "online" {
		t.Errorf("Expected network status 'online', got '%s'", net.Status)
	}
	if net.Speed.Down.Value != 850.5 {
		t.Errorf("Expected down speed 850.5, got %f", net.Speed.Down.Value)
	}
}

// TestErrorHandling explicitly ensures the client gracefully captures and
// surfaces HTTP 500 Internal Server Errors wrapped in our custom APIError struct.
func TestErrorHandling(t *testing.T) {
	// 1. Spin up the mock Eero server returning an HTTP 500 error
	mockServer := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		// Eero's meta wrapper mapping a 500 crash
		_, _ = w.Write([]byte(`{
			"meta": {"code": 500, "error": "Internal Server Error"},
			"data": {}
		}`))
	})
	defer mockServer.Close()

	// 2. Point our client to the local mock server
	client, err := eero.NewClient()
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}
	client.BaseURL = mockServer.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 3. Execute and Assert leveraging Account.Get as the arbitrary endpoint
	_, err = client.Account.Get(ctx)
	if err == nil {
		t.Fatal("Expected an error for HTTP 500, but got nil")
	}

	// Verify the surface-level custom APIError fields matched our mock wrapper
	var apiErr *eero.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected error to be of type *eero.APIError, got %T", err)
	}

	if apiErr.HTTPStatusCode != 500 {
		t.Errorf("Expected HTTP status 500, got %d", apiErr.HTTPStatusCode)
	}
	if apiErr.Code != 500 {
		t.Errorf("Expected meta API Code 500, got %d", apiErr.Code)
	}
	if apiErr.Message != "Internal Server Error" {
		t.Errorf("Expected error message 'Internal Server Error', got '%s'", apiErr.Message)
	}
}
