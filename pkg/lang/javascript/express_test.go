package javascript

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func Test_expressHandler_findAllRouterMWs(t *testing.T) {
	type mw struct {
		UseExpr      string
		ObjectName   string
		PropertyName string
		Path         string
	}
	tests := []struct {
		name    string
		source  string
		varName string
		expect  []mw
	}{
		{
			name:    "no middleware",
			source:  "const a = 1",
			varName: "b",
			expect:  nil,
		},
		{
			name:    "use middleware exports=",
			source:  `app.use(router);`,
			varName: "app",
			expect: []mw{
				{"app.use(router)", "router", "", ""},
			},
		},
		{
			name:    "use middleware export",
			source:  `app.use(get_users_1.userGet);`,
			varName: "app",
			expect: []mw{
				{"app.use(get_users_1.userGet)", "get_users_1", "userGet", ""},
			},
		},
		{
			name: "use multiple",
			source: `app.use(get_users_1.userGet);
app.use(router);`,
			varName: "app",
			expect: []mw{
				{"app.use(get_users_1.userGet)", "get_users_1", "userGet", ""},
				{"app.use(router)", "router", "", ""},
			},
		},
		{
			name:    "use with path",
			source:  `app.use("/v1", router);`,
			varName: "app",
			expect: []mw{
				{`app.use("/v1", router)`, "router", "", "/v1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var testExpressHandler = &ExpressHandler{
				log: zap.L(),
			}
			testExpressHandler.queryResources(f)
			got := testExpressHandler.findAllRouterMWs(
				tt.varName,
				f.Path())

			if !assert.Equal(len(tt.expect), len(got)) {
				return
			}
			for i, expected := range tt.expect {
				got := got[i]
				gotMw := mw{
					UseExpr:      got.UseExpr.Content(f.Program()),
					ObjectName:   got.ObjectName,
					PropertyName: got.PropertyName,
					Path:         got.Path,
				}
				assert.Equal(expected, gotMw, "at index %d", i)
			}
		})
	}
}

func Test_expressHandler_findVerbFuncs(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		varName string
		expect  []routeMethodPath
	}{
		{
			name:    "simple get",
			source:  `app.get("/")`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "get", Path: "/"},
			},
		},
		{
			name:    "single quote",
			source:  `app.get('/')`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "get", Path: "/"},
			},
		},
		{
			name:    "all changed to any",
			source:  `app.all('/')`,
			varName: "app",
			expect: []routeMethodPath{
				{Verb: "any", Path: "/"},
			},
		},
		{
			name:    "non-verb func",
			source:  `app.use(middleware)`,
			varName: "app",
			expect:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var testExpressHandler = &ExpressHandler{
				log: zap.L(),
			}
			testExpressHandler.queryResources(f)
			got := testExpressHandler.findVerbFuncs(tt.varName)
			for i := 0; i < len(tt.expect); i++ {
				if assert.NotNil(got[i]) {
					assert.Equal(f.Path(), got[i].f.Path())
					assert.Equal(tt.expect[i].Path, got[i].Path)
					assert.Equal(tt.expect[i].Verb, got[i].Verb)
				}
			}
		})
	}
}

func Test_expressHandler_handleLocalRoutes(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		varName     string
		routePrefix string
		expect      []gatewayRouteDefinition
	}{
		{
			name:        "simple get",
			source:      `app.get("/test", () => {});`,
			varName:     "app",
			routePrefix: "",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/test", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name: "multiple route verbs",
			source: `app.get("/test", () => {});
			app.post("/test", () => {});`,
			varName:     "app",
			routePrefix: "",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/test", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
				{
					Route:         core.Route{Path: "/test", Verb: "post", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name: "multiple route paths",
			source: `app.get("/test", () => {});
			app.get("/other", () => {});`,
			varName:     "app",
			routePrefix: "",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/test", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
				{
					Route:         core.Route{Path: "/other", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name: "prefix",
			source: `app.get("/test", () => {});
			app.get("/other", () => {});`,
			varName:     "app",
			routePrefix: "/v1",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/v1/test", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
				{
					Route:         core.Route{Path: "/v1/other", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name:        "prefix",
			source:      `app.use("/v1", router)`,
			varName:     "router",
			routePrefix: "/v1",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/v1", Verb: "ANY", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
				{
					Route:         core.Route{Path: "/v1/:rest*", Verb: "ANY", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name:    "route '*' is translated to '/' and '/:rest*'  '{prefix}' and '{prefix}/:rest*'",
			source:  `app.get("*")`,
			varName: "app",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
				{
					Route:         core.Route{Path: "/:rest*", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name:    "route '/*' is translated to '/' and '/:rest*'",
			source:  `app.get("/*")`,
			varName: "app",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
				{
					Route:         core.Route{Path: "/:rest*", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name:    "route '/prefix/*' is translated to '/prefix/:rest*'",
			source:  `app.get("/prefix/*")`,
			varName: "app",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/prefix/:rest*", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name:    "wildcards not processed in middle of segment",
			source:  `app.get("/prefix/br*oken")`,
			varName: "app",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/prefix/br*oken", Verb: "get", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
		{
			name:    "catch-all route",
			source:  `app.all("*")`,
			varName: "app",
			expect: []gatewayRouteDefinition{
				{
					Route:         core.Route{Path: "/", Verb: "any", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
				{
					Route:         core.Route{Path: "/:rest*", Verb: "any", ExecUnitName: "testUnit", HandledInFile: "test.js"},
					DefinedInPath: "test.js",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var testExpressHandler = &ExpressHandler{
				log: zap.L(),
			}
			testExpressHandler.queryResources(f)
			got := testExpressHandler.handleLocalRoutes(f, tt.varName, tt.routePrefix, "testUnit")
			assert.Equal(tt.expect, got)
		})
	}
}
