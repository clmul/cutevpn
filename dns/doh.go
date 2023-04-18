package dns

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

var servers = []string{
	"1.0.0.1",
	//"1.0.0.2",
	"1.0.0.3",
	"1.1.1.3",
	//"9.9.9.9",
	"149.112.112.112",
	"101.101.101.101",
}

func init() {
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		fmt.Println("dial", network, address)
		n := rand.Intn(len(servers))
		server := servers[n]
		return &dohConn{server: server}, nil
	}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial:     dial,
	}
}

var _ net.PacketConn = &dohConn{}

type dohConn struct {
	server string
	resp   *http.Response
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *dohConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, err = c.Read(p)
	return n, nil, err
}

func (c *dohConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return c.Write(p)
}

func (c *dohConn) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.resp.Body.Close()
}

func (c *dohConn) LocalAddr() net.Addr {
	return nil
}

func (c *dohConn) SetDeadline(t time.Time) error {
	c.ctx, c.cancel = context.WithDeadline(context.Background(), t)
	return nil
}

func (c *dohConn) SetReadDeadline(t time.Time) error {
	return c.SetDeadline(t)
}

func (c *dohConn) SetWriteDeadline(t time.Time) error {
	return c.SetDeadline(t)
}

func (c *dohConn) Read(b []byte) (n int, err error) {
	return c.resp.Body.Read(b)
}

func (c *dohConn) Write(b []byte) (n int, err error) {
	server := servers[rand.Intn(len(servers))]
	msg := base64.URLEncoding.EncodeToString(b)
	msg = strings.TrimRight(msg, "=")

	if c.ctx == nil {
		c.ctx = context.Background()
	}
	url := "https://" + server + "/dns-query?dns=" + msg
	req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	c.resp = resp
	return len(b), nil
}

func (c *dohConn) RemoteAddr() net.Addr {
	return nil
}
