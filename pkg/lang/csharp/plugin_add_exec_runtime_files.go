package csharp

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/multierr"
)

type (
	AddExecRuntimeFiles struct {
		runtime Runtime
		cfg     *config.Application
	}
)

func (p *AddExecRuntimeFiles) Name() string { return "AddExecRuntimeFiles:CSharp" }

func (p *AddExecRuntimeFiles) Transform(input *core.InputFiles, constructGraph *graph.Directed[core.Construct]) error {
	var errs multierr.Error
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](constructGraph) {
		if !unit.HasSourceFilesFor(CSharp) {
			continue
		}
		errs.Append(p.runtime.AddExecRuntimeFiles(unit, constructGraph))
	}
	return errs.ErrOrNil()
}
