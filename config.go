package cutevpn

import (
	"log"
	"time"
)

type Config struct {
	CIDR    string
	Gateway string
	MTU     uint32

	Cipher string
	Secret string

	Socket  string
	Routing string
	Links   []LinkConf
}

type LinkConf struct {
	Link   string
	Listen string
	Dial   string
}

func (c *LinkConf) get(vpn *VPN) (Link, error) {
	link, err := links[c.Link](vpn, c.Listen, c.Dial)
	if err != nil {
		return nil, err
	}
	vpn.Defer(func() {
		err := link.Close()
		if err != nil {
			log.Println(err)
		}
	})
	return link, nil
}

func (conf *Config) Start() (vpn *VPN, err error) {
	vpn = NewVPN()

	defer func() {
		if err != nil {
			vpn.Stop()
		}
	}()

	cipher, err := ciphers[conf.Cipher](conf.Secret)
	if err != nil {
		return vpn, err
	}
	vpn.linkCipher = cipher

	links := make([]Link, 0, len(conf.Links))
	for _, linkConf := range conf.Links {
		link, err := linkConf.get(vpn)
		if err != nil {
			return vpn, err
		}
		links = append(links, link)
	}
	time.Sleep(time.Second)

	socket, err := sockets[conf.Socket](vpn, conf.CIDR, conf.Gateway, conf.MTU)
	if err != nil {
		return vpn, err
	}

	vpn.Defer(func() {
		err := socket.Close()
		if err != nil {
			log.Println(err)
		}
	})

	conn := newConn(vpn, links)

	ip, ipnet, err := ParseCIDR(conf.CIDR)
	if err != nil {
		return vpn, err
	}

	var gateway IPv4
	if conf.Gateway != "" {
		gateway, err = ParseIPv4(conf.Gateway)
		if err != nil {
			return vpn, err
		}
	}

	routing := routings[conf.Routing](vpn, ip)
	vpn.router, err = newRouter(ip, ipnet, gateway, conn, routing, socket)
	if err != nil {
		return vpn, err
	}
	vpn.router.Start(vpn)
	for _, link := range links {
		if link.Peer() != nil {
			routing.AddIfce(link, Route{link: link, addr: link.Peer()})
		}
	}
	return vpn, nil
}
