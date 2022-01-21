package geoip

import (
	"fmt"
	"net"
	"sync"

	"github.com/oschwald/maxminddb-golang"

	_ "embed"
)

type MaxMind struct{}

type mmResult struct {
	Country struct {
		Code string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Location struct {
		Timezone string `maxminddb:"time_zone"`
	} `maxminddb:"location"`
}

func (mm *MaxMind) Geolocate(ip net.IP) (*LookupResult, error) {
	mmInit()
	var cityRes mmResult
	cityErr := citys.Lookup(ip, &cityRes)
	// if either failed, then we should return an error
	if cityErr != nil {
		return nil, fmt.Errorf("geolocate err: %w", cityErr)
	}
	return &LookupResult{
		CountryCode: cityRes.Country.Code,
		Timezone:    cityRes.Location.Timezone,
	}, nil
}

// we generate the data we need from a MaxMind GeoLite2 databases
// Of course we will need to embed the data
//go:embed maxmind-geolite2-city.mmdb
var cityData []byte
var citys *maxminddb.Reader

var mmInitOnce = sync.Once{}

func mmInit() {
	mmInitOnce.Do(func() {
		var err error
		citys, err = maxminddb.FromBytes(cityData)
		if err != nil {
			panic("Error loading maxmind city DB: " + err.Error())
		}
	})
}
