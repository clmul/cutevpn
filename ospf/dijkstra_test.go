package ospf

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindShortestPaths(t *testing.T) {
	graph := map[IPv4]map[IPv4]uint64{
		IPv4{0, 0, 0, 0}: {
			IPv4{0, 0, 0, 1}: 1,
			IPv4{0, 0, 0, 2}: 10,
		},
		IPv4{0, 0, 0, 1}: {
			IPv4{0, 0, 0, 0}: 1,
			IPv4{0, 0, 0, 2}: 2,
			IPv4{0, 0, 0, 3}: 20,
		},
		IPv4{0, 0, 0, 2}: {
			IPv4{0, 0, 0, 0}: 10,
			IPv4{0, 0, 0, 1}: 2,
			IPv4{0, 0, 0, 3}: 20,
		},
		IPv4{0, 0, 0, 3}: {
			IPv4{0, 0, 0, 1}:  20,
			IPv4{0, 0, 0, 2}:  20,
			IPv4{0, 0, 0, 20}: 20,
		},
		IPv4{9, 9, 9, 9}: {
			IPv4{8, 8, 8, 8}: 1,
		},
		IPv4{8, 8, 8, 8}: {
			IPv4{9, 9, 9, 9}: 1,
		},
	}
	from := IPv4{0, 0, 0, 0}
	answer := map[IPv4]path{
		IPv4{0, 0, 0, 0}: {Nodes: []IPv4{{0, 0, 0, 0}}, D: 0},
		IPv4{0, 0, 0, 1}: {Nodes: []IPv4{{0, 0, 0, 0}, {0, 0, 0, 1}}, D: 1},
		IPv4{0, 0, 0, 2}: {Nodes: []IPv4{{0, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 0, 2}}, D: 3},
		IPv4{0, 0, 0, 3}: {Nodes: []IPv4{{0, 0, 0, 0}, {0, 0, 0, 1}, {0, 0, 0, 3}}, D: 21},
	}
	result := dijkstra(from, graph)
	if diff := cmp.Diff(result, answer); diff != "" {
		t.Errorf("\n%s", diff)
	}
}
