package python

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestFindImports(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   Imports
	}{
		{
			name:   "import module",
			source: "import mymodule",
			want: map[string]Import{
				"mymodule": {Name: "mymodule", UsedAs: testutil.NewSet("mymodule")},
			},
		},
		{
			name:   "import module aliased",
			source: "import mymodule as m",
			want: map[string]Import{
				"mymodule": {Name: "mymodule", UsedAs: testutil.NewSet("m")},
			},
		},
		{
			name:   "import modules",
			source: "import mymodule1, mymodule2",
			want: map[string]Import{
				"mymodule1": {Name: "mymodule1", UsedAs: testutil.NewSet("mymodule1")},
				"mymodule2": {Name: "mymodule2", UsedAs: testutil.NewSet("mymodule2")},
			},
		},
		{
			name:   "import modules aliased",
			source: "import mymodule1 as m1, mymodule2 as m2",
			want: map[string]Import{
				"mymodule1": {Name: "mymodule1", UsedAs: testutil.NewSet("m1")},
				"mymodule2": {Name: "mymodule2", UsedAs: testutil.NewSet("m2")},
			},
		},
		{
			name:   "import submodule",
			source: "import mymodule.submodule",
			want: map[string]Import{
				"mymodule.submodule": {
					ParentModule: "mymodule",
					Name:         "submodule",
					UsedAs:       testutil.NewSet("mymodule.submodule")},
			},
		},
		{
			name:   "import submodule aliased",
			source: "import mymodule.submodule as w",
			want: map[string]Import{
				"mymodule.submodule": {ParentModule: "mymodule", Name: "submodule", UsedAs: testutil.NewSet("w")},
			},
		},
		{
			name:   "import aliased nested submodule",
			source: "import mymodule.submodule1.submodule2 as w",
			want: map[string]Import{
				"mymodule.submodule1.submodule2": {
					ParentModule: "mymodule.submodule1",
					Name:         "submodule2",
					UsedAs:       testutil.NewSet("w"),
				},
			},
		},
		{
			name:   "import relative module",
			source: "from .. import mymodule\nfrom ..parent import child.attribute",
			want: map[string]Import{
				"..": {
					ParentModule: "..",
					Name:         "",
					ImportedAttributes: map[string]Attribute{
						"mymodule": {Name: "mymodule", UsedAs: testutil.NewSet("mymodule")},
					}},
				"..parent": {
					ParentModule:       "..",
					Name:               "parent",
					ImportedAttributes: map[string]Attribute{"child.attribute": {Name: "child.attribute", UsedAs: testutil.NewSet("child.attribute")}},
				},
			},
		},
		{
			name:   "from module import attribute",
			source: "from mymodule import attribute",
			want: map[string]Import{
				"mymodule": {
					Name: "mymodule",
					ImportedAttributes: map[string]Attribute{
						"attribute": {Name: "attribute", UsedAs: testutil.NewSet("attribute")},
					}},
			},
		},
		{
			name: "import multiple attributes from module in separate imports",
			source: `from mymodule import attribute1
			         from mymodule import attribute2`,
			want: map[string]Import{
				"mymodule": {
					Name: "mymodule",
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", UsedAs: testutil.NewSet("attribute1")},
						"attribute2": {Name: "attribute2", UsedAs: testutil.NewSet("attribute2")},
					}},
			},
		},
		{
			name:   "from module import attributes",
			source: "from mymodule import attribute1, attribute2",
			want: map[string]Import{
				"mymodule": {
					Name: "mymodule",
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", UsedAs: testutil.NewSet("attribute1")},
						"attribute2": {Name: "attribute2", UsedAs: testutil.NewSet("attribute2")},
					}},
			},
		},
		{
			name:   "from module import attributes aliased",
			source: "from mymodule import attribute1 as a1, attribute2 as a2",
			want: map[string]Import{
				"mymodule": {
					Name: "mymodule",
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", UsedAs: testutil.NewSet("a1")},
						"attribute2": {Name: "attribute2", UsedAs: testutil.NewSet("a2")},
					}},
			},
		},
		{
			name: "from module import attributes aliased multiple times",
			source: testutil.UnIndent(`
				from mymodule import attribute1
				from mymodule import attribute1 as a1
				from mymodule import attribute1 as a2
				`),
			want: map[string]Import{
				"mymodule": {
					Name: "mymodule",
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", UsedAs: testutil.NewSet("attribute1", "a1", "a2")},
					}},
			},
		},
		{
			name:   "import sibling",
			source: "from . import mymodule",
			want: map[string]Import{
				".": {ParentModule: ".", ImportedAttributes: map[string]Attribute{"mymodule": {Name: "mymodule", UsedAs: testutil.NewSet("mymodule")}}},
			},
		},
		{
			name: "various imports",
			source: `
					import module1
					import module2.submodule1
					from module3 import attribute1, attribute2
					from .. import attribute1
			`,
			want: map[string]Import{
				"module1": {
					Name:   "module1",
					UsedAs: testutil.NewSet("module1"),
				},
				"module2.submodule1": {
					ParentModule: "module2",
					Name:         "submodule1",
					UsedAs:       testutil.NewSet("module2.submodule1"),
				},
				"module3": {
					Name: "module3",
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", UsedAs: testutil.NewSet("attribute1")},
						"attribute2": {Name: "attribute2", UsedAs: testutil.NewSet("attribute2")},
					},
				},
				"..": {
					ParentModule: "..",
					Name:         "",
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", UsedAs: testutil.NewSet("attribute1")},
					},
				},
			},
		},
		{
			name: "import module aliased twice",
			source: `
				import module1 as a
				import module1 as b
`,
			want: map[string]Import{
				"module1": {Name: "module1", UsedAs: testutil.NewSet("a", "b")},
			},
		},
		{
			name: "imported twice once with alias",
			source: `
				import module1
				import module1 as a
`,
			want: map[string]Import{
				"module1": {Name: "module1", UsedAs: testutil.NewSet("module1", "a")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}

			imports := FindImports(f)
			if len(tt.want) != len(imports) {
				t.Log(imports)
			}
			assert.Equal(len(tt.want), len(imports))
			for qualifiedName, i := range imports {
				if expected, ok := tt.want[qualifiedName]; assert.Truef(ok, "import not found for name: %s", qualifiedName) {
					validateImport(assert, f.Program(), expected, i)
				}
			}
		})
	}
}

func TestResolveFileDependencies(t *testing.T) {
	cases := []struct {
		name   string
		input  map[string]string
		expect core.FileDependencies
		// expectFailureDueTo is a string that's non-empty if we expect this test to fail
		expectFailureDueTo string
	}{
		{
			name: "import single attribute",
			input: map[string]string{
				"main.py":  `from other import my_method`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet("my_method"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import module attribute",
			input: map[string]string{
				"main.py":         `from shared import other`,
				"shared/other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"shared/other.py": testutil.NewSet[string](),
				},
				"shared/other.py": map[string]core.References{},
			},
		},
		{
			name: "import two attributes",
			input: map[string]string{
				"main.py":  `from other import method_a, method_2`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet("method_a", "method_2"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import attribute with alias",
			input: map[string]string{
				"main.py":  `from other import method_a as aaa`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet("method_a"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			input: map[string]string{
				"main.py":  `from other import method_a as aaa`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet("method_a"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import full module and use method",
			input: map[string]string{
				"main.py": `
import other
other.hello_world()
		`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet("hello_world"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import full module and use var",
			input: map[string]string{
				"main.py": `
import other
print(other.hello_world)
		`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet("hello_world"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import full module but unused",
			input: map[string]string{
				"main.py":  `import other`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet[string](),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import full module with alias",
			input: map[string]string{
				"main.py": `
import other as some_other
some_other.hello_world()
		`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet("hello_world"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import qualified module",
			input: map[string]string{
				"main.py":        `import other.hello`,
				"other/hello.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other/hello.py": testutil.NewSet[string](),
				},
				"other/hello.py": map[string]core.References{},
			},
		},
		{
			name: "import qualified module with attribute",
			input: map[string]string{
				"main.py":        `from other.hello import a`,
				"other/hello.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other/hello.py": testutil.NewSet("a"),
				},
				"other/hello.py": map[string]core.References{},
			},
		},
		{ // TODO https://github.com/klothoplatform/klotho-history/issues/492
			expectFailureDueTo: "#492",
			name:               "import qualified module and use method",
			input: map[string]string{
				"main.py": `
import other.hello
other.hello.say_hi()
		`,
				"other/hello.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other/hello.py": testutil.NewSet("say_hi"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{ // TODO https://github.com/klothoplatform/klotho-history/issues/492
			expectFailureDueTo: "#492",
			name:               "import deep qualified module and use method", // like above, but "import a.b.c" instead of "… a.b"
			input: map[string]string{
				"main.py": `
import other.hello.world
other.hello.world.say_hi()
`,
				"other/hello/world.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other/hello/world.py": testutil.NewSet("say_hi"),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "two imports",
			input: map[string]string{
				"main.py": "from foo import bar\nfrom fizz import buzz\n",
				"foo.py":  `pass`,
				"fizz.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"foo.py":  testutil.NewSet("bar"),
					"fizz.py": testutil.NewSet("buzz"),
				},
				"foo.py":  map[string]core.References{},
				"fizz.py": map[string]core.References{},
			},
		},
		{
			name: "import file missing",
			input: map[string]string{
				"main.py":  `import something_not_found`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py":  map[string]core.References{},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import sibling module",
			input: map[string]string{
				"main.py":  `from . import other`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet[string](),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import uncle module",
			input: map[string]string{
				"mod/main.py": `from .. import other`,
				"other.py":    `pass`,
			},
			expect: map[string]core.Imported{
				"mod/main.py": map[string]core.References{
					"other.py": testutil.NewSet[string](),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import sibling module short",
			input: map[string]string{
				"main.py":  `import .other`,
				"other.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"other.py": testutil.NewSet[string](),
				},
				"other.py": map[string]core.References{},
			},
		},
		{
			name: "import uncle module short",
			input: map[string]string{
				"mod/main.py": `import ..other`,
				"other.py":    `pass`,
			},
			expect: map[string]core.Imported{
				"mod/main.py": map[string]core.References{
					"other.py": testutil.NewSet[string](),
				},
				"other.py": map[string]core.References{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			inputFiles := make(map[string]core.File)
			for path, contents := range tt.input {
				file, err := NewFile(path, strings.NewReader(contents))
				if !assert.NoError(err) {
					return
				}
				inputFiles[path] = file
			}
			actual, err := ResolveFileDependencies(inputFiles)
			if assert.NoError(err) {
				if tt.expectFailureDueTo != "" {
					assert.NotEqualf(
						tt.expect,
						actual,
						"expected to fail due to %v. Do you need to un-ignore this test?",
						tt.expectFailureDueTo)
					t.Skipf("skipping due to expected failure because of %v", tt.expectFailureDueTo)
				} else {
					assert.Equal(tt.expect, actual)
				}
			}
		})
	}
}

func TestFindImportedFile(t *testing.T) {
	cases := []struct {
		name               string
		moduleName         string
		relativeToFilePath string
		files              []string
		expect             string
		expectErr          bool
	}{
		{
			name:               "absolute import exists",
			moduleName:         "foo",
			relativeToFilePath: "some_file.py",
			files:              []string{"foo.py"},
			expect:             "foo.py",
		},
		{
			name:               "absolute import doesn't exist",
			moduleName:         "foo",
			relativeToFilePath: "some_file.py",
			files:              []string{"bar.py"},
			expect:             "",
		},
		{
			name:               "absolute import ignores relative file",
			moduleName:         "foo",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"bar.py"},
			expect:             "",
		},
		{
			name:               "relative import to simple module exists",
			moduleName:         ".foo",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"path/to/foo.py"},
			expect:             "path/to/foo.py",
		},
		{
			name:               "relative import to simple module doesn't exist",
			moduleName:         ".foo",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"path/to/bar.py"},
			expect:             "",
		},
		{
			name:               "relative import goes to parent",
			moduleName:         "..foo",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"path/foo.py"},
			expect:             "path/foo.py",
		},
		{
			name:               "relative import goes to parent that doesn't exist",
			moduleName:         "..foo",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"path/to/foo.py"}, // note: moduleName implies this should be at path/foo.py!
			expect:             "",
		},
		{
			name:               "relative import goes to grandparent",
			moduleName:         "...foo",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"foo.py"},
			expect:             "foo.py",
		},
		{
			name:               "relative import goes too far up",
			moduleName:         "....foo",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"foo.py"},
			expectErr:          true,
		},
		{
			name:               "parent module",
			moduleName:         "..",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"path/__init__.py"},
			expect:             "path/__init__.py",
		},
		{
			name:               "my module",
			moduleName:         ".",
			relativeToFilePath: "path/to/some_file.py",
			files:              []string{"path/to/__init__.py"},
			expect:             "path/to/__init__.py",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			fileSet := make(map[string]struct{})
			for _, f := range tt.files {
				fileSet[f] = struct{}{}
			}
			actual, err := findImportedFile(tt.moduleName, tt.relativeToFilePath, fileSet)
			if tt.expectErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expect, actual)
		})
	}
}

func validateImport(assert *assert.Assertions, content []byte, expected Import, actual Import) {
	assert.Equal(expected.ParentModule, actual.ParentModule)
	assert.Equal(expected.Name, actual.Name)
	assert.Equal(expected.UsedAs, actual.UsedAs)

	assert.Equal(len(expected.ImportedAttributes), len(actual.ImportedAttributes))
	for i := range expected.ImportedAttributes {
		validateAttribute(assert, content, expected.ImportedAttributes[i], actual.ImportedAttributes[i])
	}
}

func validateAttribute(assert *assert.Assertions, content []byte, expected Attribute, actual Attribute) {
	assert.Equal(expected.Name, actual.Name)

}

func Test_referencesForImport(t *testing.T) {
	tests := []struct {
		name          string
		program       string
		importModules map[string]struct{}
		want          core.References
	}{
		{
			name: "imported function call",
			program: `import blah
blah.foo()`,
			importModules: testutil.NewSet("blah"),
			want:          core.References{"foo": {}},
		},
		{
			name: "imported constant",
			program: `import blah
b = blah.a + 1`,
			importModules: testutil.NewSet("blah"),
			want:          core.References{"a": {}},
		},
		{
			name: "imported subproperty",
			program: `import blah
blah.a.b()`,
			importModules: testutil.NewSet("blah"),
			want:          core.References{"a": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			parsed, err := NewFile("main.py", strings.NewReader(tt.program))
			if !assert.NoError(err) {
				return
			}

			got := referencesForImport(parsed.Tree().RootNode(), tt.importModules)
			assert.Equal(tt.want, got)
		})
	}
}

func Test_pythonModuleToPath(t *testing.T) {
	// For ease of understanding, all test cases operate in the following directory structure:
	// .
	// ├─ app/
	// │  ├─ models/
	// │  │  └─ data.py
	// │  └─ main.py
	// ├─ shared/
	// │  ├─ util.py
	// │  └─ blah.py
	// ├─ foo.py
	// └─ bar.py
	tests := []struct {
		name               string
		module             string
		relativeToFilePath string
		want               string
		wantErr            bool
	}{
		{
			name:               "absolute",
			module:             "foo",
			relativeToFilePath: "bar.py",
			want:               "foo.py",
		},
		{
			name:               "absolute in folder",
			module:             "foo",
			relativeToFilePath: "app/main.py",
			want:               "foo.py",
		},
		{
			name:               "absolute submodule",
			module:             "shared.util",
			relativeToFilePath: "bar.py",
			want:               "shared/util.py",
		},
		{
			name:               "relative",
			module:             ".foo",
			relativeToFilePath: "bar.py",
			want:               "foo.py",
		},
		{
			name:               "relative submodule",
			module:             ".shared.util",
			relativeToFilePath: "bar.py",
			want:               "shared/util.py",
		},
		{
			name:               "relative inside submodule",
			module:             ".blah",
			relativeToFilePath: "shared/util.py",
			want:               "shared/blah.py",
		},
		{
			name:               "relative parent",
			module:             "..foo",
			relativeToFilePath: "app/main.py",
			want:               "foo.py",
		},
		{
			name:               "relative parent submodule",
			module:             "..shared.util",
			relativeToFilePath: "app/main.py",
			want:               "shared/util.py",
		},
		{
			name:               "relative multi ancestor",
			module:             "...foo",
			relativeToFilePath: "app/models/data.py",
			want:               "foo.py",
		},
		{
			name:               "out of bounds error",
			module:             "...foo",
			relativeToFilePath: "foo.py",
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got, err := pythonModuleToPath(tt.module, tt.relativeToFilePath)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}
