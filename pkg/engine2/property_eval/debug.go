package property_eval

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/klothoplatform/klotho/pkg/set"
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

	setAttributes := func(key EvaluationKey, attribs map[string]string, rank int) {
		if !key.Ref.Resource.IsZero() {
			attribs["label"] = fmt.Sprintf(`(%d) %s\n%s`, rank, key.Ref.Resource, key.Ref.Property)
			attribs["shape"] = "box"
		} else if key.GraphState != "" {
			attribs["label"] = fmt.Sprintf(`(%d) %s`, rank, key.GraphState)
			attribs["shape"] = "box"
			attribs["style"] = "dashed"
		} else {
			attribs["label"] = fmt.Sprintf(`(%d) %s\n→ %s`, rank, key.Edge.Source, key.Edge.Target)
			attribs["shape"] = "parallelogram"
		}
		// attribs["group"] = fmt.Sprintf("rank%d", rank)
	}

	var extraStatements []string
	evaluated := make(set.Set[EvaluationKey])
	for rank, keys := range eval.evaluatedOrder {
		extraStatements = append(extraStatements,
			fmt.Sprintf(`"eval%d" [style="invis"]`, rank),
		)
		if rank > 0 {
			extraStatements = append(extraStatements,
				fmt.Sprintf(`"eval%d" -> "eval%d" [style="invis"]`, rank, rank-1),
			)
		}
		sb := new(strings.Builder)
		sb.WriteString("{rank = same; ")
		fmt.Fprintf(sb, `"eval%d"; `, rank)
		for _, key := range keys {
			sb.WriteString(fmt.Sprintf(`"%s"; `, key))
			evaluated.Add(key)

			_, props, err := eval.graph.VertexWithProperties(key)
			if err != nil {
				zap.S().Errorf("could not get vertex with properties: %v", err)
				continue
			}
			setAttributes(key, props.Attributes, rank)
		}
		sb.WriteString("}")
		extraStatements = append(extraStatements, sb.String())
	}

	topo, err := graph.TopologicalSort(eval.graph)
	if err != nil {
		zap.S().Errorf("could not get topological sort: %v", err)
		return
	}
	for _, key := range topo {
		if evaluated.Contains(key) {
			continue
		}
		_, props, err := eval.graph.VertexWithProperties(key)
		if err != nil {
			zap.S().Errorf("could not get vertex with properties: %v", err)
			continue
		}

		setAttributes(key, props.Attributes, 999)
		props.Attributes["style"] = "filled"
		props.Attributes["fillcolor"] = "#e87b7b"
		// props.Attributes["group"] = "unevaluated"
	}

	err = draw.DOT(eval.graph, f, func(d *draw.Description) {
		d.Attributes["rankdir"] = "BT"
		d.Attributes["ranksep"] = "1"
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
