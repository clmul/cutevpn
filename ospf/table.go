package ospf

import (
	"container/heap"
	"sync"

	"github.com/clmul/cutevpn"
)

type table struct {
	sync.Mutex
	adjaRoutes map[cutevpn.IPv4]RouteHeap
	routes     map[cutevpn.IPv4]cutevpn.IPv4
}

func (ospf *OSPF) Get(dst cutevpn.IPv4) (cutevpn.Route, error) {
	return ospf.routes.get(dst, false)
}

func newRouteTable() *table {
	rt := &table{
		adjaRoutes: make(map[cutevpn.IPv4]RouteHeap),
		routes:     make(map[cutevpn.IPv4]cutevpn.IPv4),
	}
	return rt
}

func (rt *table) get(addr cutevpn.IPv4, onlyAdja bool) (cutevpn.Route, error) {
	rt.Lock()
	defer rt.Unlock()
	if !onlyAdja {
		next, ok := rt.routes[addr]
		if ok {
			addr = next
		}
	}
	routes, ok := rt.adjaRoutes[addr]
	if !ok {
		return cutevpn.Route{}, cutevpn.NoRoute
	}
	r := routes[0].R
	routes[0].current += routes[0].Metric
	heap.Fix(&routes, 0)
	return r, nil
}

func (rt *table) Update(adjaRoutes map[cutevpn.IPv4]RouteHeap, routes map[cutevpn.IPv4]cutevpn.IPv4) {
	for _, routeHeap := range adjaRoutes {
		for _, r := range routeHeap {
			r.current = r.Metric
		}
		heap.Init(&routeHeap)
	}
	rt.Lock()
	rt.routes = routes
	rt.adjaRoutes = adjaRoutes
	rt.Unlock()
}

type RouteWithMetric struct {
	R       cutevpn.Route
	Metric  uint64
	current uint64
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
