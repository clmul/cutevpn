// +build darwin linux

package socket

import (
	"log"

	"github.com/clmul/cutevpn"
	"github.com/clmul/water"
)

type tun struct {
	ifce *water.Interface
	vpn  cutevpn.VPN
}

func openTun(vpn cutevpn.VPN, cidr string, mtu uint32) (cutevpn.Socket, error) {
	ifce, err := water.New(water.Config{})
	if err != nil {
		return nil, err
	}
	t := tun{
		ifce: ifce,
		vpn:  vpn,
	}
	err = t.setIP(cidr)
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
		select {
		case <-t.vpn.Done():
		default:
			log.Fatal(err)
		}
	}
}

func (t tun) Recv(packet []byte) int {
	n, err := t.ifce.Read(packet)
	if err != nil {
		select {
		case <-t.vpn.Done():
		default:
			log.Fatal(err)
		}
		return 0
	}
	if packet[0]>>4 != 4 {
		// a non-IPv4 packet
		return 0
	}
	return n
}
