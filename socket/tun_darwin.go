package socket

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
)

func (t tun) setIP(localCIDR string) error {
	ip, ipnet, err := net.ParseCIDR(localCIDR)
	if err != nil {
		return err
	}
	ip = ip.To4()
	if ip == nil {
		return errors.New("wrong local address")
	}
	fakePeer := make(net.IP, 4)
	copy(fakePeer, ip)
	fakePeer[3]++
	if fakePeer[3] == 0 {
		fakePeer[3]++
	}
	cmd := exec.Command("ifconfig", t.ifce.Name(), ip.String(), fakePeer.String())
	log.Println(strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	cmd = exec.Command("route", "add", ipnet.String(), fakePeer.String())
	log.Println(strings.Join(cmd.Args, " "))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
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
