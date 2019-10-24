package socket

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/clmul/cutevpn"
)

func (t tun) setIP(localCIDR string) error {
	ip, ipnet, err := net.ParseCIDR(localCIDR)
	if err != nil {
		return err
	}
	ip = ip.To4()
	if ip == nil {
		return cutevpn.ErrNoIPv6
	}
	fakePeer := make(net.IP, 4)
	copy(fakePeer, ip)
	// On macOS, tun is a point-to-point link.
	// Any two addresses can be used to configure it.
	fakePeer[3]++
	if fakePeer.Equal(ipnet.IP) {
		// if cidr is 172.16.22.255/23, fakePeer will be 172.16.22.0 here.
		// The following command `route add 172.16.22.0/23 172.16.22.0` will fail.
		fakePeer[3]++
	}
	cmd := exec.Command("ifconfig", t.ifce.Name(), ip.String(), fakePeer.String())
	log.Println(strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	// Simulate a normal subnet on the point-to-point link.
	cmd = exec.Command("route", "add", ipnet.String(), fakePeer.String())
	log.Println(strings.Join(cmd.Args, " "))
	// `route` on macOS always returns 0, so the err is always nil.
	// Just let it print to stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (t tun) setMTU(mtu uint32) error {
	cmd := exec.Command("ifconfig", t.ifce.Name(), "mtu", fmt.Sprint(mtu))
	log.Println(strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}
