package link

import (
	"fmt"
	"github.com/clmul/cutevpn"
	"net"
)

type AddrPort struct {
	ip   cutevpn.IPv4
	port int
}

func (a AddrPort) String() string {
	return fmt.Sprintf("%v:%v", a.ip, a.port)
}

func convertUDPAddr(udpAddr *net.UDPAddr) (a AddrPort) {
	a.port = udpAddr.Port
	copy(a.ip[:], udpAddr.IP.To4())
	return a
}

func convertToUDPAddr(a AddrPort) *net.UDPAddr {
	udpAddr := net.UDPAddr{
		IP:   net.IP(a.ip[:]),
		Port: int(a.port),
	}
	return &udpAddr
}

func parseAddrPort(addr string) (cutevpn.LinkAddr, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}
	return convertUDPAddr(udpAddr), nil
}

func convertTCPAddr(tcpAddr *net.TCPAddr) (a AddrPort) {
	a.port = tcpAddr.Port
	copy(a.ip[:], tcpAddr.IP.To4())
	return a
}
