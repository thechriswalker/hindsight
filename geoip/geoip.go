package geoip

import (
	"errors"
	"net"
)

// This is the data we will give back to the caller
type LookupResult struct {
	CountryCode string
	Timezone    string
}

var ErrUnknown = errors.New("unknown IP Address")

// This is the exposed interface
type Geolocater interface {
	// main lookup
	Geolocate(ip net.IP) (*LookupResult, error)
}
