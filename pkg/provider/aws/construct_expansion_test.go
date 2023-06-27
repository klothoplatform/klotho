package aws

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

type testResult struct {
	graph           coretesting.ResourcesExpectation
	mappedResources []reflect.Type
}

func convertResourcesToTypes(resources []core.Resource) []reflect.Type {
	types := []reflect.Type{}
	for _, res := range resources {
		types = append(types, reflect.TypeOf(res))
	}
	return types
}

func Test_ExpandExecutionUnit(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test", DockerfilePath: "path"}
	cases := []struct {
		name          string
		unit          *core.ExecutionUnit
		constructType string
		want          testResult
	}{
		{
			name:          "single lambda exec unit",
			unit:          eu,
			constructType: "lambda_function",
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:lambda_function:my-app-test",
						"aws:log_group:my-app-test",
					},
					Deps: []coretesting.StringDep{
						{Source: "aws:lambda_function:my-app-test", Destination: "aws:log_group:my-app-test"},
					},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.LambdaFunction{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandExecutionUnit(dag, tt.unit, tt.constructType, map[string]any{})

			if !assert.NoError(err) {
				return
			}
			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandStaticUnit(t *testing.T) {
	unit := &core.StaticUnit{Name: "test", IndexDocument: "index.html"}
	cases := []struct {
		name      string
		unit      *core.StaticUnit
		fileNames []string
		want      testResult
	}{
		{
			name:      "single lambda exec unit",
			unit:      unit,
			fileNames: []string{"index.html", "test.html"},
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:s3_bucket:my-app-test",
						"aws:s3_object:my-app-test-index.html",
						"aws:s3_object:my-app-test-test.html",
					},
					Deps: []coretesting.StringDep{
						{Source: "aws:s3_object:my-app-test-index.html", Destination: "aws:s3_bucket:my-app-test"},
						{Source: "aws:s3_object:my-app-test-test.html", Destination: "aws:s3_bucket:my-app-test"},
					},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.S3Bucket{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			for _, fileName := range tt.fileNames {
				unit.AddStaticFile(&core.FileRef{FPath: fileName})
			}

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandStaticUnit(dag, tt.unit)

			if !assert.NoError(err) {
				return
			}
			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandSecrets(t *testing.T) {
	unit := &core.Secrets{Name: "test", Secrets: []string{"secret1", "secret2"}}
	cases := []struct {
		name string
		unit *core.Secrets
		want testResult
	}{
		{
			name: "single lambda exec unit",
			unit: unit,
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:secret:my-app-secret1",
						"aws:secret:my-app-secret2",
						"aws:secret_version:my-app-secret1",
						"aws:secret_version:my-app-secret2",
					},
					Deps: []coretesting.StringDep{
						{Source: "aws:secret_version:my-app-secret1", Destination: "aws:secret:my-app-secret1"},
						{Source: "aws:secret_version:my-app-secret2", Destination: "aws:secret:my-app-secret2"},
					},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.Secret{}), reflect.TypeOf(&resources.Secret{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandSecrets(dag, tt.unit)

			if !assert.NoError(err) {
				return
			}
			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandRedisNode(t *testing.T) {
	unit := &core.RedisNode{Name: "test"}
	cases := []struct {
		name string
		unit *core.RedisNode
		want testResult
	}{
		{
			name: "single redis elasticache",
			unit: unit,
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:elasticache_cluster:my-app-test",
					},
					Deps: []coretesting.StringDep{},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.ElasticacheCluster{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandRedisNode(dag, tt.unit)

			if !assert.NoError(err) {
				return
			}

			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandOrm(t *testing.T) {
	unit := &core.Orm{Name: "test"}
	cases := []struct {
		name          string
		unit          *core.Orm
		constructType string
		want          testResult
	}{
		{
			name:          "single orm",
			unit:          unit,
			constructType: resources.RDS_INSTANCE_TYPE,
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:rds_instance:my-app-test",
					},
					Deps: []coretesting.StringDep{},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandOrm(dag, tt.unit, tt.constructType)

			if !assert.NoError(err) {
				return
			}

			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandKv(t *testing.T) {
	unit := &core.Kv{Name: "test"}
	cases := []struct {
		name          string
		unit          *core.Kv
		constructType string
		want          testResult
	}{
		{
			name:          "single lambda exec unit",
			unit:          unit,
			constructType: resources.RDS_INSTANCE_TYPE,
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:dynamodb_table:my-app-kv",
					},
					Deps: []coretesting.StringDep{},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.DynamodbTable{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandKv(dag, tt.unit)

			if !assert.NoError(err) {
				return
			}
			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandFs(t *testing.T) {
	unit := &core.Fs{Name: "test"}
	cases := []struct {
		name          string
		unit          *core.Fs
		constructType string
		want          testResult
	}{
		{
			name:          "single lambda exec unit",
			unit:          unit,
			constructType: resources.RDS_INSTANCE_TYPE,
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:s3_bucket:my-app-test",
					},
					Deps: []coretesting.StringDep{},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.S3Bucket{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandFs(dag, tt.unit)

			if !assert.NoError(err) {
				return
			}
			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandExpose(t *testing.T) {
	unit := &core.Gateway{Name: "test"}
	cases := []struct {
		name          string
		unit          *core.Gateway
		constructType string
		want          testResult
	}{
		{
			name:          "single lambda exec unit",
			unit:          unit,
			constructType: resources.API_GATEWAY_REST_TYPE,
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:api_deployment:my-app-test",
						"aws:api_stage:my-app-test",
						"aws:rest_api:my-app-test",
					},
					Deps: []coretesting.StringDep{
						{Source: "aws:api_deployment:my-app-test", Destination: "aws:rest_api:my-app-test"},
						{Source: "aws:api_stage:my-app-test", Destination: "aws:api_deployment:my-app-test"},
						{Source: "aws:api_stage:my-app-test", Destination: "aws:rest_api:my-app-test"},
					},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.RestApi{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandExpose(dag, tt.unit, tt.constructType)

			if !assert.NoError(err) {
				return
			}
			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}

func Test_ExpandConfig(t *testing.T) {
	unit := &core.Config{Name: "test", Secret: true}
	cases := []struct {
		name string
		unit *core.Config
		want testResult
	}{
		{
			name: "single lambda exec unit",
			unit: unit,
			want: testResult{
				graph: coretesting.ResourcesExpectation{
					Nodes: []string{
						"aws:secret_version:my-app-test",
					},
					Deps: []coretesting.StringDep{},
				},
				mappedResources: []reflect.Type{reflect.TypeOf(&resources.Secret{})},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				AppName: "my-app",
			}
			mappedRes, err := aws.expandConfig(dag, tt.unit)
			if !assert.NoError(err) {
				return
			}
			tt.want.graph.Assert(t, dag)
			assert.ElementsMatch(tt.want.mappedResources, convertResourcesToTypes(mappedRes))
		})
	}
}
