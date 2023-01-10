package python

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

func Test_findFastApiApp(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectAppVar   string
		expectErr      bool
		expectRootPath string
	}{
		{
			name: "simple fastapi",
			source: `
				# @klotho::expose {
				#   target = "public"
				# }
				app = FastAPI()`,
			expectAppVar:   "app",
			expectRootPath: "",
		},
		{
			name: "fastapi with root_path",
			source: `
				# @klotho::expose {
				#   target = "public"
				# }
				app = FastAPI(root_path="/root-path")`,
			expectAppVar:   "app",
			expectRootPath: "/root-path",
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
			app, _ := testRestAPIHandler.findFastAPIAppDefinition(annot, f)
			if !assert.NotNil(app.Expression, "error in test source app definition function") {
				return
			}

			assert.Equal(tt.expectAppVar, app.Identifier.Content(f.Program()))
			assert.Equal(tt.expectRootPath, app.RootPath)
		})
	}
}

func Test_fastapiHandler_handleLocalRoutes(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		varName     string
		routePrefix string
		expect      []gatewayRouteDefinition
	}{
		{
			name:        "simple get",
			source:      `@app.get("/")`,
			varName:     "app",
			routePrefix: "",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.py"},
					DefinedInPath: "test.py",
				},
			},
		},
		{
			name:    "keyword path",
			source:  `@app.get(other="val",  path="/path")`,
			varName: "app",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/path", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.py"},
					DefinedInPath: "test.py",
				},
			},
		},
		{
			name:        "root path prefix",
			source:      `@app.get(other="val",  path="/path")`,
			varName:     "app",
			routePrefix: "root-path",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "root-path/path", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.py"},
					DefinedInPath: "test.py",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := core.NewSourceFile("test.py", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			got, _ := testRestAPIHandler.findFastAPIRoutesForVar(f, tt.varName, tt.routePrefix)
			assert.Equal(tt.expect, got)
		})
	}
}

func Test_fastapiHandler_findVerbFuncs(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		varName string
		expect  []routeMethodPath
	}{
		{
			name:    "path arg",
			source:  `@app.get("/path")`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "get", Path: "/path"},
			},
		},
		{
			name: "path keyword arg",
			source: `
				@app.get(other_arg=x, path='/path1')
				@app.get(path='/path2')
				@app.get(other='/path3')
			`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "get", Path: "/path1"},
				{Verb: "get", Path: "/path2"},
			},
		},
		{
			name:    "path arg and other keyword arg",
			source:  `@app.get('/path', other='something else')`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "get", Path: "/path"},
			},
		},
		{
			// technically, this is a python runtime error
			name:    "path arg and path keyword arg",
			source:  `@app.get('/path', path='/other')`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "get", Path: "/path"},
				{Verb: "get", Path: "/other"},
			},
		},
		{
			name: "no path",
			source: `
				@app.get()
				@app.get(other="value")
			`,
			varName: "app",
		},
		{
			name: "all verbs",
			source: `
				@app.get('/path')
				@app.post('/path')
				@app.put('/path')
				@app.patch('/path')
				@app.delete('/path')
				@app.options('/path')
				@app.head('/path')
			`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "get", Path: "/path"},
				{Verb: "post", Path: "/path"},
				{Verb: "put", Path: "/path"},
				{Verb: "patch", Path: "/path"},
				{Verb: "delete", Path: "/path"},
				{Verb: "options", Path: "/path"},
				{Verb: "head", Path: "/path"},
			},
		},
		{
			name:    "non-verb func",
			source:  `@app.otherFunc()`,
			varName: "app",
		},
		{
			name:    "non-decorator get",
			source:  `app.get("/")`,
			varName: "app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			got, _ := testRestAPIHandler.findVerbFuncs(f.Tree().RootNode(), f.Program(), tt.varName)
			assert.Equal(tt.expect, got)
			assert.Equal(len(tt.expect), len(got))
		})
	}
}

func Test_sanitizeFastapiPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "no params",
			path: "/simple/path",
			want: "/simple/path",
		},
		{
			name: "one param",
			path: "/simple/{path}",
			want: "/simple/:path",
		},
		{
			name: "multiple params",
			path: "/{simple}/{path}",
			want: "/:simple/:path",
		},
		{
			name: "path param",
			path: "/my/{route:path}",
			want: "/my/:route*",
		},
		{
			name: "ignores invalid params",
			path: "/{my}/{in-valid}/{route:pathlike}",
			want: "/:my/{in-valid}/{route:pathlike}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeFastapiPath(tt.path))
		})
	}
}
