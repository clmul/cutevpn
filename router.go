package cutevpn

import (
	"context"
	"log"
	"net"
	"runtime"
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
	vpn.Loop(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case pack := <-r.conn.queue:
			r.forwardFromConn(pack, r.routing.PacketQueue())
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

func (r *router) forwardFromConn(pack Packet, routingMsgQ chan Packet) {
	dst, packet := pack.dst, pack.Payload
	if len(packet) == 0 {
		return
	}

	switch {
	case packet[0] == RoutingProtocolNumber:
		routingMsgQ <- pack
	case dst == r.ip:
		r.socket.Send(packet)
	case r.ipnet.Contains(dst[:]):
		route, err := r.routing.Get(dst)
		if err != nil {
			// no route to host
			return
		}
		r.conn.Forward(route, pack)
	default:
		log.Printf("dropped a packet whose dst (%v) is out of subnet", dst)
	}
	return
}

func (r *router) forwardFromSocket(packet []byte) {
	dst := GetDstIP(packet)
	if dst == r.ip {
		if runtime.GOOS == "darwin" {
			r.socket.Send(packet)
		} else {
			log.Println("dropped a packet from self to self")
		}
		return
	}
	if !r.ipnet.Contains(dst[:]) {
		if r.gateway == EmptyIPv4 {
			log.Printf("dropped a packet, dst is %v, gateway is empty", dst)
			return
		}
		dst = r.gateway
	}
	r.SendTo(dst, packet)
}

func (r *router) SendTo(dst IPv4, packet []byte) {
	route, err := r.routing.Get(dst)
	if err != nil {
		// no route to host
		return
	}
	r.conn.Send(route, dst, packet)
}

func (r *router) SendThrough(route Route, packet []byte) {
	r.conn.Send(route, EmptyIPv4, packet)
}

type Route struct {
	link Link
	addr LinkAddr
}

func (r Route) String() string {
	return r.link.ToString(r.addr)
}

func (r Route) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}
