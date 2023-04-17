package dns

import (
	"net"
	"testing"
)

func TestResolveIPv4(t *testing.T) {
	cases := []struct {
		arg string
		ip  net.IP
		err error
	}{
		{"1.2.3.4", net.IP{1, 2, 3, 4}, nil},
		{"1.2.3.256", nil, ErrNoSuchHost},
		{"[fe80::1]", nil, ErrExpectIPv4},
		{"fe80::1", nil, ErrExpectIPv4},
	}
	for _, c := range cases {
		ip, err := ResolveIPv4(c.arg)
		if !ip.Equal(c.ip) || err != c.err {
			t.Errorf("ResolveIPv4(%#v), got %#v %v, expect %#v %v", c.arg, ip, err, c.ip, c.err)
		}
	}
}

func TestResolveIPv6(t *testing.T) {
	cases := []struct {
		arg string
		ip  net.IP
		err error
	}{
		{"1.2.3.4", nil, ErrExpectIPv6},
		{"::1", net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, nil},
		{"[fe80::1]", net.IP{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, nil},
	}
	for _, c := range cases {
		ip, err := ResolveIPv6(c.arg)
		if !ip.Equal(c.ip) || err != c.err {
			t.Errorf("ResolveIPv6(%#v), got %#v %v, expect %#v %v", c.arg, ip, err, c.ip, c.err)
		}
	}
}
