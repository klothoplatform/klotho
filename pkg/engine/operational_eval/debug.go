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
	"github.com/klothoplatform/klotho/pkg/engine/debug"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	attribAddedIn  = "added_in"
	attribError    = "error"
	attribReady    = "ready"
	attribAddedBy  = "added_by"
	attribDuration = "duration"
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
	if debugDir := debug.GetDebugDir(eval.Solution.Context()); debugDir != "" {
		prefix = filepath.Join(debugDir, prefix)
	}
	log := logging.GetLogger(eval.Solution.Context()).Sugar()
	if err := os.MkdirAll(filepath.Dir(prefix), 0755); err != nil {
		log.Errorf("could not create debug directory %s: %v", filepath.Dir(prefix), err)
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
	log := logging.GetLogger(eval.Solution.Context()).Sugar()

	f, err := os.Create(filename + ".gv")
	if err != nil {
		log.Errorf("could not create file %s: %v", filename, err)
		return
	}
	defer f.Close()

	dotContent := new(bytes.Buffer)
	err = toDot(eval, io.MultiWriter(f, dotContent))
	if err != nil {
		log.Errorf("could not render graph to file %s: %v", filename, err)
		return
	}

	svgContent, err := dot.ExecPan(bytes.NewReader(dotContent.Bytes()))
	if err != nil {
		log.Errorf("could not run 'dot' for %s: %v", filename, err)
		return
	}

	svgFile, err := os.Create(filename + ".gv.svg")
	if err != nil {
		log.Errorf("could not create file %s: %v", filename, err)
		return
	}
	defer svgFile.Close()
	fmt.Fprint(svgFile, svgContent)
}

func (eval *Evaluator) writeExecOrder() {
	path := "exec-order.yaml"
	if debugDir := debug.GetDebugDir(eval.Solution.Context()); debugDir != "" {
		path = filepath.Join(debugDir, path)
	}
	log := logging.GetLogger(eval.Solution.Context()).Sugar()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Errorf("could not create debug directory %s: %v", filepath.Dir(path), err)
		return
	}

	f, err := os.Create(path)
	if err != nil {
		log.Errorf("could not create file %s: %v", path, err)
		return
	}
	defer f.Close()

	order := make([][]string, len(eval.evaluatedOrder))
	for i, group := range eval.evaluatedOrder {
		order[i] = make([]string, len(group))
		for j, key := range group {
			order[i][j] = key.String()
		}
	}

	err = yaml.NewEncoder(f).Encode(order)
	if err != nil {
		log.Errorf("could not write exec order to file %s: %v", path, err)
	}
}
