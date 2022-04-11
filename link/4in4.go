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

type iviniv struct {
	*net.IPConn
	protocol int
	peer     cutevpn.LinkAddr
	ctx      context.Context
	cancel   context.CancelFunc
}

var singletons [256]*net.IPConn

func new4in4(vpn cutevpn.VPN, ctx context.Context, linkURL *url.URL) (cutevpn.Link, error) {
	ctx, cancel := context.WithCancel(ctx)
	link := &iviniv{
		ctx:    ctx,
		cancel: cancel,
	}
	var err error

	link.protocol, err = strconv.Atoi(linkURL.Query().Get("protocol"))
	if err != nil {
		return nil, err
	}
	if link.protocol < 1 || link.protocol > 255 {
		return nil, fmt.Errorf("wrong protocol number %v, must between 1 and 255", link.protocol)
	}

	if linkURL.Host != "" {
		link.peer, err = resolveIPv4(linkURL.Host)
		if err != nil {
			return nil, err
		}
	}

	c := singletons[link.protocol]
	if c == nil {
		conn, err := net.ListenPacket(fmt.Sprintf("ip4:%d", link.protocol), "")
		if err != nil {
			return nil, err
		}
		c = conn.(*net.IPConn)
		singletons[link.protocol] = c
		vpn.Defer(func() {
			err := conn.Close()
			if err != nil {
				log.Println(err)
			}
		})
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

func (t *iviniv) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("4in4 %v:%v->%v", t.IPConn.LocalAddr(), t.protocol, dst)
}

func resolveIPv4(addr string) (cutevpn.IPv4, error) {
	ipAddr, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		return emptyIPv4, err
	}
	return convertIPAddr(ipAddr), nil
}

func (t *iviniv) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t *iviniv) Send(packet []byte, addr cutevpn.LinkAddr) error {
	_, err := t.WriteToIP(packet, convertToIPAddr(addr.(cutevpn.IPv4)))
	return err
}

func (t *iviniv) Recv(packet []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	n, ipAddr, err := t.ReadFromIP(packet)
	if err != nil {
		return
	}
	p = packet[:n]
	addr = convertIPAddr(ipAddr)
	return
}

func (t *iviniv) Overhead() int {
	return 20
}

func (t *iviniv) Cancel() {
	t.cancel()
}

func (t *iviniv) Done() <-chan struct{} {
	return t.ctx.Done()
}

var emptyIPv4 cutevpn.IPv4
