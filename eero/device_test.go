package eero_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

func TestDeviceService_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		networkURL   string
		mockStatus   int
		mockResponse string
		wantErr      bool
		// Expectations for decoding optional pointers correctly
		expectCount            int
		expectFirstDeviceName  *string // Pointer to allow checking for nil
		expectSecondDeviceName *string
		expectSecondDeviceIP   *string // The "offline" device should have a nil IP pointer
	}{
		{
			name:       "Success_ParsesOnlineAndOfflineDevices",
			networkURL: "/2.2/networks/55555",
			mockStatus: http.StatusOK,
			// The mock response includes an array of devices in the data envelope.
			// One is "online" containing full fields, the other is "offline" dropping Name and IP.
			mockResponse: `{
				"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
				"data": [
					{
						"url": "/2.2/networks/55555/devices/1",
						"mac": "AA:BB:CC:DD:EE:11",
						"nickname": "Arvind's iPhone",
						"ip": "192.168.4.50",
						"vlan_id": 4,
						"connectivity": {
							"score_bars": 5,
							"rx_bitrate": "15.0 MBit/s"
						},
						"ips": ["192.168.4.50", "fe80::1"],
						"connected": true,
						"device_type": "phone",
						"last_active": "2023-10-01T11:59:00Z"
					},
					{
						"url": "/2.2/networks/55555/devices/2",
						"mac": "AA:BB:CC:DD:EE:22",
						"nickname": null,
						"ip": null,
						"connected": false,
						"device_type": "laptop",
						"last_active": "2023-09-30T10:00:00Z"
					},
					{
						"url": "/2.2/networks/55555/devices/3",
						"mac": "AA:BB:CC:DD:EE:33",
						"nickname": "Test Switch",
						"connected": true,
						"manufacturer": "Nintendo"
					}
				]
			}`,
			wantErr:                false,
			expectCount:            3,
			expectFirstDeviceName:  ptr("Arvind's iPhone"),
			expectSecondDeviceName: nil, // Decoded pointer should be nil, not panicked
			expectSecondDeviceIP:   nil, // Eero omits IP for offline devices
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

			// Device.List appends "/devices" to the networkURL
			expectedRoute := tc.networkURL + "/devices"
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

			devices, err := client.Device.List(ctx, tc.networkURL)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(devices) != tc.expectCount {
				t.Fatalf("Expected %d devices, got %d", tc.expectCount, len(devices))
			}

			// Verify the first device unmarshaled correctly
			d1 := devices[0]
			if d1.Nickname == nil || *d1.Nickname != *tc.expectFirstDeviceName {
				t.Errorf("Device 1 Name mismatch. Got %v", d1.Nickname)
			}
			if !d1.Connected {
				t.Errorf("Device 1 should be connected")
			}
			if d1.LastActive.Time.IsZero() {
				t.Errorf("Device 1 failed to parse last_active custom EeroTime format")
			}
			if d1.VlanID == nil || *d1.VlanID != 4 {
				t.Errorf("Device 1 failed to unmarshal nested pointer primitive arrays")
			}
			if d1.Connectivity.ScoreBars != 5 || d1.Connectivity.RxBitrate == "" {
				t.Errorf("Device 1 failed to parse nested connectivity objects properties")
			}
			if len(d1.IPs) < 1 {
				t.Errorf("Device IPs mapping missing arrays data")
			}

			// Verify the second "offline" device gracefully handles nil pointers
			d2 := devices[1]
			if d2.Nickname != tc.expectSecondDeviceName {
				t.Errorf("Device 2 Nickname should be nil, got %v", d2.Nickname)
			}
			if d2.IP != tc.expectSecondDeviceIP {
				t.Errorf("Device 2 IP should be nil, got %v", d2.IP)
			}
			if d2.Connected {
				t.Errorf("Device 2 should be disconnected")
			}
		})
	}
}

// ptr is a helper to securely return pointers to literal strings for testing
func ptr(s string) *string {
	return &s
}
