package ospf

type path struct {
	Nodes []IPv4
	D     uint64
}

func copyLinkState(states map[IPv4]*linkState, without IPv4) map[IPv4]map[IPv4]uint64 {
	neighbors := make(map[IPv4]map[IPv4]uint64, len(states))
	for ip, state := range states {
		if ip != without {
			neighbors[ip] = state.msg.State
		}
	}
	return neighbors
}

func shortests(from, without IPv4, states map[IPv4]*linkState) map[IPv4]path {
	neighbors := copyLinkState(states, without)
	return dijkstra(from, neighbors)
}

func dijkstra(from IPv4, graph map[IPv4]map[IPv4]uint64) map[IPv4]path {
	distances := make(map[IPv4]path)
	distances[from] = path{Nodes: []IPv4{from}, D: 0}
	result := make(map[IPv4]path)
	for len(distances) != 0 {
		visit(from, distances, graph)
		result[from] = distances[from]
		delete(distances, from)
		delete(graph, from)

		var min = ^uint64(0)
		for peer, path := range distances {
			if path.D < min {
				from = peer
				min = path.D
			}
		}
	}
	return result
}

func visit(current IPv4, distances map[IPv4]path, graph map[IPv4]map[IPv4]uint64) {
	p := distances[current]
	edges := graph[current]
	d0 := p.D
	nodes0 := p.Nodes
	for adja, d := range edges {
		_, ok := graph[adja]
		if !ok {
			continue
		}
		d1 := d0 + d
		before, ok := distances[adja]
		if !ok || before.D > d1 {
			distances[adja] = path{Nodes: append(nodes0, adja), D: d1}
		}
	}
}
