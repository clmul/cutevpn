package vpn

import (
	"testing"

	"github.com/clmul/cutevpn"
)

func TestParseTable(t *testing.T) {
	var routes = []string{
		"10.10.10.10/30 via 192.168.1.2",
		"10.10.10.10 via 192.168.1.1",
		"10.10.0.0/16 via 192.168.1.4",
		"10.0.0.0/8 via 192.168.1.5",
		"10.10.10.0/24 via 192.168.1.3",
	}
	var answer = routeTable{
		{dst: ipv4net{0x0a0a0a0a, 0xffffffff}, via: cutevpn.IPv4{192, 168, 1, 1}},
		{dst: ipv4net{0x0a0a0a08, 0xfffffffc}, via: cutevpn.IPv4{192, 168, 1, 2}},
		{dst: ipv4net{0x0a0a0a00, 0xffffff00}, via: cutevpn.IPv4{192, 168, 1, 3}},
		{dst: ipv4net{0x0a0a0000, 0xffff0000}, via: cutevpn.IPv4{192, 168, 1, 4}},
		{dst: ipv4net{0x0a000000, 0xff000000}, via: cutevpn.IPv4{192, 168, 1, 5}},
	}
	_, ipnet, err := cutevpn.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatal(err)
	}
	table, err := parseRouteTable(ipnet, routes)
	if err != nil {
		t.Fatal(err)
	}
	for i := range table {
		if table[i] != answer[i] {
			t.Errorf("not equal\n%v\n%v", table[i], answer[i])
		}
	}
}
