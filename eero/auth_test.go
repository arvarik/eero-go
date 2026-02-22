package eero_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

func TestAuthService_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		// Mock server behavior
		mockStatus   int
		mockResponse string
		// Expectations
		wantErr         bool
		isAuthErr       bool
		expectUserToken string
	}{
		{
			name:       "Success_ReturnsUserToken",
			identifier: "test@example.com",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
				"data": {"user_token": "token_12345"}
			}`,
			wantErr:         false,
			expectUserToken: "token_12345",
		},
		{
			name:       "Failure_InvalidEmail",
			identifier: "bad_email",
			mockStatus: http.StatusBadRequest,
			mockResponse: `{
				"meta": {"code": 400, "error": "Invalid email format", "server_time": "2023-10-01T12:00:00Z"},
				"data": {}
			}`,
			wantErr:   true,
			isAuthErr: false, // 400 is bad request, not 401
		},
		{
			name:       "Failure_Unauthorized",
			identifier: "hacker@example.com",
			mockStatus: http.StatusUnauthorized,
			mockResponse: `{
				"meta": {"code": 401, "error": "Unauthorized user", "server_time": "2023-10-01T12:00:00Z"},
				"data": {}
			}`,
			wantErr:   true,
			isAuthErr: true, // Should trigger apiErr.IsAuthError() == true
		},
		{
			name:         "Failure_MalformedJSON",
			identifier:   "test@example.com",
			mockStatus:   http.StatusOK,
			mockResponse: `{ bad json }`,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// ─── HTTP Mock Server Setup ─────────────────────────────────────
			// Spin up a local httptest server specifically for this test case.
			// This avoids global state and allows full parallelization.
			mux := http.NewServeMux()
			mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
				// Assert request constraints
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if r.Header.Get("User-Agent") == "" {
					t.Errorf("Expected User-Agent header, but it was empty")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Verify payload matches the identifier
				var body eero.LoginRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}
				if body.Login != tc.identifier {
					t.Errorf("Expected identifier %q, got %q", tc.identifier, body.Login)
				}

				// Return the mocked response for this test case
				w.WriteHeader(tc.mockStatus)
				_, _ = w.Write([]byte(tc.mockResponse))
			})

			server := httptest.NewServer(mux)
			defer server.Close() // Automatically tear down server when test ends

			// ─── Client Setup & Execution ───────────────────────────────────
			client, err := eero.NewClient()
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			// Override BaseURL to point to our local mock server
			client.BaseURL = server.URL

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			resp, err := client.Auth.Login(ctx, tc.identifier)

			// ─── Assertions ─────────────────────────────────────────────────
			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected an error, got nil")
				}
				if tc.isAuthErr {
					var apiErr *eero.APIError
					if !errors.As(err, &apiErr) {
						t.Fatalf("Expected *eero.APIError, got %T", err)
					}
					if !apiErr.IsAuthError() {
						t.Errorf("Expected API error to report IsAuthError = true")
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
				if resp == nil {
					t.Fatal("Expected response, got nil")
				}
				if resp.UserToken != tc.expectUserToken {
					t.Errorf("Expected UserToken %q, got %q", tc.expectUserToken, resp.UserToken)
				}
			}
		})
	}
}

func TestAuthService_Verify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		code         string
		mockStatus   int
		mockResponse string
		wantErr      bool
	}{
		{
			name:       "Success_VerifyCode",
			code:       "123456",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
				"data": {}
			}`,
			wantErr: false,
		},
		{
			name:       "Failure_Rejection",
			code:       "000000",
			mockStatus: http.StatusForbidden,
			mockResponse: `{
				"meta": {"code": 403, "error": "Invalid verification code", "server_time": "2023-10-01T12:00:00Z"},
				"data": {}
			}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			mux.HandleFunc("/login/verify", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}

				var body eero.VerifyRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body.Code != tc.code {
					t.Errorf("Expected code %q, got %q", tc.code, body.Code)
				}

				w.WriteHeader(tc.mockStatus)
				_, _ = w.Write([]byte(tc.mockResponse))
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			client, _ := eero.NewClient()
			client.BaseURL = server.URL
			// In real usage, the cookie jar would send `s=user_token`.
			// Since we aren't testing cookie jar injection here, we just call Verify directly.

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := client.Auth.Verify(ctx, tc.code)

			if (err != nil) != tc.wantErr {
				t.Fatalf("Verify() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
