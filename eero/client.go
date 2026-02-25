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
	"sync"
	"time"
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
	// HTTPClient handles requests and secure cookie jar persistence.
	// The session cookie state is managed internally by the jar, making
	// it safe for concurrent use across Goroutines.
	HTTPClient *http.Client

	// BaseURL is the root URL for all API requests.
	BaseURL string

	// UserAgent is the User-Agent header sent with every request.
	UserAgent string

	// Services — each service hangs off the client.
	Auth    *AuthService
	Account *AccountService
	Network *NetworkService
	Device  *DeviceService
	Profile *ProfileService

	// originMu protects cachedOriginURL and originURLSnapshot
	originMu sync.RWMutex

	// cachedOriginURL stores the parsed origin URL (scheme + host) to avoid
	// re-parsing on every call to originURL().
	cachedOriginURL *url.URL

	// originURLSnapshot stores the BaseURL string that cachedOriginURL was
	// derived from. If BaseURL changes, we invalidate the cache.
	originURLSnapshot string
}

// NewClient creates a new eero API client with sensible defaults.
// The returned client uses a cookie jar for transparent session management
// and is secured against resource leaks and open-redirect cookie theft.
func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("eero: creating cookie jar: %w", err)
	}

	// Define a resilient custom transport instead of relying on DefaultTransport.
	// This prevents unbounded idle connection exhaustion or hanging handshakes.
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   30 * time.Second, // Fallback timeout for the entire HTTP exchange
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// SECURITY: Prevent Open-Redirect Session Hijacking.
			// If the API attempts to redirect us to a different domain, abort immediately.
			// This ensures the cookie jar never leaks the eero session key.
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				return fmt.Errorf("security policy: blocked cross-domain redirect to %s", req.URL.Host)
			}
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	c := &Client{
		HTTPClient: httpClient,
		BaseURL:    DefaultBaseURL,
		UserAgent:  DefaultUserAgent,
	}

	// Initialize the origin URL cache for the default BaseURL.
	// We ignore errors here because DefaultBaseURL is a constant known to be valid.
	if u, err := url.Parse(DefaultBaseURL); err == nil {
		c.cachedOriginURL = &url.URL{Scheme: u.Scheme, Host: u.Host}
		c.originURLSnapshot = DefaultBaseURL
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
// user_token without going through the full login flow. The underlying
// cookiejar executes safely across concurrent Goroutines.
func (c *Client) SetSessionCookie(userToken string) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("eero: parsing base URL: %w", err)
	}
	c.HTTPClient.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:   "s",
			Value:  userToken,
			Secure: true, // Enforce transit over HTTPS
		},
	})
	return nil
}

// newRequest creates an *http.Request with the appropriate headers and
// optional JSON body. The path is appended to the client's BaseURL.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	// We use simple string concatenation here because BaseURL typically contains
	// a path prefix (e.g. "/2.2") and path typically starts with "/".
	// using ResolveReference would drop the BaseURL path if the new path starts with "/".
	u := c.BaseURL + path
	return c.buildRequest(ctx, method, u, body)
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

// performRequest executes the HTTP request and reads the response body up to a limit.
func (c *Client) performRequest(req *http.Request) ([]byte, int, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("eero: executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// SECURITY: Limit payloads to 5MB to prevent memory exhaustion / DoS attacks.
	const maxBodyBytes = 5 * 1024 * 1024
	bodyReader := io.LimitReader(resp.Body, maxBodyBytes)

	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("eero: reading response body: %w", err)
	}
	return bodyBytes, resp.StatusCode, nil
}

// checkError inspects the response body and status code for API errors.
func (c *Client) checkError(bodyBytes []byte, statusCode int) error {
	var meta struct {
		Meta APIError `json:"meta"`
	}
	if err := json.Unmarshal(bodyBytes, &meta); err != nil {
		return &APIError{
			HTTPStatusCode: statusCode,
			Code:           statusCode,
			Message:        fmt.Sprintf("unparseable response body: %s", string(bodyBytes)),
		}
	}
	if statusCode < 200 || statusCode >= 300 || meta.Meta.Code >= 400 {
		apiErr := &meta.Meta
		apiErr.HTTPStatusCode = statusCode
		return apiErr
	}
	return nil
}

// do executes the given request and decodes the JSON envelope. If the API
// returns a non-2xx status or the meta.code indicates an error, a structured
// *APIError is returned. If v is non-nil, the "data" portion of the response
// envelope is decoded into it.
func (c *Client) do(req *http.Request, v any) error {
	var envelope response
	if err := c.doRaw(req, &envelope); err != nil {
		return err
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
	bodyBytes, statusCode, err := c.performRequest(req)
	if err != nil {
		return err
	}

	if err := c.checkError(bodyBytes, statusCode); err != nil {
		return err
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
func (c *Client) originURL() (*url.URL, error) {
	// Fast path: if BaseURL hasn't changed since we last parsed it, use the cache.
	c.originMu.RLock()
	cached := c.cachedOriginURL
	snapshot := c.originURLSnapshot
	c.originMu.RUnlock()

	// Note: We access c.BaseURL without a lock here because it's a public field
	// that users can modify directly. Any race on BaseURL itself is the caller's
	// responsibility. We only protect our cache consistency relative to what we see.
	if cached != nil && c.BaseURL == snapshot {
		// Return a copy to prevent callers from mutating the cached value
		u := *cached
		return &u, nil
	}

	// Slow path: acquire write lock to update cache
	c.originMu.Lock()
	defer c.originMu.Unlock()

	// Double-check: maybe another goroutine updated it while we were waiting
	if c.cachedOriginURL != nil && c.BaseURL == c.originURLSnapshot {
		u := *c.cachedOriginURL
		return &u, nil
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}

	// If the URL doesn't have a scheme or host, return as is (but don't cache).
	if u.Scheme == "" || u.Host == "" {
		return u, nil
	}

	// Create the origin URL (Scheme + Host only)
	origin := &url.URL{Scheme: u.Scheme, Host: u.Host}

	// Update cache
	c.cachedOriginURL = origin
	c.originURLSnapshot = c.BaseURL

	// Return a copy
	ret := *origin
	return &ret, nil
}

// newRequestFromURL creates an *http.Request using a full relative path
// (e.g., "/2.2/networks/12345") resolved against the API origin, rather
// than appending to BaseURL. This avoids duplicate path prefixes when the
// caller already has a complete API-relative URL.
func (c *Client) newRequestFromURL(ctx context.Context, method, relativeURL string, body any) (*http.Request, error) {
	base, err := c.originURL()
	if err != nil {
		return nil, fmt.Errorf("eero: parsing origin URL: %w", err)
	}
	rel, err := url.Parse(relativeURL)
	if err != nil {
		return nil, fmt.Errorf("eero: parsing relative URL: %w", err)
	}
	u := base.ResolveReference(rel)

	// SECURITY: Prevent SSRF by ensuring we never send credentials/requests
	// to a host other than the configured API origin.
	if u.Host != base.Host {
		return nil, fmt.Errorf("eero: security policy blocked request to %s (expected %s)", u.Host, base.Host)
	}

	uStr := u.String()

	return c.buildRequest(ctx, method, uStr, body)
}

func (c *Client) buildRequest(ctx context.Context, method, urlStr string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("eero: marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("eero: creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}
