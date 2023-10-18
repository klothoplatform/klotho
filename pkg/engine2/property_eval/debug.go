package property_eval

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"go.uber.org/zap"
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

func (eval *PropertyEval) writeGraph(filename string) {
	f, err := os.Create(filename + ".gv")
	if err != nil {
		zap.S().Errorf("could not create file %s: %v", filename, err)
	}

	var extraStatements []string
	for rank, keys := range eval.evaluatedOrder {
		sb := new(strings.Builder)
		sb.WriteString("{rank = same; ")
		for _, key := range keys {
			sb.WriteString(fmt.Sprintf(`"%s"; `, key))

			_, props, err := eval.graph.VertexWithProperties(key)
			if err != nil {
				zap.S().Errorf("could not get vertex with properties: %v", err)
				continue
			}
			if key.Ref.Resource.IsZero() {
				props.Attributes["label"] = fmt.Sprintf(`%s\n→ %s`, key.Edge.Source, key.Edge.Target)
				props.Attributes["shape"] = "parallelogram"
			} else {
				props.Attributes["label"] = fmt.Sprintf(`%s\n%s`, key.Ref.Resource, key.Ref.Property)
				props.Attributes["shape"] = "box"
			}
			props.Attributes["group"] = fmt.Sprintf("rank%d", rank)
		}
		sb.WriteString("}")
		extraStatements = append(extraStatements, sb.String())
	}

	err = draw.DOT(eval.graph, f, func(d *draw.Description) {
		d.Attributes["rankdir"] = "BT"
		d.ExtraStatements = extraStatements
	})
	if err != nil {
		zap.S().Errorf("could not render graph to file %s: %v", filename, err)
	}
	f.Close()

	cmd := exec.Command("dot", "-Tsvg", filename+".gv", "-O")
	err = cmd.Run()
	if err != nil {
		zap.S().Errorf("could not run 'dot': %v", err)
	}
}
