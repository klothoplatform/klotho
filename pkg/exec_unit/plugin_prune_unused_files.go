package execunit

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type (
	// PruneUncategorizedFiles is a plugin that performs tree-shaking on each core.ExecutionUnit in the current compilation context.
	PruneUncategorizedFiles struct {
	}
)

func (PruneUncategorizedFiles) Name() string {
	return "prune_uncategorized_files"
}

func (p PruneUncategorizedFiles) Transform(result *core.CompilationResult, _ *core.Dependencies) error {
	log := zap.L().Sugar()

	units := result.GetResourcesOfType(core.ExecutionUnitKind)
	for _, unitR := range units {
		unit, ok := unitR.(*core.ExecutionUnit)
		if !ok {
			continue
		}

		count := 0
		for path := range unit.Files() {
			_, isResource := unit.Executable.Resources[path]
			_, isStaticAsset := unit.Executable.StaticAssets[path]
			_, isSourceFile := unit.Executable.SourceFiles[path]

			if isResource || isStaticAsset || isSourceFile {
				continue
			}

			unit.Remove(path)
			count++
			log.Debugf("Removed file: '%s' from execution unit: %s", path, unit.Name)
		}
		log.Debugf("Removed %d uncategorized files from execution unit: %s", count, unit.Name)
	}
	return nil
}
