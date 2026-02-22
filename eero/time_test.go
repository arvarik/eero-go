package eero_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/arvarik/eero-go/eero"
)

func TestEeroTime_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		wantErr  bool
		expected time.Time
	}{
		{
			name:     "Success_EeroCustomFormat",
			payload:  `"2026-02-21T22:14:52+0000"`,
			wantErr:  false,
			expected: time.Date(2026, time.February, 21, 22, 14, 52, 0, time.UTC),
		},
		{
			name:     "Success_RFC3339Format",
			payload:  `"2026-02-21T22:14:52Z"`,
			wantErr:  false,
			expected: time.Date(2026, time.February, 21, 22, 14, 52, 0, time.UTC),
		},
		{
			name:     "Success_Null",
			payload:  `null`,
			wantErr:  false,
			expected: time.Time{},
		},
		{
			name:     "Success_EmptyString",
			payload:  `""`,
			wantErr:  false,
			expected: time.Time{},
		},
		{
			name:    "Failure_InvalidString",
			payload: `"not-a-date"`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var et eero.EeroTime
			err := json.Unmarshal([]byte(tc.payload), &et)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("Expected parsing error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected parsing error: %v", err)
			}

			if !et.Time.Equal(tc.expected) {
				t.Fatalf("Time parsed incorrectly. Wanted %s, got %s", tc.expected.Format(time.RFC3339), et.Time.Format(time.RFC3339))
			}
		})
	}
}
