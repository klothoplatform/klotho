package operational_eval

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/dot"
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
	if debugDir := os.Getenv("KLOTHO_DEBUG_DIR"); debugDir != "" {
		prefix = filepath.Join(debugDir, prefix)
	}
	if err := os.MkdirAll(filepath.Dir(prefix), 0755); err != nil {
		zap.S().Errorf("could not create debug directory %s: %v", filepath.Dir(prefix), err)
		return
	}
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

	svgContent, err := dot.ExecPan(bytes.NewReader(dotContent.Bytes()))
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
	fmt.Fprint(svgFile, svgContent)
}
