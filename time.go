package weheat

import (
	"strings"
	"time"
)

// Time is a time.Time that also accepts the zone-less timestamps the WeHeat backend emits.
// Zone-less values are interpreted as UTC, which is what the API documents.
type Time struct {
	time.Time
}

const zonelessLayout = "2006-01-02T15:04:05.999999999"

// UnmarshalJSON accepts RFC 3339 with or without a zone offset.
func (t *Time) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		parsed, err = time.ParseInLocation(zonelessLayout, s, time.UTC)
	}
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

// MarshalJSON writes RFC 3339 in UTC.
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.UTC().Format(time.RFC3339Nano) + `"`), nil
}
