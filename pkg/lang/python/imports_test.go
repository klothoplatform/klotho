package python

import (
	"fmt"
	"strings"
	"testing"

	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"

	"github.com/klothoplatform/klotho/pkg/core"
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
				"mymodule": {Name: "mymodule", ImportedSelf: true},
			},
		},
		{
			name:   "import module aliased",
			source: "import mymodule as m",
			want: map[string]Import{
				"mymodule": {Name: "mymodule", ImportedSelf: true, Alias: "m"},
			},
		},
		{
			name:   "import modules",
			source: "import mymodule1, mymodule2",
			want: map[string]Import{
				"mymodule1": {Name: "mymodule1", ImportedSelf: true},
				"mymodule2": {Name: "mymodule2", ImportedSelf: true},
			},
		},
		{
			name:   "import modules aliased",
			source: "import mymodule1 as m1, mymodule2 as m2",
			want: map[string]Import{
				"mymodule1": {Name: "mymodule1", ImportedSelf: true, Alias: "m1"},
				"mymodule2": {Name: "mymodule2", ImportedSelf: true, Alias: "m2"},
			},
		},
		{
			name:   "import submodule",
			source: "import mymodule.submodule",
			want: map[string]Import{
				"mymodule.submodule": {ParentModule: "mymodule", Name: "submodule", ImportedSelf: true},
			},
		},
		{
			name:   "import submodule aliased",
			source: "import mymodule.submodule as w",
			want: map[string]Import{
				"mymodule.submodule": {ParentModule: "mymodule", Name: "submodule", ImportedSelf: true, Alias: "w"},
			},
		},
		{
			name:   "import aliased nested submodule",
			source: "import mymodule.submodule1.submodule2 as w",
			want: map[string]Import{
				"mymodule.submodule1.submodule2": {
					ParentModule: "mymodule.submodule1",
					Name:         "submodule2",
					ImportedSelf: true,
					Alias:        "w",
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
					ImportedSelf: false,
					ImportedAttributes: map[string]Attribute{
						"mymodule": {Name: "mymodule"},
					}},
				"..parent": {
					ParentModule:       "..",
					Name:               "parent",
					ImportedSelf:       false,
					ImportedAttributes: map[string]Attribute{"child.attribute": {Name: "child.attribute"}},
				},
			},
		},
		{
			name:   "from module import attribute",
			source: "from mymodule import attribute",
			want: map[string]Import{
				"mymodule": {
					Name:         "mymodule",
					ImportedSelf: false,
					ImportedAttributes: map[string]Attribute{
						"attribute": {Name: "attribute"},
					}},
			},
		},
		{
			name: "import multiple attributes from module in separate imports",
			source: `from mymodule import attribute1
			         from mymodule import attribute2`,
			want: map[string]Import{
				"mymodule": {
					Name:         "mymodule",
					ImportedSelf: false,
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1"},
						"attribute2": {Name: "attribute2"},
					}},
			},
		},
		{
			name:   "from module import attributes",
			source: "from mymodule import attribute1, attribute2",
			want: map[string]Import{
				"mymodule": {
					Name:         "mymodule",
					ImportedSelf: false,
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1"},
						"attribute2": {Name: "attribute2"},
					}},
			},
		},
		{
			name:   "from module import attributes aliased",
			source: "from mymodule import attribute1 as a1, attribute2 as a2",
			want: map[string]Import{
				"mymodule": {
					Name:         "mymodule",
					ImportedSelf: false,
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1", Alias: "a1"},
						"attribute2": {Name: "attribute2", Alias: "a2"},
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
			want: map[string]Import{
				"module1": {
					Name:         "module1",
					ImportedSelf: true,
				},
				"module2.submodule1": {
					ParentModule: "module2",
					Name:         "submodule1",
					ImportedSelf: true,
				},
				"module3": {
					Name:         "module3",
					ImportedSelf: false,
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1"},
						"attribute2": {Name: "attribute2"},
					},
				},
				"..": {
					ParentModule: "..",
					Name:         "",
					ImportedSelf: false,
					ImportedAttributes: map[string]Attribute{
						"attribute1": {Name: "attribute1"},
					},
				},
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
				fmt.Println(imports)
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
		expect execunit.FileDependencies
		// expectFailureDueTo is a string that's non-empty if we expect this test to fail
		expectFailureDueTo string
	}{
		{
			name: "import single attribute",
			input: map[string]string{
				"main.py":  `from other import my_method`,
				"other.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other.py": NewSet("my_method"),
				},
				"other.py": map[string]execunit.References{},
			},
		},
		{
			name: "import two attributes",
			input: map[string]string{
				"main.py":  `from other import method_a, method_2`,
				"other.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other.py": NewSet("method_a", "method_2"),
				},
				"other.py": map[string]execunit.References{},
			},
		},
		{
			name: "import attribute with alias",
			input: map[string]string{
				"main.py":  `from other import method_a as aaa`,
				"other.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other.py": NewSet("aaa"),
				},
				"other.py": map[string]execunit.References{},
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
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other.py": NewSet[string]("hello_world"),
				},
				"other.py": map[string]execunit.References{},
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
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other.py": NewSet[string]("hello_world"),
				},
				"other.py": map[string]execunit.References{},
			},
		},
		{
			name: "import full module but unused",
			input: map[string]string{
				"main.py":  `import other`,
				"other.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other.py": NewSet[string](),
				},
				"other.py": map[string]execunit.References{},
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
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other.py": NewSet[string]("hello_world"),
				},
				"other.py": map[string]execunit.References{},
			},
		},
		{
			expectFailureDueTo: "#492",
			name:               "import qualified module and use method",
			input: map[string]string{
				"main.py": `
import other.hello
other.hello.say_hi()
`,
				"other/hello.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other/hello.py": NewSet[string]("say_hi"),
				},
				"other.py": map[string]execunit.References{},
			},
		},
		{
			expectFailureDueTo: "#492",
			name:               "import deep qualified module and use method", // like above, but "import a.b.c" instead of "… a.b"
			input: map[string]string{
				"main.py": `
import other.hello.world
other.hello.world.say_hi()
`,
				"other/hello/world.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"other/hello/world.py": NewSet[string]("say_hi"),
				},
				"other.py": map[string]execunit.References{},
			},
		},
		{
			name: "two imports",
			input: map[string]string{
				"main.py": "from foo import bar\nfrom fizz import buzz\n",
				"foo.py":  `pass`,
				"fizz.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py": map[string]execunit.References{
					"foo.py":  NewSet("bar"),
					"fizz.py": NewSet("buzz"),
				},
				"foo.py":  map[string]execunit.References{},
				"fizz.py": map[string]execunit.References{},
			},
		},
		{
			name: "import file missing",
			input: map[string]string{
				"main.py":  `import something_not_found`,
				"other.py": `pass`,
			},
			expect: map[string]execunit.Imported{
				"main.py":  map[string]execunit.References{},
				"other.py": map[string]execunit.References{},
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
			expect:             "",
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
			actual := findImportedFile(tt.moduleName, tt.relativeToFilePath, ".", fileSet)
			assert.Equal(tt.expect, actual)
		})
	}
}

func TestFindPyRoot(t *testing.T) {
	cases := []struct {
		name       string
		relativeTo string
		files      []string
		expect     string
	}{
		{
			name:       "all at root",
			relativeTo: "main.py",
			files: []string{
				"hello.py",
				"world.py",
			},
			expect: ".",
		},
		{
			name:       "all are two down",
			relativeTo: "a/b/main.py",
			files: []string{
				"a/b/hello.py",
				"a/b/world.py",
			},
			expect: "a/b",
		},
		{
			name:       "other files are lower down",
			relativeTo: "a/main.py",
			files: []string{
				"a/b/hello.py",
				"a/b/c/world.py",
			},
			expect: "a",
		},
		{
			name:       "other files are higher up",
			relativeTo: "a/b/c/d/main.py",
			files: []string{
				"a/hello.py",
				"a/world.py",
			},
			expect: "a",
		},
		{
			name:       "other files are in different sub-dirs",
			relativeTo: "a/b/c/d/main.py",
			files: []string{
				"a/e/hello.py",
				"a/f/world.py",
			},
			expect: "a",
		},
		{
			name:       "different top-level dirs",
			relativeTo: "src/main.py",
			files: []string{
				"test/hello.py",
			},
			expect: "src",
		},
		{
			name:       "no files",
			relativeTo: "a/b/main.py",
			files:      []string{},
			expect:     "a/b",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			filesMap := make(map[string]core.File)
			for _, path := range tt.files {
				pyFile, err := NewFile(path, strings.NewReader(""))
				if assert.NoError(err) {
					filesMap[path] = pyFile
				}
			}
			actual := findPyRoot(tt.relativeTo, filesMap)
			assert.Equal(tt.expect, actual)
		})
	}

}

func validateImport(assert *assert.Assertions, content []byte, expected Import, actual Import) {
	assert.Equal(expected.ParentModule, actual.ParentModule)
	assert.Equal(expected.Name, actual.Name)
	assert.Equal(expected.ImportedSelf, actual.ImportedSelf)

	assert.Equal(len(expected.ImportedAttributes), len(actual.ImportedAttributes))
	for i := range expected.ImportedAttributes {
		validateAttribute(assert, content, expected.ImportedAttributes[i], actual.ImportedAttributes[i])
	}
}

func validateAttribute(assert *assert.Assertions, content []byte, expected Attribute, actual Attribute) {
	assert.Equal(expected.Name, actual.Name)

}

func NewSet[K comparable](elements ...K) map[K]struct{} {
	result := make(map[K]struct{}, len(elements))
	for _, elem := range elements {
		result[elem] = struct{}{}
	}
	return result
}
