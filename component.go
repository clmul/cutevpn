package cutevpn

import (
	"net"
)

const RoutingProtocolNumber = 89

var links = make(map[string]LinkConstructor)
var sockets = make(map[string]SocketConstructor)
var ciphers = make(map[string]CipherConstructor)
var routings = make(map[string]RoutingConstructor)

type LinkConstructor func(listenAddr string) (Link, error)
type SocketConstructor func(vpn *VPN, cidr, gateway string, mtu int) (Socket, error)
type CipherConstructor func(secret string) (Cipher, error)
type RoutingConstructor func(vpn *VPN, ip IPv4) Routing

func RegisterLink(name string, constructor LinkConstructor) {
	links[name] = constructor
}
func RegisterSocket(name string, constructor SocketConstructor) {
	sockets[name] = constructor
}
func RegisterCipher(name string, constructor CipherConstructor) {
	ciphers[name] = constructor
}
func RegisterRouting(name string, constructor RoutingConstructor) {
	routings[name] = constructor
}

type LinkAddr interface{}

type emptyLinkAddr struct{}

func (addr emptyLinkAddr) String() string {
	return "any"
}

var EmptyLinkAddr emptyLinkAddr

type IPv4 [4]byte

var EmptyIPv4 IPv4

func (ip IPv4) String() string {
	return net.IP(ip[:]).String()
}
func (ip IPv4) MarshalText() (text []byte, err error) {
	return []byte(ip.String()), nil
}

type Link interface {
	Send(packet []byte, addr LinkAddr) error
	Recv(packet []byte) (p []byte, addr LinkAddr, err error)
	Close() error
	ParseAddr(addr string) (LinkAddr, error)
	Overhead() int
	ToString(dst LinkAddr) string
}

type Socket interface {
	Send(packet []byte)
	Recv(packet []byte) (n int)
	Close() error
}

type Cipher interface {
	Encrypt([]byte) []byte
	Decrypt([]byte) ([]byte, error)
	Overhead() int
}

type Routing interface {
	Dump() []byte
	PacketQueue() chan Packet
	AddIfce(Link, Route)
	Get(IPv4) (Route, error)
}
