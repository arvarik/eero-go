package eero

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// NetworkService provides access to eero network configuration and lifecycle.
type NetworkService struct {
	client *Client
}

// --- Response types ---

// NetworkDetails represents the full details of an eero network, including
// its name, operational status, and last measured speed.
type NetworkDetails struct {
	URL            string                `json:"url"`
	Name           string                `json:"name"`
	DisplayName    string                `json:"display_name"`
	Status         string                `json:"status"`
	Gateway        string                `json:"gateway"`
	WanIP          string                `json:"wan_ip"`
	GatewayIP      string                `json:"gateway_ip"`
	Connection     NetworkConnection     `json:"connection"`
	GeoIP          GeoIP                 `json:"geo_ip"`
	Lease          NetworkLease          `json:"lease"`
	DHCP           NetworkDHCP           `json:"dhcp"`
	DNS            NetworkDNS            `json:"dns"`
	UpnpEnabled    bool                  `json:"upnp"`
	IPv6Upstream   bool                  `json:"ipv6_upstream"`
	ThreadEnabled  bool                  `json:"thread"`
	SQMEnabled     bool                  `json:"sqm"`
	BandSteering   bool                  `json:"band_steering"`
	Wpa3           bool                  `json:"wpa3"`
	WirelessMode   string                `json:"wireless_mode"`
	MloMode        string                `json:"mlo_mode"`
	Eeros          NetworkEeros          `json:"eeros"`
	Speed          NetworkSpeed          `json:"speed"`
	Timezone       NetworkTimezone       `json:"timezone"`
	Updates        NetworkUpdates        `json:"updates"`
	Health         Health                `json:"health"`
	IPSettings     IPSettings            `json:"ip_settings"`
	PremiumDNS     PremiumDNS            `json:"premium_dns"`
	Owner          string                `json:"owner"`
	PremiumStatus  string                `json:"premium_status"`
	LastReboot     *time.Time            `json:"last_reboot"`
	IPv6Lease      IPv6Lease             `json:"ipv6_lease"`
	IPv6           NetworkIPv6           `json:"ipv6"`
	GuestNetwork   GuestNetwork          `json:"guest_network"`
	PremiumDetails NetworkPremiumDetails `json:"premium_details"`
	WanType        string                `json:"wan_type"`
}

// NetworkConnection describes the router connection mode.
type NetworkConnection struct {
	Mode string `json:"mode"`
}

// GeoIP holds geographical settings associated with the network's public IP.
type GeoIP struct {
	CountryCode string `json:"countryCode"`
	CountryName string `json:"countryName"`
	City        string `json:"city"`
	Region      string `json:"region"`
	Timezone    string `json:"timezone"`
	PostalCode  string `json:"postalCode"`
	MetroCode   int    `json:"metroCode"`
	AreaCode    *int   `json:"areaCode"`
	RegionName  string `json:"regionName"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	ASN         int    `json:"asn"`
}

// NetworkLease represents network lease details including DHCP options.
type NetworkLease struct {
	Mode string     `json:"mode"`
	DHCP *LeaseDHCP `json:"dhcp"`
}

// LeaseDHCP describes the dynamic IPs given.
type LeaseDHCP struct {
	IP     string `json:"ip"`
	Mask   string `json:"mask"`
	Router string `json:"router"`
}

// NetworkDHCP holds LAN DHCP settings mode.
type NetworkDHCP struct {
	Mode string `json:"mode"`
}

// NetworkDNS holds networking DNS setup options.
type NetworkDNS struct {
	Mode    string    `json:"mode"`
	Parent  DNSParent `json:"parent"`
	Caching bool      `json:"caching"`
}

// DNSParent holds DNS configuration properties mapping.
type DNSParent struct {
	IPs []string `json:"ips"`
}

// NetworkSpeed holds the most recent speed test results for the network.
type NetworkSpeed struct {
	Status string           `json:"status"`
	Date   time.Time        `json:"date"`
	Up     SpeedMeasurement `json:"up"`
	Down   SpeedMeasurement `json:"down"`
}

// NetworkTimezone holds the timezone data for the network.
type NetworkTimezone struct {
	Value string `json:"value"`
	GeoIP string `json:"geo_ip"`
}

// NetworkUpdates holds properties tracking when the firmware updates occur.
type NetworkUpdates struct {
	PreferredUpdateHour int       `json:"preferred_update_hour"`
	MinRequiredFirmware string    `json:"min_required_firmware"`
	TargetFirmware      string    `json:"target_firmware"`
	UpdateToFirmware    string    `json:"update_to_firmware"`
	UpdateRequired      bool      `json:"update_required"`
	CanUpdateNow        bool      `json:"can_update_now"`
	HasUpdate           bool      `json:"has_update"`
	LastUpdateStarted   time.Time `json:"last_update_started"`
	ManifestResource    string    `json:"manifest_resource"`
}

// GuestNetwork holds the guest network settings.
type GuestNetwork struct {
	URL     string `json:"url"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// IPSettings is the networking configuration of IP allocations.
type IPSettings struct {
	DoubleNAT bool   `json:"double_nat"`
	PublicIP  string `json:"public_ip"`
}

// PremiumDNS tells us if eero Secure filtering is currently in use.
type PremiumDNS struct {
	DNSPoliciesEnabled           bool            `json:"dns_policies_enabled"`
	ZscalerLocationEnabled       bool            `json:"zscaler_location_enabled"`
	AnyPoliciesEnabledForNetwork bool            `json:"any_policies_enabled_for_network"`
	DNSPolicies                  DNSPolicies     `json:"dns_policies"`
	AdBlockSettings              AdBlockSettings `json:"ad_block_settings"`
}

// DNSPolicies determines whether advanced blockers map through.
type DNSPolicies struct {
	BlockMalware bool `json:"block_malware"`
	AdBlock      bool `json:"ad_block"`
}

// AdBlockSettings enables custom domains blocks and specific rules.
type AdBlockSettings struct {
	Enabled bool `json:"enabled"`
}

// NetworkPremiumDetails carries subscription context on an associated network.
type NetworkPremiumDetails struct {
	HasPaymentInfo   bool   `json:"has_payment_info"`
	Tier             string `json:"tier"`
	PaymentMethod    string `json:"payment_method"`
	Interval         string `json:"interval"`
	IsMySubscription bool   `json:"is_my_subscription"`
}

// IPv6Lease gives a broader upstream prefix context given from an ISP.
type IPv6Lease struct {
	Prefix      string   `json:"prefix"`
	Subnets     []string `json:"subnets"`
	NameServers []string `json:"name_servers"`
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
	Value float64 `json:"value"`
	Units string  `json:"units"`
}

// Health holds the overall network health indicators.
type Health struct {
	Internet    InternetHealth `json:"internet"`
	EeroNetwork HealthDetail   `json:"eero_network"`
}

// InternetHealth is a health metric specifically for the internet connection.
type InternetHealth struct {
	Status string `json:"status"`
	ISPUp  bool   `json:"isp_up"`
}

// HealthDetail is a single health metric.
type HealthDetail struct {
	Status string `json:"status"`
}

// EeroNode represents a single eero device (gateway or extender) in the mesh.
type EeroNode struct {
	URL                   string        `json:"url"`
	Serial                string        `json:"serial"`
	Location              string        `json:"location"`
	Joined                EeroTime      `json:"joined"`
	Gateway               bool          `json:"gateway"`
	IPAddress             string        `json:"ip_address"`
	Status                string        `json:"status"`
	Model                 string        `json:"model"`
	ModelNumber           string        `json:"model_number"`
	EthernetAddresses     []string      `json:"ethernet_addresses"`
	WifiBSSIDs            []string      `json:"wifi_bssids"`
	UpdateAvailable       bool          `json:"update_available"`
	OS                    string        `json:"os"`
	OSVersion             string        `json:"os_version"`
	MeshQualityBars       int           `json:"mesh_quality_bars"`
	Wired                 bool          `json:"wired"`
	LedOn                 bool          `json:"led_on"`
	UsingWan              bool          `json:"using_wan"`
	IsPrimaryNode         bool          `json:"is_primary_node"`
	MACAddress            string        `json:"mac_address"`
	IPv6Addresses         []IPv6Address `json:"ipv6_addresses"`
	ConnectedClientsCount int           `json:"connected_clients_count"`
	HeartbeatOK           bool          `json:"heartbeat_ok"`
	LastHeartbeat         time.Time     `json:"last_heartbeat"`
	ConnectionType        string        `json:"connection_type"`
	PowerInfo             PowerInfo     `json:"power_info"`
	Bands                 []string      `json:"bands"`
	ProvidesWifi          bool          `json:"provides_wifi"`
	State                 string        `json:"state"`
}

// IPv6Address holds the IPv6 configuration details for a single node interface.
type IPv6Address struct {
	Address   string `json:"address"`
	Scope     string `json:"scope"`
	Interface string `json:"interface"`
}

// PowerInfo details connection details regarding power usage.
type PowerInfo struct {
	PowerSource string `json:"power_source"`
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
