package golang

import (
	"fmt"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_GetImportsInFile(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []Import
	}{
		{
			name: "simple import",
			source: `import(
				"os"
)`,
			want: []Import{{Package: "os"}},
		},
		{
			name:   "single import",
			source: `import "os"`,
			want:   []Import{{Package: "os"}},
		},
		{
			name:   "single import with alias",
			source: `import alias "os"`,
			want:   []Import{{Package: "os", Alias: "alias"}},
		},
		{

			name: "multiple imports",
			source: `import(
				"os"
				"github.com/go-chi/chi/v5"
)`,
			want: []Import{{Package: "os"}, {Package: "github.com/go-chi/chi/v5"}},
		},
		{
			name: "multiple imports with an alias",
			source: `import(
				"os"
				chi "github.com/go-chi/chi/v5"
)`,
			want: []Import{{Package: "os"}, {Package: "github.com/go-chi/chi/v5", Alias: "chi"}},
		},
		{
			name: "multiple import blocks",
			source: `import(
				"os"
				chi "github.com/go-chi/chi/v5"
)
	import "net/http"
	import (
		"io/fs"
	)
`,
			want: []Import{{Package: "os"}, {Package: "github.com/go-chi/chi/v5", Alias: "chi"}, {Package: "net/http"}, {Package: "io/fs"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := core.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			imports := GetImportsInFile(f)
			for _, i := range imports {
				found := false
				for _, w := range tt.want {
					if w.Package == i.Package {
						found = true
						assert.Equal(w.Alias, i.Alias)
					}
				}
				assert.True(found)
			}
		})
	}
}

func Test_UpdateImportsInFile(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		importsToAdd    []Import
		importsToRemove []Import
		want            string
	}{
		{
			name: "simple import",
			source: `package test
import(
				"os"
)
something := "something"`,
			importsToAdd: []Import{{Package: "context"}},
			want: `package test

import (
	"context"
	"os"
)

something := "something"`,
		},
		{
			name: "multiple adds import",
			source: `package test
import(
				"os"
)
import fs "io/fs"`,
			importsToAdd: []Import{{Package: "context"}, {Package: "net/http"}},
			want: `package test

import (
	"context"
	"net/http"
	"os"
	fs "io/fs"
)

`,
		},
		{
			name: "simple remove",
			source: `package test
import(
	"os"
	"path"
)`,
			importsToRemove: []Import{{Package: "path"}},
			want: `package test

import (
	"os"
)
`,
		},
		{
			name: "simple remove more than 2 imports",
			source: `package test
import(
	"os"
	ctx "context"
)
import (
	"path"
)`,
			importsToRemove: []Import{{Package: "path"}, {Package: "context"}},
			want: `package test

import (
	"os"
)

`,
		},
		{
			name: "remove and add",
			source: `package test
import(
	"os"
	"path"
	ctx "context"
	chi "github.com/go-chi/chi"
)`,
			importsToAdd:    []Import{{Package: "github.com/go-chi/chi/v5"}, {Package: "github.com/aws/aws-lambda-go/lambda"}},
			importsToRemove: []Import{{Package: "path"}, {Package: "github.com/go-chi/chi"}},
			want: `package test

import (
	"github.com/go-chi/chi/v5"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
	ctx "context"
)
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := core.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			err = UpdateImportsInFile(f, tt.importsToAdd, tt.importsToRemove)
			if !assert.NoError(err) {
				return
			}
			fmt.Println(string(f.Program()))
			assert.Equal(tt.want, string(f.Program()))
		})
	}
}
