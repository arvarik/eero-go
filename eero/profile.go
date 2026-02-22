package eero

import (
	"context"
	"fmt"
	"net/http"
)

// ProfileService manages user profiles (e.g., family members) on an eero
// network, including pausing and unpausing internet access.
type ProfileService struct {
	client *Client
}

// --- Response types ---

// Profile represents a user profile on the eero network.
type Profile struct {
	URL              string    `json:"url"`
	Name             string    `json:"name"`
	Paused           bool      `json:"paused"`
	DeviceCount      int       `json:"device_count"`
	Devices          []Device  `json:"devices"`
	BlockApps        bool      `json:"block_apps"`
	SafeSearchActive bool      `json:"safe_search_enabled"`
	Bedtime          *Schedule `json:"bedtime"`
}

// ProfileDevice is a lightweight device reference within a profile.
// Unused directly; now mapped using the detailed complete Device models.

// Schedule represents a scheduled action (e.g., bedtime) on a profile.
type Schedule struct {
	Enabled bool   `json:"enabled"`
	Time    string `json:"time"`
}

// pauseRequest is the body for pausing/unpausing a profile.
type pauseRequest struct {
	Paused bool `json:"paused"`
}

// --- Methods ---

// List returns all profiles on the specified network.
//
// The networkURL parameter should be the exact relative URL from the account
// response (e.g., "/2.2/networks/12345").
func (s *ProfileService) List(ctx context.Context, networkURL string) ([]Profile, error) {
	req, err := s.client.newRequestFromURL(ctx, http.MethodGet, networkURL+"/profiles", nil)
	if err != nil {
		return nil, fmt.Errorf("profile: creating request: %w", err)
	}

	var resp EeroResponse[[]Profile]
	if err := s.client.doRaw(req, &resp); err != nil {
		return nil, fmt.Errorf("profile: %w", err)
	}

	return resp.Data, nil
}

// Pause pauses internet access for the given profile.
//
// The profileURL parameter should be the exact relative URL from the profile
// response (e.g., "/2.2/networks/12345/profiles/67890").
func (s *ProfileService) Pause(ctx context.Context, profileURL string) error {
	return s.setPaused(ctx, profileURL, true)
}

// Unpause resumes internet access for the given profile.
//
// The profileURL parameter should be the exact relative URL from the profile
// response (e.g., "/2.2/networks/12345/profiles/67890").
func (s *ProfileService) Unpause(ctx context.Context, profileURL string) error {
	return s.setPaused(ctx, profileURL, false)
}

func (s *ProfileService) setPaused(ctx context.Context, profileURL string, paused bool) error {
	body := pauseRequest{Paused: paused}

	req, err := s.client.newRequestFromURL(ctx, http.MethodPut, profileURL, body)
	if err != nil {
		return fmt.Errorf("profile: creating pause request: %w", err)
	}

	if err := s.client.doRaw(req, nil); err != nil {
		return fmt.Errorf("profile: pause: %w", err)
	}

	return nil
}
