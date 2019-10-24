package message

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/clmul/cutevpn"
)

type IPv4 = cutevpn.IPv4

const (
	tHello           = 1
	tLinkStateUpdate = 4
	tLinkStateACK    = 5
)

type header struct {
	// The type
	t byte
	// Where does the packet come from?
	// Note that ospf packets won't be forwarded.
	Src IPv4
	// The boot timestamp of the cutevpn instance.
	// If a cutevpn instance restarts, BootTime is used to distinct it with the one before restarting.
	BootTime uint64
}

type Message interface {
	Marshal(buf []byte, src IPv4, boot uint64) []byte
}

type LinkStateUpdate struct {
	header
	Owner   IPv4
	Version uint64
	State   map[IPv4]uint64
	Name    string
}

func NewLinkStateUpdate(owner IPv4, name string, version uint64, state map[IPv4]uint64) LinkStateUpdate {
	lsu := LinkStateUpdate{
		header:  header{t: tLinkStateUpdate},
		Owner:   owner,
		Version: version,
		State:   state,
		Name:    name,
	}
	return lsu
}

type LinkStateACK struct {
	header
	Owner   IPv4
	Version uint64
}

func NewLinkStateACK(owner IPv4, version uint64) LinkStateACK {
	lsa := LinkStateACK{
		header:  header{t: tLinkStateACK},
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

func NewHello(time1, time2 uint64, forwarded uint8) Hello {
	h := Hello{
		header:    header{t: tHello},
		Time1:     time1,
		Time2:     time2,
		Forwarded: forwarded,
	}
	return h
}

func (h header) marshal(b []byte, src IPv4, boot uint64) []byte {
	h.Src = src
	h.BootTime = boot
	b = b[:0]
	b = append(b, h.t)
	b = append(b, h.Src[:]...)
	b = appendUint64(b, h.BootTime)
	return b
}

func (lsa LinkStateACK) Marshal(b []byte, src IPv4, boot uint64) []byte {
	b = lsa.header.marshal(b, src, boot)
	b = append(b, lsa.Owner[:]...)
	b = appendUint64(b, lsa.Version)
	return b
}

func (ls LinkStateUpdate) Marshal(b []byte, src IPv4, boot uint64) []byte {
	b = ls.header.marshal(b, src, boot)
	b = append(b, ls.Owner[:]...)
	b = appendUint64(b, ls.Version)
	b = appendUint16(b, uint16(len(ls.State)))
	for ip, metric := range ls.State {
		b = append(b, ip[:]...)
		b = appendUint64(b, metric)
	}
	b = appendString(b, ls.Name)
	return b
}

func (h Hello) Marshal(b []byte, src IPv4, boot uint64) []byte {
	b = h.header.marshal(b, src, boot)
	b = appendUint64(b, h.Time1)
	b = appendUint64(b, h.Time2)
	b = append(b, h.Forwarded)
	return b
}

func Unmarshal(p []byte) Message {
	var header header
	pos := 0
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
		state := make(map[IPv4]uint64)
		owner, pos := readIPv4(p, pos)
		version, pos := readUint64(p, pos)
		stateLen, pos := readUint16(p, pos)
		for i := 0; i < int(stateLen); i++ {
			var ip IPv4
			var metric uint64
			ip, pos = readIPv4(p, pos)
			metric, pos = readUint64(p, pos)
			state[ip] = metric
		}
		name, pos := readString(p, pos)
		lsu := LinkStateUpdate{
			header:  header,
			Owner:   owner,
			Version: version,
			State:   state,
			Name:    name,
		}
		return lsu
	case tLinkStateACK:
		var owner IPv4
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

func appendUint16(bs []byte, v uint16) []byte {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	return append(bs, b[:]...)
}

func appendString(bs []byte, v string) []byte {
	bs = append(bs, []byte(v)...)
	return append(bs, 0)
}

func readUint64(bs []byte, pos int) (uint64, int) {
	v := binary.LittleEndian.Uint64(bs[pos:])
	return v, pos + 8
}

func readUint16(bs []byte, pos int) (uint16, int) {
	v := binary.LittleEndian.Uint16(bs[pos:])
	return v, pos + 2
}

func readUint8(bs []byte, pos int) (uint8, int) {
	return bs[pos], pos + 1
}

func readIPv4(bs []byte, pos int) (IPv4, int) {
	var ip IPv4
	copy(ip[:], bs[pos:])
	return ip, pos + 4
}

func readString(bs []byte, pos int) (string, int) {
	for i := pos; i < len(bs); i++ {
		if bs[i] == 0 {
			return string(bs[pos:i]), i + 1
		}
	}
	panic(fmt.Sprintf("corrupt packet, %v at %v", bs, pos))
}
