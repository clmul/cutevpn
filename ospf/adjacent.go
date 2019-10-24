package ospf

import (
	"encoding/json"
	"time"

	"github.com/clmul/cutevpn"
)

type adjacent struct {
	BootTime uint64
	Metric   uint64
	Routes   map[cutevpn.Route]metric
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
		Metric: uint64(maxMetric),
		Routes: make(map[cutevpn.Route]metric),
	}
}

func (a *adjacent) GetRoutes() (routes routeHeap) {
	for r, m := range a.Routes {
		metric := m.Value()
		routes = append(routes, &routeWithMetric{
			R:       r,
			Metric:  metric,
			current: metric,
		})
	}
	routes.cut(1425) // min * 1.39
	return routes
}

func (a *adjacent) Update(route cutevpn.Route, rtt uint64) (newRoute, needUpdate bool) {
	_, ok := a.Routes[route]
	a.Routes[route] = a.Routes[route].Push(rtt)
	return !ok, a.UpdateMetric()
}

func (a *adjacent) getMinMetricAndDeleteDeadRoute() uint64 {
	var min = uint64(maxMetric)
	for r, m := range a.Routes {
		v := m.Value()
		if v == uint64(maxMetric) {
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
	old := a.Metric
	new := a.getMinMetricAndDeleteDeadRoute()
	if diff(old, new)*100 > old*updateThreshold {
		a.Metric = new
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
