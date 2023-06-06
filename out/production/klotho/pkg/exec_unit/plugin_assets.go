package execunit

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type Assets struct{}

func (p Assets) Name() string { return "Assets" }

func (p Assets) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {
	units := make(map[string]*core.ExecutionUnit)
	for _, unit := range core.GetConstructsOfType[*core.ExecutionUnit](constructGraph) {
		units[unit.ID] = unit
	}

	var errs multierr.Error
	for _, f := range input.Files() {
		astF, ok := f.(*core.SourceFile)
		if !ok {
			continue
		}

		destUnit := core.FileExecUnitName(astF)

		for _, annot := range astF.Annotations() {
			if annot.Capability.Name != annotation.AssetCapability {
				continue
			}
			log := zap.L().With(logging.FileField(f), logging.AnnotationField(annot)).Sugar()

			includes, _ := annot.Capability.Directives.StringArray("include")
			excludes, _ := annot.Capability.Directives.StringArray("exclude")

			if len(includes) == 0 {
				errs.Append(core.NewCompilerError(astF, annot, errors.New("include directive must contain at least 1 path")))
				break
			}

			matcher, err := NewAssetPathMatcher(includes, excludes, f.Path())
			if err != nil {
				errs.Append(err)
				break
			}

			matchCount := 0
			for _, asset := range input.Files() {
				if matcher.Matches(asset.Path()) {
					matchCount++
					if destUnit == "" {
						log.Infof("Adding asset '%s' to all units", asset.Path())
						for _, unit := range units {
							unit.AddStaticAsset(asset)
						}
					} else {
						log.Infof("Adding asset '%s' to unit '%s'", asset.Path(), destUnit)
						units[destUnit].AddStaticAsset(asset)
					}
				}
			}
			if matchCount == 0 {
				log.Warn("No assets found matching include/exclude rules")
			}
			if matcher.err != nil {
				errs.Append(matcher.err)
			}
		}
	}

	return errs.ErrOrNil()
}

type assetPathMatcher struct {
	include []string
	exclude []string
	err     error
}

func NewAssetPathMatcher(include []string, exclude []string, filePath string) (assetPathMatcher, error) {
	matcher := assetPathMatcher{}
	filePath = filepath.ToSlash(filePath)
	for _, pattern := range include {
		matcher.include = append(matcher.include, modifyPathIfRelative(pattern, filePath))
	}

	for _, pattern := range exclude {
		matcher.exclude = append(matcher.exclude, modifyPathIfRelative(pattern, filePath))
	}
	return matcher, nil

}

func modifyPathIfRelative(path string, currentPath string) string {
	if filepath.IsAbs(path) {
		return strings.TrimPrefix(path, "/")
	}
	return filepath.Join(filepath.Dir(currentPath), path)
}

func (m *assetPathMatcher) Matches(p string) bool {
	if m.err != nil {
		return false
	}
	p = filepath.ToSlash(p)
	//! Implementation note: use `doublestar` package over stdlib `filepath`
	//! because the std version doesn't support '**' (globstar)
	toInclude := false
	for _, pattern := range m.include {
		toInclude, m.err = doublestar.Match(pattern, p)
		if m.err != nil {
			return false
		}
		if toInclude {
			break
		}
	}
	if !toInclude {
		return false
	}

	toExclude := false
	for _, pattern := range m.exclude {
		toExclude, m.err = doublestar.Match(pattern, p)
		if m.err != nil {
			return false
		}
		if toExclude {
			break
		}
	}
	return !toExclude
}
