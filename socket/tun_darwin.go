package socket

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
)

func (t tun) setIP(localCIDR, gateway string) error {
	ip, ipnet, err := net.ParseCIDR(localCIDR)
	if err != nil {
		return err
	}
	ip = ip.To4()
	if ip == nil {
		return errors.New("wrong local address")
	}
	cmd := exec.Command("ifconfig", t.ifce.Name(), ip.String(), gateway)
	log.Println(strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	cmd = exec.Command("route", "add", ipnet.String(), gateway)
	log.Println(strings.Join(cmd.Args, " "))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}

func (t tun) setMTU(mtu int) error {
	cmd := exec.Command("ifconfig", t.ifce.Name(), "mtu", fmt.Sprint(mtu))
	log.Println(strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}
