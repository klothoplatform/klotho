package javascript

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	assert2 "github.com/stretchr/testify/assert"
)

func TestNodeJSExecutable_Transform(t *testing.T) {
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
			name:       "unit is NodeJS executable",
			otherFiles: map[string]string{"package.json": `{ "main" : "" }`},
			units: []*core.ExecutionUnit{
				execUnit("main",
					taggedFile{path: "index.js", content: "const mod = require('./module')"},
					taggedFile{path: "module.js"},
				),
			},
			expectedUnits: map[string]expectedUnit{
				"main": {
					executableType: core.ExecutableTypeNodeJS,
					expectedFiles: map[string][]string{
						"allFiles":    {"package.json", "index.js", "module.js"},
						"resources":   {"package.json"},
						"sourceFiles": {"index.js", "module.js"},
						"entrypoints": {"index.js"},
					},
				},
			},
		},
		{
			name:       "default entrypoint is resolved from package.json#main when no execution_unit annotation is present ",
			otherFiles: map[string]string{"package.json": `{ "main" : "myunit.js" }`},
			units: []*core.ExecutionUnit{
				execUnit("main",
					taggedFile{path: "myunit.js", content: "const mod = require('./module')"},
					taggedFile{path: "module.js"},
				),
			},
			expectedUnits: map[string]expectedUnit{
				"main": {
					executableType: core.ExecutableTypeNodeJS,
					expectedFiles: map[string][]string{
						"allFiles":    {"package.json", "myunit.js", "module.js"},
						"resources":   {"package.json"},
						"sourceFiles": {"myunit.js", "module.js"},
						"entrypoints": {"myunit.js"},
					},
				},
			},
		},
		{
			name:       "default entrypoint is index.js when package.json#main is not set",
			otherFiles: map[string]string{"package.json": `{ "main" : "" }`},
			units: []*core.ExecutionUnit{
				execUnit("main",
					taggedFile{path: "index.js", content: "const mod = require('./module')"},
					taggedFile{path: "module.js"}),
			},
			expectedUnits: map[string]expectedUnit{
				"main": {
					executableType: core.ExecutableTypeNodeJS,
					expectedFiles: map[string][]string{
						"allFiles":    {"package.json", "index.js", "module.js"},
						"resources":   {"package.json"},
						"sourceFiles": {"index.js", "module.js"},
						"entrypoints": {"index.js"},
					},
				},
			},
		},
		{
			name:       "upstream entrypoint is added",
			otherFiles: map[string]string{"package.json": `{ "main" : "" }`},
			units: []*core.ExecutionUnit{
				execUnit("unit1",
					taggedFile{path: "expose.js", content: `
						/* @klotho::expose {
						 *  id = "gateway"
						 * }
						 */
						const index = require('./entrypoint');`,
					},
					taggedFile{path: "entrypoint.js", content: "const mod = require('./module')", tag: "entrypoint"},
					taggedFile{path: "module.js"}),
				execUnit("unit2",
					taggedFile{path: "expose.js", content: `
						/* @klotho::expose {
						 *  id = "gateway"
						 * }
					     */
						const index = require('./index')`,
					},
					taggedFile{path: "index.js", content: "const mod = require('./module')"},
					taggedFile{path: "module.js"}),
			},
			expectedUnits: map[string]expectedUnit{
				"unit1": {
					executableType: core.ExecutableTypeNodeJS,
					expectedFiles: map[string][]string{
						"allFiles":    {"package.json", "entrypoint.js", "module.js", "expose.js"},
						"resources":   {"package.json"},
						"sourceFiles": {"entrypoint.js", "module.js", "expose.js"},
						"entrypoints": {"entrypoint.js", "expose.js"},
					},
				},
				"unit2": {
					executableType: core.ExecutableTypeNodeJS,
					expectedFiles: map[string][]string{
						"allFiles":    {"package.json", "index.js", "module.js", "expose.js"},
						"resources":   {"package.json"},
						"sourceFiles": {"index.js", "module.js", "expose.js"},
						"entrypoints": {"index.js", "expose.js"},
					},
				},
			},
		},
		{
			name:       "annotated file is added",
			otherFiles: map[string]string{"package.json": `{ "main" : "" }`},
			units: []*core.ExecutionUnit{
				execUnit("unit1",
					taggedFile{path: "expose.js", content: `
						/* @klotho::expose {
						 *  id = "gateway"
						 * }
						 */
						const index = require('./entrypoint');`,
					},
					taggedFile{path: "entrypoint.js", content: "const mod = require('./module')"},
					taggedFile{path: "module.js"}),
				execUnit("unit2",
					taggedFile{path: "expose.js", content: `
				/* @klotho::expose {
				 *  id = "gateway"
				 * }
				 */
				const index = require('./entrypoint');`,
					},
					taggedFile{path: "index.js", content: `
					/* @klotho::execution_unit {
						*  id = "unit2"
						* }
						*/
						const mod = require('./module')"}
					`, tag: "source"},
					taggedFile{path: "module.js"}),
			},
			expectedUnits: map[string]expectedUnit{
				"unit1": {
					executableType: core.ExecutableTypeNodeJS,
					expectedFiles: map[string][]string{
						"allFiles":    {"package.json", "entrypoint.js", "module.js", "expose.js"},
						"resources":   {"package.json"},
						"sourceFiles": {},
						"entrypoints": {},
					},
				},
				"unit2": {
					executableType: core.ExecutableTypeNodeJS,
					expectedFiles: map[string][]string{
						"allFiles":    {"package.json", "index.js", "module.js", "expose.js"},
						"resources":   {"package.json"},
						"sourceFiles": {"index.js", "module.js"},
						"entrypoints": {"index.js"},
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
			if !assert.NoError(NodeJSExecutable{}.Transform(inputFiles, result)) {
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
	if strings.HasSuffix(path, ".json") {
		f, err = NewPackageFile(path, strings.NewReader(content))
		if err != nil {
			panic(err)
		}
	} else if strings.HasSuffix(path, ".js") {
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
