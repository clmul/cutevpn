package ospf

import (
	"encoding/json"
	"time"

	"github.com/clmul/cutevpn"
)

type adjacent struct {
	BootTime uint64
	Metric   uint64
	Routes   map[cutevpn.Route]*metric
}

func (a *adjacent) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"BootTime": time.Unix(0, int64(a.BootTime)).In(time.UTC),
		"Metric":   time.Duration(a.Metric).String(),
		"Routes":   a.Routes,
	}
	return json.Marshal(data)
}

func newAdjacent() *adjacent {
	return &adjacent{
		Metric: uint64(MaxMetric),
		Routes: make(map[cutevpn.Route]*metric),
	}
}

func (a *adjacent) GetRoutes() (routes RouteHeap) {
	for r, m := range a.Routes {
		metric := m.Value()
		routes = append(routes, &RouteWithMetric{
			R:       r,
			Metric:  metric,
			current: metric,
		})
	}
	routes.cut(1425) // min * 1.39
	return routes
}

func (a *adjacent) Update(route cutevpn.Route, rtt uint64) bool {
	m, ok := a.Routes[route]
	if !ok {
		m = &metric{}
		a.Routes[route] = m
	}
	m.Push(rtt)
	return a.UpdateMetric()
}

func (a *adjacent) getMinMetricAndDeleteDeadRoute() uint64 {
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

func (a *adjacent) UpdateMetric() bool {
	min := a.getMinMetricAndDeleteDeadRoute()
	if diff(min, a.Metric)*128/a.Metric > UpdateThreshold {
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
