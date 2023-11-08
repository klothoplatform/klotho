package operational_eval

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/dominikbraun/graph"
	"github.com/google/pprof/third_party/svgpan"
	"go.uber.org/zap"
)

const (
	attribAddedIn = "added_in"
	attribError   = "error"
	attribReady   = "ready"
	attribAddedBy = "added_by"
)

func PrintGraph(g Graph) {
	topo, err := graph.TopologicalSort(g)
	if err != nil {
		zap.S().Errorf("could not topologically sort graph: %v", err)
		return
	}
	adj, err := g.AdjacencyMap()
	if err != nil {
		zap.S().Errorf("could not get adjacency map: %v", err)
		return
	}
	for _, v := range topo {
		for dep := range adj[v] {
			fmt.Printf("-> %s\n", dep)
		}
	}
}

func (eval *Evaluator) writeGraph(prefix string) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		writeGraph(eval, prefix, graphToClusterDOT)
	}()
	go func() {
		defer wg.Done()
		writeGraph(eval, prefix+"_flat", graphToDOT)
	}()
	wg.Wait()
}

func writeGraph(eval *Evaluator, filename string, toDot func(*Evaluator, io.Writer) error) {
	f, err := os.Create(filename + ".gv")
	if err != nil {
		zap.S().Errorf("could not create file %s: %v", filename, err)
		return
	}
	defer f.Close()

	dotContent := new(bytes.Buffer)
	err = toDot(eval, io.MultiWriter(f, dotContent))
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
		zap.S().Errorf("could not run 'dot' for %s: %v", filename, err)
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
