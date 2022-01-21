package hindsight

import (
	"time"
)

type InboundEvent struct {
	Time         time.Time
	IP           string
	Host         string
	Method       string
	Path         string
	UserAgent    string
	StatusCode   int           // must be a valid code
	BytesWritten int           // must be non-negative
	Duration     time.Duration //`json:"-"`
}

// this is the anonymised event
type Event struct {
	Key                                string // from UA/IP/current time
	Time                               time.Time
	Host, Path, Method                 string         // from request
	Device                             string         // from UA
	Browser, OS                        NameAndVersion // from UA
	CountryCode, TimeZone              string         // from IP
	StatusCode, Duration, BytesWritten int            // from response
}

type NameAndVersion struct {
	Name, Version string
}

func (nv *NameAndVersion) String() string {
	return nv.Name + " " + nv.Version
}
