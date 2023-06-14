package engine

import (
	"net/http"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/visualizer"
)

func (e *Engine) VisualizeViews() ([]core.File, error) {
	var outputFiles []core.File
	topology := visualizer.Plugin{Client: http.DefaultClient, AppName: e.Context.AppName, Provider: e.Provider.Name()}
	files, err := topology.Generate(e.Context.EndState, "iac")
	if err != nil {
		return outputFiles, err
	}
	outputFiles = append(outputFiles, files...)
	dag := e.GetDataFlowDag()
	files, err = topology.Generate(dag, "dataflow")
	if err != nil {
		return outputFiles, err
	}
	outputFiles = append(outputFiles, files...)
	return outputFiles, nil
}
