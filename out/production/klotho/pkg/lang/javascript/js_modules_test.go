package javascript

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestFindDefaultExport(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "no default export",
			source: `exports.a = a;`,
			want:   "",
		},
		{
			name:   "default export",
			source: `exports = a;`,
			want:   "a",
		},
		{
			name:   "default module.export",
			source: `module.exports = a;`,
			want:   "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			got := FindDefaultExport(f.Tree().RootNode())
			if tt.want == "" {
				assert.Nil(got)
			} else if assert.NotNil(got) {
				assert.Equal(tt.want, got.Content())
			}
		})
	}
}

func TestFindExportForVar(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		varName string
		want    string
	}{
		{
			name:    "no match default export",
			source:  `exports = a;`,
			varName: "a",
			want:    "",
		},
		{
			name:    "export var",
			source:  `exports.a = b;`,
			varName: "a",
			want:    "b",
		},
		{
			name:    "export var expression",
			source:  `exports.a = require('express').Router();`,
			varName: "a",
			want:    "exports.a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			got := FindExportForVar(f.Tree().RootNode(), tt.varName)
			if tt.want == "" {
				assert.Nil(got)
			} else if assert.NotNil(got) {
				assert.Equal(tt.want, got.Content())
			}
		})
	}
}

// TODO consider removing these test cases in favor of those in ./imports_test.go
func TestImportsFilterOfModule(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		module        string
		wantLocalName string
	}{
		{
			name:   "no import",
			source: `const a = 2`,
			module: "target",
		},
		{
			name:          "simple import",
			source:        `const a = require("./target")`,
			module:        "target",
			wantLocalName: "a",
		},
		{
			name:   "not relative import",
			source: `const a = require("target")`,
			module: "target",
		},
		{
			name:   "not import",
			source: `const a = myFunc("target")`,
			module: "target",
		},
		{
			name:          "simple import want js",
			source:        `const a = require("./target")`,
			module:        "target.js",
			wantLocalName: "a",
		},
		{
			name:          "simple import want index.js",
			source:        `const a = require("./target")`,
			module:        "target/index.js",
			wantLocalName: "a",
		},
		{
			name:          "simple import want directory",
			source:        `const a = require("./target")`,
			module:        "target/",
			wantLocalName: "a",
		},
		{
			name:          "typescript wrapped import",
			source:        `const a = __importDefault(require("./target"))`,
			module:        "target/",
			wantLocalName: "a",
		},
		{
			name:          "wrapped import",
			source:        `const a = someFunction(require("./target"))`,
			module:        "target/",
			wantLocalName: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			filteredImports := FindImportsInFile(f).Filter(filter.NewSimpleFilter(
				IsRelativeImportOfModule(tt.module),
				predicate.AnyOf(
					func(p Import) bool { return tt.wantLocalName != "" },
					IsImportOfType(ImportTypeSideEffect))))

			if tt.wantLocalName == "" {
				assert.Empty(filteredImports)
			} else {
				if !assert.Len(filteredImports, 1) {
					return
				}
				got := filteredImports[0]
				assert.Equal(tt.wantLocalName, got.ImportedAs())
			}
		})
	}
}

func TestFindFileForImport(t *testing.T) {
	type args struct {
		files        []string
		importedFrom string
		module       string
	}
	tests := []struct {
		name     string
		args     args
		wantPath string
		wantErr  bool
	}{
		{
			name:     "no match",
			args:     args{files: []string{"a.js", "b.js"}, importedFrom: "./index.js", module: "./c"},
			wantPath: "",
		},
		{
			name:     "exact match",
			args:     args{files: []string{"a.js", "b.js"}, importedFrom: "./index.js", module: "./a.js"},
			wantPath: "a.js",
		},
		{
			name:     "no extension match",
			args:     args{files: []string{"a.js", "b.js"}, importedFrom: "./index.js", module: "./a"},
			wantPath: "a.js",
		},
		{
			name:     "folder match",
			args:     args{files: []string{"a/index.js", "b.js"}, importedFrom: "./index.js", module: "./a"},
			wantPath: "a/index.js",
		},
		{
			name:     "in parent match",
			args:     args{files: []string{"a.js", "b.js"}, importedFrom: "./c/index.js", module: "../a"},
			wantPath: "a.js",
		},
		{
			name:     "prioritize exact match",
			args:     args{files: []string{"a", "a.js", "a/index.js", "b.js"}, importedFrom: ".index.js", module: "./a"},
			wantPath: "a",
		},
		{
			name:     "prioritize extension over folder match",
			args:     args{files: []string{"a.js", "a/index.js", "b.js"}, importedFrom: ".index.js", module: "./a"},
			wantPath: "a.js",
		},
		{
			name:     "this folder",
			args:     args{files: []string{"index.js", "b.js"}, importedFrom: "./a.js", module: "."},
			wantPath: "index.js",
		},
		{
			name:     "parent folder",
			args:     args{files: []string{"index.js", "b.js"}, importedFrom: "./a/index.js", module: ".."},
			wantPath: "index.js",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			files := make(map[string]core.File)
			for _, path := range tt.args.files {
				files[path] = &core.RawFile{FPath: path}
			}

			got, err := FindFileForImport(files, tt.args.importedFrom, tt.args.module)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			if tt.wantPath == "" {
				assert.Nil(got)
				return
			}
			assert.Equal(tt.wantPath, got.Path())
		})
	}
}

func TestFileToModule(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantModule string
	}{
		{
			name:       "file",
			path:       "mod.js",
			wantModule: "./mod",
		},
		{
			name:       "root index",
			path:       "index.js",
			wantModule: ".",
		},
		{
			name:       "folder index",
			path:       "a/index.js",
			wantModule: "./a",
		},
		{
			name:       "folder file",
			path:       "a/mod.js",
			wantModule: "./a/mod",
		},
		{
			name:       "parent file",
			path:       "../mod.js",
			wantModule: "../mod",
		},
		{
			name:       "parent index",
			path:       "../index.js",
			wantModule: "..",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got := FileToLocalModule(tt.path)
			assert.Equal(tt.wantModule, got)
		})
	}
}
