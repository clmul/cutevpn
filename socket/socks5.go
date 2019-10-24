package socket

import (
	"context"
	"log"
	"net"

	"github.com/clmul/cutevpn"
	"github.com/clmul/socks5"
	"github.com/google/netstack"
)

func init() {
	cutevpn.RegisterSocket("socks5", openSocks5)
}

type t struct {
	vpn    cutevpn.VPN
	stack  *netstack.Endpoint
	server *socks5.Server
}

func openSocks5(vpn cutevpn.VPN, cidr, gateway string, mtu uint32) (cutevpn.Socket, error) {
	stack, err := netstack.New(cidr, mtu)
	if err != nil {
		return nil, err
	}
	t := &t{
		vpn:   vpn,
		stack: stack,
	}
	listen := "localhost:1080"
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return t.stack.Dial(ctx, network, addr)
		},
		Resolver: t.stack,
	}
	t.server, err = socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Fatal(t.server.ListenAndServe("tcp", listen))
	}()
	return t, nil
}

func (t *t) Close() error {
	// TODO t.server.Close()
	return nil
}

func (t *t) Send(packet []byte) {
	t.stack.Inject(packet)
}

func (t *t) Recv(packet []byte) int {
	var p []byte
	select {
	case p = <-t.stack.C:
	case <-t.vpn.Done():
		return 0
	}
	return copy(packet, p)
}
