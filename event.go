package hindsight

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/0x6377/hindsight/geoip"
)

type InboundEvent struct {
	Time         time.Time
	IP           string
	Host         string
	Method       string
	Path         string
	UserAgent    string
	StatusCode   int64         // must be a valid code
	BytesWritten int64         // must be non-negative
	Duration     time.Duration //`json:"-"`
}

func (in *InboundEvent) UnmarshalJSON(b []byte) error {
	// make a map[string]interface{} and then be more specific
	// with the keys. this way we can ensure all fields are present
	// and cased correctly.
	m := make(map[string]interface{}, 9)
	// We actually want to use the "decoder" so we can "useNumber" to get big numbers.

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	err := dec.Decode(&m)
	if err != nil {
		return err // plain bad JSON
	}
	// We should have 9 fields
	if len(m) > 9 {
		return fmt.Errorf("event has unexpected fields")
	}
	// now for each field.
	// Time should be an RFC3339 string
	if err := unmarshalStringField(m, "Time", func(s string) error {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			in.Time = t
			return nil
		} else {
			return fmt.Errorf("event 'Time' key not a string formatted in RFC3339: %w", err)
		}
	}); err != nil {
		return err
	}
	// IP
	if err := unmarshalStringField(m, "IP", func(s string) error {
		// Should I validate?
		if ip := net.ParseIP(s); ip == nil {
			return fmt.Errorf("event 'IP' key did not contain a valid IP")
		}
		in.IP = s
		return nil
	}); err != nil {
		return err
	}
	// Host
	if err := unmarshalStringField(m, "Host", func(s string) error {
		// Should I normalise? remove trailing dot, etc...
		in.Host = s
		return nil
	}); err != nil {
		return err
	}
	// Method
	if err := unmarshalStringField(m, "Method", func(s string) error {
		// Should I whitelist?
		in.Method = s
		return nil
	}); err != nil {
		return err
	}
	// Path
	if err := unmarshalStringField(m, "Path", func(s string) error {
		in.Path = s
		return nil
	}); err != nil {
		return err
	}
	// UserAgent
	if err := unmarshalStringField(m, "UserAgent", func(s string) error {
		in.UserAgent = s
		return nil
	}); err != nil {
		return err
	}
	// StatusCode
	if err := unmarshalIntField(m, "StatusCode", func(n int64) error {
		if n < 100 || n >= 600 {
			return fmt.Errorf("event 'StatusCode' should be between 100 and 599")
		}
		in.StatusCode = n
		return nil
	}); err != nil {
		return err
	}
	// BytesWritten
	if err := unmarshalIntField(m, "BytesWritten", func(n int64) error {
		if n < 0 {
			return fmt.Errorf("event 'BytesWritten' should be non-negative")
		}
		in.BytesWritten = n
		return nil
	}); err != nil {
		return err
	}
	// Duration (in millisecnds)
	if err := unmarshalIntField(m, "Duration", func(n int64) error {
		if n < 0 {
			return fmt.Errorf("event 'BytesWritten' should be non-negative")
		}
		in.Duration = time.Duration(n) * time.Millisecond
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func unmarshalStringField(m map[string]interface{}, key string, fn func(s string) error) error {
	if i, ok := m[key]; !ok {
		return fmt.Errorf("event missing the %q key", key)
	} else {
		if s, ok := i.(string); !ok {
			return fmt.Errorf("event %q was not a string", key)
		} else {
			return fn(s)
		}
	}
}
func unmarshalIntField(m map[string]interface{}, key string, fn func(n int64) error) error {
	if i, ok := m[key]; !ok {
		return fmt.Errorf("event missing the %q key", key)
	} else {
		// they will actually come out as a json.Number
		if s, ok := i.(json.Number); !ok {
			return fmt.Errorf("event %q was not an integer", key)
		} else {
			if n, err := s.Int64(); err != nil {
				return fmt.Errorf("event %q was not an integer", key)
			} else {
				return fn(n)
			}
		}
	}
}

// this is the anonymised event
type Event struct {
	Key                                string // from UA/IP/current time
	Time                               time.Time
	Host, Path, Method                 string         // from request
	Device                             string         // from UA
	Browser, OS                        NameAndVersion // from UA
	CountryCode, TimeZone              string         // from IP
	StatusCode, Duration, BytesWritten int64          // from response
}

type NameAndVersion struct {
	Name, Version string
}

func (nv *NameAndVersion) String() string {
	return nv.Name + " " + nv.Version
}

func mapInboundEvent(c *Config, in *InboundEvent) *Event {
	uainfo := DecodeUserAgent(in.UserAgent)
	loc := geoip.MustGeolocate(net.ParseIP(in.IP))
	return &Event{
		Key:  UniqueKey(c, in),
		Time: in.Time,

		Device:  string(uainfo.Device),
		Browser: uainfo.Browser,
		OS:      uainfo.OS,

		CountryCode: loc.CountryCode,
		TimeZone:    loc.Timezone,

		Host:   in.Host, // should we clean/canonicalise it?
		Method: in.Method,
		Path:   in.Path, // should we clean/canonicalise it?

		Duration:     int64(in.Duration / time.Millisecond),
		BytesWritten: in.BytesWritten,
		StatusCode:   in.StatusCode,
	}

}
