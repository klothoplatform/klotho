package property_eval

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/google/pprof/third_party/svgpan"
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
	defer f.Close()

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
		for key := range keys {
			sb.WriteString(fmt.Sprintf(`"%s"; `, key))
			evaluated.Add(key)

			_, props, err := eval.graph.VertexWithProperties(key)
			if err != nil {
				zap.S().Errorf("error getting vertex properties for property eval debug output: %v", err)
				continue
			}
			setAttributes(key, props.Attributes, rank)
		}
		sb.WriteString("}")
		extraStatements = append(extraStatements, sb.String())
	}

	// don't trust eval.unevaluated because if there's anything here, it means something went wrong, which could
	// have impacted what the unevaluated graph contains.
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
			zap.S().Errorf("error getting unevaluated vertex properties for property eval debug output: %v", err)
			continue
		}

		setAttributes(key, props.Attributes, 999)
		props.Attributes["style"] = "filled"
		props.Attributes["fillcolor"] = "#e87b7b"
	}

	dotContent := new(bytes.Buffer)
	err = draw.DOT(eval.graph, io.MultiWriter(f, dotContent), func(d *draw.Description) {
		d.Attributes["rankdir"] = "BT"
		d.Attributes["ranksep"] = "1"
		d.ExtraStatements = extraStatements
	})
	if err != nil {
		zap.S().Errorf("could not render graph to file %s: %v", filename, err)
		return
	}

	svgContent := new(bytes.Buffer)
	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = dotContent
	cmd.Stdout = svgContent
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		zap.S().Errorf("could not run 'dot': %v", err)
		return
	}

	svgFile, err := os.Create(filename + ".gv.svg")
	if err != nil {
		zap.S().Errorf("could not create file %s: %v", filename, err)
		return
	}
	defer svgFile.Close()
	fmt.Fprint(svgFile, massageSVG(svgContent.String()))
}

// THe following adds SVG pan to the SVG output from DOT, taken from
// https://github.com/google/pprof/blob/main/internal/driver/svg.go

var (
	viewBox  = regexp.MustCompile(`<svg\s*width="[^"]+"\s*height="[^"]+"\s*viewBox="[^"]+"`)
	graphID  = regexp.MustCompile(`<g id="graph\d"`)
	svgClose = regexp.MustCompile(`</svg>`)
)

// massageSVG enhances the SVG output from DOT to provide better
// panning inside a web browser. It uses the svgpan library, which is
// embedded into the svgpan.JSSource variable.
func massageSVG(svg string) string {
	// Work around for dot bug which misses quoting some ampersands,
	// resulting on unparsable SVG.
	svg = strings.Replace(svg, "&;", "&amp;;", -1)

	// Dot's SVG output is
	//
	//    <svg width="___" height="___"
	//     viewBox="___" xmlns=...>
	//    <g id="graph0" transform="...">
	//    ...
	//    </g>
	//    </svg>
	//
	// Change it to
	//
	//    <svg width="100%" height="100%"
	//     xmlns=...>

	//    <script type="text/ecmascript"><![CDATA[` ..$(svgpan.JSSource)... `]]></script>`
	//    <g id="viewport" transform="translate(0,0)">
	//    <g id="graph0" transform="...">
	//    ...
	//    </g>
	//    </g>
	//    </svg>

	if loc := viewBox.FindStringIndex(svg); loc != nil {
		svg = svg[:loc[0]] +
			`<svg width="100%" height="100%"` +
			svg[loc[1]:]
	} else {
		zap.S().Warn("could not find viewBox in SVG")
	}

	if loc := graphID.FindStringIndex(svg); loc != nil {
		svg = svg[:loc[0]] +
			`<script type="text/ecmascript"><![CDATA[` + svgpan.JSSource + `]]></script>` +
			`<g id="viewport" transform="scale(0.5,0.5) translate(0,0)">` +
			svg[loc[0]:]
	} else {
		zap.S().Warn("could not find graph ID in SVG")
	}

	if loc := svgClose.FindStringIndex(svg); loc != nil {
		svg = svg[:loc[0]] +
			`</g>` +
			svg[loc[0]:]
	} else {
		zap.S().Warn("could not find svgClose in SVG")
	}

	return svg
}
