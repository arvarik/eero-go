// Package eero provides a Go client for the eero router REST API.
package eero

import "fmt"

// APIError represents an error returned by the eero API.
// Eero responses include a "meta" envelope with a status code and optional
// error message. This struct captures both the HTTP-level and API-level error
// information.
type APIError struct {
	// HTTPStatusCode is the HTTP status code of the response.
	HTTPStatusCode int `json:"-"`
	// Code is the API-level status code from the "meta" envelope.
	Code int `json:"code"`
	// Message is the human-readable error message from the API.
	Message string `json:"error"`
	// ServerTime is the server timestamp from the "meta" envelope.
	ServerTime string `json:"server_time"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("eero: HTTP %d, API code %d: %s", e.HTTPStatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("eero: HTTP %d, API code %d", e.HTTPStatusCode, e.Code)
}

// IsAuthError reports whether the API error indicates an authentication failure.
func (e *APIError) IsAuthError() bool {
	return e.HTTPStatusCode == 401 || e.Code == 401
}
