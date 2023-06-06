package execunit

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	UnitFileDependencyResolver func(unit *core.ExecutionUnit) (core.FileDependencies, error)

	SourceFilesResolver struct {
		UnitFileDependencyResolver UnitFileDependencyResolver
		UpstreamAnnotations        []string
	}
)

func (resolver SourceFilesResolver) Resolve(unit *core.ExecutionUnit) (map[string]struct{}, error) {

	var entrypoints []string
	for entrypoint := range unit.Executable.Entrypoints {
		entrypoints = append(entrypoints, entrypoint)
	}

	// Including the upstream check here ensures that the files containing annotations
	// representing upstream entrypoints relative to unit's existing entrypoints
	// are always included in the dependency graph.
	var upstreamFiles []string
	for _, fileR := range unit.Files() {
		for _, annotation := range resolver.UpstreamAnnotations {
			if upstreamFile, ok := fileR.(*core.SourceFile); ok && upstreamFile.IsAnnotatedWith(annotation) {
				upstreamDeps, err := resolver.resolveDependencies(unit, []string{upstreamFile.Path()})
				if err != nil {
					return nil, err
				}
				for _, entrypoint := range entrypoints {
					if _, dependsOnEntrypoint := upstreamDeps[entrypoint]; dependsOnEntrypoint {
						upstreamFiles = append(upstreamFiles, upstreamFile.Path())
					}
				}
			}
		}
	}

	return resolver.resolveDependencies(unit, append(entrypoints, upstreamFiles...))
}

func (resolver SourceFilesResolver) resolveDependencies(unit *core.ExecutionUnit, entrypoints []string) (map[string]struct{}, error) {
	fileDeps, err := resolver.UnitFileDependencyResolver(unit)
	if err != nil {
		return nil, err
	}
	resolvedDeps := map[string]struct{}{}

	for _, entrypoint := range entrypoints {
		resolvedDeps[entrypoint] = struct{}{}
	}

	for _, fileR := range unit.Files() {
		if _, ok := resolvedDeps[fileR.Path()]; ok {
			continue
		}

		file := fileR

		var unprocessedDeps []string
		for dep := range resolvedDeps {
			unprocessedDeps = append(unprocessedDeps, dep)
		}

		processedDeps := make(map[string]struct{})
		for len(unprocessedDeps) > 0 {

			dep := unprocessedDeps[0]
			unprocessedDeps = unprocessedDeps[1:]
			imports := fileDeps[dep]

			for importedFilePath := range imports {
				if unit.Get(importedFilePath) == nil {
					return nil, fmt.Errorf("file '%s' imported by '%s' not found", importedFilePath, dep)
				}
				if file.Path() == importedFilePath {
					resolvedDeps[importedFilePath] = struct{}{}
				}
				// add any deps to the queue that haven't already been processed (avoiding circular imports)
				if _, ok := processedDeps[importedFilePath]; !ok {
					unprocessedDeps = append(unprocessedDeps, importedFilePath)
				}
			}
			processedDeps[dep] = struct{}{}
		}
	}
	return resolvedDeps, nil
}
