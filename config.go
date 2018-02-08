package cutevpn

import (
	"context"
	"log"
	"sync"
)

type Config struct {
	CIDR    string
	Gateway string

	Cipher string
	Secret string

	Socket string

	Routing string

	MTU int

	Links []LinkConf
}

type LinkConf struct {
	Link   string `toml:"link"`
	Listen string `toml:"listen"`
	Dial   string `toml:"dial"`
}

func (c *LinkConf) get(vpn *VPN) (Link, LinkAddr, error) {
	link, err := links[c.Link](c.Listen)
	if err != nil {
		return nil, nil, err
	}
	vpn.Defer(func() {
		err := link.Close()
		if err != nil {
			log.Println(err)
		}
	})
	var peer LinkAddr = EmptyLinkAddr
	if c.Dial != "" {
		peer, err = link.ParseAddr(c.Dial)
		if err != nil {
			return nil, nil, err
		}
	}
	return link, peer, nil
}

func defaultConf(conf *Config) {
	if conf.Socket == "" {
		conf.Socket = "tun"
	}
	if conf.MTU == 0 {
		conf.MTU = 1400
	}
	if conf.Routing == "" {
		conf.Routing = "ospf"
	}
	for i := range conf.Links {
		if conf.Links[i].Link == "" {
			conf.Links[i].Link = "udp4"
		}
	}
}

func (conf *Config) Start() (vpn *VPN, err error) {
	defaultConf(conf)
	vpn = new(VPN)
	vpn.ctx, vpn.cancel = context.WithCancel(context.Background())
	vpn.wg = new(sync.WaitGroup)

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

	var links []Link
	var peers []LinkAddr
	for _, linkConf := range conf.Links {
		link, peer, err := linkConf.get(vpn)
		if err != nil {
			return vpn, err
		}
		links = append(links, link)
		peers = append(peers, peer)
	}

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

	conn := newConn(vpn, links, peers)

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
	for i, link := range links {
		peer := peers[i]
		if peer != EmptyLinkAddr {
			routing.AddIfce(link, Route{link: link, addr: peer})
		}
	}
	return vpn, nil
}
