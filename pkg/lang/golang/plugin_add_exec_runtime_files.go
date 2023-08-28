package golang

import (
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/multierr"
)

type (
	AddExecRuntimeFiles struct {
		runtime Runtime
		cfg     *config.Application
	}
)

func (p *AddExecRuntimeFiles) Name() string { return "AddExecRuntimeFiles:Go" }

func (p *AddExecRuntimeFiles) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
	var errs multierr.Error
	for _, unit := range construct.GetConstructsOfType[*types.ExecutionUnit](constructGraph) {
		if !unit.HasSourceFilesFor(goLang) {
			continue
		}

		errs.Append(p.runtime.AddExecRuntimeFiles(unit, constructGraph))
	}

	return errs.ErrOrNil()
}
