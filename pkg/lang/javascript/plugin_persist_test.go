package javascript

import (
	"fmt"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func Test_missingAwait(t *testing.T) {
	observingCore, logObserver := observer.New(zap.WarnLevel)
	observingWrapper := func(c zapcore.Core) zapcore.Core {
		// "c" here is the core from zaptest.NewLogger below.
		// If you want to see the log lines in the testing output, replace this with "zapcore.NewTee(c, observingCore)".
		// But, we're going to replace (rather than wrap) the logger, so that we don't get scary-looking WARN messages.
		return observingCore
	}
	zapLogger := zaptest.NewLogger(t, zaptest.WrapOptions(zap.WrapCore(observingWrapper)))
	defer zap.ReplaceGlobals(zapLogger)()

	tests := []struct {
		name      string
		sources   map[string]string
		expectLog string
	}{
		{
			name: "single file missing await",
			sources: map[string]string{
				"test.js": `// @klotho::persist
const p = new Map();
exports.p = p;
const v = p.get('foo');`, // note: using `get` here, and `set` in the imported test below
			},
			expectLog: "warn file{path: test.js} node@{start-row: 3, start-column: 10, end-row: 3, end-column: 22} Call is async, but is missing \"await\"",
		},
		{
			name: "imported persist missing await",
			sources: map[string]string{
				"test1.js": `// @klotho::persist

const p = new Map();
exports.p = p;`,
				"test2.js": `const other = require('./test1');
other.p.set('bar', 'myval');`, // note: using `set` here, and `get` in the single-file test above
			},
			expectLog: "warn file{path: test2.js} node@{start-row: 1, start-column: 0, end-row: 1, end-column: 27} Call is async, but is missing \"await\"",
		},
		{
			name: "imported persist using named import missing await",
			sources: map[string]string{
				"test1.js": `// @klotho::persist { id="kv" }
const p = new Map();
exports.p = p;`,
				"test2.js": `const {p: local} = require('./test1');
local.set('bar', 'myval');`, // note: using `set` here, and `get` in the single-file test above
			},
			expectLog: "warn file{path: test2.js} node@{start-row: 1, start-column: 0, end-row: 1, end-column: 25} Call is async, but is missing \"await\"",
		},
		{
			name: "single file has await",
			sources: map[string]string{
				"test.js": `// @klotho::persist
const p = new Map();
exports.p = p;
const v = await p.get('foo');`,
			},
			expectLog: "",
		},
		{
			name: "don't need await",
			sources: map[string]string{
				"test.js": `// @klotho::persist
const p = new Map();
exports.p = p;
const v = p.uninteresting('foo');`, // don't need an await
			},
			expectLog: "",
		},
		{
			name: "unexported persist not awaited",
			sources: map[string]string{
				"test.js": `//@klotho::persist
const p = new Map();
p.get('foo')`,
			},
			expectLog: "warn file{path: test.js} node@{start-row: 2, start-column: 0, end-row: 2, end-column: 12} Call is async, but is missing \"await\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			var execUnit core.ExecutionUnit
			for filename, content := range tt.sources {
				f, err := NewFile(filename, strings.NewReader(content))
				if !assert.NoError(err) {
					return
				}
				execUnit.Add(f)
			}
			p := persister{
				result: &core.CompilationResult{},
			}
			p.result.Add(&execUnit)

			p.findUnawaitedCalls(&execUnit)
			logSb := strings.Builder{}
			for _, logEntry := range logObserver.TakeAll() {
				klothoFields := logging.DescribeKlothoFields(logEntry.Context, "file", "node")
				fmt.Fprintf(&logSb, "%s file%s node@%s %s\n", logEntry.Level.String(), klothoFields["file"], klothoFields["node"], logEntry.Message)
			}
			if tt.expectLog == "" {
				assert.Empty(logSb.String())
			} else {
				assert.Contains(logSb.String(), tt.expectLog)
			}
		})
	}
}

func Test_queryKV(t *testing.T) {
	defer zap.ReplaceGlobals(zaptest.NewLogger(t))()

	tests := []struct {
		name            string
		source          string
		matchName       string
		matchExpression string
	}{
		{
			name:            "const match",
			source:          "const users = new Map();",
			matchName:       "users",
			matchExpression: "new Map()",
		},
		{
			name:            "const match with export",
			source:          "exports.quoteStore = new Map({\"key\",\"value\"});",
			matchName:       "quoteStore",
			matchExpression: "new Map({\"key\",\"value\"})",
		},
		{
			name:            "let match",
			source:          "let users = new Map();",
			matchName:       "users",
			matchExpression: "new Map()",
		},
		{
			name:            "no let",
			source:          "store.a = new Map()",
			matchName:       "",
			matchExpression: "",
		},
		{
			name:            "const store = new Map(), blah = new Map();",
			source:          "const store = new Map(), blah = new Map();",
			matchName:       "",
			matchExpression: "",
		},
		{
			name:            "non match",
			source:          "a = 1",
			matchName:       "",
			matchExpression: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			cap := &core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability},
				Node:       f.Tree().RootNode(),
			}

			p := persister{}

			kvResult := p.queryKV(f, cap, true)

			if tt.matchExpression != "" || tt.matchName != "" {
				if assert.NotNil(kvResult) {
					assert.Equal(tt.matchExpression, kvResult.expression.Content())
					assert.Equal(tt.matchName, kvResult.name)
				}
			} else {
				assert.Nil(kvResult)
			}
		})
	}
}

func Test_transformKV(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    string
		wantErr bool
	}{
		{
			name: "plain",
			source: `// @klotho::persist
const m = new Map()`,
			want: `// @klotho::persist
const m = new keyvalueRuntime.dMap()`,
		},
		{
			name: "directives",
			source: `/* @klotho::persist {
 *   versioned = true
 * }
 */
const m = new Map()`,
			want: `/* @klotho::persist {
 *   versioned = true
 * }
 */
const m = new keyvalueRuntime.dMap({"versioned":true})`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			var cap *core.Annotation
			for _, v := range f.Annotations() {
				cap = v
				break
			}
			// assuming aws runtime
			p := persister{
				runtime: NoopRuntime{},
			}

			ptype, pres := p.determinePersistType(f, cap)

			if !assert.Equal(core.PersistKVKind, ptype) {
				return
			}

			_, err = p.transformKV(f, cap, pres)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, string(f.Program()))
		})
	}
}

func Test_queryFS(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		matchName       string
		matchExpression string
	}{
		{
			name:            "require fs/promises",
			source:          `const fs = require("fs/promises");`,
			matchExpression: `require("fs/promises")`,
			matchName:       "fs",
		},
		{
			name:            "require fs.promises",
			source:          `const fs = require("fs").promises;`,
			matchExpression: `require("fs")`,
			matchName:       "fs",
		},
		{
			name:            "property promises of fs",
			source:          `const {promises: fs} = require("fs");`,
			matchExpression: `require("fs")`,
			matchName:       "fs",
		},
		{
			name:            "property promises of fs",
			source:          `const {readFile: rf, promises: fs} = require("fs");`,
			matchExpression: `require("fs")`,
			matchName:       "fs",
		},
		{
			name:            "not using fs promise",
			source:          `const fs = require("fs");`,
			matchExpression: "",
		},
		{
			name:            "invalid require package",
			source:          `const fs = require("notfs");`,
			matchExpression: "",
			matchName:       "",
		},
		{
			name:            "not requiring promises",
			source:          `const fs = require("fs").notpromises;`,
			matchExpression: "",
			matchName:       "",
		},
		{
			name:            "not requiring property promises on fs",
			source:          `const {notpromises: fs} = require("fs");`,
			matchExpression: "",
			matchName:       "",
		},
		{
			name:            "typescript generated default require",
			source:          `const fs = __importDefault(require("fs/promises"));`,
			matchExpression: `require("fs/promises")`,
			matchName:       "fs",
		},
		{
			name:            "typescript generated default incorrect require",
			source:          `const fs = __importDefault(require("notfs"));`,
			matchExpression: "",
			matchName:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			p := persister{}

			fsResult := p.queryFS(f, &core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability},
				Node:       f.Tree().RootNode(),
			})
			if tt.matchExpression != "" {
				if !assert.NotNil(fsResult) {
					return
				}
				assert.Equal(tt.matchExpression, fsResult.expression.Content())
				assert.Equal(tt.matchName, fsResult.name)
			} else {
				assert.Nil(fsResult)
			}
		})
	}
}

func Test_queryORM(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		matchName       string
		matchExpression string
	}{
		{
			name:            "sqlite memory no options",
			source:          `const seq = new Sequelize('sqlite::memory:');`,
			matchExpression: `'sqlite::memory:'`,
			matchName:       "seq",
		},
		{
			name:            "postgres no options",
			source:          `const seq = new Sequelize('postgres://user:pass@example.com:5432/dbname');`,
			matchExpression: `'postgres://user:pass@example.com:5432/dbname'`,
			matchName:       "seq",
		},
		{
			name:            "sqlite3 with options",
			source:          `const blah = new Sequelize('sqlite::memory:', {"option1": "value1"});`,
			matchExpression: `'sqlite::memory:'`,
			matchName:       "blah",
		},
		{
			name:            "import sequelize directly",
			source:          `const client = new sequelize.Sequelize('sqlite::memory:')`,
			matchExpression: `'sqlite::memory:'`,
			matchName:       "client",
		},
		{
			name:            "assign directly to export",
			source:          `exports.client = new sequelize.Sequelize('sqlite::memory:')`,
			matchExpression: `'sqlite::memory:'`,
			matchName:       "client",
		},
		{
			name:            "typeorm postgres",
			source:          `const orm = new typeorm.DataSource({type: "postgres"})`,
			matchExpression: `{type: "postgres"}`,
			matchName:       "orm",
		},
		{
			name:            "typeorm export assignment postgres",
			source:          `exports.orm = new typeorm.DataSource({type: "postgres"})`,
			matchExpression: `{type: "postgres"}`,
			matchName:       "orm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			p := persister{}

			fsResult := p.queryORM(f, &core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability},
				Node:       f.Tree().RootNode(),
			}, true)
			if tt.matchExpression != "" {
				assert.NotNil(fsResult)
				assert.Equal(tt.matchExpression, fsResult.expression.Content())
				assert.Equal(tt.matchName, fsResult.name)
			} else {
				assert.Nil(fsResult)
			}
		})
	}
}

func Test_queryRedis(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		matchName       string
		matchExpression string
	}{
		{
			name:            "variable declaration with args",
			source:          `const client = createClient({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "compiled variable declaration with args",
			source:          `const client = (0, redis_1.createClient)({url: 'redis://alice:foobared@awesome.redis.server:6380'});`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "variable declaration no arguments",
			source:          `const client = (0, redis_1.createClient)();`,
			matchExpression: `()`,
			matchName:       "client",
		},
		{
			name:            "redis compiled assignment expression with args",
			source:          `client = (0, redis_1.createClient)({url: 'redis://alice:foobared@awesome.redis.server:6380'});`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "redis assignment expression with args",
			source:          `client = createClient({url: 'redis://alice:foobared@awesome.redis.server:6380'});`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "redis assignment expression no args",
			source:          `client = createClient();`,
			matchExpression: `()`,
			matchName:       "client",
		},
		{
			name:            "redis export assignment expression with args",
			source:          `exports.client = createClient({url: 'redis://alice:foobared@awesome.redis.server:6380'});`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "redis compiled export assignment expression with args",
			source:          `exports.client = (0, redis_1.createClient)({url: 'redis://alice:foobared@awesome.redis.server:6380'});`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "redis export assignment expression no args",
			source:          `exports.client = (0, redis_1.createClient)();`,
			matchExpression: `()`,
			matchName:       "client",
		},
		{
			name:            "cluster variable declaration with args",
			source:          `const client = createCluster({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "cluster compiled variable declaration with args",
			source:          `const client = (0, redis_1.createCluster)({url: 'redis://alice:foobared@awesome.redis.server:6380'});`,
			matchExpression: `({url: 'redis://alice:foobared@awesome.redis.server:6380'})`,
			matchName:       "client",
		},
		{
			name:            "cluster variable declaration no arguments",
			source:          `const client = (0, redis_1.createCluster)();`,
			matchExpression: `()`,
			matchName:       "client",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			p := persister{}

			_, fsResult := p.queryRedis(f, &core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability},
				Node:       f.Tree().RootNode(),
			}, true)
			if tt.matchExpression != "" {
				assert.Equal(tt.matchExpression, fsResult.expression.Content())
				assert.Equal(tt.matchName, fsResult.name)
			} else {
				assert.Nil(fsResult)
			}
		})
	}
}

func Test_transformRedis(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    string
		wantErr bool
	}{
		{
			name: "node",
			source: `
/**
* @klotho::persist {
*   id = "redis"
* }
*/
const client = createClient({ socket: {
	host: process.env.REDIS_HOST,
	port: port,
	keepAlive: 5000
}})`,
			want: `
/**
* @klotho::persist {
*   id = "redis"
* }
*/
const client = createClient(redis_nodeRuntime.getParams("REDIS_PERSIST_REDIS_HOST", "REDIS_PERSIST_REDIS_PORT", { socket: {
	host: process.env.REDIS_HOST,
	port: port,
	keepAlive: 5000
}}))`,
		},
		{
			name: "cluster",
			source: `
/**
* @klotho::persist {
*   id = "redis"
* }
*/
const client = createCluster({
	rootNodes:[
		{
			url: 'redis://127.0.0.1:8001'
		}
	],
})`,
			want: `
/**
* @klotho::persist {
*   id = "redis"
* }
*/
const client = createCluster(redis_clusterRuntime.getParams("REDIS_PERSIST_REDIS_HOST", "REDIS_PERSIST_REDIS_PORT", {
	rootNodes:[
		{
			url: 'redis://127.0.0.1:8001'
		}
	],
}))`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			var cap *core.Annotation
			for _, v := range f.Annotations() {
				cap = v
				break
			}
			// assuming aws runtime
			p := persister{
				runtime: NoopRuntime{},
			}

			pKind, pres := p.determinePersistType(f, cap)
			unit := &core.ExecutionUnit{}
			_, err = p.transformRedis(unit, f, cap, pres, pKind)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, string(f.Program()))
			assert.Len(unit.EnvironmentVariables, 2)
		})
	}
}

func Test_querySecretFileName(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		object       string
		wantErr      bool
		matchSecrets []string
	}{
		{
			name:         "fs readFile with secret file",
			source:       `const f = await fs.readFile("my_secret.key");`,
			object:       "fs",
			matchSecrets: []string{"my_secret.key"},
		},
		{
			name:         "fs readFile with multiple secret files",
			source:       `const f = await fs.readFile("my_secret1.key"); const g = await fs.readFile("my_secret2.key");`,
			object:       "fs",
			matchSecrets: []string{"my_secret1.key", "my_secret2.key"},
		},
		{
			name:         "incorrect object name",
			source:       `const f = await fs.readFile("my_secret1.key"); const g = await fs.readFile("my_secret2.key");`,
			object:       "secretStore",
			matchSecrets: []string{},
		},
		{
			name:         "fs readFile no assign",
			source:       `return await fs.readFile("my_secret.key");`,
			object:       "fs",
			matchSecrets: []string{"my_secret.key"},
		},
		{
			name:         "fs readFile with encoding",
			source:       `return await fs.readFile("my_secret.key", "utf-8");`,
			object:       "fs",
			matchSecrets: []string{"my_secret.key"},
		},
		{
			name:    "fs with invalid method",
			source:  `fs.stat();`,
			object:  "fs",
			wantErr: true,
		},
		{
			name:    "fs readfile with no secret name",
			source:  `fs.readFile();`,
			object:  "fs",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}

			p := persister{}

			secrets, err := p.querySecretName(f, tt.object)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			if len(tt.matchSecrets) > 0 {
				assert.Equal(len(tt.matchSecrets), len(secrets))
				for _, secret := range secrets {
					assert.Contains(tt.matchSecrets, secret)
				}
			} else {
				assert.Empty(secrets)
			}
		})
	}
}

func Test_inferType(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		matchType       string
		matchExpression string
	}{
		{
			name:            "fs readFile with secret file",
			source:          `const f = await fs.readFile("my_secret.key");`,
			matchType:       "",
			matchExpression: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := NewFile("test.js", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			p := persister{}

			fsResult := p.queryFS(f, &core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability},
				Node:       f.Tree().RootNode(),
			})
			if tt.matchExpression != "" {

			} else {
				assert.Nil(fsResult)
			}
		})
	}
}
