package envvar

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	"github.com/stretchr/testify/assert"
)

func Test_envVarPlugin(t *testing.T) {
	type testResult struct {
		resource core.CloudResource
		envVars  []core.EnvironmentVariable
	}
	tests := []struct {
		name    string
		source  string
		want    testResult
		wantErr bool
	}{
		{
			name: "simple redis node",
			source: `
/*
* @klotho::persist {
*   id = "myRedisNode"
*   [environment_variables]
*     REDIS_NODE_HOST = "redis_node.host"
*     REDIS_NODE_PORT = "redis_node.port"
* }
*/
const a = 1`,
			want: testResult{
				resource: &core.Persist{Kind: core.PersistRedisNodeKind, Name: "myRedisNode"},
				envVars: []core.EnvironmentVariable{{
					Name:       "REDIS_NODE_HOST",
					Kind:       "persist_redis_node",
					ResourceID: "myRedisNode",
					Value:      "host",
				},
					{
						Name:       "REDIS_NODE_PORT",
						Kind:       "persist_redis_node",
						ResourceID: "myRedisNode",
						Value:      "port",
					},
				},
			},
		},
		{
			name: "simple redis cluster",
			source: `
/*
* @klotho::persist {
*   id = "myRedisCluster"
*   [environment_variables]
*     REDIS_HOST = "redis_cluster.host"
*     REDIS_PORT = "redis_cluster.port"
* }
*/
const a = 1`,
			want: testResult{
				resource: &core.Persist{Kind: core.PersistRedisClusterKind, Name: "myRedisCluster"},
				envVars: []core.EnvironmentVariable{{
					Name:       "REDIS_HOST",
					Kind:       "persist_redis_cluster",
					ResourceID: "myRedisCluster",
					Value:      "host",
				},
					{
						Name:       "REDIS_PORT",
						Kind:       "persist_redis_cluster",
						ResourceID: "myRedisCluster",
						Value:      "port",
					},
				},
			},
		},
		{
			name: "simple orm",
			source: `
/*
* @klotho::persist {
*   id = "myOrm"
*   [environment_variables]
*     ORM_CONNECTION_STRING = "orm.connection_string"
* }
*/
const a = 1`,
			want: testResult{
				resource: &core.Persist{Kind: core.PersistORMKind, Name: "myOrm"},
				envVars: []core.EnvironmentVariable{{
					Name:       "ORM_CONNECTION_STRING",
					Kind:       "persist_orm",
					ResourceID: "myOrm",
					Value:      "connection_string",
				},
				},
			},
		},
		{
			name: "error no id",
			source: `
/*
* @klotho::persist {
*   [environment_variables]
*     REDIS_HOST = "redis_cluster.host"
*     REDIS_PORT = "redis_cluster.port"
* }
*/
const a = 1`,
			wantErr: true,
		},
		{
			name: "error no environment variables",
			source: `
/*
* @klotho::persist {
*   [environment_variables]
* }
*/
const a = 1`,
			wantErr: true,
		},
		{
			name: "error invalid kind",
			source: `
/*
* @klotho::persist {
*   [environment_variables]
*     REDIS_HOST = "invalid.host"
* }
*/
const a = 1`,
			wantErr: true,
		},
		{
			name: "error invalid value",
			source: `
/*
* @klotho::persist {
*   [environment_variables]
*     REDIS_HOST = "redis_node.invalid"
* }
*/
const a = 1`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := javascript.NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			p := EnvVarInjection{}

			unit := &core.ExecutionUnit{Name: "unit"}
			unit.Add(f)
			result := &core.CompilationResult{}
			result.Add(unit)
			deps := &core.Dependencies{}
			err = p.Transform(result, deps)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			resources := result.GetResourcesOfType(tt.want.resource.Key().Kind)
			assert.Len(resources, 1)
			assert.Equal(tt.want.resource, resources[0])

			downstreamDeps := deps.Downstream(unit.Key())
			assert.Len(downstreamDeps, 1)
			assert.Equal(tt.want.resource.Key(), downstreamDeps[0])

			assert.Len(unit.EnvironmentVariables, len(tt.want.envVars))
			for _, envVar := range tt.want.envVars {
				for _, unitVar := range unit.EnvironmentVariables {
					if envVar.Name == unitVar.Name {
						assert.Equal(envVar, unitVar)
					}
				}
			}

		})
	}
}

func Test_parseDirectiveToEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    EnvironmentVariableDirectiveResult
		wantErr bool
	}{
		{
			name: "simple happy path",
			source: `
/*
* @klotho::persist {
*   id = "myRedisNode"
*   [environment_variables]
*     REDIS_NODE_HOST = "redis_node.host"
*     REDIS_NODE_PORT = "redis_node.port"
* }
*/
const a = 1`,
			want: EnvironmentVariableDirectiveResult{
				kind: string(core.PersistRedisNodeKind),
				variables: []core.EnvironmentVariable{{
					Name:       "REDIS_NODE_HOST",
					Kind:       "persist_redis_node",
					ResourceID: "myRedisNode",
					Value:      "host",
				},
					{
						Name:       "REDIS_NODE_PORT",
						Kind:       "persist_redis_node",
						ResourceID: "myRedisNode",
						Value:      "port",
					},
				},
			},
		},
		{
			name: "kind mistmatch",
			source: `
/*
* @klotho::persist {
*   id = "myRedisCluster"
*   [environment_variables]
*     REDIS_HOST = "redis_cluster.host"
*     REDIS_PORT = "redis_node.port"
* }
*/
const a = 1`,
			wantErr: true,
		},
		{
			name: "invalid env value",
			source: `
/*
* @klotho::persist {
*   id = "myRedisCluster"
*   [environment_variables]
*     REDIS_HOST = "redis_cluster.host.thisisnotallowed"
* }
*/
const a = 1`,
			wantErr: true,
		},
		{
			name: "error invalid kind",
			source: `
/*
* @klotho::persist {
*   [environment_variables]
*     REDIS_HOST = "invalid.host"
* }
*/
const a = 1`,
			wantErr: true,
		},
		{
			name: "error invalid value",
			source: `
/*
* @klotho::persist {
*   [environment_variables]
*     REDIS_HOST = "redis_node.invalid"
* }
*/
const a = 1`,
			wantErr: true,
		},
		{
			name: "error invalid value and kind for one env var",
			source: `
/*
* @klotho::persist {
*   [environment_variables]
*     REDIS_HOST = "redis_node.host"
*     INVALID = "invalid.invalid"
* }
*/
const a = 1`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := javascript.NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var annot *core.Annotation
			for _, v := range f.Annotations() {
				annot = v
				break
			}
			cap := annot.Capability
			result, err := ParseDirectiveToEnvVars(cap)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want.kind, result.kind)

			assert.Len(result.variables, len(tt.want.variables))
			for _, envVar := range tt.want.variables {
				for _, unitVar := range result.variables {
					if envVar.Name == unitVar.Name {
						assert.Equal(envVar, unitVar)
					}
				}
			}

		})
	}
}
