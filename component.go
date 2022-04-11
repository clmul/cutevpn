package cutevpn

import (
	"context"
	"errors"
	"net"
)

type Config struct {
	Name    string
	CIDR    string
	MTU     uint32
	Gateway string
	Routes  []string

	CACert string
	Cert   string
	Key    string

	Socket string
	Links  []string

	DefaultRoute bool
}

type VPN interface {
	Name() string
	// Call f in a new goroutine and wait it to return before exiting.
	Go(f func())
	Done() <-chan struct{}
	Loop(f func(context.Context) error)
	Defer(f func())
	OnCancel(ctx context.Context, f func())

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
