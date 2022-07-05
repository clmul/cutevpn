package ospf

import (
	"fmt"
	"time"

	// for go:linkname
	_ "unsafe"
)

//go:linkname nanotime runtime.nanotime
func nanotime() uint64

// The cost of a route
type metric []rtt

type rtt struct {
	// nanoseconds
	d uint64
	// the return value of runtime.nanotime()
	t uint64
}

func (m metric) Value() uint64 {
	if len(m) == 0 {
		return uint64(maxMetric)
	}
	lastUpdate := m[len(m)-1].t
	age := time.Duration(nanotime() - lastUpdate)
	if age > routerDeadInterval {
		return uint64(maxMetric)
	}

	// avg(rtts) * (packet received rate) ** (-3)
	var sum uint64
	for _, rtt := range m {
		sum += rtt.d
	}
	interval := uint64(helloInterval)
	d := nanotime() - m[0].t
	if d > uint64(time.Second) {
		d -= uint64(time.Second)
	}
	helloTimes := d/interval + 1
	receivedTimes := uint64(len(m))
	if helloTimes < receivedTimes {
		helloTimes = receivedTimes
	}
	ratio := helloTimes * 128 / receivedTimes
	ratio = ratio * ratio * ratio
	return sum / uint64(len(m)) * ratio / 128 / 128 / 128
}

func (m metric) String() string {
	v := m.Value()
	d := nanotime() - m[len(m)-1].t
	return fmt.Sprintf("%v at %v ago", time.Duration(v), time.Duration(d))
}

func (m metric) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m metric) Push(d uint64) metric {
	rtt := rtt{d: d, t: nanotime()}
	if len(m) < averageWindow {
		return append(m, rtt)
	} else {
		return append(m[1:], rtt)
	}
}
