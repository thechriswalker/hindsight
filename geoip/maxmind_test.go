package geoip

import (
	"fmt"
	"net"
	"testing"
)

func TestMaxmind(*testing.T) {
	ip := net.ParseIP("2001:41d0:a:218f::1")
	//ip := net.ParseIP("37.187.97.143")
	//ip := net.ParseIP("212.69.49.161")
	mm := MaxMind{}
	res, err := mm.Geolocate(ip)

	fmt.Printf("err: %s\n", err)
	fmt.Printf("res: %#v\n", res)
}
