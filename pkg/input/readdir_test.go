package input

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestReadDir(t *testing.T) {
	defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	tests := []struct {
		name string
		// All of the files in the mock filesystem
		files map[string]string
		// If specified, a dir within the fs to effectively "cd" into before running ReadDir
		fsRoot string
		// The rootPath arg to pass into ReadDir
		rootPath string
		// The Path()s of the expected files
		want []string
	}{
		{
			name: "js: simple",
			files: map[string]string{
				"fizz/one.js":       "",
				"fizz/package.json": "{}",
			},
			rootPath: "fizz",
			want: []string{
				"one.js",
				"package.json",
			},
		},
		{
			name: "js: everything is one dir down",
			files: map[string]string{
				"fizz/src/one.js":       "",
				"fizz/src/package.json": "{}",
			},
			rootPath: "fizz",
			want: []string{
				"src/one.js",
				"src/package.json",
			},
		},
		{
			name: "js: package is in root's parent",
			files: map[string]string{
				"fizz/one.js":  "",
				"package.json": "{}",
			},
			rootPath: "fizz",
			want: []string{
				"one.js",
				"package.json",
			},
		},
		{
			name: "js: package is in subdir",
			files: map[string]string{
				"fizz/one.js":               "",
				"fizz/foo/bar/package.json": "{}",
			},
			rootPath: "fizz",
			want: []string{
				"one.js",
				"foo/bar/package.json",
			},
		},
		{
			name: "js: don't look above root",
			files: map[string]string{
				"fizz/one.js":       "",
				"fizz/package.json": "{}",
				"two.js":            "{}",
			},
			rootPath: "fizz",
			want: []string{
				"one.js",
				"package.json",
			},
		},
		{
			name: "js: no package",
			files: map[string]string{
				"parent/src/one.js":   "",   // will be within ./new_cwd/src
				"parent/package.json": "{}", // will be within ./ (so, above cwd)
			},
			fsRoot:   "parent",
			rootPath: "parent/src",
			want:     nil, // expect an err due to no package.json in parent/src
		},
		{
			name: "py: simple",
			files: map[string]string{
				"fizz/one.py":           "",
				"fizz/requirements.txt": "",
			},
			rootPath: "fizz",
			want: []string{
				"one.py",
				"requirements.txt",
			},
		},
		{
			name: "multi-language",
			files: map[string]string{
				"fizz/js/one.js":        "",
				"fizz/package.json":     "{}",
				"fizz/py/two.py":        "",
				"fizz/requirements.txt": "",
			},
			rootPath: "fizz",
			want: []string{
				"js/one.js",
				"package.json",
				"py/two.py",
				"requirements.txt",
			},
		},
		{
			name: "file refs for assets",
			files: map[string]string{
				"example.txt":    "",
				"big.zip":        "",
				"subfolder/file": "",
			},
			rootPath: ".",
			want: []string{
				"example.txt",
				"big.zip",
				"subfolder/file",
			},
		},
		{
			name: "excludes config file",
			files: map[string]string{
				"example.yaml": "",
				"klotho.yaml":  "",
			},
			rootPath: ".",
			want: []string{
				"example.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			// fullMockFs is the "full" mock of the fs. Once we create it, we essentially cd
			// into tt.fsRoot. This lets us test that we never cd higher than that in looking
			// for package files.
			fullMockFs := make(fstest.MapFS)
			for path, contents := range tt.files {
				fullMockFs[path] = &fstest.MapFile{
					Data:    []byte(contents),
					Mode:    0700,
					ModTime: time.Now(),
					Sys:     struct{}{},
				}
			}
			if tt.fsRoot == "" {
				tt.fsRoot = "."
			}
			mockFs, err := fs.Sub(fullMockFs, tt.fsRoot)
			if !assert.NoError(err) {
				return
			}
			app := config.Application{
				Path: tt.rootPath,
			}
			files, err := ReadDir(mockFs, app, "klotho.yaml")

			if tt.want == nil {
				assert.Error(err)
			} else {
				if !assert.NoError(err) {
					return
				}
				var actual []string
				for _, f := range files.Files() {
					actual = append(actual, f.Path())
				}
				assert.ElementsMatch(tt.want, actual)
			}
		})
	}
}

func Test_splitPathRoot(t *testing.T) {
	tests := []struct {
		name     string
		cfgPath  string
		wantRoot string
		wantPath string
	}{
		{
			name:     "dot",
			cfgPath:  ".",
			wantRoot: ".",
			wantPath: ".",
		},
		{
			name:     "relative dir",
			cfgPath:  "dist",
			wantRoot: ".",
			wantPath: "dist",
		},
		{
			name:     "absolute dir",
			cfgPath:  "/path/to/source",
			wantRoot: "/",
			wantPath: "path/to/source",
		},
		{
			name:     "has parent dirs",
			cfgPath:  "../../source",
			wantRoot: "../..",
			wantPath: "source",
		},
		{
			name:     "cleans the path",
			cfgPath:  "././././source/sub/..",
			wantRoot: ".",
			wantPath: "source",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			gotRoot, gotPath := splitPathRoot(tt.cfgPath)
			assert.Equal(tt.wantRoot, gotRoot, "root")
			assert.Equal(tt.wantPath, gotPath, "path")
		})
	}
}
