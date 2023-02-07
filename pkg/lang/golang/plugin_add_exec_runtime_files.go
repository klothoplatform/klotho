package golang

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

func (p *AddExecRuntimeFiles) Name() string { return "AddExecRuntimeFiles:Go" }

func (p *AddExecRuntimeFiles) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		if !unit.HasSourceFilesFor(goLang) {
			continue
		}

		errs.Append(p.runtime.AddExecRuntimeFiles(unit, result, deps))
	}

	return errs.ErrOrNil()
}
