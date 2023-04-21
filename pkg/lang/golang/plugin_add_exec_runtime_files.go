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

func (p *AddExecRuntimeFiles) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {
	var errs multierr.Error
	for _, unit := range core.GetConstructsOfType[*core.ExecutionUnit](constructGraph) {
		if !unit.HasSourceFilesFor(goLang) {
			continue
		}

		errs.Append(p.runtime.AddExecRuntimeFiles(unit, constructGraph))
	}

	return errs.ErrOrNil()
}
