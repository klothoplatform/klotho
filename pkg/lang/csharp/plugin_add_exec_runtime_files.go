package csharp

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
)

type (
	AddExecRuntimeFiles struct {
		runtime Runtime
		cfg     *config.Application
	}
)

func (p *AddExecRuntimeFiles) Name() string { return "AddExecRuntimeFiles:CSharp" }

func (p *AddExecRuntimeFiles) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !(ok && unit.HasSourceFilesFor(CSharp)) {
			continue
		}
		errs.Append(p.runtime.AddExecRuntimeFiles(unit))
	}
	return errs.ErrOrNil()
}
