package staticunit

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/klothoplatform/klotho/pkg/config"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type (
	StaticUnitSplit struct {
		Config *config.Application
	}
)

func (p StaticUnitSplit) Name() string { return "StaticUnitSplit" }

func (p StaticUnitSplit) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error

	inputR := result.GetFirstResource(core.InputFilesKind)
	if inputR == nil {
		//? Already split?
		return nil
	}
	input := inputR.(*core.InputFiles)
	for _, f := range input.Files() {

		log := zap.L().With(logging.FileField(f)).Sugar()
		ast, ok := f.(*core.SourceFile)
		if !ok {
			// Non-source files can't have any annotations therefore we don't know what
			// is intended to be included or not. Default to not including and let other plugins decide.
			log.Debug("Skipping non-source file")
			continue
		}

		for _, annot := range ast.Annotations() {
			cap := annot.Capability
			if cap.Name == annotation.StaticUnitCapability {
				var cause error
				if cap.ID == "" {
					cause = errors.New("'id' is required")
					errs.Append(cause)
				}
				newUnit := &core.StaticUnit{
					Name: cap.ID,
				}

				indexDocument, ok := cap.Directives.String("index_document")
				if ok {
					newUnit.IndexDocument = indexDocument
				}

				matcher := staticAssetPathMatcher{}
				matcher.staticFiles, _ = cap.Directives.StringArray("static_files")
				matcher.sharedFiles, _ = cap.Directives.StringArray("shared_files")
				err := matcher.ModifyPathsForAnnotatedFile(f.Path())
				if err != nil {
					errs.Append(err)
					break
				}
				matchCount := 0
				for _, asset := range input.Files() {
					ref := &core.FileRef{
						FPath:          asset.Path(),
						RootConfigPath: p.Config.Path,
					}
					static, shared := matcher.Matches(asset.Path())
					if shared {
						matchCount++
						newUnit.AddSharedFile(ref)
					} else if static || asset.Path() == indexDocument {
						matchCount++
						newUnit.AddStaticFile(ref)
						// replace with ref to exclude from further processing
						log.Debug("Replacing input file with reference path: " + asset.Path())
						input.Add(ref)
					}
				}
				log.Debug("Including " + strconv.Itoa(matchCount) + " files in static unit")
				if matcher.err != nil {
					errs.Append(matcher.err)
				}
				result.Add(newUnit)
			}
		}
	}
	return errs.ErrOrNil()
}

type staticAssetPathMatcher struct {
	staticFiles []string
	sharedFiles []string
	err         error
}

// ModifyPathsForAnnotatedFile transforms the staticAssetPathMatcher sharedFiles and staticFiles to be absolute paths from the klotho project root, without the prefix '/'.
// This is done to conform to the file path structure of input files.
func (m *staticAssetPathMatcher) ModifyPathsForAnnotatedFile(path string) error {
	newInclude := []string{}
	for _, pattern := range m.staticFiles {
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
	m.staticFiles = newInclude

	newExclude := []string{}
	for _, pattern := range m.sharedFiles {
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
	m.sharedFiles = newExclude
	return nil
}

func (m *staticAssetPathMatcher) Matches(p string) (bool, bool) {
	if m.err != nil {
		return false, false
	}

	//! Implementation note: use `doublestar` package over stdlib `filepath`
	//! because the std version doesn't support '**' (globstar)
	staticFile := false
	for _, pattern := range m.staticFiles {
		staticFile, m.err = doublestar.PathMatch(pattern, p)
		if staticFile {
			break
		}
		if m.err != nil {
			return false, false
		}
	}

	sharedFile := false
	for _, pattern := range m.sharedFiles {
		sharedFile, m.err = doublestar.PathMatch(pattern, p)
		if sharedFile {
			break
		}
		if m.err != nil {
			return false, false
		}
	}
	return staticFile, sharedFile
}
