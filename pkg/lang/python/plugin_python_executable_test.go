package python

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	assert2 "github.com/stretchr/testify/assert"
)

func TestPythonExecutable_Transform(t *testing.T) {
	type expectedUnit struct {
		executableType core.ExecutableType
		expectedFiles  map[string][]string
	}

	cases := []struct {
		name          string
		otherFiles    map[string]string
		units         []*core.ExecutionUnit
		expectedUnits map[string]expectedUnit
	}{
		{
			name:       "unit is Python executable",
			otherFiles: map[string]string{"requirements.txt": ""},
			units: []*core.ExecutionUnit{
				execUnit("main",
					taggedFile{path: "app/main.py", content: "import app.module"},
					taggedFile{path: "app/module.py"},
				),
			},
			expectedUnits: map[string]expectedUnit{
				"main": {
					executableType: core.ExecutableTypePython,
					expectedFiles: map[string][]string{
						"allFiles":    {"requirements.txt", "app/main.py", "app/module.py"},
						"resources":   {"requirements.txt"},
						"sourceFiles": {"app/main.py", "app/module.py"},
						"entrypoints": {"app/main.py"},
					},
				},
			},
		},
		{
			name:       "default entrypoint is main.py",
			otherFiles: map[string]string{"requirements.txt": ""},
			units: []*core.ExecutionUnit{
				execUnit("main",
					taggedFile{path: "app/main.py", content: "import app.module"},
					taggedFile{path: "app/module.py"}),
			},
			expectedUnits: map[string]expectedUnit{
				"main": {
					executableType: core.ExecutableTypePython,
					expectedFiles: map[string][]string{
						"allFiles":    {"requirements.txt", "app/main.py", "app/module.py"},
						"resources":   {"requirements.txt"},
						"sourceFiles": {"app/main.py", "app/module.py"},
						"entrypoints": {"app/main.py"},
					},
				},
			},
		},
		{
			name:       "upstream entrypoint is added",
			otherFiles: map[string]string{"requirements.txt": ""},
			units: []*core.ExecutionUnit{
				execUnit("unit1",
					taggedFile{path: "app/expose.py", content: `
					     # @klotho::expose {
						 #  id = "gateway"
						 # }
						import app.entrypoint`,
					},
					taggedFile{path: "app/entrypoint.py", content: "import app.module", tag: "entrypoint"},
					taggedFile{path: "app/module.py"}),
				execUnit("unit2",
					taggedFile{path: "app/expose.py", content: `
						# @klotho::expose {
						#  id = "gateway"
					    # }
						import app.main`,
					},
					taggedFile{path: "app/main.py", content: "import app.module"},
					taggedFile{path: "app/module.py"}),
			},
			expectedUnits: map[string]expectedUnit{
				"unit1": {
					executableType: core.ExecutableTypePython,
					expectedFiles: map[string][]string{
						"allFiles":    {"requirements.txt", "app/entrypoint.py", "app/module.py", "app/expose.py"},
						"resources":   {"requirements.txt"},
						"sourceFiles": {"app/entrypoint.py", "app/module.py", "app/expose.py"},
						"entrypoints": {"app/entrypoint.py", "app/expose.py"},
					},
				},
				"unit2": {
					executableType: core.ExecutableTypePython,
					expectedFiles: map[string][]string{
						"allFiles":    {"requirements.txt", "app/main.py", "app/module.py", "app/expose.py"},
						"resources":   {"requirements.txt"},
						"sourceFiles": {"app/main.py", "app/module.py", "app/expose.py"},
						"entrypoints": {"app/main.py", "app/expose.py"},
					},
				},
			},
		},
		{
			name:       "annotated file is added",
			otherFiles: map[string]string{"requirements.txt": ""},
			units: []*core.ExecutionUnit{
				execUnit("unit1",
					taggedFile{path: "app/expose.py", content: `
					     # @klotho::expose {
						 #  id = "gateway"
						 # }
						import app.entrypoint`,
					},
					taggedFile{path: "app/entrypoint.py", content: `
					# @klotho::execution_unit { id = "unit1" }
					import app.module`},
					taggedFile{path: "app/module.py"}),
				execUnit("unit2",
					taggedFile{path: "app/expose.py", content: `
						# @klotho::expose {
						#  id = "gateway"
					    # }
						import app.main`,
					},
					taggedFile{path: "app/main.py", content: `
					# @klotho::execution_unit { id = "unit2" }
					import app.module`},
					taggedFile{path: "app/module.py"}),
			},
			expectedUnits: map[string]expectedUnit{
				"unit1": {
					executableType: core.ExecutableTypePython,
					expectedFiles: map[string][]string{
						"allFiles":    {"requirements.txt", "app/entrypoint.py", "app/module.py", "app/expose.py"},
						"resources":   {"requirements.txt"},
						"sourceFiles": {"app/entrypoint.py", "app/module.py", "app/expose.py"},
						"entrypoints": {"app/entrypoint.py", "app/expose.py"},
					},
				},
				"unit2": {
					executableType: core.ExecutableTypePython,
					expectedFiles: map[string][]string{
						"allFiles":    {"requirements.txt", "app/main.py", "app/module.py", "app/expose.py"},
						"resources":   {"requirements.txt"},
						"sourceFiles": {"app/main.py", "app/module.py", "app/expose.py"},
						"entrypoints": {"app/main.py", "app/expose.py"},
					},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert2.New(t)

			inputFiles := &core.InputFiles{}
			for p, c := range tt.otherFiles {
				inputFiles.Add(file(p, c))
			}

			result := core.NewConstructGraph()
			for _, unit := range tt.units {
				result.AddConstruct(unit)
				for _, f := range unit.Files() {
					inputFiles.Add(f)
				}
			}
			if !assert.NoError(PythonExecutable{}.Transform(inputFiles, &core.FileDependencies{}, result)) {
				return
			}
			assert.Equal(len(tt.expectedUnits), len(tt.units))

			for _, unit := range tt.units {
				eu := tt.expectedUnits[unit.ID]
				assert.Equal(eu.executableType, unit.Executable.Type)
				assert.ElementsMatch(eu.expectedFiles["allFiles"], keys(unit.Files()))
				assert.ElementsMatch(eu.expectedFiles["entrypoints"], keys(unit.Executable.Entrypoints))
				assert.ElementsMatch(eu.expectedFiles["sourceFiles"], keys(unit.Executable.SourceFiles))
				assert.ElementsMatch(eu.expectedFiles["resources"], keys(unit.Executable.Resources))
				assert.ElementsMatch(eu.expectedFiles["staticAssets"], keys(unit.Executable.StaticAssets))
			}
		})
	}
}

func keys[K comparable, V any](m map[K]V) []K {
	var ks []K
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func execUnit(name string, files ...taggedFile) *core.ExecutionUnit {
	unit := core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: name, Capability: annotation.ExecutionUnitCapability}, Executable: core.NewExecutable()}
	for _, tf := range files {
		f := file(tf.path, tf.content)

		switch tf.tag {
		case "source":
			unit.AddSourceFile(f)
		case "entrypoint":
			unit.AddEntrypoint(f)
		case "resource":
			unit.AddResource(f)
		case "asset":
			unit.AddStaticAsset(f)
		default:
			unit.Add(f)
		}
	}
	return &unit
}

func file(path string, content string) core.File {
	var f core.File
	var err error
	if strings.HasSuffix(path, ".txt") {
		f, err = NewRequirementsTxt(path, strings.NewReader(content))
		if err != nil {
			panic(err)
		}
	} else if strings.HasSuffix(path, ".py") {
		f, err = core.NewSourceFile(path, strings.NewReader(content), Language)
		if err != nil {
			panic(err)
		}
	} else {
		f = &core.FileRef{FPath: path}
	}
	return f
}

type taggedFile struct {
	path    string
	content string
	tag     string
}
