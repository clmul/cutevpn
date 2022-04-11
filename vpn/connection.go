package vpn

import (
	"context"
	"fmt"
	"log"

	"github.com/clmul/checksum"
	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/ipv4"
)

type conn struct {
	vpn *VPN
	// incoming packet queue
	queue chan packet
}

const (
	flagRouting  = 0x10
	flagHopLimit = 0x0f
	flagDefault  = 0x0f
)

const tailSize = 9

type packet struct {
	// For ingress Packets, the packet is coming from the route
	// For egress Packets, the packet will be sent through the route
	route cutevpn.Route
	// flags, dst and via are header fields.
	flags uint8
	// The final destination
	dst cutevpn.IPv4
	// The peer before the final destination
	via cutevpn.IPv4

	payload []byte
}

func newConn(vpn *VPN) *conn {
	c := &conn{
		vpn:   vpn,
		queue: make(chan packet, 16),
	}
	return c
}

func (c *conn) AddLink(link cutevpn.Link) {
	msg := fmt.Sprintf("link %v connected", link.ToString(link.Peer()))
	if link.Overhead() >= 0 {
		overhead := link.Overhead() + tailSize
		msg += fmt.Sprintf(", overhead is %v", overhead)
	}
	log.Println(msg)
	c.vpn.Loop(func(ctx context.Context) error {
		payload := make([]byte, 2048)
		payload, linkAddr, err := link.Recv(payload)
		if err != nil {
			log.Println(err)
			link.Cancel()
			return cutevpn.ErrStopLoop
		}
		if len(payload) == 0 {
			return nil
		}

		payload, tail := payload[:len(payload)-tailSize], payload[len(payload)-tailSize:]
		p := packet{
			payload: payload,
			route:   cutevpn.Route{Link: link, Addr: linkAddr},
			flags:   tail[0],
		}
		copy(p.dst[:], tail[1:])
		copy(p.via[:], tail[5:])
		c.queue <- p
		return nil
	})
}

func (c *conn) Forward(self cutevpn.IPv4, route cutevpn.Route, pack packet) {
	if pack.flags&flagHopLimit <= 1 {
		log.Printf("drop a packet because hop limit is 0, dst is %v", pack.dst)
		return
	}
	const ttlOffset = 8
	ttl := pack.payload[ttlOffset]
	if ttl <= 1 {
		src := cutevpn.GetSrcIP(pack.payload)
		reply := ipv4.TimeExceeded(self, src, pack.payload)
		c.Send(packet{route: pack.route, flags: flagDefault, dst: src, via: emptyIPv4, payload: reply})
		return
	}
	checksum.UpdateByte(pack.payload, ttlOffset, ttl-1)
	pack.route = route
	pack.flags--
	c.Send(pack)
}

func (c *conn) Send(p packet) {
	var tail [tailSize]byte
	tail[0] = p.flags
	copy(tail[1:], p.dst[:])
	copy(tail[5:], p.via[:])
	payload := append(p.payload, tail[:]...)

	route := p.route
	err := route.Link.Send(payload, route.Addr)
	if err != nil {
		log.Println(err)
		route.Link.Cancel()
	}
}
