package socket

import (
	"fmt"

	"github.com/clmul/cutevpn"
)

func New(name string, vpn cutevpn.VPN, cidr string, mtu uint32) (cutevpn.Socket, error) {
	switch name {
	case "tun":
		return openTun(vpn, cidr, mtu)
	//case "socks5":
	default:
		return nil, fmt.Errorf("unknown socket %s", name)
	}
}
