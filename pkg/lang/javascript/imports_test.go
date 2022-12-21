package javascript

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestFindImports(t *testing.T) {

	type file struct {
		Path    string
		Content string
	}

	tests := []struct {
		name       string
		sourceFile file
		want       FileImports
	}{
		{
			name:       "ES: import side-effect",
			sourceFile: file{Path: "my-module.js", Content: `import "./module-name";`},
			want: FileImports{
				"./module-name": []Import{{
					Source: "./module-name",
					Scope:  ImportScopeModule,
					Type:   ImportTypeSideEffect,
					Kind:   ImportKindES,
				}},
			},
		},
		{
			name:       "ES: source file extension is not modified when specified",
			sourceFile: file{Path: "my-module.js", Content: `import "./module-name.js";`},
			want: FileImports{
				"./module-name.js": []Import{{
					Source: "./module-name.js",
					Scope:  ImportScopeModule,
					Type:   ImportTypeSideEffect,
					Kind:   ImportKindES,
				}},
			},
		},
		{
			name:       "ES: import aliased default",
			sourceFile: file{Path: "my-module.js", Content: `import defaultExport from "./module-name";`},
			want: FileImports{
				"./module-name": []Import{{
					Source: "./module-name",
					Name:   "default",
					Alias:  "defaultExport",
					Scope:  ImportScopeModule,
					Type:   ImportTypeDefault,
					Kind:   ImportKindES,
				}},
			},
		},
		{
			name:       "ES: import named default",
			sourceFile: file{Path: "my-module.js", Content: `import { default as named } from "./module-name";`},
			want: FileImports{
				"./module-name": []Import{{
					Source: "./module-name",
					Name:   "default",
					Alias:  "named",
					Scope:  ImportScopeModule,
					Type:   ImportTypeDefault,
					Kind:   ImportKindES,
				}},
			},
		},
		{
			// TODO: revisit this case once https://github.com/tree-sitter/tree-sitter-javascript/pull/234 is merged
			name:       "ES: import named string alias is treated as a side-effect import",
			sourceFile: file{Path: "my-module.js", Content: `import {"string-name" as alias } from "./module-name";`},
			want: FileImports{
				"./module-name": []Import{{
					Source: "./module-name",
					Name:   "",
					Alias:  "",
					Scope:  ImportScopeModule,
					Type:   ImportTypeSideEffect,
					Kind:   ImportKindES,
				}},
			},
		},
		{
			name:       "ES: import namespace",
			sourceFile: file{Path: "my-module.js", Content: `import * as mod from "./module-name";`},
			want: FileImports{
				"./module-name": []Import{{
					Source: "./module-name",
					Name:   "*",
					Alias:  "mod",
					Scope:  ImportScopeModule,
					Type:   ImportTypeNamespace,
					Kind:   ImportKindES,
				}},
			},
		},
		{
			name:       "ES: find default + namespace imports",
			sourceFile: file{Path: "my-module.js", Content: `import defaultExport, * as mod from "./module-name";`},
			want: FileImports{
				"./module-name": []Import{
					{
						Source: "./module-name",
						Name:   "*",
						Alias:  "mod",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindES,
					},
					{
						Source: "./module-name",
						Name:   "default",
						Alias:  "defaultExport",
						Scope:  ImportScopeModule,
						Type:   ImportTypeDefault,
						Kind:   ImportKindES,
					}},
			},
		},
		{
			name:       "ES: find named imports",
			sourceFile: file{Path: "my-module.js", Content: `import { a, b as c } from "./module-name";`},
			want: FileImports{
				"./module-name": []Import{
					{
						Source: "./module-name",
						Name:   "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
					{
						Source: "./module-name",
						Name:   "b",
						Alias:  "c",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
				},
			},
		},
		{
			name:       "ES: find default + named imports",
			sourceFile: file{Path: "my-module.js", Content: `import defaultExport, { a } from "./module-name";`},
			want: FileImports{
				"./module-name": []Import{
					{
						Source: "./module-name",
						Name:   "default",
						Alias:  "defaultExport",
						Scope:  ImportScopeModule,
						Type:   ImportTypeDefault,
						Kind:   ImportKindES,
					},
					{
						Source: "./module-name",
						Name:   "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
				},
			},
		},
		{
			name:       "ES: local import scope",
			sourceFile: file{Path: "my-module.js", Content: `function func() {import { a } from "./module-name"; }`},
			want: FileImports{
				"./module-name": []Import{
					{
						Source: "./module-name",
						Name:   "a",
						Scope:  ImportScopeLocal,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
				},
			},
		},
		{
			// This can be a node module or a Webpack/Babel absolute import
			name:       "ES: absolute import",
			sourceFile: file{Path: "my-module.js", Content: `function func() {import { a } from "module-name"; }`},
			want: FileImports{
				"module-name": []Import{
					{
						Source: "module-name",
						Name:   "a",
						Scope:  ImportScopeLocal,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
				},
			},
		},
		{
			name: "ES: multiple import statements",
			sourceFile: file{Path: "my-module.js", Content: `
import { a, b as c } from "./module-name";
import defaultExport from "./module-name";
import "./module-name";

function func() {
	import { a } from "./module-name";
}`},
			want: FileImports{
				"./module-name": []Import{
					{
						Source: "./module-name",
						Name:   "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
					{
						Source: "./module-name",
						Name:   "b",
						Alias:  "c",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
					{
						Source: "./module-name",
						Name:   "default",
						Alias:  "defaultExport",
						Scope:  ImportScopeModule,
						Type:   ImportTypeDefault,
						Kind:   ImportKindES,
					},
					{
						Source: "./module-name",
						Scope:  ImportScopeModule,
						Type:   ImportTypeSideEffect,
						Kind:   ImportKindES,
					},
					{
						Source: "./module-name",
						Name:   "a",
						Scope:  ImportScopeLocal,
						Type:   ImportTypeNamed,
						Kind:   ImportKindES,
					},
				},
			},
		},
		{
			name:       "CJS: no import",
			sourceFile: file{Path: "my-module.js", Content: `"const a = 1; const b = require(1)"`},
			want:       FileImports{},
		},
		{
			name:       "CJS: const namespace import",
			sourceFile: file{Path: "my-module.js", Content: "const a = require('./a');"},
			want: FileImports{
				"./a": []Import{
					{
						Source: "./a",
						Name:   "*",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: var namespace import",
			sourceFile: file{Path: "my-module.js", Content: "var a = require('./a');"},
			want: FileImports{
				"./a": []Import{
					{
						Source: "./a",
						Name:   "*",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: named property import",
			sourceFile: file{Path: "my-module.js", Content: "const a = require('./a').prop;"},
			want: FileImports{
				"./a": []Import{
					{
						Source: "./a",
						Name:   "prop",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: default property import",
			sourceFile: file{Path: "my-module.js", Content: "const a = require('./a').default;"},
			want: FileImports{
				"./a": []Import{
					{
						Source: "./a",
						Name:   "default",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeDefault,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: TS __importStar() namespace import",
			sourceFile: file{Path: "my-module.js", Content: "const a = __importStar(require('./module'));"},
			want: FileImports{
				"./module": []Import{
					{
						Source: "./module",
						Name:   "*",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: TS __importDefault() default import",
			sourceFile: file{Path: "my-module.js", Content: "const a = __importDefault(require('./module'));"},
			want: FileImports{
				"./module": []Import{
					{
						Source: "./module",
						Name:   "default",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeDefault,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: absolute import",
			sourceFile: file{Path: "my-module.js", Content: "const a = require('module');"},
			want: FileImports{
				"module": []Import{
					{
						Source: "module",
						Name:   "*",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name: "CJS: side-effect import",
			sourceFile: file{Path: "my-module.js", Content: `
require('module1');
const x = { y: require('module2') };
require('module3').name;
require('module4').name.field1.field2;
`},
			want: FileImports{
				"module1": []Import{
					{
						Source: "module1",
						Name:   "",
						Alias:  "",
						Scope:  ImportScopeModule,
						Type:   ImportTypeSideEffect,
						Kind:   ImportKindCommonJS,
					},
				},
				"module2": []Import{
					{
						Source: "module2",
						Name:   "",
						Alias:  "",
						Scope:  ImportScopeLocal,
						Type:   ImportTypeSideEffect,
						Kind:   ImportKindCommonJS,
					},
				},
				"module3": []Import{
					{
						Source: "module3",
						Name:   "name",
						Alias:  "",
						Scope:  ImportScopeModule,
						Type:   ImportTypeSideEffect,
						Kind:   ImportKindCommonJS,
					},
				},
				"module4": []Import{
					{
						Source: "module4",
						Name:   "name.field1.field2",
						Alias:  "",
						Scope:  ImportScopeModule,
						Type:   ImportTypeSideEffect,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name: "CJS: multiple declaration",
			sourceFile: file{Path: "my-module.js", Content: `
const a = require('module1'), b = require('module2');
const {c, d: e} = require('module3'), f = require('module4').g, h = require('module4').i.j";
`},
			want: FileImports{
				"module1": []Import{
					{
						Source: "module1",
						Name:   "*",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				}, "module2": []Import{
					{
						Source: "module2",
						Name:   "*",
						Alias:  "b",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				}, "module3": []Import{
					{
						Source: "module3",
						Name:   "c",
						Alias:  "",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					},
					{
						Source: "module3",
						Name:   "d",
						Alias:  "e",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					},
				}, "module4": []Import{
					{
						Source: "module4",
						Name:   "g",
						Alias:  "f",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					}, {
						Source: "module4",
						Name:   "i.j",
						Alias:  "h",
						Scope:  ImportScopeModule,
						Type:   ImportTypeField,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: multiple assignment in sequence",
			sourceFile: file{Path: "my-module.js", Content: `a = require('module1'), b = require('module2').c, d = require('module3').e.f;`},
			want: FileImports{
				"module1": []Import{
					{
						Source: "module1",
						Name:   "*",
						Alias:  "a",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				}, "module2": []Import{
					{
						Source: "module2",
						Name:   "c",
						Alias:  "b",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					},
				}, "module3": []Import{
					{
						Source: "module3",
						Name:   "e.f",
						Alias:  "d",
						Scope:  ImportScopeModule,
						Type:   ImportTypeField,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name: "CJS: assignment import",
			sourceFile: file{Path: "my-module.js", Content: `
a1 = require('module1');
a2 = require('module2').name;
a3 = require('module3').default;
`},
			want: FileImports{
				"module1": []Import{
					{
						Source: "module1",
						Name:   "*",
						Alias:  "a1",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamespace,
						Kind:   ImportKindCommonJS,
					},
				},
				"module2": []Import{
					{
						Source: "module2",
						Name:   "name",
						Alias:  "a2",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					},
				},
				"module3": []Import{
					{
						Source: "module3",
						Name:   "default",
						Alias:  "a3",
						Scope:  ImportScopeModule,
						Type:   ImportTypeDefault,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name:       "CJS: non-string source not imported",
			sourceFile: file{Path: "my-module.js", Content: "const a = require(var);"},
			want:       FileImports{},
		},
		{
			name:       "CJS: multiple source args not imported",
			sourceFile: file{Path: "my-module.js", Content: "const a = require('module1', 'module2'); require('module1', 'module2');"},
			want:       FileImports{},
		},
		{
			name:       "CJS: destructured imports",
			sourceFile: file{Path: "my-module.js", Content: "const {exp, src: local, default: myDefault} = require('./module');"},
			want: FileImports{
				"./module": []Import{
					{
						Source: "./module",
						Name:   "exp",
						Alias:  "",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					},
					{
						Source: "./module",
						Name:   "src",
						Alias:  "local",
						Scope:  ImportScopeModule,
						Type:   ImportTypeNamed,
						Kind:   ImportKindCommonJS,
					},
					{
						Source: "./module",
						Name:   "default",
						Alias:  "myDefault",
						Scope:  ImportScopeModule,
						Type:   ImportTypeDefault,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
		{
			name: "CJS: field imports",
			sourceFile: file{
				Path: "my-module.js",
				Content: `
const { field, field2: alias } = require('module1').prop;
const alias = require('module2').prop.field1.field2;
			`},
			want: FileImports{
				"module1": []Import{
					{
						Source: "module1",
						Name:   "prop.field",
						Alias:  "",
						Scope:  ImportScopeModule,
						Type:   ImportTypeField,
						Kind:   ImportKindCommonJS,
					},
					{
						Source: "module1",
						Name:   "prop.field2",
						Alias:  "alias",
						Scope:  ImportScopeModule,
						Type:   ImportTypeField,
						Kind:   ImportKindCommonJS,
					},
				},
				"module2": []Import{
					{
						Source: "module2",
						Name:   "prop.field1.field2",
						Alias:  "alias",
						Scope:  ImportScopeModule,
						Type:   ImportTypeField,
						Kind:   ImportKindCommonJS,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile(tt.sourceFile.Path, strings.NewReader(tt.sourceFile.Content), Language)
			if !assert.NoError(err) {
				return
			}

			actualImports := FindImportsInFile(f)

			if !assert.Equal(len(tt.want), len(actualImports)) {
				spew.Dump(actualImports)
				return
			}

			for path, actualImportsForFile := range actualImports {
				expectedImportsForFile := tt.want[path]

				// sort imports since order query result order is not helpful (maybe we should do this in FindImportsInFile?)
				sort.Slice(expectedImportsForFile, func(i, j int) bool {
					lhs, rhs := expectedImportsForFile[i], expectedImportsForFile[j]
					byAlias := compare(lhs.Alias, rhs.Alias)
					byName := compare(lhs.Name, rhs.Name)
					byType := compare(lhs.Type, rhs.Type)
					byScope := compare(lhs.Scope, rhs.Scope)
					byKind := compare(lhs.Kind, rhs.Kind)
					return sortBy(byAlias, byName, byType, byScope, byKind)
				})
				sort.Slice(actualImportsForFile, func(i, j int) bool {
					lhs, rhs := actualImportsForFile[i], actualImportsForFile[j]
					byAlias := compare(lhs.Alias, rhs.Alias)
					byName := compare(lhs.Name, rhs.Name)
					byType := compare(lhs.Type, rhs.Type)
					byScope := compare(lhs.Scope, rhs.Scope)
					byKind := compare(lhs.Kind, rhs.Kind)
					return sortBy(byAlias, byName, byType, byScope, byKind)
				})

				assert.Equalf(len(expectedImportsForFile), len(actualImportsForFile), "wrong number of imports for file: %s", path)
				if !assert.Equal(len(expectedImportsForFile), len(actualImportsForFile)) {
					spew.Dump(actualImportsForFile)
					return
				}
				for i, actualImport := range actualImportsForFile {
					validateImport(assert, expectedImportsForFile[i], actualImport)
				}
			}
		})
	}
}

func TestFindImportForVar(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		varName string
		want    string
	}{
		{
			name:    "no import",
			source:  "const a = 1;",
			varName: "a",
			want:    "",
		},
		{
			name:    "import",
			source:  "const a = require('./a');",
			varName: "a",
			want:    "./a",
		},
		{
			name:    "import wrong var",
			source:  "const a = require('./a');",
			varName: "b",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			p := FindImportForVar(f.Tree().RootNode(), f.Program(), tt.varName)
			assert.Equal(tt.want, p.Source)
		})
	}
}

func validateImport(assert *assert.Assertions, expected Import, actual Import) {
	assert.Equalf(expected.Source, actual.Source, "Source '%s' does not match '%s' for '%s.%s'", actual.Source, expected.Source, expected.Source, expected.Name)
	assert.Equalf(expected.Name, actual.Name, "Name '%s' does not match '%s' for '%s.%s'", actual.Name, expected.Name, expected.Source, expected.Name)
	assert.Equalf(expected.Alias, actual.Alias, "Alias '%s' does not match '%s' for '%s.%s'", actual.Alias, expected.Alias, expected.Source, expected.Name)
	assert.Equalf(expected.Scope, actual.Scope, "Scope '%s' does not match '%s' for '%s.%s'", actual.Scope, expected.Scope, expected.Source, expected.Name)
	assert.Equalf(expected.Type, actual.Type, "Type '%s' does not match '%s' for '%s.%s'", actual.Type, expected.Type, expected.Source, expected.Name)
	assert.Equalf(expected.Kind, actual.Kind, "Kind '%s' does not match '%s' for '%s.%s'", actual.Kind, expected.Kind, expected.Source, expected.Name)

	assert.NotNilf(actual.ImportNode, "ImportNode is nil for '%s.s'", expected.Source, expected.Name)
}

func compare(a, b interface{}) int {
	return strings.Compare(fmt.Sprintf("%s", a), fmt.Sprintf("%s", b))
}
func sortBy(sc ...int) bool {
	for _, c := range sc {
		if c != 0 {
			return c < 0
		}
	}
	return sc[len(sc)-1] < 0
}
