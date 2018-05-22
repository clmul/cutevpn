package cutevpn

import (
	"context"
	"log"

	"github.com/clmul/checksum"
	"github.com/clmul/cutevpn/ipv4"
)

type conn struct {
	vpn   *VPN
	queue chan Packet
}

type Packet struct {
	Payload []byte
	Route   Route
	dst     IPv4
	through IPv4
	flags   uint8
}

const (
	flagRouting  = 0x10
	flagHopLimit = 0x0f
	flagDefault  = 0x0f
)

const TailSize = 9

func newConn(vpn *VPN, links []Link) *conn {
	c := &conn{
		vpn:   vpn,
		queue: make(chan Packet, 16),
	}
	cipherOverhead := vpn.linkCipher.Overhead()
	for _, link := range links {
		if link.Overhead() >= 0 {
			log.Printf("overhead of %v is %v", link.ToString(link.Peer()), link.Overhead()+cipherOverhead+TailSize)
		}
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
				Payload: packet,
				Route:   Route{link: link, addr: linkAddr},
				flags:   tail[0],
			}
			copy(p.dst[:], tail[1:])
			copy(p.through[:], tail[5:])
			c.queue <- p
			return nil
		})
	}

	return c
}

func (c *conn) Forward(self IPv4, route Route, pack Packet) {
	if pack.flags&flagHopLimit <= 1 {
		log.Printf("drop a packet because hop limit is 0, dst is %v", pack.dst)
		return
	}
	const ttlOffset = 8
	ttl := pack.Payload[ttlOffset]
	if ttl <= 1 {
		src := GetSrcIP(pack.Payload)
		reply := ipv4.TimeExceeded(self, src, pack.Payload)
		c.Send(pack.Route, src, EmptyIPv4, flagDefault, reply)
		return
	}
	checksum.UpdateByte(pack.Payload, ttlOffset, ttl-1)
	c.Send(route, pack.dst, pack.through, pack.flags-1, pack.Payload)
}

func (c *conn) Send(route Route, dst, through IPv4, flags uint8, packet []byte) {
	var tail [TailSize]byte
	tail[0] = flags
	copy(tail[1:], dst[:])
	copy(tail[5:], through[:])
	packet = append(packet, tail[:]...)

	packet = c.vpn.linkCipher.Encrypt(packet)
	err := route.link.Send(packet, route.addr)

	if err != nil {
		c.vpn.LinkSendErr(err)
	}
}
