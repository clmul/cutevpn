package link

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/clmul/cutevpn"
)

type stream struct {
	conn net.Conn
	out  chan []byte

	peer   cutevpn.LinkAddr
	local  string
	remote string

	ctx    context.Context
	cancel context.CancelFunc

	scanner *bufio.Scanner
}

func newStream(ctx context.Context, vpn cutevpn.VPN, conn net.Conn, peer cutevpn.LinkAddr) *stream {
	scanner := bufio.NewScanner(conn)
	scanner.Split(split)
	ctx, cancel := context.WithCancel(ctx)
	d := &stream{
		conn:    conn,
		out:     make(chan []byte, 4),
		peer:    peer,
		local:   conn.LocalAddr().String(),
		remote:  conn.RemoteAddr().String(),
		ctx:     ctx,
		cancel:  cancel,
		scanner: scanner,
	}
	vpn.Go(func() {
		d.sendLoop()
	})
	vpn.OnCancel(ctx, func() {
		conn.Close()
	})
	return d
}

func (d *stream) sendLoop() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case packet := <-d.out:
			err := send(d.conn, packet)
			if err != nil {
				log.Println(err)
				d.cancel()
				return
			}
		}
	}
}

func send(conn net.Conn, packet []byte) error {
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, err := conn.Write(addSep(packet))
	return err
}

func recv(d *stream, buffer []byte) ([]byte, error) {
	d.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if !d.scanner.Scan() {
		return nil, d.scanner.Err()
	}
	payload := d.scanner.Bytes()
	n := copy(buffer, payload)
	return buffer[:n], nil
}

func (d *stream) Send(packet []byte, dst cutevpn.LinkAddr) error {
	select {
	case d.out <- packet:
	default:
	}
	return nil
}

func (d *stream) Recv(buffer []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	p, err = recv(d, buffer)
	return p, d.remote, err
}

func (d *stream) Peer() cutevpn.LinkAddr {
	return d.peer
}

func (d *stream) Overhead() int {
	return -1
}

func (d *stream) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("tls %v->%v", d.local, d.remote)
}

func (d *stream) Cancel() {
	d.cancel()
}

func (d *stream) Done() <-chan struct{} {
	return d.ctx.Done()
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

func addSep(p []byte) []byte {
	var sep0 [4]byte
	sep := sep0[:1]
	for ; bytes.Index(p, sep) >= 0; sep = nextSep(sep) {
	}
	r := make([]byte, 0, 2048)
	r = append(r, sep...)
	r = append(r, p...)
	r = append(r, sep...)
	return r
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
