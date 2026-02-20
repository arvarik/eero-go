package eero

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const (
	// DefaultBaseURL is the base URL for the eero API.
	DefaultBaseURL = "https://api-user.e2ro.com/2.2"

	// DefaultUserAgent mimics the eero iOS app.
	DefaultUserAgent = "eero/3.0 (iPhone; iOS 17.0)"
)

// Client is the top-level eero API client. It holds the HTTP client (with a
// cookie jar for automatic session management), the base URL, and references
// to each service.
type Client struct {
	// HTTPClient is the underlying HTTP client. It is initialized with a
	// cookie jar so that session cookies (Cookie: s=<user_token>) are
	// stored and replayed automatically.
	HTTPClient *http.Client

	// BaseURL is the root URL for all API requests.
	BaseURL string

	// UserAgent is the User-Agent header sent with every request.
	UserAgent string

	// userToken is stored after login so it can be set as a cookie.
	userToken string

	// Services — each service hangs off the client.
	Auth    *AuthService
	Account *AccountService
	Network *NetworkService
	Device  *DeviceService
	Profile *ProfileService
}

// NewClient creates a new eero API client with sensible defaults.
// The returned client uses a cookie jar for transparent session management.
func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("eero: creating cookie jar: %w", err)
	}

	c := &Client{
		HTTPClient: &http.Client{Jar: jar},
		BaseURL:    DefaultBaseURL,
		UserAgent:  DefaultUserAgent,
	}

	c.Auth = &AuthService{client: c}
	c.Account = &AccountService{client: c}
	c.Network = &NetworkService{client: c}
	c.Device = &DeviceService{client: c}
	c.Profile = &ProfileService{client: c}

	return c, nil
}

// SetSessionCookie programmatically sets the eero session cookie on the
// client's cookie jar. This is useful when restoring a previously obtained
// user_token without going through the full login flow.
func (c *Client) SetSessionCookie(userToken string) error {
	c.userToken = userToken
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("eero: parsing base URL: %w", err)
	}
	c.HTTPClient.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "s",
			Value: userToken,
		},
	})
	return nil
}

// newRequest creates an *http.Request with the appropriate headers and
// optional JSON body. The path is appended to the client's BaseURL.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	u := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("eero: marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("eero: creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// response is the non-generic envelope used internally by the do() helper.
// All responses have a "meta" field; the "data" field varies by endpoint.
type response struct {
	Meta APIError        `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// EeroResponse is a generic envelope for type-safe JSON unmarshaling of eero
// API responses. Use this when you want the compiler to enforce the data type
// at the call site — e.g., EeroResponse[[]Device] for list endpoints.
type EeroResponse[T any] struct {
	Meta APIError `json:"meta"`
	Data T        `json:"data"`
}

// do executes the given request and decodes the JSON envelope. If the API
// returns a non-2xx status or the meta.code indicates an error, a structured
// *APIError is returned. If v is non-nil, the "data" portion of the response
// envelope is decoded into it.
func (c *Client) do(req *http.Request, v any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("eero: executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("eero: reading response body: %w", err)
	}

	var envelope response
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		// If we can't parse the envelope at all, wrap the raw status.
		return &APIError{
			HTTPStatusCode: resp.StatusCode,
			Code:           resp.StatusCode,
			Message:        fmt.Sprintf("unparseable response body: %s", string(bodyBytes)),
		}
	}

	// Check for API-level errors.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Meta.Code >= 400 {
		apiErr := &envelope.Meta
		apiErr.HTTPStatusCode = resp.StatusCode
		return apiErr
	}

	// Decode the data payload if a target was provided.
	if v != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, v); err != nil {
			return fmt.Errorf("eero: decoding data payload: %w", err)
		}
	}

	return nil
}

// doRaw executes the given request and unmarshals the entire JSON response
// body into v. Unlike do(), this method does not separate the "meta" and
// "data" fields — it is intended for use with EeroResponse[T] where the
// caller controls the full envelope type. Error checking is performed by
// inspecting the HTTP status and parsing a meta envelope from the raw bytes.
func (c *Client) doRaw(req *http.Request, v any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("eero: executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("eero: reading response body: %w", err)
	}

	// Check for API-level errors by peeking at the meta envelope.
	var meta struct {
		Meta APIError `json:"meta"`
	}
	if err := json.Unmarshal(bodyBytes, &meta); err != nil {
		return &APIError{
			HTTPStatusCode: resp.StatusCode,
			Code:           resp.StatusCode,
			Message:        fmt.Sprintf("unparseable response body: %s", string(bodyBytes)),
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || meta.Meta.Code >= 400 {
		apiErr := &meta.Meta
		apiErr.HTTPStatusCode = resp.StatusCode
		return apiErr
	}

	// Unmarshal the full response into the caller's target.
	if v != nil {
		if err := json.Unmarshal(bodyBytes, v); err != nil {
			return fmt.Errorf("eero: decoding response: %w", err)
		}
	}

	return nil
}

// originURL returns the scheme+host portion of BaseURL (e.g.,
// "https://api-user.e2ro.com") so that callers can build URLs from full
// relative paths like "/2.2/networks/12345" without double-prefixing the
// version segment.
func (c *Client) originURL() string {
	// BaseURL is expected to be like "https://api-user.e2ro.com/2.2".
	// We strip from the third slash onward.
	if idx := strings.Index(c.BaseURL[8:], "/"); idx >= 0 {
		return c.BaseURL[:8+idx]
	}
	return c.BaseURL
}

// newRequestFromURL creates an *http.Request using a full relative path
// (e.g., "/2.2/networks/12345") resolved against the API origin, rather
// than appending to BaseURL. This avoids duplicate path prefixes when the
// caller already has a complete API-relative URL.
func (c *Client) newRequestFromURL(ctx context.Context, method, relativeURL string, body any) (*http.Request, error) {
	u := c.originURL() + relativeURL

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("eero: marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("eero: creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}
