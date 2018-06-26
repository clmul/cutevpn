package link

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/clmul/cutevpn"
)

type tcp4 struct {
	loop   cutevpn.Looper
	in     chan packet
	out    chan packet
	listen string
	peer   AddrPort

	listener *net.TCPListener
	conns    sync.Map
}

type packet struct {
	addr    cutevpn.LinkAddr
	payload []byte
}

func init() {
	cutevpn.RegisterLink("tcp4", newTCP)
}

func newTCP(loop cutevpn.Looper, listen, dial string) (cutevpn.Link, error) {
	t := &tcp4{
		loop:   loop,
		in:     make(chan packet, 16),
		out:    make(chan packet, 16),
		listen: listen,
	}
	if dial != "" {
		peer, err := parseAddrPort(dial)
		if err != nil {
			return nil, err
		}
		t.peer = peer.(AddrPort)
		t.dial()
	}

	if listen != "" {
		ln, err := net.Listen("tcp4", listen)
		if err != nil {
			return nil, err
		}
		t.listener = ln.(*net.TCPListener)
		loop.Loop(t.accept)
	}
	loop.Loop(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case p := <-t.out:
			t.send(p.payload, p.addr)
		}
		return nil
	})
	return t, nil
}

func (t *tcp4) dial() {
	log.Printf("dialing %v", t.peer)
	t.loop.Loop(func(ctx context.Context) error {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp4", t.peer.String())
		if err != nil {
			log.Println(err)
			select {
			case <-ctx.Done():
			case <-time.After(time.Second * 16):
			}
			return nil
		}
		log.Printf("connected to %v", conn.RemoteAddr())
		c := conn.(*net.TCPConn)
		t.conns.Store(t.peer, c)
		t.poll(c, t.peer)
		return cutevpn.ErrStopLoop
	})
}

func (t *tcp4) accept(ctx context.Context) error {
	conn, err := t.listener.AcceptTCP()
	if err != nil {
		return err
	}
	addr := convertTCPAddr(conn.RemoteAddr().(*net.TCPAddr))
	t.conns.Store(addr, conn)
	t.poll(conn, addr)
	return nil
}

func (t *tcp4) onErr(err error, addr AddrPort) {
	log.Printf("err %v on %v", err, addr)
	c, ok := t.conns.Load(addr)
	if ok {
		t.conns.Delete(addr)
		c.(*net.TCPConn).Close()
	}
	if addr == t.peer {
		t.dial()
	}
}

func (t *tcp4) poll(conn *net.TCPConn, addr AddrPort) {
	scanner := bufio.NewScanner(conn)
	scanner.Split(split)
	t.loop.Loop(func(ctx context.Context) error {
		err := conn.SetReadDeadline(time.Now().Add(time.Minute * 4))
		if err != nil {
			t.onErr(err, addr)
			return cutevpn.ErrStopLoop
		}
		if !scanner.Scan() {
			t.onErr(scanner.Err(), addr)
			return cutevpn.ErrStopLoop
		}
		payload := scanner.Bytes()
		p := make([]byte, len(payload))
		copy(p, payload)
		t.in <- packet{payload: p, addr: addr}
		return nil
	})
}

func (t *tcp4) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("tcp4 %v->%v", t.listen, dst)
}

func (t *tcp4) ParseAddr(addr string) (cutevpn.LinkAddr, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return nil, err
	}
	return convertTCPAddr(tcpAddr), nil
}

func (t *tcp4) Peer() cutevpn.LinkAddr {
	return t.peer
}

func (t *tcp4) Send(payload []byte, addr cutevpn.LinkAddr) error {
	p := packet{
		payload: payload,
		addr:    addr,
	}
	select {
	case t.out <- p:
	default:
	}
	return nil
}

func nextSep(sep []byte) []byte {
	for i := 0; i < len(sep); i++ {
		sep[i]++
		if sep[i] != 0 {
			break
		}
		sep[i]++
	}
	if sep[len(sep)-1] != 0 {
		sep = append(sep, 0)
	}
	return sep
}

func addSep(p []byte) []byte {
	sep := []byte{0}
	for ; bytes.Index(p, sep) >= 0; sep = nextSep(sep) {
	}
	r := make([]byte, 0, 2048)
	r = append(r, sep...)
	r = append(r, p...)
	r = append(r, sep...)
	return r
}

func split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF {
		return 0, nil, io.ErrUnexpectedEOF
	}
	sepLen := bytes.IndexByte(data, 0) + 1
	if sepLen <= 0 {
		return 0, nil, nil
	}
	sep := data[:sepLen]
	end := bytes.Index(data[sepLen:], sep)
	if end < 0 {
		return 0, nil, nil
	}
	token = data[sepLen : end+sepLen]
	advance = len(token) + 2*sepLen
	return
}

func (t *tcp4) send(packet []byte, addr cutevpn.LinkAddr) {
	var c *net.TCPConn
	conn, ok := t.conns.Load(addr)
	if ok {
		c = conn.(*net.TCPConn)
	} else {
		log.Printf("unknown address %v", addr)
		return
	}
	err := c.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
	if err != nil {
		c.Close()
		log.Println(err)
		return
	}
	packet = addSep(packet)
	_, err = c.Write(packet)
	if err != nil {
		c.Close()
		log.Println(err)
	}
}

func (t *tcp4) Recv(buffer []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	packet := <-t.in
	copy(buffer, packet.payload)
	return buffer[:len(packet.payload)], packet.addr, nil
}

func (t *tcp4) Close() error {
	close(t.in)
	if t.listener != nil {
		t.listener.Close()
	}
	t.conns.Range(func(addr, conn interface{}) bool {
		c := conn.(*net.TCPConn)
		c.Close()
		return true
	})
	return nil
}

func (t *tcp4) Overhead() int {
	return -1
}
