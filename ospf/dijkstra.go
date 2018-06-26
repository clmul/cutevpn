package ospf

import (
	"github.com/clmul/cutevpn"
	"github.com/kirves/godijkstra/common/structs"
	"github.com/kirves/godijkstra/dijkstra"
)

type graph struct {
	m    map[cutevpn.IPv4]*linkState
	self cutevpn.IPv4
}

func (g graph) SuccessorsForNode(node string) []dijkstrastructs.Connection {
	var ip cutevpn.IPv4
	copy(ip[:], node)
	var result []dijkstrastructs.Connection
	if _, ok := g.m[ip]; !ok {
		return nil
	}
	for dst, metric := range g.m[ip].msg.State {
		if dst == g.self {
			continue
		}
		result = append(result, dijkstrastructs.Connection{
			Destination: string(dst[:]),
			// TODO: 3e9 nanoseconds overflow int on 32-bit machines
			Weight: int(metric),
		})
	}
	return result
}

func (g graph) PredecessorsFromNode(node string) []dijkstrastructs.Connection {
	panic("unused")
}

func (g graph) EdgeWeight(n1, n2 string) int {
	panic("unused")
}

func findShortest(self, from, to cutevpn.IPv4, neighbors map[cutevpn.IPv4]*linkState) (zero, one, last cutevpn.IPv4, metric uint64) {
	g := graph{m: neighbors, self: self}
	path, ok := dijkstra.SearchPath(g, string(from[:]), string(to[:]), dijkstra.VANILLA)
	if !ok {
		return cutevpn.EmptyIPv4, cutevpn.EmptyIPv4, cutevpn.EmptyIPv4, 0
	}
	copy(zero[:], path.Path[0].Node)
	copy(one[:], path.Path[1].Node)
	copy(last[:], path.Path[len(path.Path)-2].Node)
	return zero, one, last, uint64(path.Weight)
}
