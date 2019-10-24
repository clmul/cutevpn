package cutevpn

import (
	"fmt"
	"net"
)

func ParseIPv4(addr string) (ipv4 IPv4, err error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return ipv4, ErrInvalidIP
	}
	ip = ip.To4()
	if ip == nil {
		return ipv4, ErrNoIPv6
	}
	copy(ipv4[:], ip)
	return ipv4, nil
}

func ParseCIDR(cidr string) (ipv4 IPv4, ipnet *net.IPNet, err error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return
	}
	ip = ip.To4()
	if ip == nil {
		return ipv4, ipnet, ErrNoIPv6
	}
	copy(ipv4[:], ip)
	return
}

func GetSrcIP(packet []byte) IPv4 {
	var ip IPv4
	copy(ip[:], packet[12:16])
	return ip
}

func GetDstIP(packet []byte) IPv4 {
	var ip IPv4
	copy(ip[:], packet[16:20])
	return ip
}

type AddrPort struct {
	IP   IPv4
	Port int
}

func (ap AddrPort) String() string {
	return fmt.Sprintf("%v:%v", ap.IP, ap.Port)
}

func ConvertNetAddr(ip net.IP, port int) AddrPort {
	r := AddrPort{}
	copy(r.IP[:], ip.To4())
	r.Port = port
	return r
}

func ConvertToNetAddr(ap AddrPort) (ip net.IP, port int) {
	return ap.IP[:], ap.Port
}
