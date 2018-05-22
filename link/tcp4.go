package link

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"encoding/binary"
	"github.com/clmul/cutevpn"
	"time"
)

type tcp4 struct {
	loop   cutevpn.Looper
	in     chan packet
	out    chan packet
	listen string
	peer   AddrPort

	conn     *net.TCPConn
	listener *net.TCPListener
	clients  sync.Map
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
		time.Sleep(time.Second)
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
		t.conn = conn.(*net.TCPConn)
		t.loop.Loop(func(ctx context.Context) error {
			err := t.poll(ctx, t.conn, t.peer)
			if err != nil {
				t.onErr(err, t.peer)
				return cutevpn.StopLoop
			}
			return nil
		})
		return cutevpn.StopLoop
	})
}

func (t *tcp4) accept(ctx context.Context) error {
	conn, err := t.listener.AcceptTCP()
	if err != nil {
		return err
	}
	addr := convertTCPAddr(conn.RemoteAddr().(*net.TCPAddr))
	t.clients.Store(addr, conn)
	t.loop.Loop(func(ctx context.Context) error {
		err := t.poll(ctx, conn, addr)
		if err != nil {
			t.onErr(err, addr)
			return cutevpn.StopLoop
		}
		return nil
	})
	return nil
}

func (t *tcp4) onErr(err error, addr AddrPort) {
	log.Printf("err %v on %v", err, addr)
	if addr == t.peer {
		t.conn.Close()
		t.conn = nil
		t.dial()
		return
	}
	c, ok := t.clients.Load(addr)
	if !ok {
		return
	}
	t.clients.Delete(addr)
	c.(*net.TCPConn).Close()
}

func (t *tcp4) poll(ctx context.Context, conn *net.TCPConn, addr cutevpn.LinkAddr) (err error) {
	size := make([]byte, 2)
	err = conn.SetReadDeadline(time.Now().Add(time.Minute * 4))
	if err != nil {
		return err
	}
	_, err = io.ReadFull(conn, size)
	if err != nil {
		return err
	}
	payload := make([]byte, binary.LittleEndian.Uint16(size))
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return err
	}
	t.in <- packet{payload: payload, addr: addr}
	return nil
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

func (t *tcp4) send(packet []byte, addr cutevpn.LinkAddr) {
	var c *net.TCPConn
	if addr == t.peer && t.conn != nil {
		c = t.conn
	} else {
		conn, ok := t.clients.Load(addr)
		if ok {
			c = conn.(*net.TCPConn)
		} else {
			log.Printf("unknown address %v", addr)
			return
		}
	}
	err := c.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
	if err != nil {
		c.Close()
		log.Println(err)
		return
	}
	buffer := make([]byte, 2048)
	binary.LittleEndian.PutUint16(buffer, uint16(len(packet)))
	// TODO: encrypt size
	n := copy(buffer[2:], packet)
	_, err = c.Write(buffer[:2+n])
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
	if t.conn != nil {
		t.conn.Close()
	}
	t.clients.Range(func(addr, conn interface{}) bool {
		c := conn.(*net.TCPConn)
		c.Close()
		return true
	})
	return nil
}

func (t *tcp4) Overhead() int {
	return -1
}
