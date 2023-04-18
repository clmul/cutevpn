package link

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"

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
		ips, err := net.LookupIP(linkURL.Hostname())
		if err != nil {
			return err
		}
		if len(ips) == 0 {
			return fmt.Errorf("known host %v", linkURL.Hostname())
		}
		addr := ips[0]
		port, err := strconv.Atoi(linkURL.Port())
		if err != nil {
			return fmt.Errorf("%v is not a valid port", linkURL.Port())
		}
		t.peer = convertNetAddr(addr, port)
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
	ip, port := convertToNetAddr(addr.(AddrPort))
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
	return packet, convertNetAddr(udpAddr.IP, udpAddr.Port), nil
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

type AddrPort struct {
	IP   [16]byte
	Port int
}

func (ap AddrPort) String() string {
	return fmt.Sprintf("%v:%v", net.IP(ap.IP[:]), ap.Port)
}

func convertNetAddr(ip net.IP, port int) AddrPort {
	r := AddrPort{}
	copy(r.IP[:], ip.To16())
	r.Port = port
	return r
}

func convertToNetAddr(ap AddrPort) (ip net.IP, port int) {
	return ap.IP[:], ap.Port
}
