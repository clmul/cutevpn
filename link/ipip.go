package link

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/clmul/cutevpn"
)

type ipip struct {
	*net.IPConn
	cipher cutevpn.Cipher
	peer   cutevpn.LinkAddr
	ctx    context.Context
	cancel context.CancelFunc
}

var singleton *net.IPConn

func newIPIP(vpn cutevpn.VPN, linkURL *url.URL, cipher cutevpn.Cipher) error {
	ctx, cancel := context.WithCancel(vpn.Context())
	link := &ipip{
		ctx:    ctx,
		cancel: cancel,
		cipher: cipher,
	}
	var err error

	if linkURL.Hostname() != "" {
		link.peer, err = resolveIPv4(linkURL.Hostname())
		if err != nil {
			return err
		}
	}

	if singleton == nil {
		conn, err := net.ListenPacket("ip4:4", "")
		if err != nil {
			return err
		}
		singleton = conn.(*net.IPConn)
		vpn.OnCancel(vpn.Context(), func() {
			err := conn.Close()
			if err != nil {
				log.Println(err)
			}
		})
	}
	link.IPConn = singleton
	vpn.AddLink(link)
	return nil
}

func convertIPAddr(addr *net.IPAddr) (a cutevpn.IPv4) {
	copy(a[:], addr.IP.To4())
	return a
}

func convertToIPAddr(a cutevpn.IPv4) *net.IPAddr {
	return &net.IPAddr{IP: a[:]}
}

func (t *ipip) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("ipip %v->%v", t.IPConn.LocalAddr(), dst)
}

func resolveIPv4(addr string) (cutevpn.IPv4, error) {
	ipAddr, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		return emptyIPv4, err
	}
	return convertIPAddr(ipAddr), nil
}

func (t *ipip) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t *ipip) Send(packet []byte, addr cutevpn.LinkAddr) error {
	packet = t.cipher.Encrypt(packet)
	_, err := t.WriteToIP(packet, convertToIPAddr(addr.(cutevpn.IPv4)))
	return err
}

func (t *ipip) Recv(packet []byte) ([]byte, cutevpn.LinkAddr, error) {
	n, ipAddr, err := t.ReadFromIP(packet)
	if err != nil {
		return nil, nil, err
	}
	packet = packet[:n]
	packet, err = t.cipher.Decrypt(packet)
	if err != nil {
		log.Println(err)
		return packet[:0], nil, nil
	}
	return packet, convertIPAddr(ipAddr), nil
}

func (t *ipip) Overhead() int {
	return 20 + t.cipher.Overhead()
}

func (t *ipip) Cancel() {
	t.cancel()
}

func (t *ipip) Done() <-chan struct{} {
	return t.ctx.Done()
}

var emptyIPv4 cutevpn.IPv4
