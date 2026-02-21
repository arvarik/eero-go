package eero

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// DeviceService provides access to devices connected to an eero network.
type DeviceService struct {
	client *Client
}

// --- Response types ---

// Device represents a single client device connected to the eero network.
// Optional fields that the API may omit for offline devices use pointer types
// so that missing JSON keys decode to nil rather than zero values.
type Device struct {
	URL            string        `json:"url"`
	MAC            string        `json:"mac"`
	Nickname       *string       `json:"nickname"`
	Hostname       *string       `json:"hostname"`
	DisplayName    *string       `json:"display_name"`
	IP             *string       `json:"ip"`
	ConnectionType *string       `json:"connection_type"`
	Connected      bool          `json:"connected"`
	Wireless       bool          `json:"wireless"`
	DeviceType     string        `json:"device_type"`
	Manufacturer   *string       `json:"manufacturer"`
	Source         *DeviceSource `json:"source"`
	LastActive     time.Time     `json:"last_active"`
	Profile        *DeviceRef    `json:"profile"`
	Usage          *Usage        `json:"usage"`
	Band           *string       `json:"frequency_band"`
}

// DeviceRef is a lightweight reference to a profile from within a device.
type DeviceRef struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// DeviceSource holds information about what eero node this device is connected to.
type DeviceSource struct {
	Location     string `json:"location"`
	IsGateway    bool   `json:"is_gateway"`
	Model        string `json:"model"`
	DisplayName  string `json:"display_name"`
	SerialNumber string `json:"serial_number"`
	URL          string `json:"url"`
}

// Usage holds bandwidth usage statistics for a device.
type Usage struct {
	Download float64 `json:"download"`
	Upload   float64 `json:"upload"`
	Units    string  `json:"units"`
}

// --- Methods ---

// List returns all devices connected to the specified network.
//
// The networkURL parameter should be the exact relative URL from the account
// response (e.g., "/2.2/networks/12345"). The "/devices" suffix is appended
// automatically.
//
// The response is unmarshaled into EeroResponse[[]Device], but only the
// []Device slice is returned to the caller.
func (s *DeviceService) List(ctx context.Context, networkURL string) ([]Device, error) {
	req, err := s.client.newRequestFromURL(ctx, http.MethodGet, networkURL+"/devices", nil)
	if err != nil {
		return nil, fmt.Errorf("device: creating request: %w", err)
	}

	var resp EeroResponse[[]Device]
	if err := s.client.doRaw(req, &resp); err != nil {
		return nil, fmt.Errorf("device: %w", err)
	}

	return resp.Data, nil
}
