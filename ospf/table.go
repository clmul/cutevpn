package ospf

import (
	"container/heap"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/clmul/cutevpn"
)

type IPv4 = cutevpn.IPv4

type table struct {
	sync.Mutex
	adja     map[IPv4]*routeHeap
	balance  map[IPv4]*routeHeap
	shortest map[IPv4]IPv4
}

func (ospf *OSPF) GetBalance(dst IPv4) (cutevpn.Route, IPv4, error) {
	ospf.routes.Lock()
	defer ospf.routes.Unlock()
	return ospf.routes.getBalance(dst)
}

func (ospf *OSPF) GetAdja(adja IPv4) (cutevpn.Route, error) {
	ospf.routes.Lock()
	defer ospf.routes.Unlock()
	return ospf.routes.getAdja(adja)
}

func (ospf *OSPF) GetShortest(dst IPv4) (cutevpn.Route, error) {
	ospf.routes.Lock()
	defer ospf.routes.Unlock()
	return ospf.routes.getShortest(dst)
}

func newRouteTable() *table {
	rt := &table{
		adja:     make(map[IPv4]*routeHeap),
		balance:  make(map[IPv4]*routeHeap),
		shortest: make(map[IPv4]IPv4),
	}
	return rt
}

func (rt *table) getBalance(addr IPv4) (cutevpn.Route, IPv4, error) {
	proutes, ok := rt.balance[addr]
	if !ok {
		return cutevpn.Route{}, emptyIPv4, cutevpn.ErrNoRoute
	}
	routes := *proutes
	next := routes[0].Next
	through := routes[0].Through
	routes[0].current += routes[0].Metric * routes[0].Metric
	heap.Fix(&routes, 0)
	r, err := rt.getAdja(next)
	return r, through, err
}

func (rt *table) getAdja(addr IPv4) (cutevpn.Route, error) {
	proutes, ok := rt.adja[addr]
	if !ok {
		return cutevpn.Route{}, cutevpn.ErrNoRoute
	}
	routes := *proutes
	r := routes[0].R
	routes[0].current += routes[0].Metric
	heap.Fix(&routes, 0)
	return r, nil
}

func (rt *table) getShortest(addr IPv4) (cutevpn.Route, error) {
	next, ok := rt.shortest[addr]
	if !ok {
		return cutevpn.Route{}, cutevpn.ErrNoRoute
	}
	return rt.getAdja(next)
}

func calcShortest(selfIP IPv4, boot uint64, states map[IPv4]*linkState) map[IPv4]IPv4 {
	shortest := shortests(selfIP, emptyIPv4, states)
	delete(shortest, selfIP)
	r := make(map[IPv4]IPv4)
	for dst, p := range shortest {
		r[dst] = p.Nodes[1]
	}
	return r
}

func calcBalance(selfIP IPv4, adjacents map[IPv4]*adjacent, states map[IPv4]*linkState) map[IPv4]*routeHeap {
	balance := make(map[IPv4]*routeHeap)
	selfLinkState, ok := states[selfIP]
	if !ok {
		return balance
	}
	shortestsFromAdja := make(map[IPv4]map[IPv4]path)
	for adjaIP := range adjacents {
		shortestsFromAdja[adjaIP] = shortests(adjaIP, selfIP, states)
	}
	for neighIP := range states {
		if neighIP == selfIP {
			continue
		}
		var routes routeHeap
		for adjaIP, paths := range shortestsFromAdja {
			metric0 := selfLinkState.msg.State[adjaIP]
			through := adjaIP
			if neighIP != adjaIP {
				path, ok := paths[neighIP]
				if !ok {
					continue
				}
				through = path.Nodes[len(path.Nodes)-2]
			}
			metric := paths[neighIP].D
			routes = append(routes, &routeWithMetric{
				Next:    adjaIP,
				Through: through,
				Metric:  metric + metric0,
				current: metric + metric0,
			})
		}
		if len(routes) > 0 {
			routes.cut(1425) // min * 1.39
			heap.Init(&routes)
			balance[neighIP] = &routes
		}
	}
	return balance
}

func (rt *table) Update(selfIP IPv4, boot uint64, adjacents map[IPv4]*adjacent, states map[IPv4]*linkState) {
	adjaRoutes := make(map[IPv4]*routeHeap)
	for ip, adja := range adjacents {
		routes := adja.GetRoutes()
		heap.Init(&routes)
		adjaRoutes[ip] = &routes
	}

	shortest := calcShortest(selfIP, boot, states)
	balance := calcBalance(selfIP, adjacents, states)
	rt.Lock()
	rt.shortest = shortest
	rt.adja = adjaRoutes
	rt.balance = balance
	rt.Unlock()
}

type routeWithMetric struct {
	R       cutevpn.Route
	Next    IPv4
	Through IPv4
	Metric  uint64
	current uint64
}

func (r *routeWithMetric) String() string {
	if r.R.IsEmpty() {
		return fmt.Sprintf("next %v through %v, %v", r.Next, r.Through, time.Duration(r.Metric))
	}
	return fmt.Sprintf("%v %v", r.R, time.Duration(r.Metric))
}

func (r *routeWithMetric) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

type routeHeap []*routeWithMetric

func (h routeHeap) Len() int           { return len(h) }
func (h routeHeap) Less(i, j int) bool { return h[i].current < h[j].current }
func (h routeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *routeHeap) Push(x interface{}) {
	panic("won't push")
}
func (h *routeHeap) Pop() interface{} {
	panic("won't pop")
}

func (h *routeHeap) cut(mul uint64) {
	sort.Sort(h)
	routes := *h
	threshold := routes[0].Metric * mul / 1024
	for i := 1; i < len(routes); i++ {
		if routes[i].Metric > threshold {
			routes = routes[:i]
			break
		}
	}
	*h = routes
}

var emptyIPv4 IPv4
