package ospf

import (
	"log"

	"github.com/clmul/cutevpn"
	"github.com/kirves/godijkstra/common/structs"
	"github.com/kirves/godijkstra/dijkstra"
)

type graph map[cutevpn.IPv4]*linkState

func (g graph) SuccessorsForNode(node string) []dijkstrastructs.Connection {
	var ip cutevpn.IPv4
	copy(ip[:], node)
	var result []dijkstrastructs.Connection
	if _, ok := g[ip]; !ok {
		return nil
	}
	for dst, metric := range g[ip].msg.State {
		result = append(result, dijkstrastructs.Connection{
			Destination: string(dst[:]),
			// TODO: 3e9 nanoseconds overflow int on 32-bit machines
			Weight: int(metric),
		})
	}
	return result
}

func (g graph) PredecessorsFromNode(node string) []dijkstrastructs.Connection {
	return nil
}

func (g graph) EdgeWeight(n1, n2 string) int {
	var ip1, ip2 cutevpn.IPv4
	copy(ip1[:], n1)
	copy(ip2[:], n2)
	return int(g[ip1].msg.State[ip2])

}

func findPaths(self cutevpn.IPv4, neighbors graph) map[cutevpn.IPv4]cutevpn.IPv4 {
	result := make(map[cutevpn.IPv4]cutevpn.IPv4)

	for dst := range neighbors {
		if dst == self {
			continue
		}
		path, ok := dijkstra.SearchPath(neighbors, string(self[:]), string(dst[:]), dijkstra.VANILLA)
		if !ok {
			log.Printf("no path to %v", dst)
			continue
		}
		var next cutevpn.IPv4
		copy(next[:], path.Path[1].Node)
		result[dst] = next
	}
	return result
}
