package engine

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/visualizer"
)

func (e *Engine) VisualizeViews() ([]core.File, error) {
	iac_topo := &visualizer.File{
		FilenamePrefix: "iac-",
		AppName:        e.Context.AppName,
		Provider:       "aws",
		DAG:            e.Context.EndState,
	}
	dataflow_topo := &visualizer.File{
		FilenamePrefix: "dataflow-",
		AppName:        e.Context.AppName,
		Provider:       "aws",
		DAG:            e.GetDataFlowDag(),
	}
	return []core.File{iac_topo, dataflow_topo}, nil
}
