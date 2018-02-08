package message

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/clmul/cutevpn"
)

var bootTime = uint64(time.Now().UnixNano())

const (
	tHello           = 1
	tLinkStateUpdate = 4
	tLinkStateACK    = 5
)

type Message interface {
	Marshal([]byte) []byte
}

type header struct {
	t        byte
	Src      cutevpn.IPv4
	BootTime uint64
}

type LinkStateUpdate struct {
	header
	Owner   cutevpn.IPv4
	Version uint64
	State   map[cutevpn.IPv4]uint64
}

func NewLinkStateUpdate(src cutevpn.IPv4, owner cutevpn.IPv4, version uint64, state map[cutevpn.IPv4]uint64) LinkStateUpdate {
	lsu := LinkStateUpdate{
		header:  header{t: tLinkStateUpdate, Src: src},
		Owner:   owner,
		Version: version,
		State:   state,
	}
	return lsu
}

type LinkStateACK struct {
	header
	Owner   cutevpn.IPv4
	Version uint64
}

func NewLinkStateACK(src, owner cutevpn.IPv4, version uint64) LinkStateACK {
	lsa := LinkStateACK{
		header:  header{t: tLinkStateACK, Src: src},
		Owner:   owner,
		Version: version,
	}
	return lsa

}

type Hello struct {
	header
	Time1, Time2 uint64
	Forwarded    uint8
}

func NewHello(src cutevpn.IPv4, time1, time2 uint64, forwarded uint8) Hello {
	h := Hello{
		header: header{
			Src: src,
			t:   tHello,
		},
		Time1:     time1,
		Time2:     time2,
		Forwarded: forwarded,
	}
	return h
}

func (h header) marshal(b []byte) []byte {
	b = b[:0]
	b = append(b, cutevpn.RoutingProtocolNumber)
	b = append(b, h.t)
	b = append(b, h.Src[:]...)
	b = appendUint64(b, bootTime)
	return b
}

func (lsa LinkStateACK) Marshal(b []byte) []byte {
	b = lsa.header.marshal(b)
	b = append(b, lsa.Owner[:]...)
	b = appendUint64(b, lsa.Version)
	return b
}

func (ls LinkStateUpdate) Marshal(b []byte) []byte {
	b = ls.header.marshal(b)
	b = append(b, ls.Owner[:]...)
	b = appendUint64(b, ls.Version)
	for ip, metric := range ls.State {
		b = append(b, ip[:]...)
		b = appendUint64(b, metric)
	}
	return b
}

func (h Hello) Marshal(b []byte) []byte {
	b = h.header.marshal(b)
	b = appendUint64(b, h.Time1)
	b = appendUint64(b, h.Time2)
	b = append(b, h.Forwarded)
	return b
}

func Unmarshal(p []byte) Message {
	if p[0] != cutevpn.RoutingProtocolNumber {
		log.Fatal("wrong protocol number")
	}
	var header header
	pos := 1
	header.t, pos = readUint8(p, pos)
	header.Src, pos = readIPv4(p, pos)
	header.BootTime, pos = readUint64(p, pos)
	switch header.t {
	case tHello:
		var hello Hello
		hello.header = header
		hello.Time1, pos = readUint64(p, pos)
		hello.Time2, pos = readUint64(p, pos)
		hello.Forwarded, pos = readUint8(p, pos)
		return hello
	case tLinkStateUpdate:
		linkStateUpdate := LinkStateUpdate{
			header:  header,
			Version: 0,
			State:   make(map[cutevpn.IPv4]uint64),
		}
		linkStateUpdate.Owner, pos = readIPv4(p, pos)
		linkStateUpdate.Version, pos = readUint64(p, pos)
		for pos < len(p) {
			var ip cutevpn.IPv4
			var metric uint64
			ip, pos = readIPv4(p, pos)
			metric, pos = readUint64(p, pos)
			linkStateUpdate.State[ip] = metric
		}
		return linkStateUpdate
	case tLinkStateACK:
		var owner cutevpn.IPv4
		var version uint64
		owner, pos = readIPv4(p, pos)
		version, pos = readUint64(p, pos)
		lsa := LinkStateACK{
			header:  header,
			Owner:   owner,
			Version: version,
		}
		return lsa
	default:
		log.Fatal("wrong packet type")
	}
	return nil
}

func appendUint64(bs []byte, v uint64) []byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	return append(bs, b[:]...)
}

func readUint64(bs []byte, pos int) (uint64, int) {
	v := binary.LittleEndian.Uint64(bs[pos:])
	return v, pos + 8
}

func readUint8(bs []byte, pos int) (uint8, int) {
	return bs[pos], pos + 1
}

func readIPv4(bs []byte, pos int) (cutevpn.IPv4, int) {
	var ip cutevpn.IPv4
	copy(ip[:], bs[pos:])
	return ip, pos + 4
}
