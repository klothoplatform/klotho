package execunit

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
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

func (p PruneUncategorizedFiles) Transform(input *core.InputFiles, constructGraph *graph.Directed[core.Construct]) error {
	log := zap.L().Sugar()

	units := core.GetResourcesOfType[*core.ExecutionUnit](constructGraph)
	for _, unit := range units {
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
			log.Debugf("Removed file: '%s' from execution unit: %s", path, unit.ID)
		}
		log.Debugf("Removed %d uncategorized files from execution unit: %s", count, unit.ID)
	}
	return nil
}
