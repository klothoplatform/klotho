package golang

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_querySecrets(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    *persistResult
		wantErr bool
	}{
		{
			name: "simple runtime var",
			source: `
import (
	"gocloud.dev/runtimevar"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
v, err := runtimevar.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))`,
			want: &persistResult{
				varName: "v",
			},
		},
		{
			name: "simple var runtime var",
			source: `
import (
	"gocloud.dev/runtimevar"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
var v, err = runtimevar.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))`,
			want: &persistResult{
				varName: "v",
			},
		},
		{
			name: "simple var declaration",
			source: `
import (
	"gocloud.dev/runtimevar"
)
var v *runtimevar.Variable
var err error
/**
* @klotho::persist {
*	id = "test"
* }
*/
v, err = runtimevar.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))`,
			want: &persistResult{
				varName: "v",
			},
		},
		{
			name: "aliased file blob",
			source: `
import (
	alias "gocloud.dev/runtimevar"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
v, err := alias.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))`,
			want: &persistResult{
				varName: "v",
			},
		},
		{
			name: "wrong import no match",
			source: `
import (
	"gocloud.dev/runtimevarrrrr"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
v, err := runtimevarrrrr.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := types.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			annot, ok := f.Annotations()[types.AnnotationKey{Capability: "persist", ID: "test"}]

			if !assert.True(ok) {
				return
			}
			result := querySecret(f, annot)
			if tt.want == nil {
				assert.Nil(result)
				return
			}
			assert.Equal(tt.want.varName, result.varName)
		})
	}
}

func Test_TransformSecrets(t *testing.T) {
	type testResult struct {
		resource types.Config
		content  string
	}
	tests := []struct {
		name    string
		source  string
		want    testResult
		wantErr bool
	}{
		{
			name: "simple open var",
			source: `package fs
import (
	"gocloud.dev/runtimevar"
)
/**
* @klotho::config {
*	id = "test"
*   secret = true
* }
*/
v, err := runtimevar.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))
`,
			want: testResult{
				resource: types.Config{Name: "test"},
				content: `package fs

import (
	_ "gocloud.dev/runtimevar/awssecretsmanager"
	"gocloud.dev/runtimevar"
)

/**
* @klotho::config {
*	id = "test"
*   secret = true
* }
*/
klothoRuntimePathSub := fmt.Sprintf("file://%s?decoder=string", path)
klothoRuntimePathSubChunks := strings.SplitN(klothoRuntimePathSub, "?", 2)
var queryParams string
	if len(klothoRuntimePathSubChunks) == 2 {
		queryParams = "&" + klothoRuntimePathSubChunks[1]
	}
	v, err := runtimevar.OpenVariable(context.TODO(), "awssecretsmanager://" + os.Getenv("TEST_CONFIG_SECRET") + "?region=" + os.Getenv("AWS_REGION") + queryParams)
`,
			},
		},
		{
			name: "long var open var",
			source: `package fs
import (
	"gocloud.dev/runtimevar"
)
/**
* @klotho::config {
*	id = "test"
*   secret = true
* }
*/
var v, err = runtimevar.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))
`,
			want: testResult{
				resource: types.Config{Name: "test"},
				content: `package fs

import (
	_ "gocloud.dev/runtimevar/awssecretsmanager"
	"gocloud.dev/runtimevar"
)

/**
* @klotho::config {
*	id = "test"
*   secret = true
* }
*/
klothoRuntimePathSub := fmt.Sprintf("file://%s?decoder=string", path)
klothoRuntimePathSubChunks := strings.SplitN(klothoRuntimePathSub, "?", 2)
var queryParams string
	if len(klothoRuntimePathSubChunks) == 2 {
		queryParams = "&" + klothoRuntimePathSubChunks[1]
	}
	var v, err = runtimevar.OpenVariable(context.TODO(), "awssecretsmanager://" + os.Getenv("TEST_CONFIG_SECRET") + "?region=" + os.Getenv("AWS_REGION") + queryParams)
`,
			},
		},
		{
			name: "var declaration open var",
			source: `package fs
import (
	"gocloud.dev/runtimevar"
)
var v *runtimevar.Variable
var err error
/**
* @klotho::config {
*	id = "test"
*   secret = true
* }
*/
v, err = runtimevar.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))
`,
			want: testResult{
				resource: types.Config{Name: "test"},
				content: `package fs

import (
	_ "gocloud.dev/runtimevar/awssecretsmanager"
	"gocloud.dev/runtimevar"
)

var v *runtimevar.Variable
var err error
/**
* @klotho::config {
*	id = "test"
*   secret = true
* }
*/
klothoRuntimePathSub := fmt.Sprintf("file://%s?decoder=string", path)
klothoRuntimePathSubChunks := strings.SplitN(klothoRuntimePathSub, "?", 2)
var queryParams string
	if len(klothoRuntimePathSubChunks) == 2 {
		queryParams = "&" + klothoRuntimePathSubChunks[1]
	}
	v, err = runtimevar.OpenVariable(context.TODO(), "awssecretsmanager://" + os.Getenv("TEST_CONFIG_SECRET") + "?region=" + os.Getenv("AWS_REGION") + queryParams)
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			cfg := config.Application{AppName: "app"}
			p := PersistSecretsPlugin{runtime: NoopRuntime{}, config: &cfg}
			unit := types.ExecutionUnit{}

			f, err := types.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			annot, ok := f.Annotations()[types.AnnotationKey{Capability: "config", ID: "test"}]

			if !assert.True(ok) {
				return
			}
			queryResult := querySecret(f, annot)
			result, err := p.transformSecret(f, annot, queryResult, &unit)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want.resource.Id(), result.Id())
			assert.Equal(tt.want.content, string(f.Program()))
		})
	}
}
