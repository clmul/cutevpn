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
	helloInterval      = 4096 * time.Millisecond
	routerDeadInterval = 256 * time.Second
	adjaCheckInterval  = 1500 * time.Millisecond
	retryInterval      = 1800 * time.Millisecond
	floodInterval      = 2048 * time.Millisecond
	updateThreshold    = 32 // 32%
	maxMetric          = time.Hour
	averageWindow      = 9
)

type Neighbor struct {
	Name string
	Addr IPv4
}

type Packet struct {
	Route   cutevpn.Route
	Payload []byte
}

type deadRoute struct {
	adja  IPv4
	route cutevpn.Route
}

type OSPF struct {
	vpn cutevpn.VPN
	in  chan Packet
	out chan Packet

	ip     IPv4
	routes *table
	leaf   bool
	boot   uint64

	adjacents map[IPv4]*adjacent
	neighbors map[IPv4]*linkState

	deadRoutes chan deadRoute
	tasks      chan func()

	pendingFlood bool
}

type linkState struct {
	msg   message.LinkStateUpdate
	acked map[IPv4]uint64
}

func (ls linkState) MarshalJSON() ([]byte, error) {
	acked := make(map[IPv4]time.Time)
	for ip := range ls.acked {
		acked[ip] = time.Unix(0, int64(ls.acked[ip])).In(time.UTC)
	}
	data := map[string]interface{}{
		"db":      ls.msg.State,
		"name":    ls.msg.Name,
		"version": time.Unix(0, int64(ls.msg.Version)).In(time.UTC),
		"acked":   acked,
	}
	return json.Marshal(data)
}

func New(vpn cutevpn.VPN, ip IPv4, isLeaf bool) *OSPF {
	ospf := &OSPF{
		in:         make(chan Packet, 16),
		out:        make(chan Packet, 16),
		vpn:        vpn,
		ip:         ip,
		leaf:       isLeaf,
		boot:       uint64(time.Now().UnixNano()),
		routes:     newRouteTable(),
		adjacents:  make(map[IPv4]*adjacent),
		neighbors:  make(map[IPv4]*linkState),
		deadRoutes: make(chan deadRoute),
		tasks:      make(chan func()),
	}
	adjaCheckTick := time.NewTicker(adjaCheckInterval)
	vpn.Defer(adjaCheckTick.Stop)
	retryTick := time.NewTicker(retryInterval)
	vpn.Defer(retryTick.Stop)
	floodTick := time.NewTicker(floodInterval)
	vpn.Defer(floodTick.Stop)
	vpn.Loop(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case packet := <-ospf.in:
			ospf.handlePacket(packet)
		case <-adjaCheckTick.C:
			ospf.checkAdja()
		case <-retryTick.C:
			ospf.sendPendingLSDB()
		case <-floodTick.C:
			ospf.floodLinkState()
		case r := <-ospf.deadRoutes:
			ospf.removeRoute(r)
		case f := <-ospf.tasks:
			f()
		}
		return nil
	})
	return ospf
}

func (ospf *OSPF) Neighbors() []Neighbor {
	ch := make(chan []Neighbor)
	ospf.tasks <- func() {
		ospf.routes.Lock()
		result := make([]Neighbor, 0, len(ospf.neighbors))
		for addr, state := range ospf.neighbors {
			result = append(result, Neighbor{Addr: addr, Name: state.msg.Name})
		}
		ospf.routes.Unlock()
		ch <- result
	}
	return <-ch
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

func (ospf *OSPF) Inject(p Packet) {
	select {
	case ospf.in <- p:
	default:
	}
}

func (ospf *OSPF) SendQueue() chan Packet {
	return ospf.out
}

func (ospf *OSPF) AddLink(peer cutevpn.Route) {
	sendHello := func() {
		msg := message.NewHello(nanotime(), 0, 0)
		packet := msg.Marshal(make([]byte, 2048), ospf.ip, ospf.boot)
		select {
		case ospf.out <- Packet{Payload: packet, Route: peer}:
		case <-ospf.vpn.Done():
		}
	}
	sendHello()

	ospf.vpn.Go(func() {
		tick := time.NewTicker(helloInterval)
		defer tick.Stop()
		for {
			select {
			case <-ospf.vpn.Done():
				return
			case <-tick.C:
				sendHello()
			}
		}
	})
}

func (ospf *OSPF) handlePacket(packet Packet) {
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
	bootTime := hello.BootTime
	switch hello.Forwarded {
	case 0:
		hello.Time2 = nanotime()
		hello.Forwarded = 1
		hello.Src = ospf.ip
		hello.BootTime = ospf.boot
		packet := hello.Marshal(make([]byte, 2048), ospf.ip, ospf.boot)
		ospf.out <- Packet{Payload: packet, Route: route}
		return
	case 1:
		start = hello.Time1
		hello.Forwarded = 2
		hello.Src = ospf.ip
		hello.BootTime = ospf.boot
		packet := hello.Marshal(make([]byte, 2048), ospf.ip, ospf.boot)
		ospf.out <- Packet{Payload: packet, Route: route}
	case 2:
		start = hello.Time2
	}
	rtt := nanotime() - start
	ospf.updateMetric(src, route, bootTime, rtt)
}

func (ospf *OSPF) updateMetric(src IPv4, route cutevpn.Route, bootTime, rtt uint64) {
	adja, ok := ospf.adjacents[src]
	if !ok {
		adja = newAdjacent()
		ospf.adjacents[src] = adja
	}
	adja.BootTime = bootTime
	newRoute, needUpdate := adja.Update(route, rtt)
	if newRoute {
		ospf.vpn.Go(func() {
			<-route.Link.Done()
			select {
			case <-ospf.vpn.Done():
			case ospf.deadRoutes <- deadRoute{src, route}:
			}
		})
	}
	if needUpdate {
		ospf.pendingFlood = true
	}
	ospf.updateRouteTable()
}

func (ospf *OSPF) removeRoute(dr deadRoute) {
	log.Printf("remove dead route to %v, %v", dr.adja, dr.route)
	adja := ospf.adjacents[dr.adja]
	delete(adja.Routes, dr.route)
	if len(adja.Routes) == 0 {
		log.Println("remove dead adjacent", dr.adja)
		delete(ospf.adjacents, dr.adja)
	}
	ospf.updateRouteTable()
	ospf.pendingFlood = true
}

func (ospf *OSPF) checkAdja() {
	for ip, adja := range ospf.adjacents {
		updated := adja.UpdateMetric()
		if updated {
			ospf.pendingFlood = true
		}
		if len(adja.Routes) == 0 {
			delete(ospf.adjacents, ip)
			ospf.pendingFlood = true
		}
	}
	ospf.updateRouteTable()
}

func (ospf *OSPF) updateRouteTable() {
	ospf.routes.Update(ospf.ip, ospf.boot, ospf.adjacents, ospf.neighbors)
}

func (ospf *OSPF) linkState() map[IPv4]uint64 {
	db := make(map[IPv4]uint64)
	for ip, adja := range ospf.adjacents {
		db[ip] = adja.Metric
	}
	return db
}

func (ospf *OSPF) floodLinkState() {
	if !ospf.pendingFlood {
		return
	}
	state := ospf.linkState()
	version := uint64(time.Now().UnixNano())
	msg := message.NewLinkStateUpdate(ospf.ip, ospf.vpn.Name(), version, state)
	linkState := linkState{
		msg:   msg,
		acked: make(map[IPv4]uint64),
	}
	ospf.neighbors[ospf.ip] = &linkState

	ospf.pendingFlood = false
}

func (ospf *OSPF) sendPendingLSDB() {
	for owner, state := range ospf.neighbors {
		for adjaIP, adja := range ospf.adjacents {
			if bootTime, ok := state.acked[adjaIP]; !ok || bootTime < adja.BootTime {
				msg := state.msg
				msg.Src = ospf.ip
				if owner == ospf.ip && ospf.leaf {
					msg.State = make(map[IPv4]uint64)
				}
				route, err := ospf.GetAdja(adjaIP)
				if err != nil {
					continue
				}
				ospf.out <- Packet{
					Payload: msg.Marshal(make([]byte, 2048), ospf.ip, ospf.boot),
					Route:   route,
				}
			}
		}
	}
}

func (ospf *OSPF) ack(msg message.LinkStateUpdate) {
	ackPacket := message.NewLinkStateACK(msg.Owner, msg.Version)
	route, err := ospf.GetAdja(msg.Src)
	if err != nil {
		return
	}
	ospf.out <- Packet{Payload: ackPacket.Marshal(make([]byte, 2048), ospf.ip, ospf.boot), Route: route}
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
	// new neighbor or new link state
	if !ok || oldState.msg.Version < msg.Version {
		state := linkState{
			msg:   msg,
			acked: make(map[IPv4]uint64),
		}
		state.acked[msg.Src] = msg.BootTime
		state.acked[msg.Owner] = ^uint64(0)
		ospf.neighbors[msg.Owner] = &state
		return
	}
	// The link state was received before. Mark src as that it has acked the link state.
	if oldState.msg.Version == msg.Version && oldState.acked[msg.Src] < msg.BootTime {
		oldState.acked[msg.Src] = msg.BootTime
	}
}
