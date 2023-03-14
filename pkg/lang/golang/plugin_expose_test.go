package golang

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var testRestAPIHandler = &restAPIHandler{
	log:  zap.L(),
	Unit: &core.ExecutionUnit{Name: "testUnit"},
}

func Test_findHttpListenServe(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectAppVar string
		expectErr    bool
	}{
		{
			name: "simple http listen and serve",
			source: `
				/* @klotho::expose {
				*   target = "public"
				*   id = "app"
				* }
				*/
				http.ListenAndServe(":3000", r)`,
			expectAppVar: "r",
		},
		{
			name: "incorrect http listen and serve",
			source: `
				/* @klotho::expose {
				*   target = "public"
				*   id = "app"
				* }
				*/
				http.TalkAndTake(":3000", r)`,
			expectAppVar: "",
			expectErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			var annot *core.Annotation
			for _, v := range f.Annotations() {
				annot = v
				break
			}
			listener, err := testRestAPIHandler.findHttpListenAndServe(annot, f)

			if tt.expectErr {
				assert.Error(err)
				return
			}

			if !assert.NotNil(listener.Expression, "error in test source app definition function") {
				return
			}

			assert.Equal(tt.expectAppVar, listener.Identifier.Content())
		})
	}
}

func Test_findChiRouterDefinition(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		expectAppVar string
		expectErr    bool
	}{
		{
			name:         "simple chi router definition",
			source:       `r := chi.NewRouter()`,
			expectAppVar: "r",
			expectErr:    false,
		},
		// Right now we assume the var router found in the listen and serve is the same var with the router definition
		// In the future we may want to account for var reassignment
		{
			name:         "incorrect router name",
			source:       `r := chi.NewRouter()`,
			expectAppVar: "test",
			expectErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			router, err := testRestAPIHandler.findChiRouterDefinition(f, tt.expectAppVar)
			if tt.expectErr {
				assert.Error(err)
				return
			}

			assert.Equal(tt.expectAppVar, router.Identifier.Content())
		})
	}
}

func Test_removeNetHttpImport(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "simple identifier",
			source: `package test
import (
	"fmt"
	"net/http"
)

http.ListenAndServe(":3000", r)
			`,
			want: `package test
import (
	"fmt"
	"net/http"
)

http.ListenAndServe(":3000", r)
			`,
		},
		{
			name: "simple package identifier",
			source: `package test
import (
	"fmt"
	"net/http"
)

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello!"))
})
			`,
			want: `package test
import (
	"fmt"
	"net/http"
)

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello!"))
})
			`,
		},
		{
			name: "should remove import",
			source: `package test
import (
	"fmt"
	"net/http"
)

random.ListenAndServe(":3000", r)
			`,
			want: `package test

import (
	"fmt"
)


random.ListenAndServe(":3000", r)
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			err = testRestAPIHandler.removeNetHttpImport(f)

			assert.NoError(err)
			assert.Equal(tt.want, string(f.Program()))
		})
	}
}

func Test_findImports(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "simple import block",
			source: `
			import (
				"fmt"
				"net/http"
					
				"github.com/go-chi/chi"
				"github.com/go-chi/chi/middleware"
			)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			importsNode, _ := testRestAPIHandler.FindImports(f)

			assert.NotNil(importsNode)
		})
	}
}

func Test_findChiRouterMounts(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		routerName string
		want       []routerMount
	}{
		{
			name:       "Basic single mount",
			source:     `r.Mount("/test", routes.TestRoutes())`,
			routerName: "r",
			want: []routerMount{
				{
					Path:     "/test",
					FuncName: "TestRoutes",
					PkgAlias: "routes",
				},
			},
		},
		{
			name:       "Incorrect router name",
			source:     `wrong.Mount("/test", routes.TestRoutes())`,
			routerName: "r",
			want:       []routerMount{},
		},
		{
			name: "Multiple router mount",
			source: `
			r.Mount("/test", routes.TestRoutes())
			r.Mount("/test2", routes.TestRoutes2())
			`,
			routerName: "r",
			want: []routerMount{
				{
					Path:     "/test",
					FuncName: "TestRoutes",
					PkgAlias: "routes",
				},
				{
					Path:     "/test2",
					FuncName: "TestRoutes2",
					PkgAlias: "routes",
				},
			},
		},
		{
			name: "Multiple router wrong router mount",
			source: `
			r.Mount("/test", routes.TestRoutes())
			wrong.Mount("/test2", routes.TestRoutes2())
			`,
			routerName: "r",
			want: []routerMount{
				{
					Path:     "/test",
					FuncName: "TestRoutes",
					PkgAlias: "routes",
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
			mounts := testRestAPIHandler.findChiRouterMounts(f, tt.routerName)

			assert.Equal(tt.want, mounts)
		})
	}
}

func Test_findChiRouterMountPackage(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		mount   routerMount
		want    string
		wantErr bool
	}{
		{
			name: "Block import no alias",
			source: `
			import (
				"fmt"
				"net/http"
			
				"github.com/go-chi/chi/v5"
				"github.com/go-chi/chi/v5/middleware"
				"github.com/klothoplatform/demo-app/pkg/routes"
			)`,
			mount: routerMount{PkgAlias: "routes"},
			want:  "routes",
		},
		{
			name: "Block import with matching alias",
			source: `
			import (
				"fmt"
				"net/http"
			
				"github.com/go-chi/chi/v5"
				"github.com/go-chi/chi/v5/middleware"
				routes "github.com/klothoplatform/demo-app/pkg/routes"
			)`,
			mount: routerMount{PkgAlias: "routes"},
			want:  "routes",
		},
		{
			name: "Block import with non matching alias",
			source: `
			import (
				"fmt"
				"net/http"
			
				"github.com/go-chi/chi/v5"
				"github.com/go-chi/chi/v5/middleware"
				test "github.com/klothoplatform/demo-app/pkg/routes"
			)`,
			mount: routerMount{PkgAlias: "test"},
			want:  "routes",
		},
		{
			name: "Block import missing import",
			source: `
			import (
				"fmt"
				"net/http"
			)`,
			mount:   routerMount{PkgAlias: "routes"},
			want:    "",
			wantErr: true,
		},
		{
			name: "Block import with non multi alias",
			source: `
			import (
				"fmt"
				"net/http"
			
				chi "github.com/go-chi/chi/v5"
				middleware "github.com/go-chi/chi/v5/middleware"
				test "github.com/klothoplatform/demo-app/pkg/routes"
			)`,
			mount: routerMount{PkgAlias: "test"},
			want:  "routes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			err = testRestAPIHandler.findChiRouterMountPackage(f, &tt.mount)
			if tt.wantErr {
				assert.Error(err)
				return
			}

			assert.Equal(tt.want, tt.mount.PkgName)
		})
	}
}

func Test_findFilesForFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		sources  map[string]string
		funcName string
		wantErr  bool
	}{
		{
			name: "One file with function",
			sources: map[string]string{
				"file1.go": `func TestRoutes() chi.Router {}`,
			},
			funcName: "TestRoutes",
		},
		{
			name: "One file with multiple functions",
			sources: map[string]string{
				"file1.go": `
				func TestRoutes() chi.Router {}
				func WrongRoutes() chi.Router {}
				func MoreWrongRoutes() chi.Router {}`,
			},
			funcName: "TestRoutes",
		},
		{
			name: "Multiple files",
			sources: map[string]string{
				"file1.go": `func WrongRoutes() chi.Router {}`,
				"file2.go": `func MoreWrongRoutes() chi.Router {}`,
				"file3.go": `func TestRoutes() chi.Router {}`,
			},
			funcName: "TestRoutes",
		},
		{
			name: "Multiple files no matching function",
			sources: map[string]string{
				"file1.go": `func WrongRoutes() chi.Router {}`,
				"file2.go": `func MoreWrongRoutes() chi.Router {}`,
				"file3.go": `func EvenMoreWrongRoutes() chi.Router {}`,
			},
			funcName: "TestRoutes",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			var files = make([]*core.SourceFile, 0)
			for path, src := range tt.sources {
				f, err := core.NewSourceFile(path, strings.NewReader(src), Language)
				if !assert.NoError(err) {
					return
				}
				files = append(files, f)
			}

			file, node := testRestAPIHandler.findFileForFunctionName(files, tt.funcName)
			if tt.wantErr {
				assert.Nil(file)
				assert.Nil(node)
				return
			}
			assert.NotNil(file)
			assert.NotNil(node)
		})
	}
}
