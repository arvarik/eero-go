package eero_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

func TestNetworkService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		networkURL   string
		mockStatus   int
		mockResponse string
		wantErr      bool
		// Expected parsed field values for successful decoding check
		expectName       string
		expectStatus     string
		expectDownSpeed  float64
		expectDownUnits  string
		expectEeroCount  int
		expectEeroSerial string
	}{
		{
			name:       "Success_FullNetworkPayload",
			networkURL: "/2.2/networks/44444",
			mockStatus: http.StatusOK,
			// The mock server returns a JSON payload matching an eero response envelope
			mockResponse: `{
				"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
				"data": {
					"name": "Home Mesh",
					"url": "/2.2/networks/44444",
					"status": "online",
					"speed": {
						"down": {"value": 850.5, "units": "Mbps"},
						"up": {"value": 940.2, "units": "Mbps"}
					},
					"eeros": [
						{
							"serial": "111-222",
							"model": "eero Pro 6",
							"gateway": true,
							"ip_address": "192.168.4.1"
						}
					],
					"health": {
						"internet": {"status": "green"}
					}
				}
			}`,
			wantErr:          false,
			expectName:       "Home Mesh",
			expectStatus:     "online",
			expectDownSpeed:  850.5,
			expectDownUnits:  "Mbps",
			expectEeroCount:  1,
			expectEeroSerial: "111-222",
		},
		{
			name:         "Failure_NetworkNotFound",
			networkURL:   "/2.2/networks/99999",
			mockStatus:   http.StatusNotFound,
			mockResponse: `{"meta": {"code": 404, "error": "Network not found"}, "data": {}}`,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// ─── HTTP Mock Server Setup ─────────────────────────────────────
			mux := http.NewServeMux()

			// The NetworkService uses newRequestFromURL, which resolves relative
			// API paths using the origin of the BaseURL. Therefore, we should
			// map the exact networkURL from our test case.
			mux.HandleFunc(tc.networkURL, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				// Verify the session cookie is propagated via the jar correctly
				// Our test client will manually seed this.
				cookie, err := r.Cookie("s")
				if err != nil || cookie.Value != "test_session_active" {
					t.Errorf("Expected session cookie 's=test_session_active' on request, got %v", cookie)
				}

				w.WriteHeader(tc.mockStatus)
				w.Write([]byte(tc.mockResponse))
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			// ─── Client Setup & Execution ───────────────────────────────────
			client, err := eero.NewClient()
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			// Replace BaseURL with the mock server URL so newRequestFromURL works correctly.
			client.BaseURL = server.URL + "/2.2" // Setup the origin URL resolution properly

			// Seed the cookie jar to simulate an active session. We bypass
			// SetSessionCookie() here because it enforces Secure: true, which
			// the http.Client correctly refuses to send over plain http:// tests.
			testURL, _ := url.Parse(client.BaseURL)
			client.HTTPClient.Jar.SetCookies(testURL, []*http.Cookie{
				{Name: "s", Value: "test_session_active"},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			netDetails, err := client.Network.Get(ctx, tc.networkURL)

			// ─── Assertions ─────────────────────────────────────────────────
			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
				if netDetails == nil {
					t.Fatal("Expected network details, got nil")
				}
				if netDetails.Name != tc.expectName {
					t.Errorf("Name = %v, want %v", netDetails.Name, tc.expectName)
				}
				if netDetails.Status != tc.expectStatus {
					t.Errorf("Status = %v, want %v", netDetails.Status, tc.expectStatus)
				}
				if netDetails.Speed.Down.Value != tc.expectDownSpeed {
					t.Errorf("Down speed = %v, want %v", netDetails.Speed.Down.Value, tc.expectDownSpeed)
				}
				if netDetails.Speed.Down.Units != tc.expectDownUnits {
					t.Errorf("Down units = %v, want %v", netDetails.Speed.Down.Units, tc.expectDownUnits)
				}
				if len(netDetails.Eeros) != tc.expectEeroCount {
					t.Fatalf("Eero count = %v, want %v", len(netDetails.Eeros), tc.expectEeroCount)
				}
				if netDetails.Eeros[0].Serial != tc.expectEeroSerial {
					t.Errorf("Eero serial = %v, want %v", netDetails.Eeros[0].Serial, tc.expectEeroSerial)
				}
			}
		})
	}
}
