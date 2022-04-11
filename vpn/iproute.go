package vpn

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func run(cmds []string) error {
	for _, cmd := range cmds {
		log.Println(cmd)
		output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
		if err != nil {
			log.Println(string(output))
			return err
		}
	}
	return nil
}

func (v *VPN) addDefaultRoute(gateway string) error {
	pid := os.Getpid()
	up := []string{
		"mkdir -p /sys/fs/cgroup/net_cls/cutevpn",
		"echo 0x20202021 > /sys/fs/cgroup/net_cls/cutevpn/net_cls.classid",
		fmt.Sprintf("echo %v > /sys/fs/cgroup/net_cls/cutevpn/cgroup.procs", pid),

		"iptables -t mangle -A OUTPUT -m cgroup --cgroup 0x20202021 -j MARK --set-mark 2020",
		"iptables -t nat -A POSTROUTING -m mark --mark 2020 -j MASQUERADE",

		"ip rule add table 19088",
		"ip rule add table main suppress_prefixlength 0",
		"ip rule add fwmark 2020 table main",

		fmt.Sprintf("ip route add default via %s table 19088", gateway),
		"mv /etc/resolv.conf /etc/resolve.conf.orig",
		"echo 'nameserver 1.1.1.1\nnameserver 8.8.8.8' > /etc/resolv.conf",
	}
	down := []string{
		"ip rule delete table 19088",
		"ip rule delete table main suppress_prefixlength 0",
		"ip rule delete fwmark 2020 table main",

		"iptables -t mangle -D OUTPUT -m cgroup --cgroup 0x20202021 -j MARK --set-mark 2020",
		"iptables -t nat -D POSTROUTING -m mark --mark 2020 -j MASQUERADE",

		"for pid in $(cat /sys/fs/cgroup/net_cls/cutevpn/cgroup.procs); do echo $pid > /sys/fs/cgroup/net_cls/cgroup.procs; done",
		"rmdir /sys/fs/cgroup/net_cls/cutevpn",
		"mv /etc/resolv.conf.orig /etc/resolv.conf",
	}
	v.Defer(func() {
		run(down)
	})
	return run(up)
}
