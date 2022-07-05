package socket

import (
	"context"
	"fmt"
	"log"
	"net"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/dnsresolver"
	"github.com/clmul/socks5"
)

var nicID tcpip.NICID = 1

type netstack struct {
	ep *channel.Endpoint
	s  *stack.Stack
}

type socks5proxy struct {
	vpn    cutevpn.VPN
	server *socks5.Server
	s      netstack
}

type stackErr struct {
	e tcpip.Error
}

func (e stackErr) Error() string {
	return e.e.String()
}

func newStack(cidr string, mtu uint32) (*netstack, error) {
	s := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
	})

	ep := channel.New(16, mtu, "")

	e := s.CreateNIC(nicID, ep)
	if e != nil {
		return nil, stackErr{e}
	}

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("can't parse cidr %v, %v", cidr, err)
	}
	ip = ip.To4()
	if ip == nil {
		return nil, cutevpn.ErrNoIPv6
	}
	prefix, _ := ipnet.Mask.Size()

	e = s.AddProtocolAddress(nicID,
		tcpip.ProtocolAddress{
			Protocol: ipv4.ProtocolNumber,
			AddressWithPrefix: tcpip.AddressWithPrefix{
				Address:   tcpip.Address(ip),
				PrefixLen: prefix,
			},
		},
		stack.AddressProperties{})
	if e != nil {
		return nil, stackErr{e}
	}
	subnet, err := tcpip.NewSubnet(tcpip.Address([]byte{0, 0, 0, 0}), tcpip.AddressMask([]byte{0, 0, 0, 0}))
	if err != nil {
		return nil, err
	}
	s.SetRouteTable([]tcpip.Route{
		{
			Destination: subnet,
			Gateway:     tcpip.Address([]byte{0, 0, 0, 0}),
			NIC:         nicID,
		},
	})

	return &netstack{ep: ep, s: s}, nil
}

func resolveFullAddress(address string) (*tcpip.FullAddress, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	if err != nil {
		return nil, err
	}
	return &tcpip.FullAddress{
		NIC:  nicID,
		Addr: tcpip.Address(tcpAddr.IP),
		Port: uint16(tcpAddr.Port),
	}, nil
}

func (s *netstack) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	addr, err := resolveFullAddress(address)
	if err != nil {
		return nil, err
	}

	switch network {
	case "tcp", "tcp4":
		return gonet.DialContextTCP(ctx, s.s, *addr, ipv4.ProtocolNumber)
	case "udp", "udp4":
		return gonet.DialUDP(s.s, nil, addr, ipv4.ProtocolNumber)
	default:
		return nil, fmt.Errorf("network %v is not supported", network)
	}
}

func (s *netstack) Listen(ctx context.Context, network, address string) (net.Listener, error) {
	addr, err := resolveFullAddress(address)
	if err != nil {
		return nil, err
	}
	switch network {
	case "tcp", "tcp4":
		return gonet.ListenTCP(s.s, *addr, ipv4.ProtocolNumber)
	default:
		return nil, fmt.Errorf("network %v is not supported", network)
	}
}

func (s *netstack) ListenPacket(ctx context.Context, network, address string) (net.PacketConn, error) {
	addr, err := resolveFullAddress(address)
	if err != nil {
		return nil, err
	}
	switch network {
	case "udp", "udp4":
		return gonet.DialUDP(s.s, addr, nil, ipv4.ProtocolNumber)
	default:
		return nil, fmt.Errorf("network %v is not supported", network)
	}
}

func openSocks5(vpn cutevpn.VPN, cidr string, mtu uint32) (cutevpn.Socket, error) {
	s, err := newStack(cidr, mtu)
	if err != nil {
		return nil, err
	}
	resolver, err := dnsresolver.New("8.8.8.8:53", s.Dial)
	if err != nil {
		return nil, err
	}
	conf := &socks5.Config{
		Dial:     s.Dial,
		Resolver: resolver,
	}
	server, err := socks5.New(conf)
	if err != nil {
		return nil, err
	}
	t := &socks5proxy{
		vpn:    vpn,
		server: server,
		s:      *s,
	}
	listen := "localhost:1080"

	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Fatal(t.server.ListenAndServe("tcp", listen))
	}()
	return t, nil
}

func (t *socks5proxy) Close() error {
	// TODO t.server.Close()
	return nil
}

func (t *socks5proxy) Send(packet []byte) {
	packetBuffer := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.NewWithData(packet),
	})
	t.s.ep.InjectInbound(ipv4.ProtocolNumber, packetBuffer)
}

func (t *socks5proxy) Recv(packet []byte) int {
	packetBuffer := t.s.ep.ReadContext(t.vpn.Context())
	if packetBuffer == nil {
		return 0
	}
	n := 0
	for _, buf := range packetBuffer.Slices() {
		n += copy(packet[n:], buf)
	}
	return n
}
