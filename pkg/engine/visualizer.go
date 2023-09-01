package engine

import (
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/visualizer"
)

func (e *Engine) VisualizeViews() ([]klotho_io.File, error) {
	iac_topo := &visualizer.File{
		FilenamePrefix: "iac-",
		AppName:        e.Context.AppName,
		Provider:       "aws",
		DAG:            e.Context.Solution.ResourceGraph,
	}
	dataflow_topo := &visualizer.File{
		FilenamePrefix: "dataflow-",
		AppName:        e.Context.AppName,
		Provider:       "aws",
		DAG:            e.GetDataFlowDag(),
	}
	return []klotho_io.File{iac_topo, dataflow_topo}, nil
}
