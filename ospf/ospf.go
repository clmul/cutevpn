package ospf

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/clmul/cutevpn"
	"github.com/clmul/cutevpn/ospf/message"
)

const (
	HelloInterval      = 1024 * time.Millisecond
	RouterDeadInterval = 256 * time.Second
	AdjaCheckInterval  = 1500 * time.Millisecond
	RetryInterval      = 1800 * time.Millisecond
	UpdateThreshold    = 32 // 32/128
	MaxMetric          = time.Hour
	AverageWindow      = 9
)

func init() {
	cutevpn.RegisterRouting("ospf", func(loop cutevpn.Looper, ip cutevpn.IPv4) cutevpn.Routing {
		return newOSPF(loop, ip, false)
	})
	cutevpn.RegisterRouting("leaf", func(loop cutevpn.Looper, ip cutevpn.IPv4) cutevpn.Routing {
		return newOSPF(loop, ip, true)
	})
}

type OSPF struct {
	in   chan cutevpn.Packet
	out  chan cutevpn.Packet
	loop cutevpn.Looper

	ip     cutevpn.IPv4
	routes *table
	leaf   bool

	adjacents map[cutevpn.IPv4]*adjacent
	neighbors map[cutevpn.IPv4]*linkState

	tasks chan func()
}

type linkState struct {
	msg   message.LinkStateUpdate
	acked map[cutevpn.IPv4]uint64
}

func (ls linkState) MarshalJSON() ([]byte, error) {
	acked := make([]cutevpn.IPv4, 0, len(ls.acked))
	for ip := range ls.acked {
		acked = append(acked, ip)
	}
	data := map[string]interface{}{
		"db":      ls.msg.State,
		"version": time.Unix(0, int64(ls.msg.Version)).In(time.UTC),
		"acked":   acked,
	}
	return json.Marshal(data)
}

func newOSPF(loop cutevpn.Looper, ip cutevpn.IPv4, isLeaf bool) cutevpn.Routing {
	ospf := &OSPF{
		in:        make(chan cutevpn.Packet, 16),
		out:       make(chan cutevpn.Packet, 16),
		loop:      loop,
		ip:        ip,
		leaf:      isLeaf,
		routes:    newRouteTable(),
		adjacents: make(map[cutevpn.IPv4]*adjacent),
		neighbors: make(map[cutevpn.IPv4]*linkState),
		tasks:     make(chan func()),
	}
	adjaCheckTick := time.NewTicker(AdjaCheckInterval)
	loop.Defer(adjaCheckTick.Stop)
	retryTick := time.NewTicker(RetryInterval)
	loop.Defer(retryTick.Stop)
	loop.Loop(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case packet := <-ospf.in:
			ospf.handlePacket(packet)
		case <-adjaCheckTick.C:
			ospf.checkAdja()
		case <-retryTick.C:
			ospf.sendPendingLSDB()
		case f := <-ospf.tasks:
			f()
		}
		return nil
	})
	return ospf
}

func (ospf *OSPF) Dump() []byte {
	result := make(chan []byte)
	ospf.tasks <- func() {
		ospf.routes.Lock()
		data, err := json.Marshal(map[string]interface{}{
			"IP":             ospf.ip,
			"adjacents":      ospf.adjacents,
			"neighbors":      ospf.neighbors,
			"adjaRoutes":     ospf.routes.adja,
			"shortestRoutes": ospf.routes.shortest,
			"balanceRoutes":  ospf.routes.balance,
		})
		ospf.routes.Unlock()
		if err != nil {
			panic(err)
		}
		result <- data
	}
	return <-result
}

func (ospf *OSPF) Inject(p cutevpn.Packet) {
	select {
	case ospf.in <- p:
	default:
	}
}

func (ospf *OSPF) SendQueue() chan cutevpn.Packet {
	return ospf.out
}

func (ospf *OSPF) AddIfce(ifce cutevpn.Link, peer cutevpn.Route) {
	sendHello := func() {
		msg := message.NewHello(ospf.ip, nanotime(), 0, 0)
		packet := msg.Marshal(make([]byte, 2048))
		ospf.out <- cutevpn.Packet{Payload: packet, Route: peer}
	}
	sendHello()

	tick := time.NewTicker(HelloInterval)
	ospf.loop.Defer(tick.Stop)
	ospf.loop.Loop(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case <-tick.C:
			sendHello()
		}
		return nil
	})
}

func (ospf *OSPF) handlePacket(packet cutevpn.Packet) {
	msg := message.Unmarshal(packet.Payload)
	switch m := msg.(type) {
	case message.Hello:
		ospf.handleHello(m, packet.Route)
	case message.LinkStateUpdate:
		ospf.handleLinkState(m)
	case message.LinkStateACK:
		ospf.handleACK(m)
	default:
		log.Fatal("wrong message type")
	}
}

func (ospf *OSPF) handleHello(hello message.Hello, route cutevpn.Route) {
	var start uint64
	src := hello.Src
	switch hello.Forwarded {
	case 0:
		hello.Time2 = nanotime()
		hello.Forwarded = 1
		hello.Src = ospf.ip
		packet := hello.Marshal(make([]byte, 2048))
		ospf.out <- cutevpn.Packet{Payload: packet, Route: route}
		return
	case 1:
		start = hello.Time1
		hello.Forwarded = 2
		hello.Src = ospf.ip
		packet := hello.Marshal(make([]byte, 2048))
		ospf.out <- cutevpn.Packet{Payload: packet, Route: route}
	case 2:
		start = hello.Time2
	}
	rtt := nanotime() - start
	ospf.updateMetric(src, route, hello.BootTime, rtt)
}

func (ospf *OSPF) updateMetric(src cutevpn.IPv4, route cutevpn.Route, startTime, rtt uint64) {
	adja, ok := ospf.adjacents[src]
	if !ok {
		adja = newAdjacent()
		ospf.adjacents[src] = adja
	}
	adja.BootTime = startTime
	if adja.Update(route, rtt) {
		ospf.floodLinkState()
	}
	ospf.updateRouteTable()
}

func (ospf *OSPF) checkAdja() {
	shouldFlood := false
	for ip, adja := range ospf.adjacents {
		updated := adja.UpdateMetric()
		if updated {
			shouldFlood = true
		}
		if len(adja.Routes) == 0 {
			delete(ospf.adjacents, ip)
			shouldFlood = true
		}
	}
	if shouldFlood {
		ospf.floodLinkState()
	}
	ospf.updateRouteTable()
}

func (ospf *OSPF) updateRouteTable() {
	ospf.routes.Update(ospf.ip, ospf.adjacents, ospf.neighbors)
}

func (ospf *OSPF) linkState() map[cutevpn.IPv4]uint64 {
	db := make(map[cutevpn.IPv4]uint64)
	for ip, adja := range ospf.adjacents {
		db[ip] = adja.Metric
	}
	return db
}

func (ospf *OSPF) floodLinkState() {
	state := ospf.linkState()
	version := uint64(time.Now().UnixNano())
	msg := message.NewLinkStateUpdate(ospf.ip, ospf.ip, version, state)
	linkState := linkState{
		msg:   msg,
		acked: make(map[cutevpn.IPv4]uint64),
	}
	ospf.neighbors[ospf.ip] = &linkState
}

func (ospf *OSPF) sendPendingLSDB() {
	for owner, state := range ospf.neighbors {
		for adjaIP, adja := range ospf.adjacents {
			if bootTime, ok := state.acked[adjaIP]; !ok || bootTime < adja.BootTime {
				log.Printf("send %v's LinkState to %v", owner, adjaIP)
				msg := state.msg
				msg.Src = ospf.ip
				if owner == ospf.ip && ospf.leaf {
					msg.State = make(map[cutevpn.IPv4]uint64)
				}
				route, err := ospf.GetAdja(adjaIP)
				if err != nil {
					continue
				}
				ospf.out <- cutevpn.Packet{Payload: msg.Marshal(make([]byte, 2048)), Route: route}
			}
		}
	}
}

func (ospf *OSPF) ack(msg message.LinkStateUpdate) {
	ackPacket := message.NewLinkStateACK(ospf.ip, msg.Owner, msg.Version)
	route, err := ospf.GetAdja(msg.Src)
	if err != nil {
		return
	}
	ospf.out <- cutevpn.Packet{Payload: ackPacket.Marshal(make([]byte, 2048)), Route: route}
}

func (ospf *OSPF) handleACK(ack message.LinkStateACK) {
	src := ack.Src
	owner := ack.Owner
	state, ok := ospf.neighbors[owner]
	if !ok {
		return
	}
	if ack.Version >= state.msg.Version {
		state.acked[src] = ack.BootTime
	}
}

func (ospf *OSPF) handleLinkState(msg message.LinkStateUpdate) {
	ospf.ack(msg)
	if msg.Owner == ospf.ip {
		log.Println("bug: receive a LinkStateUpdate packet whose owner is myself")
		return
	}
	ospf.updateLSDB(msg)
}

func (ospf *OSPF) updateLSDB(msg message.LinkStateUpdate) {
	oldState, ok := ospf.neighbors[msg.Owner]
	if !ok || oldState.msg.Version < msg.Version {
		state := linkState{
			msg:   msg,
			acked: make(map[cutevpn.IPv4]uint64),
		}
		state.acked[msg.Src] = msg.BootTime
		state.acked[msg.Owner] = ^uint64(0)
		ospf.neighbors[msg.Owner] = &state
	} else {
		if oldState.acked[msg.Src] < msg.BootTime {
			oldState.acked[msg.Src] = msg.BootTime
		}
	}
}
