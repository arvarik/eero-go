package eero

import (
	"context"
	"fmt"
	"net/http"
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
	URL                      string             `json:"url"`
	MAC                      string             `json:"mac"`
	EUI64                    string             `json:"eui64"`
	Manufacturer             *string            `json:"manufacturer"`
	IP                       *string            `json:"ip"`
	IPs                      []string           `json:"ips"`
	IPv6Addresses            []IPv6Address      `json:"ipv6_addresses"`
	Nickname                 *string            `json:"nickname"`
	Hostname                 *string            `json:"hostname"`
	Connected                bool               `json:"connected"`
	Wireless                 bool               `json:"wireless"`
	ConnectionType           string             `json:"connection_type"`
	Source                   DeviceSource       `json:"source"`
	LastActive               EeroTime           `json:"last_active"`
	FirstActive              EeroTime           `json:"first_active"`
	Connectivity             DeviceConnectivity `json:"connectivity"`
	Interface                DeviceInterface    `json:"interface"`
	Usage                    *Usage             `json:"usage"`
	Profile                  DeviceRef          `json:"profile"`
	DeviceType               string             `json:"device_type"`
	Blacklisted              bool               `json:"blacklisted"`
	Dropped                  bool               `json:"dropped"`
	Homekit                  Homekit            `json:"homekit"`
	IsGuest                  bool               `json:"is_guest"`
	Paused                   bool               `json:"paused"`
	Channel                  int                `json:"channel"`
	Auth                     string             `json:"auth"`
	IsPrivate                bool               `json:"is_private"`
	SecondaryWanDenyAccess   bool               `json:"secondary_wan_deny_access"`
	RingLTE                  RingLTE            `json:"ring_lte"`
	IPv4                     string             `json:"ipv4"`
	IsProxiedNode            bool               `json:"is_proxied_node"`
	ManufacturerDeviceTypeID *string            `json:"manufacturer_device_type_id"`
	AmazonDevicesDetail      any                `json:"amazon_devices_detail"`
	SSID                     string             `json:"ssid"`
	SubnetKind               string             `json:"subnet_kind"`
	VlanID                   *int               `json:"vlan_id"`
	VlanName                 string             `json:"vlan_name"`
	DisplayName              *string            `json:"display_name"`
	ModelName                *string            `json:"model_name"`
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

// DeviceConnectivity holds wireless performance ratings for connected nodes.
type DeviceConnectivity struct {
	RxBitrate      string         `json:"rx_bitrate"`
	Signal         string         `json:"signal"`
	SignalAvg      *string        `json:"signal_avg"`
	Score          float64        `json:"score"`
	ScoreBars      int            `json:"score_bars"`
	Frequency      int            `json:"frequency"`
	RxRateInfo     RateInfo       `json:"rx_rate_info"`
	TxRateInfo     RateInfo       `json:"tx_rate_info"`
	EthernetStatus EthernetStatus `json:"ethernet_status"`
}

// RateInfo tracks Wi-Fi specifications and modulation info for clients.
type RateInfo struct {
	RateBps       *int64  `json:"rate_bps"`
	MCS           *int    `json:"mcs"`
	NSS           *int    `json:"nss"`
	GuardInterval *string `json:"guard_interval"`
	ChannelWidth  *string `json:"channel_width"`
	PhyType       *string `json:"phy_type"`
}

// EthernetStatus describes a wired link.
type EthernetStatus struct {
	Value any `json:"value"` // Abstract generic field due to API variances.
}

// DeviceInterface captures what frequencies the node represents over transmission.
type DeviceInterface struct {
	Frequency     string `json:"frequency"`
	FrequencyUnit string `json:"frequency_unit"`
}

// Homekit dictates whether the router isolates or enables tracking routing.
type Homekit struct {
	Registered     bool   `json:"registered"`
	ProtectionMode string `json:"protection_mode"`
}

// RingLTE shows alarm pro integration.
type RingLTE struct {
	IsNotPausable bool `json:"is_not_pausable"`
	RingManaged   bool `json:"ring_managed"`
	LTEEnabled    bool `json:"lte_enabled"`
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
