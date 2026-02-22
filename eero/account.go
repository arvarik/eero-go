package eero

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// AccountService provides access to the authenticated user's eero account.
type AccountService struct {
	client *Client
}

// --- Response types ---

// Account represents the authenticated user's eero account, including the
// list of networks they have access to.
type Account struct {
	Name                      string          `json:"name"`
	Phone                     AccountPhone    `json:"phone"`
	Email                     AccountEmail    `json:"email"`
	LogID                     string          `json:"log_id"`
	OrganizationID            *string         `json:"organization_id"`
	ImageAssets               any             `json:"image_assets"`
	Networks                  AccountNetworks `json:"networks"`
	Auth                      AccountAuth     `json:"auth"`
	Role                      string          `json:"role"`
	IsBetaBugReporterEligible bool            `json:"is_beta_bug_reporter_eligible"`
	ReportIssue               ReportIssue     `json:"report_issue"`
	CanTransfer               bool            `json:"can_transfer"`
	IsOwner                   bool            `json:"is_owner"`
	IsPremiumCapable          bool            `json:"is_premium_capable"`
	PaymentFailed             bool            `json:"payment_failed"`
	PremiumStatus             string          `json:"premium_status"`
	PremiumDetails            PremiumDetails  `json:"premium_details"`
	PushSettings              PushSettings    `json:"push_settings"`
	TrustCertificatesEtag     string          `json:"trust_certificates_etag"`
	Consents                  Consents        `json:"consents"`
	CanMigrateToAmazonLogin   bool            `json:"can_migrate_to_amazon_login"`
	EeroForBusiness           bool            `json:"eero_for_business"`
	MduProgram                bool            `json:"mdu_program"`
	BusinessDetails           any             `json:"business_details"`
}

// AccountEmail holds email-related account fields.
type AccountEmail struct {
	Value    string `json:"value"`
	Verified bool   `json:"verified"`
}

// AccountPhone holds phone-related account fields.
type AccountPhone struct {
	Value          string `json:"value"`
	CountryCode    string `json:"country_code"`
	NationalNumber string `json:"national_number"`
	Verified       bool   `json:"verified"`
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
	URL              string     `json:"url"`
	Name             string     `json:"name"`
	Created          time.Time  `json:"created"`
	NicknameLabel    *string    `json:"nickname_label"`
	AccessExpiresOn  *time.Time `json:"access_expires_on"`
	AmazonDirectedID string     `json:"amazon_directed_id"`
}

// AccountAuth represents the auth details for the account.
type AccountAuth struct {
	Type       string  `json:"type"`
	ProviderID *string `json:"provider_id"`
	ServiceID  *string `json:"service_id"`
}

// ReportIssue represents the feature flag/availability.
type ReportIssue struct {
	Enabled bool `json:"enabled"`
}

// PremiumDetails holds eero Plus/Secure subscription information.
type PremiumDetails struct {
	TrialEnds            *time.Time `json:"trial_ends"`
	HasPaymentInfo       bool       `json:"has_payment_info"`
	Tier                 string     `json:"tier"`
	SubscribedSince      *time.Time `json:"subscribed_since"`
	IsIapCustomer        bool       `json:"is_iap_customer"`
	PaymentMethod        string     `json:"payment_method"`
	Interval             string     `json:"interval"`
	NextBillingEventDate *time.Time `json:"next_billing_event_date"`
}

// PushSettings holds push notification preferences.
type PushSettings struct {
	NetworkOffline bool `json:"networkOffline"`
	NodeOffline    bool `json:"nodeOffline"`
}

// Consents holds user consent preferences.
type Consents struct {
	MarketingEmails MarketingEmailsConsent `json:"marketing_emails"`
}

// MarketingEmailsConsent holds the consent flag for marketing emails.
type MarketingEmailsConsent struct {
	Consented bool `json:"consented"`
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
