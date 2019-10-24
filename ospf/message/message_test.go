package message

import (
	"fmt"
	"testing"
)

func init() {
	bootTime = 127
}

func TestMarshalHello(t *testing.T) {
	p0 := NewHello(IPv4{192, 168, 123, 234}, 1, 2, 3)
	p0.BootTime = bootTime
	marshaled := p0.Marshal(make([]byte, 2048))
	p1 := Unmarshal(marshaled)

	if p0 != p1 {
		t.Errorf("expect\n%#v, got\n%#v", p0, p1)
	}
}

func TestMarshalLinkStateAck(t *testing.T) {
	p0 := NewLinkStateACK(IPv4{1, 1, 1, 1}, IPv4{8, 8, 8, 8}, 1345245240)
	p0.BootTime = bootTime
	marshaled := p0.Marshal(make([]byte, 2048))
	p1 := Unmarshal(marshaled)

	if p0 != p1 {
		t.Errorf("expect\n%#v, got\n%#v", p0, p1)
	}
}

func TestMarshalLinkStateUpdate(t *testing.T) {
	p0 := NewLinkStateUpdate(IPv4{1, 1, 1, 1}, "test", 1345245240, map[IPv4]uint64{
		{1, 1, 1, 1}:   135,
		{2, 1, 1, 1}:   135246,
		{3, 1, 1, 1}:   135246789,
		{3, 1, 1, 255}: 1352467890,
	})
	p0.BootTime = bootTime
	marshaled := p0.Marshal(make([]byte, 2048))
	p1 := Unmarshal(marshaled)

	if fmt.Sprint(p0) != fmt.Sprint(p1) {
		t.Errorf("expect\n%#v, got\n%#v", p0, p1)
	}
}
