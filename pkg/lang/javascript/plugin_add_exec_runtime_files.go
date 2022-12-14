package javascript

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
)

type (
	AddExecRuntimeFiles struct {
		runtime Runtime
	}
)

func (p AddExecRuntimeFiles) Name() string { return "AddExecRuntimeFiles:JavaScript" }

func (p AddExecRuntimeFiles) Transform(result *core.CompilationResult, deps *core.Dependencies) error {

	var errs multierr.Error
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !ok {
			continue
		}
		if !unit.HasSourceFilesFor(Language.ID) {
			continue
		}

		errs.Append(p.runtime.AddExecRuntimeFiles(unit, result, deps))
	}

	return errs.ErrOrNil()
}
