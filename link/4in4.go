package link

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/clmul/cutevpn"
)

type iviniv struct {
	*net.IPConn
	protocol uint8
	peer     cutevpn.LinkAddr
}

var singletons [256]*net.IPConn

func init() {
	cutevpn.RegisterLink("4in4", new4in4)
}

func new4in4(_ cutevpn.Looper, listen, dial string) (cutevpn.Link, error) {
	link := &iviniv{}

	ipAddr, protocol, err := parse(listen)
	if err != nil {
		return nil, err
	}
	link.protocol = protocol

	if dial != "" {
		link.peer, err = parseIPv4(dial)
		if err != nil {
			return nil, err
		}
	}

	c := singletons[protocol]
	if c == nil {
		c, err = net.ListenIP(fmt.Sprintf("ip4:%d", protocol), ipAddr)
		if err != nil {
			return nil, err
		}
		singletons[protocol] = c
	}
	link.IPConn = c
	return link, nil
}

func convertIPAddr(addr *net.IPAddr) (a cutevpn.IPv4) {
	copy(a[:], addr.IP.To4())
	return a
}

func convertToIPAddr(a cutevpn.IPv4) *net.IPAddr {
	return &net.IPAddr{IP: net.IP(a[:])}
}

func (t iviniv) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("4in4 %v:%v->%v", t.IPConn.LocalAddr(), t.protocol, dst)
}

func parse(addr string) (*net.IPAddr, uint8, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return nil, 0, fmt.Errorf("wrong addr format %v", addr)
	}
	host, protocol := parts[0], parts[1]
	number, err := strconv.ParseUint(protocol, 10, 8)
	if err != nil {
		return nil, 0, fmt.Errorf("wrong protocol number: %v, it must between 0 and 255", protocol)
	}
	ipAddr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return nil, 0, err
	}
	return ipAddr, uint8(number), nil
}

func parseIPv4(addr string) (cutevpn.LinkAddr, error) {
	ipAddr, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		return nil, err
	}
	return convertIPAddr(ipAddr), nil
}

func (t iviniv) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t iviniv) Send(packet []byte, addr cutevpn.LinkAddr) error {
	_, err := t.WriteToIP(packet, convertToIPAddr(addr.(cutevpn.IPv4)))
	return err
}

func (t iviniv) Recv(packet []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	n, ipAddr, err := t.ReadFromIP(packet)
	if err != nil {
		return
	}
	p = packet[:n]
	addr = convertIPAddr(ipAddr)
	return
}

func (t iviniv) Close() error {
	return t.IPConn.Close()
}

func (t iviniv) Overhead() int {
	return 20
}
