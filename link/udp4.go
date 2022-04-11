package link

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/clmul/cutevpn"
)

type udp4 struct {
	listen string
	peer   cutevpn.LinkAddr
	conn   *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
}

func newUDP(vpn cutevpn.VPN, ctx context.Context, linkURL *url.URL) (cutevpn.Link, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &udp4{
		ctx:    ctx,
		cancel: cancel,
	}

	if linkURL.Query().Get("listen") == "1" {
		t.listen = linkURL.Host
	} else {
		addr, err := net.ResolveUDPAddr("udp4", linkURL.Host)
		if err != nil {
			return nil, err
		}
		t.peer = cutevpn.ConvertNetAddr(addr.IP, addr.Port)
	}
	c, err := net.ListenPacket("udp4", t.listen)
	if err != nil {
		return nil, err
	}
	vpn.OnCancel(ctx, func() {
		err := c.Close()
		if err != nil {
			log.Println(err)
		}
	})
	t.conn = c.(*net.UDPConn)

	return t, nil
}

func (t *udp4) ToString(dst cutevpn.LinkAddr) string {
	if dst == nil {
		dst = "any"
	}
	return fmt.Sprintf("udp4 %v->%v", t.conn.LocalAddr(), dst)
}

func (t *udp4) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t *udp4) Send(packet []byte, addr cutevpn.LinkAddr) error {
	ip, port := cutevpn.ConvertToNetAddr(addr.(cutevpn.AddrPort))
	_, err := t.conn.WriteToUDP(packet, &net.UDPAddr{IP: ip, Port: port})
	return err
}

func (t *udp4) Recv(packet []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	n, udpAddr, err := t.conn.ReadFromUDP(packet)
	if err != nil {
		return nil, nil, err
	}
	return packet[:n], cutevpn.ConvertNetAddr(udpAddr.IP, udpAddr.Port), nil
}

func (t *udp4) Overhead() int {
	return 20 + 8
}

func (t *udp4) Cancel() {
	t.cancel()
}

func (t *udp4) Done() <-chan struct{} {
	return t.ctx.Done()
}
