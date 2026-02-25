package eero_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

func TestAccountService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mockStatus   int
		mockResponse string
		wantErr      bool
		// Expected parsed field values
		expectName         string
		expectEmail        string
		expectNetworkCount int
		expectNetworkName  string
		expectNetworkURL   string
	}{
		{
			name:       "Success_FullAccountPayload",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
				"data": {
					"name": "Test User",
					"email": {
						"value": "test@example.com",
						"verified": true
					},
					"phone": {
						"value": "+1234567890",
						"verified": true
					},
					"networks": {
						"count": 1,
						"data": [
							{
								"url": "/2.2/networks/123",
								"name": "My Network",
								"created": "2023-01-01T00:00:00Z",
								"amazon_directed_id": "amzn1.account.AHX"
							}
						]
					},
					"role": "owner"
				}
			}`,
			wantErr:            false,
			expectName:         "Test User",
			expectEmail:        "test@example.com",
			expectNetworkCount: 1,
			expectNetworkName:  "My Network",
			expectNetworkURL:   "/2.2/networks/123",
		},
		{
			name:         "Failure_InternalServerError",
			mockStatus:   http.StatusInternalServerError,
			mockResponse: `{"meta": {"code": 500, "error": "Internal Server Error"}, "data": {}}`,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 1. Setup Mock Server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/account" {
					t.Errorf("Expected path /account, got %s", r.URL.Path)
				}
				if r.Method != http.MethodGet {
					t.Errorf("Expected method GET, got %s", r.Method)
				}
				w.WriteHeader(tc.mockStatus)
				_, _ = w.Write([]byte(tc.mockResponse))
			}))
			defer server.Close()

			// 2. Setup Client
			client, err := eero.NewClient()
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			client.BaseURL = server.URL

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// 3. Execute
			account, err := client.Account.Get(ctx)

			// 4. Assert
			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
				if account == nil {
					t.Fatal("Expected account data, got nil")
				}
				if account.Name != tc.expectName {
					t.Errorf("Name = %v, want %v", account.Name, tc.expectName)
				}
				if account.Email.Value != tc.expectEmail {
					t.Errorf("Email = %v, want %v", account.Email.Value, tc.expectEmail)
				}
				if len(account.Networks.Data) != tc.expectNetworkCount {
					t.Fatalf("Network count = %v, want %v", len(account.Networks.Data), tc.expectNetworkCount)
				}
				if account.Networks.Data[0].Name != tc.expectNetworkName {
					t.Errorf("Network Name = %v, want %v", account.Networks.Data[0].Name, tc.expectNetworkName)
				}
				if account.Networks.Data[0].URL != tc.expectNetworkURL {
					t.Errorf("Network URL = %v, want %v", account.Networks.Data[0].URL, tc.expectNetworkURL)
				}
			}
		})
	}
}
