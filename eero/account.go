package eero

import (
	"context"
	"fmt"
	"net/http"
)

// AccountService provides access to the authenticated user's eero account.
type AccountService struct {
	client *Client
}

// --- Response types ---

// Account represents the authenticated user's eero account, including the
// list of networks they have access to.
type Account struct {
	Name          string          `json:"name"`
	Phone         string          `json:"phone"`
	Email         AccountEmail    `json:"email"`
	LogID         string          `json:"log_id"`
	Networks      AccountNetworks `json:"networks"`
	Role          string          `json:"role"`
	CanTransfer   bool            `json:"can_transfer"`
	IsOwner       bool            `json:"is_owner"`
	PremiumStatus string          `json:"premium_status"`
}

// AccountEmail holds email-related account fields.
type AccountEmail struct {
	Value    string `json:"value"`
	Verified bool   `json:"verified"`
}

// AccountNetworks holds the network count and the list of network references.
type AccountNetworks struct {
	Count int              `json:"count"`
	Data  []NetworkSummary `json:"data"`
}

// NetworkSummary is a lightweight reference to a network, returned within the
// account payload. The URL field contains the full relative API path
// (e.g., "/2.2/networks/12345") that should be passed directly to
// NetworkService.Get or DeviceService.List.
type NetworkSummary struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// --- Methods ---

// Get retrieves the authenticated user's account information, including the
// list of networks they have access to.
//
// The returned Account.Networks.Data entries contain a URL field
// (e.g., "/2.2/networks/12345") that can be passed directly to
// NetworkService.Get and DeviceService.List.
func (s *AccountService) Get(ctx context.Context) (*Account, error) {
	req, err := s.client.newRequest(ctx, http.MethodGet, "/account", nil)
	if err != nil {
		return nil, fmt.Errorf("account: creating request: %w", err)
	}

	var resp EeroResponse[Account]
	if err := s.client.doRaw(req, &resp); err != nil {
		return nil, fmt.Errorf("account: %w", err)
	}

	return &resp.Data, nil
}
