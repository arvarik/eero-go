package eero_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

func TestProfileService_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		networkURL    string
		mockStatus    int
		mockResponse  string
		wantErr       bool
		expectCount   int
		expectName    string
		expectDevices int
		expectMac     string
	}{
		{
			name:       "Success_ProfilesDevices",
			networkURL: "/2.2/networks/55555",
			mockStatus: http.StatusOK,
			mockResponse: `{
				"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
				"data": [
					{
						"url": "/2.2/networks/4296130/profiles/15859226",
						"name": "Unassigned",
						"paused": false,
						"devices": [
							{
								"url": "/2.2/networks/4296130/devices/bcdf5800c734",
								"mac": "bc:df:58:00:c7:34",
								"connected": true,
								"device_type": "digital_assistant",
								"last_active": "2026-02-21T22:14:52.249Z",
								"first_active": "2026-02-21T12:58:06.480Z"
							}
						]
					}
				]
			}`,
			wantErr:       false,
			expectCount:   1,
			expectName:    "Unassigned",
			expectDevices: 1,
			expectMac:     "bc:df:58:00:c7:34",
		},
		{
			name:         "Failure_BadGateway",
			networkURL:   "/2.2/networks/55555",
			mockStatus:   http.StatusBadGateway,
			mockResponse: `{}`,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()

			expectedRoute := tc.networkURL + "/profiles"
			mux.HandleFunc(expectedRoute, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				w.WriteHeader(tc.mockStatus)
				_, _ = w.Write([]byte(tc.mockResponse))
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			client, _ := eero.NewClient()
			client.BaseURL = server.URL + "/2.2"

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			profiles, err := client.Profile.List(ctx, tc.networkURL)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(profiles) != tc.expectCount {
				t.Fatalf("Expected %d profiles, got %d", tc.expectCount, len(profiles))
			}

			p1 := profiles[0]
			if p1.Name != tc.expectName {
				t.Errorf("Profile Name mismatch. Got %v", p1.Name)
			}
			if len(p1.Devices) != tc.expectDevices {
				t.Fatalf("Expected devices in profile: %d", tc.expectDevices)
			}
			if p1.Devices[0].MAC != tc.expectMac {
				t.Errorf("Profile device MAC mismatch: %s", p1.Devices[0].MAC)
			}
		})
	}
}

func TestProfileService_Pause(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	profileURL := "/2.2/networks/55555/profiles/111"

	mux.HandleFunc(profileURL, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"meta": {"code": 200}, "data": {}}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, _ := eero.NewClient()
	client.BaseURL = server.URL + "/2.2"

	err := client.Profile.Pause(context.Background(), profileURL)
	if err != nil {
		t.Fatalf("Expected no error pausing profile, got: %v", err)
	}

	err = client.Profile.Unpause(context.Background(), profileURL)
	if err != nil {
		t.Fatalf("Expected no error unpausing profile, got: %v", err)
	}
}

func TestProfileService_Unpause(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	profileURL := "/2.2/networks/55555/profiles/111"

	mux.HandleFunc(profileURL, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		expected := `{"paused":false}`
		if string(body) != expected {
			t.Errorf("Expected body %s, got %s", expected, string(body))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"meta": {"code": 200}, "data": {}}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, _ := eero.NewClient()
	client.BaseURL = server.URL + "/2.2"

	err := client.Profile.Unpause(context.Background(), profileURL)
	if err != nil {
		t.Fatalf("Expected no error unpausing profile, got: %v", err)
	}
}
