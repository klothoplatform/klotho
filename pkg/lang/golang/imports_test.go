package golang

import (
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := core.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			imports := GetImportsInFile(f)
			assert.ElementsMatch(tt.want, imports)
		})
	}
}

func Test_UpdateImportsInFile(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		importsToAdd    []string
		importsToRemove []string
		want            string
	}{
		{
			name: "simple import",
			source: `import(
				"os"
)`,
			importsToAdd: []string{"context"},
			want: `
import (
	"os"
	"context"
)`,
		},
		{
			name: "simple remove",
			source: `import(
				"os"
				"path"
)`,
			importsToRemove: []string{"path"},
			want: `
import (
	"os"
)`,
		},
		{
			name: "simple remove more than 2 imports",
			source: `import(
				"os"
				"path"
				ctx "context"
)`,
			importsToRemove: []string{"path"},
			want: `
import (
	"os"
	ctx "context"
)`,
		},
		{
			name: "remove and add",
			source: `import(
				"os"
				"path"
				ctx "context"
				chi "github.com/go-chi/chi"
)`,
			importsToAdd:    []string{"github.com/go-chi/chi/v5", "github.com/aws/aws-lambda-go/lambda"},
			importsToRemove: []string{"path", "github.com/go-chi/chi"},
			want: `
import (
	"os"
	ctx "context"
	"github.com/go-chi/chi/v5"
	"github.com/aws/aws-lambda-go/lambda"
)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := core.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			newFile := UpdateImportsInFile(f, tt.importsToAdd, tt.importsToRemove)
			assert.Equal(tt.want, newFile)
		})
	}
}
