package tcp4

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/clmul/cutevpn"
)

type link struct {
	conn net.Conn
	out  chan []byte

	peer   cutevpn.LinkAddr
	local  string
	remote string

	vpn    cutevpn.VPN
	ctx    context.Context
	cancel context.CancelFunc
}

func (d *link) sendLoop() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case packet := <-d.out:
			err := send(d.vpn, d.conn, packet)
			if err != nil {
				log.Println(err)
				d.cancel()
				return
			}
		}
	}
}

func send(vpn cutevpn.VPN, conn net.Conn, packet []byte) error {
	lengthBuf := make([]byte, 2+vpn.Cipher().Overhead())
	binary.LittleEndian.PutUint16(lengthBuf, uint16(len(packet)))
	lengthBuf = vpn.Cipher().Encrypt(lengthBuf[:2])

	conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, err := (&net.Buffers{lengthBuf, packet}).WriteTo(conn)
	return err
}

func recv(vpn cutevpn.VPN, conn net.Conn, buffer []byte) ([]byte, error) {
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buffer = buffer[:vpn.Cipher().Overhead()+2]
	_, err := io.ReadFull(conn, buffer)
	if err != nil {
		return nil, err
	}
	lengthBuf, err := vpn.Cipher().Decrypt(buffer)
	if err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint16(lengthBuf)
	buffer = buffer[:length]
	_, err = io.ReadFull(conn, buffer)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (d *link) Send(packet []byte, dst cutevpn.LinkAddr) error {
	select {
	case d.out <- packet:
	default:
	}
	return nil
}

func (d *link) Recv(buffer []byte) (p []byte, addr cutevpn.LinkAddr, err error) {
	p, err = recv(d.vpn, d.conn, buffer)
	return p, d.remote, err
}

func (d *link) Peer() cutevpn.LinkAddr {
	return d.peer
}

func (d *link) Overhead() int {
	return 20 + 20 + 2 + d.vpn.Cipher().Overhead()
}

func (d *link) ToString(dst cutevpn.LinkAddr) string {
	return fmt.Sprintf("tcp4 %v->%v", d.local, d.remote)
}

func (d *link) Cancel() {
	d.cancel()
}

func (d *link) Done() <-chan struct{} {
	return d.ctx.Done()
}
