package eero

import (
	"context"
	"fmt"
	"net/http"
)

// NetworkService provides access to eero network configuration and lifecycle.
type NetworkService struct {
	client *Client
}

// --- Response types ---

// NetworkDetails represents the full details of an eero network, including
// its name, operational status, and last measured speed.
type NetworkDetails struct {
	URL           string          `json:"url"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	Timezone      NetworkTimezone `json:"timezone"`
	Speed         NetworkSpeed    `json:"speed"`
	GuestNetwork  GuestNetwork    `json:"guest_network"`
	SquadID       string          `json:"squad_id"`
	UpnpEnabled   bool            `json:"upnp"`
	BandSteering  bool            `json:"band_steering"`
	ThreadEnabled bool            `json:"thread"`
	IPv6          NetworkIPv6     `json:"ipv6"`
	Eeros         NetworkEeros    `json:"eeros"`
	Health        Health          `json:"health"`
}

// NetworkSpeed holds the most recent speed test results for the network.
type NetworkSpeed struct {
	Down SpeedMeasurement `json:"down"`
	Up   SpeedMeasurement `json:"up"`
}

// NetworkTimezone holds the timezone data for the network.
type NetworkTimezone struct {
	Value string `json:"value"`
}

// GuestNetwork holds the guest network settings.
type GuestNetwork struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// NetworkIPv6 holds the IPv6 configuration details for the network.
type NetworkIPv6 struct {
	NameServers struct {
		Mode string `json:"mode"`
	} `json:"name_servers"`
}

// NetworkEeros holds the eeros count and the list of eero nodes.
type NetworkEeros struct {
	Count int        `json:"count"`
	Data  []EeroNode `json:"data"`
}

// SpeedMeasurement is a single directional speed measurement.
type SpeedMeasurement struct {
	Units string  `json:"units"`
	Value float64 `json:"value"`
}

// Health holds the overall network health indicators.
type Health struct {
	Internet HealthDetail `json:"internet"`
	Eero     HealthDetail `json:"eero"`
}

// HealthDetail is a single health metric.
type HealthDetail struct {
	Status string `json:"status"`
}

// EeroNode represents a single eero device (gateway or extender) in the mesh.
type EeroNode struct {
	URL          string  `json:"url"`
	Serial       string  `json:"serial"`
	Name         *string `json:"name"`
	Model        string  `json:"model"`
	Location     *string `json:"location"`
	Status       string  `json:"status"`
	Gateway      bool    `json:"gateway"`
	IPAddress    *string `json:"ip_address"`
	MACAddress   string  `json:"mac_address"`
	Firmware     string  `json:"os_version"`
	UpdatedAt    string  `json:"updated_at"`
	ConnectionTo *string `json:"connection_to"`
	MeshQuality  *int    `json:"mesh_quality_bars"`
}

// --- Methods ---

// Get retrieves full details for the specified network.
//
// The networkURL parameter should be the exact relative URL from the account
// response (e.g., "/2.2/networks/12345"). Do not manually construct the path.
func (s *NetworkService) Get(ctx context.Context, networkURL string) (*NetworkDetails, error) {
	req, err := s.client.newRequestFromURL(ctx, http.MethodGet, networkURL, nil)
	if err != nil {
		return nil, fmt.Errorf("network: creating request: %w", err)
	}

	var resp EeroResponse[NetworkDetails]
	if err := s.client.doRaw(req, &resp); err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}

	return &resp.Data, nil
}

// Reboot triggers a reboot of all eero devices in the specified network.
//
// The networkURL parameter should be the exact relative URL from the account
// response (e.g., "/2.2/networks/12345").
func (s *NetworkService) Reboot(ctx context.Context, networkURL string) error {
	req, err := s.client.newRequestFromURL(ctx, http.MethodPost, networkURL+"/reboot", nil)
	if err != nil {
		return fmt.Errorf("network: creating reboot request: %w", err)
	}

	if err := s.client.doRaw(req, nil); err != nil {
		return fmt.Errorf("network: reboot: %w", err)
	}

	return nil
}
