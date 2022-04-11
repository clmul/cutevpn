package vpn

import (
	"errors"
	"log"
	"net/url"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/link"
	"github.com/clmul/cutevpn/ospf"
	"github.com/clmul/cutevpn/socket"
)

func checkConfig(conf *cutevpn.Config) error {
	if conf.DefaultRoute {
		if conf.Gateway == "" {
			return errors.New("no gateway to set default route")
		}
	}
	return nil
}

func StartWithSocket(conf *cutevpn.Config, vpn *VPN, sock cutevpn.Socket) (err error) {
	defer func() {
		if err != nil {
			vpn.Stop()
		}
	}()

	vpn.conn = newConn(vpn)

	ip, ipnet, err := cutevpn.ParseCIDR(conf.CIDR)
	if err != nil {
		return err
	}

	var gateway cutevpn.IPv4
	if conf.Gateway != "" {
		gateway, err = cutevpn.ParseIPv4(conf.Gateway)
		if err != nil {
			return err
		}
	}

	vpn.routing = ospf.New(vpn, ip, false)
	vpn.router, err = newRouter(ip, ipnet, gateway, conf.Routes, vpn.conn, vpn.routing, sock)
	if err != nil {
		return err
	}
	vpn.router.Start(vpn)

	for _, linkURL := range conf.Links {
		parsedURL, err := url.Parse(linkURL)
		if err != nil {
			return err
		}
		li, err := link.New(vpn, parsedURL, conf.CACert, conf.Cert, conf.Key)
		if err != nil {
			return err
		}
		if li != nil {
			vpn.AddLink(li)
		}
	}
	if conf.DefaultRoute {
		err = vpn.addDefaultRoute(conf.Gateway)
	}
	return err
}

func Start(conf *cutevpn.Config) (*VPN, error) {
	err := checkConfig(conf)
	if err != nil {
		return nil, err
	}
	vpn := NewVPN(conf.Name)
	sock, err := socket.New(conf.Socket, vpn, conf.CIDR, conf.MTU)
	if err != nil {
		return nil, err
	}
	vpn.Defer(func() {
		err := sock.Close()
		if err != nil {
			log.Println(err)
		}
	})
	return vpn, StartWithSocket(conf, vpn, sock)
}
