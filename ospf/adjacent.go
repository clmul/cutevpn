package ospf

import "github.com/clmul/cutevpn"

type adjacent struct {
	BootTime uint64
	Metric   uint64
	Routes   map[cutevpn.Route]*metric
}

func newAdjacent() *adjacent {
	return &adjacent{
		Metric: uint64(MaxMetric),
		Routes: make(map[cutevpn.Route]*metric),
	}
}

func (a *adjacent) GetRoutes() (routes RouteHeap) {
	for r, m := range a.Routes {
		routes = append(routes, &RouteWithMetric{
			R:      r,
			Metric: m.Value(),
		})
	}
	return routes
}

func (a *adjacent) Update(route cutevpn.Route, rtt uint64) bool {
	m, ok := a.Routes[route]
	if !ok {
		m = &metric{}
		a.Routes[route] = m
	}
	m.Push(rtt)
	return a.updateMetric()
}

func (a *adjacent) GetMinMetricAndDeleteDeadRoute() uint64 {
	var min = uint64(MaxMetric)
	for r, m := range a.Routes {
		v := m.Value()
		if v == uint64(MaxMetric) {
			delete(a.Routes, r)
			continue
		}
		if v < min {
			min = v
		}
	}
	return min
}

func (a *adjacent) updateMetric() bool {
	min := a.GetMinMetricAndDeleteDeadRoute()
	if diff(min, a.Metric)*100/a.Metric > UpdateThreshold {
		a.Metric = min
		return true
	}
	return false
}

func diff(x, y uint64) uint64 {
	if x > y {
		return x - y
	}
	return y - x
}
