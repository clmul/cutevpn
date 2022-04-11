package link

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/clmul/cutevpn"
)

func newTLS(vpn cutevpn.VPN, linkURL *url.URL, cert tls.Certificate, ca *x509.CertPool) (cutevpn.Link, error) {
	if linkURL.Query().Get("listen") == "1" {
		return newTLSListener(vpn, linkURL, cert, ca)
	}
	return newTLSDialer(vpn, linkURL, cert, ca)
}

func newTLSListener(vpn cutevpn.VPN, linkURL *url.URL, cert tls.Certificate, ca *x509.CertPool) (cutevpn.Link, error) {
	listener, err := tls.Listen("tcp", linkURL.Host, &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    ca,
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"http/1.1"},
	})
	if err != nil {
		return nil, err
	}
	vpn.Defer(func() {
		listener.Close()
	})
	fake := newFakeListener()
	vpn.Loop(func(ctx context.Context) error {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		err = conn.(*tls.Conn).Handshake()
		if err != nil {
			log.Println(err)
			return nil
		}
		if len(conn.(*tls.Conn).ConnectionState().VerifiedChains) == 0 {
			log.Println("response HTTPS")
			fake.ch <- conn
			return nil
		}
		peer := newStream(ctx, vpn, conn, nil)
		vpn.AddLink(peer)
		return nil
	})
	return nil, nil
}

func newTLSDialer(vpn cutevpn.VPN, linkURL *url.URL, cert tls.Certificate, ca *x509.CertPool) (cutevpn.Link, error) {
	vpn.Loop(func(ctx context.Context) error {
		return connect(ctx, vpn, linkURL, cert, ca)
	})
	return nil, nil
}

func tlsDialContext(ctx context.Context, addr string, config *tls.Config) (*tls.Conn, error) {
	rawConn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	conn := tls.Client(rawConn, config)
	return conn, nil
}

func connect(ctx context.Context, vpn cutevpn.VPN, linkURL *url.URL, cert tls.Certificate, ca *x509.CertPool) error {
	for i := 1; ; i++ {
		conn, err := tlsDialContext(ctx, linkURL.Host, &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      ca,
			MinVersion:   tls.VersionTLS13,
			ServerName:   linkURL.Hostname(),
		})
		if err != nil {
			log.Println(err)
			select {
			case <-time.After(time.Second * 5):
			case <-vpn.Done():
				return nil
			}
			continue
		}
		peer := newStream(ctx, vpn, conn, linkURL.Host)
		vpn.AddLink(peer)
		<-peer.Done()
		return nil
	}
}

type fakeListener struct {
	ch chan net.Conn
}

func newFakeListener() *fakeListener {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Index Page</title>
</head>
<body>
<h1>Hello, world!</h1>
</body>
</html>`))
	})
	ln := &fakeListener{ch: make(chan net.Conn)}
	go func() {
		log.Fatal(http.Serve(ln, handler))
	}()
	return ln
}

func (ln *fakeListener) Accept() (net.Conn, error) {
	return <-ln.ch, nil
}

func (ln *fakeListener) Close() error {
	return nil
}

func (ln *fakeListener) Addr() net.Addr {
	return &net.TCPAddr{
		IP:   []byte{127, 0, 0, 1},
		Port: 443,
		Zone: "",
	}
}
