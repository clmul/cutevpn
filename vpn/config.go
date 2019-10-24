package vpn

import (
	"context"
	"log"
	"net/url"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/ospf"
)

func Start(conf *cutevpn.Config) (vpn *VPN, err error) {
	vpn = NewVPN(conf.Name)

	defer func() {
		if err != nil {
			vpn.Stop()
		}
	}()

	Cipher, err := cutevpn.GetCipher(conf.Cipher)
	if err != nil {
		return vpn, err
	}
	cipher, err := Cipher(conf.Secret)
	if err != nil {
		return vpn, err
	}
	vpn.cipher = cipher

	Socket, err := cutevpn.GetSocket(conf.Socket)
	if err != nil {
		return vpn, err
	}
	socket, err := Socket(vpn, conf.CIDR, conf.Gateway, conf.MTU)
	if err != nil {
		return vpn, err
	}

	vpn.Defer(func() {
		err := socket.Close()
		if err != nil {
			log.Println(err)
		}
	})

	vpn.conn = newConn(vpn)

	ip, ipnet, err := cutevpn.ParseCIDR(conf.CIDR)
	if err != nil {
		return vpn, err
	}

	var gateway cutevpn.IPv4
	if conf.Gateway != "" {
		gateway, err = cutevpn.ParseIPv4(conf.Gateway)
		if err != nil {
			return vpn, err
		}
	}

	vpn.routing = ospf.New(vpn, ip, false)
	vpn.router, err = newRouter(ip, ipnet, gateway, vpn.conn, vpn.routing, socket)
	if err != nil {
		return vpn, err
	}
	vpn.router.Start(vpn)

	for _, linkURL := range conf.Links {
		parsedURL, err := url.Parse(linkURL)
		if err != nil {
			return vpn, err
		}
		ctx, cancel := context.WithCancel(vpn.ctx)
		Link, err := cutevpn.GetLink(parsedURL.Scheme)
		if err != nil {
			return vpn, err
		}
		link, err := Link(vpn, ctx, cancel, parsedURL)
		if err != nil {
			return vpn, err
		}
		if link != nil {
			vpn.AddLink(link)
		}
	}
	return vpn, nil
}
