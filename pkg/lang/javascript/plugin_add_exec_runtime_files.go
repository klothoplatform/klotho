package javascript

import (
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/multierr"
)

type (
	AddExecRuntimeFiles struct {
		runtime Runtime
	}
)

func (p AddExecRuntimeFiles) Name() string { return "AddExecRuntimeFiles:JavaScript" }

func (p AddExecRuntimeFiles) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {

	var errs multierr.Error
	for _, unit := range construct.GetConstructsOfType[*types.ExecutionUnit](constructGraph) {
		if !unit.HasSourceFilesFor(Language.ID) {
			continue
		}

		errs.Append(p.runtime.AddExecRuntimeFiles(unit, constructGraph))
	}

	return errs.ErrOrNil()
}
