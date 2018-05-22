package link

import (
	"fmt"
	"net"

	"github.com/clmul/cutevpn"
)

type udp4 struct {
	conn *net.UDPConn
	peer cutevpn.LinkAddr
}

func init() {
	cutevpn.RegisterLink("udp4", newUDP)
}

func newUDP(_ cutevpn.Looper, listen, dial string) (cutevpn.Link, error) {
	var peer cutevpn.LinkAddr
	var err error
	if dial != "" {
		peer, err = parseAddrPort(dial)
		if err != nil {
			return nil, err
		}
	}
	c, err := net.ListenPacket("udp4", listen)
	if err != nil {
		return nil, err
	}
	t := &udp4{
		conn: c.(*net.UDPConn),
		peer: peer,
	}
	return t, nil
}

func (t *udp4) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("udp4 %v->%v", t.conn.LocalAddr(), dst)
}

func (t *udp4) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t *udp4) Send(packet []byte, addr cutevpn.LinkAddr) error {
	_, err := t.conn.WriteToUDP(packet, convertToUDPAddr(addr.(AddrPort)))
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
