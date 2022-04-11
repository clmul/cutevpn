package vpn

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/ospf"
)

type VPN struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	conn    *conn
	router  *router
	routing *ospf.OSPF

	http httpServer
}

func NewVPN(name string) *VPN {
	ctx, cancel := context.WithCancel(context.Background())
	return &VPN{
		name:   name,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (v *VPN) Name() string {
	return v.name
}

func (v *VPN) AddLink(link cutevpn.Link) {
	v.conn.AddLink(link)
	peer := link.Peer()
	if peer != nil {
		v.routing.AddLink(cutevpn.Route{Link: link, Addr: peer})
	}
}

func (v *VPN) Stop() {
	v.cancel()
	v.wg.Wait()
}

func (v *VPN) StartHTTP(addr string) {
	v.http = startHTTPServer(addr)
	v.http.RegisterHandler(v.router)
}

func (v *VPN) StopHTTP() {
	err := v.http.Close()
	if err != nil {
		log.Println(err)
	}
}

// used by Android app
func (v *VPN) UpdateGateway(gateway string) {
	v.router.gatewayUpdateCh <- gateway
}

// used by Android app
func (v *VPN) Neighbors() []ospf.Neighbor {
	return v.routing.Neighbors()
}

func (v *VPN) Done() <-chan struct{} {
	return v.ctx.Done()
}

func (v *VPN) Defer(f func()) {
	v.OnCancel(v.ctx, f)
}

func (v *VPN) OnCancel(ctx context.Context, f func()) {
	v.wg.Add(1)
	go func() {
		<-ctx.Done()
		f()
		v.wg.Done()
	}()
}

func (v *VPN) Go(f func()) {
	v.wg.Add(1)
	go func() {
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
			if err == cutevpn.ErrStopLoop {
				return
			}
			if err != nil {
				select {
				case <-v.ctx.Done():
					return
				default:
				}
				log.Println(caller, err)
				v.cancel()
				return
			}
		}
	}()
}
