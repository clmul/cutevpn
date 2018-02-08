package ospf

import (
	"fmt"
	"time"
	_ "unsafe"
)

type metric struct {
	rtts []rtt
}

type rtt struct {
	d uint64
	t uint64
}

func (m *metric) Value() uint64 {
	if len(m.rtts) == 0 {
		return uint64(MaxMetric)
	}
	lastUpdate := m.rtts[len(m.rtts)-1].t
	age := time.Duration(nanotime() - lastUpdate)
	if age > RouterDeadInterval {
		return uint64(MaxMetric)
	}
	if len(m.rtts) == 1 {
		return m.rtts[0].d
	}

	// avg(rtts) * (packet received rate) ** (-2)
	var sum uint64
	for _, rtt := range m.rtts {
		sum += rtt.d
	}
	interval := uint64(HelloInterval)
	d := m.rtts[len(m.rtts)-1].t - m.rtts[0].t
	d += interval / 2
	helloTimes := d/interval + 1
	receivedTimes := uint64(len(m.rtts))
	ratio := helloTimes * 128 / receivedTimes
	ratio *= ratio
	return sum / uint64(len(m.rtts)) * ratio / 128 / 128
}

func (m *metric) String() string {
	v := m.Value()
	d := nanotime() - m.rtts[len(m.rtts)-1].t
	return fmt.Sprintf("%v at %v ago", time.Duration(v), time.Duration(d))
}

func (m *metric) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *metric) Push(d uint64) {
	rtt := rtt{d: d, t: nanotime()}
	if len(m.rtts) < AverageWindow {
		m.rtts = append(m.rtts, rtt)
	} else {
		m.rtts = append(m.rtts[1:], rtt)
	}
}

//go:linkname nanotime runtime.nanotime
func nanotime() uint64
