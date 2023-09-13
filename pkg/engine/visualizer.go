package engine

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/visualizer"
)

type (
	View string
	Tag  string
)

const (
	DataflowView View = "dataflow"
	IACView      View = "iac"

	ParentIconTag Tag = "parent"
	BigIconTag    Tag = "big"
	SmallIconTag  Tag = "small"
	NoRenderTag   Tag = "no-render"
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

func (e *Engine) GetResourceVizTag(view string, resource construct.Resource) Tag {
	template := e.GetTemplateForResource(resource)
	if template == nil {
		return NoRenderTag
	}
	tag, found := template.Views[view]
	if !found {
		return NoRenderTag
	}
	return Tag(tag)
}
