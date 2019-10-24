package cutevpn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
)

var links = make(map[string]LinkConstructor)
var sockets = make(map[string]SocketConstructor)
var ciphers = make(map[string]CipherConstructor)

type LinkConstructor func(vpn VPN, ctx context.Context, cancel context.CancelFunc, linkURL *url.URL) (Link, error)
type SocketConstructor func(vpn VPN, cidr, gateway string, mtu uint32) (Socket, error)
type CipherConstructor func(secret string) (Cipher, error)

func RegisterLink(name string, constructor LinkConstructor) {
	links[name] = constructor
}
func RegisterSocket(name string, constructor SocketConstructor) {
	sockets[name] = constructor
}
func RegisterCipher(name string, constructor CipherConstructor) {
	ciphers[name] = constructor
}

func GetLink(name string) (LinkConstructor, error) {
	link, ok := links[name]
	if !ok {
		return nil, fmt.Errorf("no link %v", name)
	}
	return link, nil
}

func GetSocket(name string) (SocketConstructor, error) {
	socket, ok := sockets[name]
	if !ok {
		return nil, fmt.Errorf("no socket %v", name)
	}
	return socket, nil
}

func GetCipher(name string) (CipherConstructor, error) {
	cipher, ok := ciphers[name]
	if !ok {
		return nil, fmt.Errorf("no cipher %v", name)
	}
	return cipher, nil
}

type Config struct {
	Name    string
	CIDR    string
	Gateway string
	MTU     uint32

	Cipher string
	Secret string

	Socket  string
	Routing string
	Links   []string
}

type VPN interface {
	Name() string
	// Call f in a new goroutine and wait it to return before exiting.
	Go(f func())
	Done() <-chan struct{}
	Loop(f func(context.Context) error)
	Defer(f func())
	OnCancel(ctx context.Context, f func())

	Cipher() Cipher
	AddLink(link Link)
}

type IPv4 [4]byte

func (ip IPv4) String() string {
	return net.IP(ip[:]).String()
}

func (ip IPv4) MarshalText() (text []byte, err error) {
	return []byte(ip.String()), nil
}

// LinkAddr is something like MAC address.
// It is used as map keys,
// so it must be comparable and equality means equality.
type LinkAddr interface{}

// Link is a connection between two peers.
// It acts like the link layer.
// Link is used as map keys.
type Link interface {
	// Send the packet through the Link via dst.
	// This method is called on the main event loop, so it must be non-blocking.
	Send(packet []byte, dst LinkAddr) error
	// Receive a packet from the Link.
	// buffer is a []byte whose length is 2048.
	// Returns the packet and the source address.
	// called on the link's own loop, can block
	Recv(buffer []byte) (p []byte, addr LinkAddr, err error)
	// If the Link is something like a TCP dialer, Peer should return
	// the remote address. If it is something like a TCP listener,
	// Peer should return nil.
	Peer() LinkAddr
	// The bytes which is used by the Link in IP packets of the host network.
	// This is for calculating mtu of tun interface.
	Overhead() int
	// A string used for debugging purposes.
	ToString(dst LinkAddr) string
	// This will be called when either Send or Recv returns a non-nil error.
	Cancel()
	Done() <-chan struct{}
}

// cutevpn interacts with the OS through Socket.
// It can be a tun interface or SOCKS5 server.
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

type Route struct {
	// The two fields are like 'ip route add 1.2.3.4 via addr dev link'
	Link Link
	Addr LinkAddr
}

func (r Route) IsEmpty() bool {
	return r.Link == nil && r.Addr == nil
}

func (r Route) String() string {
	return r.Link.ToString(r.Addr)
}

func (r Route) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

var ErrStopLoop = errors.New("stop")
var ErrNoRoute = errors.New("no route to host")

var ErrNoIPv6 = errors.New("IPv6 is not supported")
var ErrInvalidIP = errors.New("invalid IP address")
