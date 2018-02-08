package cutevpn

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type VPN struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	router     *router
	linkCipher Cipher
	http       HTTPServer
}

func (v *VPN) Stop() {
	v.cancel()
	v.wg.Wait()
}

func (v *VPN) StartHTTP() {
	v.http = StartHTTPServer(v.router.ip.String(), 19088)
	v.http.RegisterHandler(v.router)
}

func (v *VPN) StopHTTP() {
	err := v.http.Close()
	if err != nil {
		log.Println(err)
	}
}

func (v *VPN) IP() IPv4 {
	return v.router.ip
}

func (v *VPN) SocketErr(err error) {
	select {
	case <-v.ctx.Done():
		return
	default:
		// TODO
		log.Fatal(err)
	}
}

func (v *VPN) CipherErr(err error) {
	select {
	case <-v.ctx.Done():
		return
	default:
		log.Println(err)
	}
}

func (v *VPN) LinkRecvErr(err error) {
	select {
	case <-v.ctx.Done():
		return
	default:
		log.Fatal(err)
	}
}

func (v *VPN) LinkSendErr(err error) {
	opErr, ok := err.(*net.OpError)
	if !ok {
		log.Fatal(err)
	}
	syscallErr, ok := opErr.Err.(*os.SyscallError)
	if !ok {
		log.Fatal(err)
	}
	errno, ok := syscallErr.Err.(syscall.Errno)
	if !ok {
		log.Fatal(err)
	}
	switch errno {
	case unix.ENETUNREACH:
	default:
		log.Println(int(errno))
		log.Fatal(err)
	}
	// write udp4 0.0.0.0:58010->35.194.178.51:15234: sendto: network is down
	// write udp4 0.0.0.0:61147->35.194.178.51:15234: sendto: can't assign requested address
}

func (v *VPN) SendThrough(route Route, packet []byte) {
	v.router.SendThrough(route, packet)
}

func (v *VPN) Defer(f func()) {
	v.wg.Add(1)
	go func() {
		<-v.ctx.Done()
		f()
		v.wg.Done()
	}()
}

func (v *VPN) Loop(f func(context.Context) error) {
	v.wg.Add(1)
	_, file, line, _ := runtime.Caller(1)
	caller := fmt.Sprintf("%v:%v", filepath.Base(file), line)
	go func() {
		defer v.wg.Done()
		for {
			select {
			case <-v.ctx.Done():
				return
			default:
			}
			err := f(v.ctx)
			if err == StopLoop {
				return
			}
			if err != nil {
				log.Println(caller, err)
				v.cancel()
				return
			}
		}
	}()
}
