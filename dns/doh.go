package dns

import (
	"encoding/base64"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"

	"github.com/miekg/dns"
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

var ErrExpectIPv4 = errors.New("not ipv4 address")
var ErrExpectIPv6 = errors.New("not ipv6 address")
var ErrNoSuchHost = errors.New("no such host")

func resolve(server, host string, dnsType uint16) (net.IP, error) {
	host = dns.Fqdn(host)
	m := new(dns.Msg)
	m.SetQuestion(host, dnsType)
	buf, err := m.Pack()
	if err != nil {
		return nil, err
	}
	resp, err := http.Get("https://" + server + "/dns-query?dns=" + base64.URLEncoding.EncodeToString(buf))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	n := new(dns.Msg)
	err = n.Unpack(body)
	if err != nil {
		return nil, err
	}
	for _, rr := range n.Answer {
		switch a := rr.(type) {
		case *dns.A:
			return a.A, nil
		case *dns.AAAA:
			return a.AAAA, nil
		}
	}
	return nil, ErrNoSuchHost
}

func resolve1(host string, dnsType uint16) (ip net.IP, err error) {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		ip = net.ParseIP(host[1 : len(host)-1])
	} else {
		ip = net.ParseIP(host)
	}
	if ip != nil {
		ipv4 := ip.To4()
		if dnsType == dns.TypeA && ipv4 != nil {
			return ipv4, nil
		}
		if dnsType == dns.TypeA && ipv4 == nil {
			return nil, ErrExpectIPv4
		}
		if dnsType == dns.TypeAAAA && ipv4 == nil {
			return ip, nil
		}
		if dnsType == dns.TypeAAAA && ipv4 != nil {
			return nil, ErrExpectIPv6
		}
	}
	for i := 0; i < 3; i++ {
		n := rand.Intn(len(servers))
		server := servers[n]
		ip, err = resolve(server, host, dnsType)
		if err == nil {
			return ip, nil
		}
	}
	return ip, err
}

func ResolveIPv4(host string) (net.IP, error) {
	return resolve1(host, dns.TypeA)
}

func ResolveIPv6(host string) (net.IP, error) {
	return resolve1(host, dns.TypeAAAA)
}
