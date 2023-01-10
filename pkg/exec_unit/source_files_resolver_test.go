package execunit

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/stretchr/testify/assert"
)

func TestSourceFilesResolver_Resolve(t *testing.T) {

	tests := []struct {
		name                string
		entrypoints         []string
		fileUnits           map[string]string
		unitDeps            []testFileDep
		upstreamAnnotations []string
		expectedSourceFiles []string
	}{
		{
			name:        "source file dependencies are resolved starting from the unit's entrypoints excluding files that are not imported",
			entrypoints: []string{"main"},
			fileUnits: map[string]string{
				"main":   "execution_unit:main",
				"expose": "expose:gateway",
				"dep1":   "",
				"dep2":   "",
				"orphan": "",
			},
			unitDeps: []testFileDep{
				{
					filePath: "expose",
					imports:  map[string][]string{"main": {}},
				},
				{
					filePath: "main",
					imports:  map[string][]string{"dep1": {}},
				},
				{
					filePath: "dep1",
					imports:  map[string][]string{"dep2": {}},
				},
				{filePath: "dep2"},
				{filePath: "orphan"},
			},
			expectedSourceFiles: []string{"main", "dep1", "dep2"},
		},
		{
			name:        "dependencies of upstream annotations are included in a unit's source files",
			entrypoints: []string{"main"},
			fileUnits: map[string]string{
				"main":       "execution_unit:main",
				"expose":     "expose:gateway",
				"expose_dep": "",
			},
			unitDeps: []testFileDep{
				{
					filePath: "expose",
					imports:  map[string][]string{"main": {}, "expose_dep": {}},
				},
				{
					filePath: "main",
				},
				{filePath: "expose_dep"},
			},
			expectedSourceFiles: []string{"main", "expose", "expose_dep"},
			upstreamAnnotations: []string{annotation.ExposeCapability},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testUnit := core.ExecutionUnit{Name: "main", Executable: core.NewExecutable()}
			for path, unit := range tt.fileUnits {
				f, err := core.NewSourceFile(path, strings.NewReader(unit), testAnnotationLang)
				if assert.Nil(err) {
					testUnit.Add(f)
				}
			}
			for _, entrypoint := range tt.entrypoints {
				testUnit.Executable.Entrypoints[entrypoint] = struct{}{}
			}

			resolver := SourceFilesResolver{
				UnitFileDependencyResolver: testFileDepResolver(tt.unitDeps),
				UpstreamAnnotations:        tt.upstreamAnnotations,
			}
			got, err := resolver.Resolve(&testUnit)
			assert.NoError(err)
			var gotPaths []string
			for path := range got {
				gotPaths = append(gotPaths, path)
			}
			assert.ElementsMatch(tt.expectedSourceFiles, gotPaths)
		})
	}
}

type testFileDep struct {
	filePath string
	imports  map[string][]string
}

func testFileDepResolver(testFileDeps []testFileDep) UnitFileDependencyResolver {
	return func(unit *core.ExecutionUnit) (FileDependencies, error) {
		fileDeps := FileDependencies{}
		for _, fileDep := range testFileDeps {
			imported := Imported{}
			for importPath, importedRefs := range fileDep.imports {
				refs := References{}
				for _, ref := range importedRefs {
					refs[ref] = struct{}{}
				}
				imported[importPath] = refs
			}
			fileDeps[fileDep.filePath] = imported
		}
		return fileDeps, nil
	}
}

type testMultipleCapabilityFinder struct{}

var testAnnotationLang = core.SourceLanguage{
	ID:               core.LanguageId("test_annotation_lang"),
	Sitter:           javascript.GetLanguage(), // we don't actually care about the language, but we do need a non-nil one
	CapabilityFinder: &testMultipleCapabilityFinder{},
}

func (t *testMultipleCapabilityFinder) FindAllCapabilities(sf *core.SourceFile) (core.AnnotationMap, error) {
	body := string(sf.Program())
	rawAnnots := strings.SplitN(body, "|", 2)
	annots := make(core.AnnotationMap)
	if body == "" {
		return annots, nil
	}
	for _, rawAnnot := range rawAnnots {
		annotParts := strings.SplitN(rawAnnot, ":", 2)
		if len(annotParts) != 2 {
			continue
		}
		annots.Add(&core.Annotation{
			Capability: &annotation.Capability{
				Name: strings.TrimSpace(annotParts[0]),
				ID:   strings.TrimSpace(annotParts[1]),
			},
		})

	}
	return annots, nil
}
