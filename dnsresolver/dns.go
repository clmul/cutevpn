package dnsresolver

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type future[V any] struct {
	ch    chan struct{}
	value V
}

type Cache[V any] struct {
	v  map[string]*future[V]
	mu sync.Mutex
}

func (c *Cache[V]) loadOrNew(host string) (f *future[V], loaded bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.v[host]
	if ok {
		return f, true
	}
	f = &future[V]{
		ch: make(chan struct{}),
	}
	c.v[host] = f
	return f, false
}

func (c *Cache[V]) Get(host string, job func() (V, time.Duration)) V {
	f, loaded := c.loadOrNew(host)
	if loaded {
		<-f.ch
		return f.value
	}
	value, ttl := job()
	f.value = value
	close(f.ch)
	go func() {
		time.Sleep(ttl)
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.v, host)
	}()
	return value
}

type result struct {
	addr net.IP
	err  error
}

type resolver struct {
	cache  Cache[result]
	dial   func(context.Context, string, string) (net.Conn, error)
	server string
}

func New(server string, dial func(context.Context, string, string) (net.Conn, error)) (*resolver, error) {
	r := &resolver{
		cache:  Cache[result]{v: make(map[string]*future[result])},
		dial:   dial,
		server: server,
	}
	return r, nil
}

func (r *resolver) Resolve(ctx context.Context, host string) (net.IP, error) {
	host = dns.Fqdn(host)

	job := func() (result, time.Duration) {
		conn, err := r.dial(ctx, "udp", r.server)
		if err != nil {
			return result{nil, err}, 0
		}
		dnsConn := &dns.Conn{Conn: conn}
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(host), dns.TypeA)
		err = dnsConn.WriteMsg(m)
		if err != nil {
			return result{nil, err}, 0
		}
		var msg *dns.Msg
		for {
			err = dnsConn.SetReadDeadline(time.Now().Add(time.Second))
			if err != nil {
				return result{nil, err}, 0
			}
			msg, err = dnsConn.ReadMsg()
			if err == nil {
				break
			}
			select {
			case <-ctx.Done():
				return result{nil, err}, 0
			default:
			}
			if err1, ok := err.(*net.OpError); ok && err1.Timeout() {
				err = dnsConn.WriteMsg(m)
				if err != nil {
					return result{nil, err}, 0
				}
				continue
			}
			return result{nil, err}, 0
		}
		ttl := -1
		for _, rr := range msg.Answer {
			ttl1 := int(rr.Header().Ttl)
			if ttl < 0 || ttl1 < ttl {
				ttl = ttl1
			}
			switch r := rr.(type) {
			case *dns.CNAME:
			case *dns.A:
				return result{r.A, nil}, time.Second * time.Duration(ttl)
			}
		}
		return result{nil, &net.DNSError{IsNotFound: true}}, 0
	}
	re := r.cache.Get(host, job)
	return re.addr, re.err
}
