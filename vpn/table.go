package vpn

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"net"
	"sort"
	"strings"

	"github.com/clmul/cutevpn"
)

type ipv4net struct {
	ip   uint32
	mask uint32
}

type route struct {
	dst ipv4net
	via cutevpn.IPv4
}

type routeTable []route

func (t routeTable) Len() int {
	return len(t)
}
func (t routeTable) Less(i, j int) bool {
	return bits.TrailingZeros32(t[i].dst.mask) < bits.TrailingZeros32(t[j].dst.mask)
}

func (t routeTable) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t routeTable) Get(dstIP cutevpn.IPv4) cutevpn.IPv4 {
	dst := binary.BigEndian.Uint32(dstIP[:])
	for _, r := range t {
		if dst&r.dst.mask == r.dst.ip {
			return r.via
		}
	}
	return emptyIPv4
}

func parseCIDRorIP(s string) (ipv4net, error) {
	var dstnet ipv4net
	_, ipnet, err := cutevpn.ParseCIDR(s)
	if err == nil {
		dstnet.ip = binary.BigEndian.Uint32(ipnet.IP)
		dstnet.mask = binary.BigEndian.Uint32(ipnet.Mask)
		return dstnet, nil
	}
	ip, err := cutevpn.ParseIPv4(s)
	if err == nil {
		dstnet.ip = binary.BigEndian.Uint32(ip[:])
		dstnet.mask = 0xffffffff
		return dstnet, nil
	}
	return dstnet, fmt.Errorf("wrong destination %s", s)
}

func parseRouteTable(ipnet *net.IPNet, routes []string) (routeTable, error) {
	var table routeTable
	for _, r := range routes {
		if len(r) == 0 {
			continue
		}
		parts := strings.Split(r, " via ")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid route, %s", r)
		}
		var dstnet ipv4net
		dst, via := parts[0], parts[1]
		dstnet, err := parseCIDRorIP(dst)
		if err != nil {
			return nil, fmt.Errorf("invalid route, %w", err)
		}
		viaIP, err := cutevpn.ParseIPv4(via)
		if err != nil {
			return nil, fmt.Errorf("invalid route, %w", err)
		}
		if !ipnet.Contains(viaIP[:]) {
			return nil, fmt.Errorf("destination %s is not in vpn network %s", viaIP, ipnet)
		}
		table = append(table, route{
			dst: dstnet,
			via: viaIP,
		})
	}
	sort.Sort(table)
	return table, nil
}
