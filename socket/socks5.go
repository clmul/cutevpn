package socket

import (
	"context"
	"log"
	"net"

	"github.com/clmul/cutevpn"
	"github.com/clmul/netstack"
	"github.com/clmul/socks5"
)

func init() {
	cutevpn.RegisterSocket("socks5", openSocks5)
}

type t struct {
	loop  cutevpn.Looper
	stack *netstack.Endpoint
}

func openSocks5(loop cutevpn.Looper, cidr, gateway string, mtu uint32) (cutevpn.Socket, error) {
	stack, err := netstack.New(cidr, mtu)
	if err != nil {
		return nil, err
	}
	t := &t{
		loop:  loop,
		stack: stack,
	}

	go func(addr string) {
		conf := &socks5.Config{
			Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return t.stack.Dial(ctx, network, addr)
			},
			Resolver: t.stack,
		}
		server, err := socks5.New(conf)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal(server.ListenAndServe("tcp", addr))
	}("localhost:1081")
	return t, nil
}

func (t *t) Close() error {
	return nil
}

func (t *t) Send(packet []byte) {
	t.stack.Inject(packet)
}

func (t *t) Recv(packet []byte) int {
	var p []byte
	select {
	case p = <-t.stack.C:
	case <-t.loop.Done():
		return 0
	}
	return copy(packet, p)
}
