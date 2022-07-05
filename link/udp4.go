package link

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/clmul/cutevpn"
)

type udp struct {
	cipher cutevpn.Cipher
	peer   cutevpn.LinkAddr
	conn   *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
}

func newUDP(vpn cutevpn.VPN, linkURL *url.URL, cipher cutevpn.Cipher) error {
	ctx, cancel := context.WithCancel(vpn.Context())
	t := &udp{
		cipher: cipher,
		ctx:    ctx,
		cancel: cancel,
	}

	var listen string
	if linkURL.Hostname() == "" {
		listen = linkURL.Host
	} else {
		addr, err := net.ResolveUDPAddr("udp", linkURL.Host)
		if err != nil {
			return err
		}
		t.peer = cutevpn.ConvertNetAddr(addr.IP, addr.Port)
	}
	c, err := net.ListenPacket("udp", listen)
	if err != nil {
		return err
	}
	vpn.OnCancel(ctx, func() {
		err := c.Close()
		if err != nil {
			log.Println(err)
		}
	})
	t.conn = c.(*net.UDPConn)
	vpn.AddLink(t)

	return nil
}

func (t *udp) ToString(dst cutevpn.LinkAddr) string {
	if dst == nil {
		dst = "any"
	}
	return fmt.Sprintf("udp %v->%v", t.conn.LocalAddr(), dst)
}

func (t *udp) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t *udp) Send(packet []byte, addr cutevpn.LinkAddr) error {
	ip, port := cutevpn.ConvertToNetAddr(addr.(cutevpn.AddrPort))
	_, err := t.conn.WriteToUDP(t.cipher.Encrypt(packet), &net.UDPAddr{IP: ip, Port: port})
	return err
}

func (t *udp) Recv(packet []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	n, udpAddr, err := t.conn.ReadFromUDP(packet)
	if err != nil {
		return nil, nil, err
	}
	packet = packet[:n]
	packet, err = t.cipher.Decrypt(packet)
	if err != nil {
		log.Println(err)
		return packet[:0], nil, nil
	}
	return packet, cutevpn.ConvertNetAddr(udpAddr.IP, udpAddr.Port), nil
}

func (t *udp) Overhead() int {
	return 20 + 8 + t.cipher.Overhead()
}

func (t *udp) Cancel() {
	t.cancel()
}

func (t *udp) Done() <-chan struct{} {
	return t.ctx.Done()
}
