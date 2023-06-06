package javascript

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

// UnitFileDependencyResolver resolves the execunit.FileDependencies for the provided core.ExecutionUnit.
func UnitFileDependencyResolver(unit *core.ExecutionUnit) (core.FileDependencies, error) {
	return ResolveFileDependencies(unit.Files())
}

func ResolveFileDependencies(fs map[string]core.File) (core.FileDependencies, error) {
	fileDeps := make(core.FileDependencies)
	for _, f := range fs {
		jsF, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}
		dependencies, err := localImports(fs, jsF)
		if err != nil {
			return nil, err
		}
		fileDeps[f.Path()] = dependencies
	}

	return fileDeps, nil
}

func localImports(input map[string]core.File, f *core.SourceFile) (core.Imported, error) {
	imports := make(core.Imported)
	var errs multierr.Error

	localImports := FindImportsInFile(f).Filter(filter.NewSimpleFilter(IsRelativeImport))

	for _, imp := range localImports {
		src := imp.Source
		importFile, err := FindFileForImport(input, f.Path(), src)
		if err != nil {
			errs.Append(core.WrapErrf(err, "failed to find file for module %s", src))
			continue
		} else if importFile == nil {
			// Debug rather than Warn since importFile may be part of a different execution unit
			zap.L().With(logging.FileField(f)).Sugar().Debugf("failed to find file for module %s", src)
			continue
		}

		uses := ImportUsageQuery(f.Tree().RootNode(), imp.ImportedAs())

		useNames := make(core.References)
		for _, use := range uses {
			name := use.Content()
			useNames.Add(name)
		}
		refs, ok := imports[importFile.Path()]
		if !ok {
			refs = make(core.References)
			imports[importFile.Path()] = refs
		}
		for name := range useNames {
			refs.Add(name)
		}
	}

	return imports, errs.ErrOrNil()
}
