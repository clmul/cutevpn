package cutevpn

import (
	"context"
	"log"
	"net"
)

type router struct {
	conn   *conn
	socket Socket

	ip      IPv4
	ipnet   *net.IPNet
	gateway IPv4

	socketQueue chan []byte
	routing     Routing
}

func newRouter(ip IPv4, ipnet *net.IPNet, gateway IPv4, conn *conn, routing Routing, socket Socket) (*router, error) {
	r := &router{
		conn:   conn,
		socket: socket,

		ip:      ip,
		ipnet:   ipnet,
		gateway: gateway,

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
		case pack := <-routingQ:
			r.sendRoutingMessage(pack)
		case pack := <-r.conn.queue:
			r.forwardFromConn(pack, r.routing)
		case packet := <-r.socketQueue:
			r.forwardFromSocket(packet)
		}
		return nil
	})
}

func (r *router) readSocket(ctx context.Context) error {
	packet := make([]byte, 2048)
	n := r.socket.Recv(packet)
	if n == 0 {
		return nil
	}
	r.socketQueue <- packet[:n]
	return nil
}

func (r *router) forwardFromConn(pack Packet, routing Routing) {
	if len(pack.Payload) == 0 {
		return
	}
	switch {
	case pack.flags&flagRouting != 0:
		routing.Inject(pack)
	case pack.dst == r.ip:
		r.socket.Send(pack.Payload)
	case r.ipnet.Contains(pack.dst[:]):
		var err error
		var route Route

		switch {
		case pack.through == r.ip:
			route, err = r.routing.GetAdja(pack.dst)
			if err != nil {
				// no route to host
				return
			}
		case pack.through == EmptyIPv4:
			route, pack.through, err = r.routing.GetBalance(pack.dst)
			if err != nil {
				return
			}
		default:
			route, err = r.routing.GetShortest(pack.through)
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

func (r *router) forwardFromSocket(packet []byte) {
	dst := GetDstIP(packet)
	if dst == r.ip {
		r.socket.Send(packet)
		return
	}
	if !r.ipnet.Contains(dst[:]) {
		if r.gateway == EmptyIPv4 {
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
	r.conn.Send(route, dst, through, flagDefault, packet)
}

func (r *router) sendRoutingMessage(pack Packet) {
	r.conn.Send(pack.Route, EmptyIPv4, EmptyIPv4, flagRouting|flagHopLimit, pack.Payload)
}

type Route struct {
	link Link
	addr LinkAddr
}

func (r Route) IsEmpty() bool {
	return r.link == nil && r.addr == nil
}

func (r Route) String() string {
	return r.link.ToString(r.addr)
}

func (r Route) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}
