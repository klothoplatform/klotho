package property_eval

import (
	"fmt"

	"github.com/dominikbraun/graph"
)

func PrintGraph(g Graph) {
	nodes, err := graph.TopologicalSort(g)
	if err != nil {
		panic(err)
	}

	adj, err := g.AdjacencyMap()
	if err != nil {
		panic(err)
	}

	for _, node := range nodes {
		fmt.Println(node)
		for adj := range adj[node] {
			fmt.Printf("  -> %s\n", adj)
		}
	}
}
