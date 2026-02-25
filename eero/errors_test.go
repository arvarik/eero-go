package eero_test

import (
	"testing"

	"github.com/arvarik/eero-go/eero"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  eero.APIError
		want string
	}{
		{
			name: "With message",
			err: eero.APIError{
				HTTPStatusCode: 400,
				Code:           1001,
				Message:        "invalid input",
			},
			want: "eero: HTTP 400, API code 1001: invalid input",
		},
		{
			name: "Without message",
			err: eero.APIError{
				HTTPStatusCode: 500,
				Code:           5001,
			},
			want: "eero: HTTP 500, API code 5001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("APIError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError_IsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  eero.APIError
		want bool
	}{
		{
			name: "HTTP 401",
			err: eero.APIError{
				HTTPStatusCode: 401,
				Code:           0,
			},
			want: true,
		},
		{
			name: "API code 401",
			err: eero.APIError{
				HTTPStatusCode: 200,
				Code:           401,
			},
			want: true,
		},
		{
			name: "Neither 401",
			err: eero.APIError{
				HTTPStatusCode: 400,
				Code:           1001,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsAuthError(); got != tt.want {
				t.Errorf("APIError.IsAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}
