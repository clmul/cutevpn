package vpn

import (
	"context"
	"log"
	"net"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/ospf"
)

type router struct {
	conn   *conn
	socket cutevpn.Socket

	ip      cutevpn.IPv4
	ipnet   *net.IPNet
	gateway cutevpn.IPv4

	gatewayUpdateCh chan string

	socketQueue chan []byte
	routing     *ospf.OSPF
}

func newRouter(ip cutevpn.IPv4, ipnet *net.IPNet, gateway cutevpn.IPv4, conn *conn, routing *ospf.OSPF, socket cutevpn.Socket) (*router, error) {
	r := &router{
		conn:   conn,
		socket: socket,

		ip:      ip,
		ipnet:   ipnet,
		gateway: gateway,

		gatewayUpdateCh: make(chan string, 1),

		socketQueue: make(chan []byte, 16),
		routing:     routing,
	}
	return r, nil
}

func (r *router) Start(vpn *VPN) {
	vpn.Loop(r.readSocket)
	routingQ := r.routing.SendQueue()
	vpn.Loop(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case newGateway := <-r.gatewayUpdateCh:
			gatewayIP, err := cutevpn.ParseIPv4(newGateway)
			if err != nil {
				log.Fatalf("wrong gateway, %v", newGateway)
			}
			r.gateway = gatewayIP
		case p := <-routingQ:
			r.conn.Send(packet{
				route:   p.Route,
				flags:   flagRouting | flagHopLimit,
				dst:     emptyIPv4,
				via:     emptyIPv4,
				payload: p.Payload,
			})
		case pack := <-r.conn.queue:
			r.forwardFromConn(pack, r.routing)
		case payload := <-r.socketQueue:
			r.forwardFromSocket(payload)
		}
		return nil
	})
}

func (r *router) readSocket(ctx context.Context) error {
	payload := make([]byte, 2048)
	n := r.socket.Recv(payload)
	if n == 0 {
		return nil
	}
	r.socketQueue <- payload[:n]
	return nil
}

func (r *router) forwardFromConn(pack packet, routing *ospf.OSPF) {
	if len(pack.payload) == 0 {
		return
	}
	switch {
	case pack.flags&flagRouting != 0:
		routing.Inject(ospf.Packet{Route: pack.route, Payload: pack.payload})
	case pack.dst == r.ip:
		r.socket.Send(pack.payload)
	case r.ipnet.Contains(pack.dst[:]):
		var err error
		var route cutevpn.Route

		switch {
		case pack.via == r.ip:
			route, err = r.routing.GetAdja(pack.dst)
			if err != nil {
				// no route to host
				return
			}
		case pack.via == emptyIPv4:
			route, pack.via, err = r.routing.GetBalance(pack.dst)
			if err != nil {
				return
			}
		default:
			route, err = r.routing.GetShortest(pack.via)
			if err != nil {
				return
			}
		}

		r.conn.Forward(r.ip, route, pack)
	default:
		log.Printf("dropped a packet whose dst %v is out of subnet", pack.dst)
	}
	return
}

func (r *router) forwardFromSocket(payload []byte) {
	dst := cutevpn.GetDstIP(payload)
	if dst == r.ip {
		r.socket.Send(payload)
		return
	}
	if !r.ipnet.Contains(dst[:]) {
		if r.gateway == emptyIPv4 {
			log.Printf("dropped a packet, dst is %v, gateway is empty", dst)
			return
		}
		dst = r.gateway
	}
	route, through, err := r.routing.GetBalance(dst)
	if err != nil {
		// no route to host
		return
	}
	r.conn.Send(packet{route: route, flags: flagDefault, dst: dst, via: through, payload: payload})
}

var emptyIPv4 cutevpn.IPv4
