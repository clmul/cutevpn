package tcp4

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/clmul/cutevpn"
)

func init() {
	cutevpn.RegisterLink("tcp4", newTCP)
}

func newTCP(vpn cutevpn.VPN, ctx context.Context, cancel context.CancelFunc, linkURL *url.URL) (cutevpn.Link, error) {
	if listenrange := linkURL.Query().Get("listenrange"); listenrange != "" {
		host := linkURL.Host
		var port0, port1 uint16
		_, err := fmt.Sscanf(listenrange, "%d-%d", &port0, &port1)
		if err != nil {
			return nil, err
		}
		for i := port0; i <= port1; i++ {
			linkURL.Host = fmt.Sprintf("%s:%d", host, i)
			_, err = newTCP4Listener(vpn, linkURL)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if linkURL.Query().Get("listen") == "1" {
		return newTCP4Listener(vpn, linkURL)
	}
	return newTCP4Dialer(vpn, linkURL)
}

func newTCP4Listener(vpn cutevpn.VPN, linkURL *url.URL) (cutevpn.Link, error) {
	tcpListener, err := net.Listen("tcp4", linkURL.Host)
	if err != nil {
		return nil, err
	}
	vpn.Loop(func(ctx context.Context) error {
		conn, err := tcpListener.Accept()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(ctx)
		peer := &link{
			conn:   conn,
			out:    make(chan []byte, 4),
			peer:   nil,
			local:  conn.LocalAddr().String(),
			remote: conn.RemoteAddr().String(),
			vpn:    vpn,
			ctx:    ctx,
			cancel: cancel,
		}
		vpn.AddLink(peer)
		vpn.Go(func() {
			peer.sendLoop()
		})
		vpn.OnCancel(ctx, func() {
			conn.Close()
		})
		return nil
	})
	vpn.Defer(func() {
		tcpListener.Close()
	})
	return nil, nil
}

func newTCP4Dialer(vpn cutevpn.VPN, linkURL *url.URL) (cutevpn.Link, error) {
	peer, err := net.ResolveTCPAddr("tcp4", linkURL.Host)
	if err != nil {
		return nil, err
	}
	vpn.Loop(func(ctx context.Context) error {
		return connect(ctx, vpn, peer.String())
	})
	return nil, nil
}

func connect(ctx context.Context, vpn cutevpn.VPN, dial string) error {
	for i := 1; ; i++ {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp4", dial)
		if err != nil {
			log.Println(err)
			select {
			case <-time.After(time.Second * 5):
			case <-vpn.Done():
				return nil
			}
			continue
		}
		ctx, cancel := context.WithCancel(ctx)
		peer := &link{
			conn:   conn,
			out:    make(chan []byte, 4),
			peer:   dial,
			local:  conn.LocalAddr().String(),
			remote: conn.RemoteAddr().String(),
			vpn:    vpn,
			ctx:    ctx,
			cancel: cancel,
		}
		vpn.AddLink(peer)
		vpn.Go(func() {
			peer.sendLoop()
		})
		<-ctx.Done()
		conn.Close()
		return nil
	}
}
