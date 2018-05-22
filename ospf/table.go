package ospf

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/clmul/cutevpn"
	"sort"
)

type table struct {
	sync.Mutex
	adja     map[cutevpn.IPv4]*RouteHeap
	balance  map[cutevpn.IPv4]*RouteHeap
	shortest map[cutevpn.IPv4]cutevpn.IPv4
}

func (ospf *OSPF) GetBalance(dst cutevpn.IPv4) (cutevpn.Route, cutevpn.IPv4, error) {
	ospf.routes.Lock()
	defer ospf.routes.Unlock()
	return ospf.routes.getBalance(dst)
}

func (ospf *OSPF) GetAdja(adja cutevpn.IPv4) (cutevpn.Route, error) {
	ospf.routes.Lock()
	defer ospf.routes.Unlock()
	return ospf.routes.getAdja(adja)
}

func (ospf *OSPF) GetShortest(through cutevpn.IPv4) (cutevpn.Route, error) {
	ospf.routes.Lock()
	defer ospf.routes.Unlock()
	return ospf.routes.getShortest(through)
}

func newRouteTable() *table {
	rt := &table{
		adja:     make(map[cutevpn.IPv4]*RouteHeap),
		balance:  make(map[cutevpn.IPv4]*RouteHeap),
		shortest: make(map[cutevpn.IPv4]cutevpn.IPv4),
	}
	return rt
}

func (rt *table) getBalance(addr cutevpn.IPv4) (cutevpn.Route, cutevpn.IPv4, error) {
	proutes, ok := rt.balance[addr]
	if !ok {
		return cutevpn.Route{}, cutevpn.EmptyIPv4, cutevpn.NoRoute
	}
	routes := *proutes
	next := routes[0].Next
	through := routes[0].Through
	routes[0].current += routes[0].Metric * routes[0].Metric
	heap.Fix(&routes, 0)
	r, err := rt.getAdja(next)
	return r, through, err
}

func (rt *table) getAdja(addr cutevpn.IPv4) (cutevpn.Route, error) {
	proutes, ok := rt.adja[addr]
	if !ok {
		return cutevpn.Route{}, cutevpn.NoRoute
	}
	routes := *proutes
	r := routes[0].R
	routes[0].current += routes[0].Metric
	heap.Fix(&routes, 0)
	return r, nil
}

func (rt *table) getShortest(addr cutevpn.IPv4) (cutevpn.Route, error) {
	next, ok := rt.shortest[addr]
	if !ok {
		return cutevpn.Route{}, cutevpn.NoRoute
	}
	return rt.getAdja(next)
}

func (rt *table) Update(selfIP cutevpn.IPv4, adjacents map[cutevpn.IPv4]*adjacent, neighbors map[cutevpn.IPv4]*linkState) {
	shortest := make(map[cutevpn.IPv4]cutevpn.IPv4)
	for neighIP := range neighbors {
		if selfIP == neighIP {
			continue
		}
		_, next, _, metric := findShortest(selfIP, selfIP, neighIP, neighbors)
		if metric != 0 {
			shortest[neighIP] = next
		} else {
			neigh := neighbors[neighIP]
			if nanotime()-neigh.msg.Version > uint64(RouterDeadInterval) {
				delete(neighbors, neighIP)
			}
		}
	}

	adjaRoutes := make(map[cutevpn.IPv4]*RouteHeap)
	for ip, adja := range adjacents {
		routes := adja.GetRoutes()
		heap.Init(&routes)
		adjaRoutes[ip] = &routes
	}

	balance := make(map[cutevpn.IPv4]*RouteHeap)
	for neighIP := range neighbors {
		var routes RouteHeap
		for adjaIP := range adjacents {
			metric0 := neighbors[selfIP].msg.State[adjaIP]
			if neighIP == adjaIP {
				routes = append(routes, &RouteWithMetric{
					Next:    adjaIP,
					Through: adjaIP,
					Metric:  metric0,
					current: metric0,
				})
				continue
			}
			next, _, last, metric := findShortest(selfIP, adjaIP, neighIP, neighbors)
			if metric == 0 {
				continue
			}
			routes = append(routes, &RouteWithMetric{
				Next:    next,
				Through: last,
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

	rt.Lock()
	rt.shortest = shortest
	rt.adja = adjaRoutes
	rt.balance = balance
	rt.Unlock()
}

type RouteWithMetric struct {
	R       cutevpn.Route
	Next    cutevpn.IPv4
	Through cutevpn.IPv4
	Metric  uint64
	current uint64
}

func (r *RouteWithMetric) String() string {
	if r.R.IsEmpty() {
		return fmt.Sprintf("next %v through %v, %v", r.Next, r.Through, time.Duration(r.Metric))
	}
	return fmt.Sprintf("%v %v", r.R, time.Duration(r.Metric))
}

func (r *RouteWithMetric) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

type RouteHeap []*RouteWithMetric

func (h RouteHeap) Len() int           { return len(h) }
func (h RouteHeap) Less(i, j int) bool { return h[i].current < h[j].current }
func (h RouteHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *RouteHeap) Push(x interface{}) {
	panic("won't push")
}
func (h *RouteHeap) Pop() interface{} {
	panic("won't pop")
}

func (h *RouteHeap) cut(mul uint64) {
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
