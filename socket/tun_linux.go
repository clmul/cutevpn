package socket

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func (t tun) setIP(localCIDR, _ string) error {
	cmd := exec.Command("ip", "link", "set", t.ifce.Name(), "up")
	log.Println(strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	cmd = exec.Command("ip", "address", "add", localCIDR, "dev", t.ifce.Name())
	log.Println(strings.Join(cmd.Args, " "))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}

func (t tun) setMTU(mtu int) error {
	cmd := exec.Command("ip", "link", "set", t.ifce.Name(), "mtu", fmt.Sprint(mtu))
	log.Println(strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}
