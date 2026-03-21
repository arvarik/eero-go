package eero

import (
	"bytes"
	"encoding/json"
	"time"
)

// EeroTime handles eero's custom timestamp formats that do not strictly comply
// with RFC3339, such as "2006-01-02T15:04:05+0000".
// It will try to parse using this custom format first, and fallback to
// time.RFC3339 format if it fails.
type EeroTime struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *EeroTime) UnmarshalJSON(b []byte) error {
	// 1. Handle explicit nulls safely
	if bytes.Equal(b, []byte("null")) {
		return nil
	}

	// 2. Decode the JSON string value (handling quotes, escapes, etc.)
	var s string
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' && !bytes.ContainsRune(b, '\\') {
		// Fast path for simple quoted strings
		s = string(b[1 : len(b)-1])
	} else {
		// Fallback for complex escapes or invalid formats
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
	}

	// 3. Handle empty strings
	if s == "" {
		return nil
	}

	// 4. Attempt parsing
	parsed, err := time.Parse("2006-01-02T15:04:05Z0700", s)
	if err != nil {
		// Fallback to strict format
		parsed, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return err
		}
	}
	t.Time = parsed
	return nil
}
