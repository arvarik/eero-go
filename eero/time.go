package eero

import "time"

// EeroTime handles eero's custom timestamp formats that do not strictly comply
// with RFC3339, such as "2006-01-02T15:04:05+0000".
// It will try to parse using this custom format first, and fallback to
// time.RFC3339 format if it fails.
type EeroTime struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *EeroTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" || s == `""` {
		return nil
	}
	s = s[1 : len(s)-1] // strip quotes
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
