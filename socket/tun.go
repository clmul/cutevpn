// +build darwin linux

package socket

import (
	"github.com/clmul/cutevpn"
	"github.com/songgao/water"
)

func init() {
	cutevpn.RegisterSocket("tun", openTun)
}

type tun struct {
	ifce *water.Interface
	vpn  *cutevpn.VPN
}

func openTun(vpn *cutevpn.VPN, cidr, gateway string, mtu int) (cutevpn.Socket, error) {
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return nil, err
	}
	t := tun{
		ifce: ifce,
		vpn:  vpn,
	}
	err = t.setIP(cidr, gateway)
	if err != nil {
		return nil, err
	}
	err = t.setMTU(mtu)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t tun) Close() error {
	return t.ifce.Close()
}

func (t tun) Send(packet []byte) {
	_, err := t.ifce.Write(packet)
	if err != nil {
		t.vpn.SocketErr(err)
	}
}

func (t tun) Recv(packet []byte) int {
	n, err := t.ifce.Read(packet)
	if err != nil {
		t.vpn.SocketErr(err)
		return 0
	}
	if packet[0]>>4 != 4 {
		// a non-IPv4 packet
		return 0
	}
	return n
}
