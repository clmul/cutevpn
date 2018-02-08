package message

import (
	"bytes"
	"github.com/clmul/cutevpn"
	"testing"
)

func TestMarshalHello(t *testing.T) {
	h := NewHello(cutevpn.IPv4{192, 168, 123, 234}, 1, 2, 3, nil)

	got := h.Marshal(make([]byte, 2048))
	expected := []byte{
		cutevpn.RoutingProtocolNumber, tHello, 192, 168, 123, 234,
		1, 0, 0, 0, 0, 0, 0, 0,
		2, 0, 0, 0, 0, 0, 0, 0,
		3,
	}

	if !bytes.Equal(got, expected) {
		t.Errorf("wrong Hello binary packet,\nexpect %v,\ngot    %v", expected, got)
	}

}
