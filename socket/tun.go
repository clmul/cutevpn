// +build darwin linux

package socket

import (
	"log"

	"github.com/clmul/cutevpn"
	"github.com/clmul/water"
)

func init() {
	cutevpn.RegisterSocket("tun", openTun)
}

type tun struct {
	ifce *water.Interface
	loop cutevpn.Looper
}

func openTun(loop cutevpn.Looper, cidr, gateway string, mtu uint32) (cutevpn.Socket, error) {
	ifce, err := water.New(water.Config{})
	if err != nil {
		return nil, err
	}
	t := tun{
		ifce: ifce,
		loop: loop,
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
		case <-t.loop.Done():
		default:
			log.Fatal(err)
		}
	}
}

func (t tun) Recv(packet []byte) int {
	n, err := t.ifce.Read(packet)
	if err != nil {
		if err != nil {
			select {
			case <-t.loop.Done():
			default:
				log.Fatal(err)
			}
		}
		return 0
	}
	if packet[0]>>4 != 4 {
		// a non-IPv4 packet
		return 0
	}
	return n
}
