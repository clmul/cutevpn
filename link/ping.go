package link

import (
	"crypto/rand"
	"fmt"
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/clmul/cutevpn"
)

type ping struct {
	conn *icmp.PacketConn
	peer cutevpn.LinkAddr
}

const (
	DialerSeq   = 65123
	ListenerSeq = 65321
)

func init() {
	cutevpn.RegisterLink("ping", newPing)
}

func newPing(_ cutevpn.Looper, listen, dial string) (cutevpn.Link, error) {
	link := &ping{}

	var err error
	if dial != "" {
		ip, err := parseIPv4(dial)
		if err != nil {
			return nil, err
		}
		idBuf := make([]byte, 2)
		_, err = rand.Read(idBuf)
		if err != nil {
			return nil, err
		}
		//id := int(idBuf[0])<<8 | int(idBuf[1])
		peer := AddrPort{
			ip:   ip.(cutevpn.IPv4),
			port: 6,
		}
		link.peer = peer
	}

	link.conn, err = icmp.ListenPacket("ip4:icmp", listen)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func (t *ping) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("ping %v->%v", t.conn.LocalAddr(), dst)
}

func (t *ping) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t *ping) Send(packet []byte, addr cutevpn.LinkAddr) error {
	addrPort := addr.(AddrPort)
	var icmpType icmp.Type
	var seq int
	if addr == t.peer {
		icmpType = ipv4.ICMPTypeEcho
		seq = DialerSeq
	} else {
		icmpType = ipv4.ICMPTypeEchoReply
		seq = ListenerSeq
	}
	msg := icmp.Message{
		Type: icmpType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   addrPort.port,
			Seq:  seq,
			Data: packet,
		},
	}
	msgData, err := msg.Marshal(nil)
	if err != nil {
		return err
	}
	_, err = t.conn.WriteTo(msgData, &net.IPAddr{IP: addrPort.ip[:]})
	return err
}

func (t *ping) Recv(packet []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	n, udpAddr, err := t.conn.ReadFrom(packet)
	if err != nil {
		return
	}
	p = packet[:n]
	msg, err := icmp.ParseMessage(1, p)
	if err != nil {
		return
	}
	body, ok := msg.Body.(*icmp.Echo)
	if !ok {
		return t.Recv(packet)
	}
	ip := convertIPAddr(udpAddr.(*net.IPAddr))
	port := body.ID
	addrPort := AddrPort{
		ip: ip, port: port,
	}

	var seq int
	if addrPort == t.peer {
		seq = ListenerSeq
	} else {
		seq = DialerSeq
	}
	if body.Seq != seq {
		return t.Recv(packet)
	}

	return body.Data, addrPort, nil
}

func (t *ping) Close() error {
	return t.conn.Close()
}

func (t *ping) Overhead() int {
	return 20 + 8
}
