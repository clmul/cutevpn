package link

import (
	"fmt"
	"net"

	"github.com/clmul/cutevpn"
)

type udp4 struct {
	conn *net.UDPConn
}

type udp4Addr struct {
	ip   cutevpn.IPv4
	port int
}

func (a udp4Addr) String() string {
	return fmt.Sprintf("%v:%v", a.ip, a.port)
}

func init() {
	cutevpn.RegisterLink("udp4", newUDP)
}

func newUDP(listen string) (cutevpn.Link, error) {
	c, err := net.ListenPacket("udp4", listen)
	if err != nil {
		return nil, err
	}
	t := &udp4{
		conn: c.(*net.UDPConn),
	}
	return t, nil
}

func convertUDPAddr(udpAddr *net.UDPAddr) (a udp4Addr) {
	a.port = udpAddr.Port
	copy(a.ip[:], udpAddr.IP.To4())
	return a
}

func convertToUDPAddr(a udp4Addr) *net.UDPAddr {
	udpAddr := net.UDPAddr{
		IP:   net.IP(a.ip[:]),
		Port: int(a.port),
	}
	return &udpAddr
}

func (t *udp4) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("udp4 %v->%v", t.conn.LocalAddr(), dst)
}

func (t *udp4) ParseAddr(addr string) (cutevpn.LinkAddr, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}
	return convertUDPAddr(udpAddr), nil
}

func (t *udp4) Send(packet []byte, addr cutevpn.LinkAddr) error {
	_, err := t.conn.WriteToUDP(packet, convertToUDPAddr(addr.(udp4Addr)))
	return err
}

func (t *udp4) Recv(packet []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	n, udpAddr, err := t.conn.ReadFromUDP(packet)
	if err != nil {
		return
	}
	p = packet[:n]
	addr = convertUDPAddr(udpAddr)
	return
}

func (t *udp4) Close() error {
	return t.conn.Close()
}

func (t *udp4) Overhead() int {
	return 20 + 8
}
