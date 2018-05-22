package cutevpn

import (
	"context"
	"net"
)

var links = make(map[string]LinkConstructor)
var sockets = make(map[string]SocketConstructor)
var ciphers = make(map[string]CipherConstructor)
var routings = make(map[string]RoutingConstructor)

type LinkConstructor func(loop Looper, listenAddr, dialAddr string) (Link, error)
type SocketConstructor func(loop Looper, cidr, gateway string, mtu uint32) (Socket, error)
type CipherConstructor func(secret string) (Cipher, error)
type RoutingConstructor func(loop Looper, ip IPv4) Routing

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

type Looper interface {
	Defer(func())
	Loop(func(context.Context) error)
	Done() <-chan struct{}
}

type LinkAddr interface{}

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
	Peer() LinkAddr
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
	Inject(Packet)
	SendQueue() chan Packet
	AddIfce(Link, Route)
	GetBalance(dst IPv4) (Route, IPv4, error)
	GetAdja(adja IPv4) (Route, error)
	GetShortest(through IPv4) (Route, error)
}
