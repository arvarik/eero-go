package eero

import (
	"context"
	"net/http"
)

// AuthService handles authentication against the eero API.
// The login flow is a two-step process:
//  1. Login sends an identifier (email or phone) and receives a user_token.
//  2. Verify sends the verification code (received via email/SMS) to complete
//     authentication and activate the session.
type AuthService struct {
	client *Client
}

// --- Request / Response types ---

// LoginRequest is the body sent to POST /login.
type LoginRequest struct {
	Login string `json:"login"`
}

// LoginResponse is the response from POST /login.
type LoginResponse struct {
	UserToken string `json:"user_token"`
}

// VerifyRequest is the body sent to POST /login/verify.
type VerifyRequest struct {
	Code string `json:"code"`
}

// --- Methods ---

// Login initiates the authentication challenge by sending an email address or
// phone number. Eero will send a verification code to the provided identifier.
// The returned user_token is automatically stored on the client and set as the
// session cookie for subsequent requests.
func (s *AuthService) Login(ctx context.Context, identifier string) (*LoginResponse, error) {
	body := LoginRequest{Login: identifier}

	req, err := s.client.newRequest(ctx, http.MethodPost, "/login", body)
	if err != nil {
		return nil, err
	}

	var res LoginResponse
	if err := s.client.do(req, &res); err != nil {
		return nil, err
	}

	// Store the user_token and set it as the session cookie so all
	// subsequent requests are authenticated.
	if err := s.client.SetSessionCookie(res.UserToken); err != nil {
		return nil, err
	}

	return &res, nil
}

// Verify completes the two-step authentication by sending the verification
// code that was delivered to the user's email or phone. After a successful
// verification, the session cookie is fully activated and all subsequent API
// calls will be authenticated.
func (s *AuthService) Verify(ctx context.Context, verificationCode string) error {
	body := VerifyRequest{Code: verificationCode}

	req, err := s.client.newRequest(ctx, http.MethodPost, "/login/verify", body)
	if err != nil {
		return err
	}

	return s.client.do(req, nil)
}
