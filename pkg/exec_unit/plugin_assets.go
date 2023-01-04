package execunit

import (
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

func (p Assets) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	input := result.GetFirstResource(core.InputFilesKind).(*core.InputFiles)

	units := make(map[string]*core.ExecutionUnit)
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !ok {
			continue
		}
		units[unit.Name] = unit
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

			matcher := assetPathMatcher{}
			matcher.include, _ = annot.Capability.Directives.StringArray("include")
			matcher.exclude, _ = annot.Capability.Directives.StringArray("exclude")

			err := matcher.ModifyPathsForAnnotatedFile(f.Path())
			if err != nil {
				errs.Append(err)
				break
			}

			matchCount := 0
			for _, asset := range input.Files() {
				if matcher.Matches(asset.Path()) {
					zap.L().With(logging.FileField(f), logging.AnnotationField(annot)).Sugar().Infof("Adding asset '%s' to unit '%s'", asset.Path(), destUnit)
					matchCount++
					if destUnit == "" {
						for _, unit := range units {
							unit.AddStaticAsset(asset)
						}
					} else {
						units[destUnit].AddStaticAsset(asset)
					}
				}
			}
			if matchCount == 0 {
				zap.L().With(logging.FileField(f), logging.AnnotationField(annot)).Warn("No assets found matching include/exclude rules")
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

func (m *assetPathMatcher) ModifyPathsForAnnotatedFile(path string) error {
	newInclude := []string{}
	for _, pattern := range m.include {
		absPath, err := filepath.Abs(pattern)
		if err != nil {
			return err
		}
		if absPath == pattern {
			newInclude = append(newInclude, strings.TrimPrefix(pattern, "/"))
			continue
		}
		relPath, err := filepath.Rel(filepath.Dir("."), filepath.Join(filepath.Dir(path), pattern))
		if err != nil {
			return err
		}
		newInclude = append(newInclude, relPath)
	}
	m.include = newInclude

	newExclude := []string{}
	for _, pattern := range m.exclude {
		absPath, err := filepath.Abs(pattern)
		if err != nil {
			return err
		}
		if absPath == pattern {
			newExclude = append(newExclude, strings.TrimPrefix(pattern, "/"))
			continue
		}
		relPath, err := filepath.Rel(filepath.Dir("."), filepath.Join(filepath.Dir(path), pattern))
		if err != nil {
			return err
		}
		newExclude = append(newExclude, relPath)
	}
	m.exclude = newExclude
	return nil
}

func (m *assetPathMatcher) Matches(p string) bool {
	if m.err != nil {
		return false
	}
	//! Implementation note: use `doublestar` package over stdlib `filepath`
	//! because the std version doesn't support '**' (globstar)
	toInclude := false
	for _, pattern := range m.include {
		toInclude, m.err = doublestar.PathMatch(pattern, p)
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
		toExclude, m.err = doublestar.PathMatch(pattern, p)
		if m.err != nil {
			return false
		}
		if toExclude {
			break
		}
	}
	return !toExclude
}
