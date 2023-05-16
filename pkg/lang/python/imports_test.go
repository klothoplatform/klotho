package python

import (
	"bytes"
	"fmt"
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
			want: Imports{
				"mymodule": {Name: "mymodule", UsedAs: testutil.NewSet("mymodule")},
			},
		},
		{
			name:   "import module aliased",
			source: "import mymodule as m",
			want: Imports{
				"mymodule": {Name: "mymodule", UsedAs: testutil.NewSet("m")},
			},
		},
		{
			name:   "import modules",
			source: "import mymodule1, mymodule2",
			want: Imports{
				"mymodule1": {Name: "mymodule1", UsedAs: testutil.NewSet("mymodule1")},
				"mymodule2": {Name: "mymodule2", UsedAs: testutil.NewSet("mymodule2")},
			},
		},
		{
			name:   "import modules aliased",
			source: "import mymodule1 as m1, mymodule2 as m2",
			want: Imports{
				"mymodule1": {Name: "mymodule1", UsedAs: testutil.NewSet("m1")},
				"mymodule2": {Name: "mymodule2", UsedAs: testutil.NewSet("m2")},
			},
		},
		{
			name:   "import submodule",
			source: "import mymodule.submodule",
			want: Imports{
				"mymodule.submodule": {
					ParentModule: "mymodule",
					Name:         "submodule",
					UsedAs:       testutil.NewSet("mymodule.submodule")},
			},
		},
		{
			name:   "import submodule aliased",
			source: "import mymodule.submodule as w",
			want: Imports{
				"mymodule.submodule": {ParentModule: "mymodule", Name: "submodule", UsedAs: testutil.NewSet("w")},
			},
		},
		{
			name:   "import aliased nested submodule",
			source: "import mymodule.submodule1.submodule2 as w",
			want: Imports{
				"mymodule.submodule1.submodule2": {
					ParentModule: "mymodule.submodule1",
					Name:         "submodule2",
					UsedAs:       testutil.NewSet("w"),
				},
			},
		},
		{
			name:   "import relative parent module",
			source: "from .. import mymodule\nfrom ..parent import child.attribute",
			want: Imports{
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
			name:   "import relative sibling module",
			source: "from . import foo\nfrom .foo import bar\nfrom .x.y.z import a",
			want: Imports{
				".": {
					ParentModule: ".",
					Name:         "",
					ImportedAttributes: map[string]Attribute{
						"foo": {Name: "foo", UsedAs: testutil.NewSet("foo")},
					}},
				".foo": {
					ParentModule: ".",
					Name:         "foo",
					ImportedAttributes: map[string]Attribute{
						"bar": {Name: "bar", UsedAs: testutil.NewSet("bar")},
					}},
				".x.y.z": {
					ParentModule: ".x.y",
					Name:         "z",
					ImportedAttributes: map[string]Attribute{
						"a": {Name: "a", UsedAs: testutil.NewSet("a")},
					}},
			},
		},
		{
			name:   "from module import attribute",
			source: "from mymodule import attribute",
			want: Imports{
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
			want: Imports{
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
			want: Imports{
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
			want: Imports{
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
			want: Imports{
				"mymodule": {
					Name: "mymodule",
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", UsedAs: testutil.NewSet("attribute1", "a1", "a2")},
					}},
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
			want: Imports{
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
			want: Imports{
				"module1": {Name: "module1", UsedAs: testutil.NewSet("a", "b")},
			},
		},
		{
			name: "imported twice once with alias",
			source: `
				import module1
				import module1 as a
`,
			want: Imports{
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

			imports := FindFileImports(f)
			assertImportsEqual(t, f.Program(), tt.want, imports)
		})
	}
}

func assertImportsEqual(t *testing.T, program []byte, expect, actual Imports) {
	t.Helper()
	assert := assert.New(t)
	expectKeys := make([]string, 0, len(expect))
	actualKeys := make([]string, 0, len(actual))
	for k := range expect {
		expectKeys = append(expectKeys, k)
	}
	for k := range actual {
		actualKeys = append(actualKeys, k)
	}
	if !assert.ElementsMatch(expectKeys, actualKeys, "import keys") {
		return
	}

	for k, expectV := range expect {
		actualV := actual[k]
		validateImport(t, program, expectV, actualV)
	}
}

func validateImport(t *testing.T, content []byte, expected Import, actual Import) {
	t.Helper()
	assert := assert.New(t)
	assert.Equal(expected.ParentModule, actual.ParentModule, "ParentModule")
	assert.Equal(expected.Name, actual.Name, "Name")
	assert.Equal(expected.UsedAs, actual.UsedAs, "UsedAs")

	expectedAttrs := make([]string, 0, len(expected.ImportedAttributes))
	actualAttrs := make([]string, 0, len(actual.ImportedAttributes))
	for i := range expected.ImportedAttributes {
		expectedAttrs = append(expectedAttrs, expected.ImportedAttributes[i].Name)
	}
	for i := range actual.ImportedAttributes {
		actualAttrs = append(actualAttrs, actual.ImportedAttributes[i].Name)
	}
	assert.ElementsMatch(expectedAttrs, actualAttrs, "imported attributes")
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
		{
			name: "import multiple with submodule",
			input: map[string]string{
				"main.py":    `from foo import bar, baz`,
				"foo.py":     `pass`,
				"foo/bar.py": `pass`,
			},
			expect: map[string]core.Imported{
				"main.py": map[string]core.References{
					"foo.py":     testutil.NewSet("baz"),
					"foo/bar.py": testutil.NewSet[string](),
				},
				"foo.py":     map[string]core.References{},
				"foo/bar.py": map[string]core.References{},
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

func Test_dependenciesForImport(t *testing.T) {
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
	baseFiles := map[string]core.File{
		"app/main.py":        nil,
		"app/models/data.py": nil,
		"shared/util.py":     nil,
		"shared/blah.py":     nil,
		"foo.py":             nil,
		"bar.py":             nil,
	}
	tests := []struct {
		name           string
		relativeToPath string
		spec           Import
		want           core.Imported
		wantErr        bool
	}{
		{
			name: "direct import",
			// import foo
			spec:           Import{Name: "foo"},
			relativeToPath: "bar.py",
			want:           core.Imported{"foo.py": {}},
		},
		{
			name: "reference imports",
			// import foo
			spec:           Import{Name: "foo"},
			relativeToPath: "bar.py",
			want:           core.Imported{"foo.py": {"x": {}}},
		},
		{
			name: "import non-module attributes",
			// from foo import x
			spec:           Import{Name: "foo", ImportedAttributes: map[string]Attribute{"x": {}}},
			relativeToPath: "bar.py",
			want:           core.Imported{"foo.py": {"x": {}}},
		},
		{
			name: "import sibling module",
			// from . import foo
			spec:           Import{ParentModule: ".", ImportedAttributes: map[string]Attribute{"foo": {}}},
			relativeToPath: "bar.py",
			want:           core.Imported{"foo.py": {}},
		},
		{
			name: "import module attributes",
			// from .models import data
			spec:           Import{ParentModule: ".", Name: "models", ImportedAttributes: map[string]Attribute{"data": {}}},
			relativeToPath: "app/main.py",
			want:           core.Imported{"app/models/data.py": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			// Make a copy of the base files so the test can add the 'relativeToPath' file
			// with content based on the expected references.
			files := make(map[string]core.File)
			for p, f := range baseFiles {
				files[p] = f
			}

			if tt.spec.UsedAs == nil {
				tt.spec.UsedAs = map[string]struct{}{"test": {}}
			}

			// Create a file with the expected references for use by any
			// referencesForImport calls.
			fileBuf := new(bytes.Buffer)
			for used := range tt.spec.UsedAs {
				fmt.Fprintf(fileBuf, "import %s as %s\n", tt.spec.FullyQualifiedModule(), used)
				for _, refs := range tt.want {
					for ref := range refs {
						fmt.Fprintf(fileBuf, "%s.%s\n", used, ref)
					}
				}
			}

			// Fill in the ImportedAttributes to allow more short-hand specification
			// in the test cases.
			for attrName, attr := range tt.spec.ImportedAttributes {
				attr.Name = attrName
				if len(attr.UsedAs) == 0 {
					attr.UsedAs = map[string]struct{}{attr.Name: {}}
				}
				tt.spec.ImportedAttributes[attrName] = attr
			}

			file, err := NewFile(tt.relativeToPath, fileBuf)
			if !assert.NoError(err) {
				return
			}
			files[tt.relativeToPath] = file

			got, err := dependenciesForImport(tt.relativeToPath, tt.spec, files)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}
			assertImportedEqual(t, tt.want, got)
		})
	}
}

func assertImportedEqual(t *testing.T, want, got core.Imported) {
	t.Helper()

	assert := assert.New(t)

	gotKeys := make([]string, 0, len(got))
	wantKeys := make([]string, 0, len(want))
	for k := range got {
		gotKeys = append(gotKeys, k)
	}
	for k := range want {
		wantKeys = append(wantKeys, k)
	}
	if !assert.ElementsMatch(wantKeys, gotKeys) {
		return
	}

	for k, wantV := range want {
		gotV := got[k]
		assert.Equal(wantV, gotV, "key '%s'", k)
	}
}

func TestImport_FullyQualifiedModule(t *testing.T) {
	tests := []struct {
		name string
		imp  Import
		want string
	}{
		{
			name: "name only",
			imp:  Import{Name: "foo"},
			want: "foo",
		},
		{
			name: "parent only",
			imp:  Import{ParentModule: "."},
			want: ".",
		},
		{
			name: "name and parent relative",
			imp:  Import{ParentModule: ".", Name: "foo"},
			want: ".foo",
		},
		{
			name: "name and parent absolute",
			imp:  Import{ParentModule: "blah", Name: "foo"},
			want: "blah.foo",
		},
		{
			name: "many parents",
			imp:  Import{ParentModule: "x.y.z", Name: "foo"},
			want: "x.y.z.foo",
		},
		{
			name: "many parents relative",
			imp:  Import{ParentModule: ".x.y.z", Name: "foo"},
			want: ".x.y.z.foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.imp.FullyQualifiedModule()
			assert.Equal(t, tt.want, got)
		})
	}
}
