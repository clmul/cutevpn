package cutevpn

import (
	"context"
	"log"
)

type conn struct {
	vpn             *VPN
	defaultHopLimit uint8
	queue           chan Packet
}

type Packet struct {
	Payload  []byte
	Route    Route
	dst      IPv4
	hopLimit uint8
}

const TailSize = 5

func newConn(vpn *VPN, links []Link, peers []LinkAddr) *conn {
	c := &conn{
		vpn:             vpn,
		defaultHopLimit: 8,
		queue:           make(chan Packet, 16),
	}
	cipherOverhead := vpn.linkCipher.Overhead()
	for i, link := range links {
		log.Printf("overhead of %v is %v", link.ToString(peers[i]), link.Overhead()+cipherOverhead+TailSize)
		link := link
		vpn.Loop(func(ctx context.Context) error {
			packet := make([]byte, 2048)
			packet, linkAddr, err := link.Recv(packet)
			if err != nil {
				vpn.LinkRecvErr(err)
				return nil
			}

			packet, err = vpn.linkCipher.Decrypt(packet)
			if err != nil {
				vpn.CipherErr(err)
				return nil
			}

			if err != nil {
				log.Println("dropped a packet, err is", err)
				return nil
			}

			packet, tail := packet[:len(packet)-TailSize], packet[len(packet)-TailSize:]
			p := Packet{
				Payload:  packet,
				Route:    Route{link: link, addr: linkAddr},
				hopLimit: tail[4],
			}
			copy(p.dst[:], tail[:4])
			c.queue <- p
			return nil
		})
	}

	return c
}

func (c *conn) Forward(route Route, pack Packet) {
	if pack.hopLimit <= 1 {
		log.Printf("drop a packet because hop limit is 0, dst is %v", pack.dst)
		return
	}
	c.send(route, pack.dst, pack.hopLimit-1, pack.Payload)
}

func (c *conn) Send(route Route, dst IPv4, packet []byte) {
	c.send(route, dst, c.defaultHopLimit, packet)
}

func (c *conn) send(route Route, dst IPv4, hopLimit uint8, packet []byte) {
	var tail [TailSize]byte
	copy(tail[:], dst[:])
	tail[4] = hopLimit
	packet = append(packet, tail[:]...)

	packet = c.vpn.linkCipher.Encrypt(packet)
	err := route.link.Send(packet, route.addr)

	if err != nil {
		c.vpn.LinkSendErr(err)
	}
}
