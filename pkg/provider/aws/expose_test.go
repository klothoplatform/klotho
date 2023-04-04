package aws

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_CreateRestApi(t *testing.T) {
	appName := "test"
	unit1 := &core.ExecutionUnit{
		AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
	}
	unit2 := &core.ExecutionUnit{
		AnnotationKey: core.AnnotationKey{ID: "test2", Capability: annotation.ExecutionUnitCapability},
	}
	type testResult struct {
		resourceIds []string
		deps        []graph.Edge[string]
		err         bool
	}
	cases := []struct {
		name                   string
		gw                     *core.Gateway
		units                  []*core.ExecutionUnit
		constructIdToResources map[string][]core.Resource
		cfg                    config.Application
		want                   testResult
	}{
		{
			name: "simple base route and single lambda",
			gw: &core.Gateway{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability},
				Routes: []core.Route{
					{
						Path:         "/",
						Verb:         "Get",
						ExecUnitName: "test",
					},
				},
			},
			units: []*core.ExecutionUnit{
				unit1,
			},
			constructIdToResources: map[string][]core.Resource{
				"execution_unit:test": {
					resources.NewLambdaFunction(unit1, appName, &resources.IamRole{}, &resources.EcrImage{}),
				},
			},
			cfg: config.Application{
				Defaults: config.Defaults{
					ExecutionUnit: config.KindDefaults{Type: Lambda},
				},
				AppName: appName,
			},
			want: testResult{
				resourceIds: []string{
					"aws:api_deployment:test-test", "aws:api_integration:test-test-GET", "aws:api_method:test-test-GET", "aws:api_stage:test-test-$default", "aws:lambda_function:test_test",
					"aws:lambda_permission:test_test_awsrest_apitest_test", "aws:rest_api:test-test",
				},
				deps: []graph.Edge[string]{
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_stage:test-test-$default", Destination: "aws:api_deployment:test-test"},
					{Source: "aws:lambda_permission:test_test_awsrest_apitest_test", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:lambda_function:test_test"},
				},
			},
		},
		{
			name: "multiple routes and lambda",
			gw: &core.Gateway{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability},
				Routes: []core.Route{
					{
						Path:         "/",
						Verb:         "Get",
						ExecUnitName: "test",
					},
					{
						Path:         "/test",
						Verb:         "POST",
						ExecUnitName: "test",
					},
					{
						Path:         "/test",
						Verb:         "Get",
						ExecUnitName: "test2",
					},
					{
						Path:         "/test/:id",
						Verb:         "Get",
						ExecUnitName: "test2",
					},
				},
			},
			units: []*core.ExecutionUnit{
				unit1, unit2,
			},
			constructIdToResources: map[string][]core.Resource{
				"execution_unit:test": {
					resources.NewLambdaFunction(unit1, appName, &resources.IamRole{}, &resources.EcrImage{}),
				},
				"execution_unit:test2": {
					resources.NewLambdaFunction(unit2, appName, &resources.IamRole{}, &resources.EcrImage{}),
				},
			},
			cfg: config.Application{
				AppName: appName,
				Defaults: config.Defaults{
					ExecutionUnit: config.KindDefaults{Type: Lambda},
				},
			},
			want: testResult{
				resourceIds: []string{
					"aws:api_deployment:test-test", "aws:api_integration:test-test-GET", "aws:api_integration:test-test-test/-POST", "aws:api_integration:test-test-test/-id-/-GET",
					"aws:api_integration:test-test-test/-GET", "aws:api_method:test-test-GET", "aws:api_method:test-test-test/-POST", "aws:api_method:test-test-test/-id-/-GET",
					"aws:api_method:test-test-test/-GET", "aws:api_resource:test-test-test/", "aws:api_resource:test-test-test/-id-/", "aws:api_stage:test-test-$default", "aws:lambda_function:test_test",
					"aws:lambda_function:test_test2", "aws:lambda_permission:test_test2_awsrest_apitest_test", "aws:lambda_permission:test_test_awsrest_apitest_test", "aws:rest_api:test-test",
				},
				deps: []graph.Edge[string]{
					{Source: "aws:api_resource:test-test-test/-id-/", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_resource:test-test-test/-id-/", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-POST"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-id-/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-id-/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-POST"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-GET"},
					{Source: "aws:api_stage:test-test-$default", Destination: "aws:api_deployment:test-test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_method:test-test-test/-GET", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_method:test-test-test/-id-/-GET", Destination: "aws:api_resource:test-test-test/-id-/"},
					{Source: "aws:api_resource:test-test-test/", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:lambda_function:test_test2"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:api_method:test-test-test/-GET"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:api_method:test-test-test/-POST"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:api_method:test-test-test/-id-/-GET"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:lambda_function:test_test2"},
					{Source: "aws:api_method:test-test-test/-POST", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:lambda_permission:test_test_awsrest_apitest_test", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:lambda_permission:test_test2_awsrest_apitest_test", Destination: "aws:lambda_function:test_test2"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config:                 &tt.cfg,
				constructIdToResources: tt.constructIdToResources,
			}
			dag := core.NewResourceGraph()

			for _, resources := range tt.constructIdToResources {
				for _, res := range resources {
					dag.AddResource(res)
				}
			}
			result := core.NewConstructGraph()
			result.AddConstruct(tt.gw)
			for _, unit := range tt.units {
				result.AddConstruct(unit)
				result.AddDependency(unit.Id(), tt.gw.Id())
			}

			err := aws.CreateRestApi(tt.gw, result, dag)
			if tt.want.err {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			for _, node := range tt.want.resourceIds {
				found := false
				for _, res := range dag.ListResources() {
					if res.Id() == node {
						found = true
					}
				}
				assert.True(found, fmt.Sprintf("Did not find resource with id, %s, in graph", node))
			}

			for _, dep := range tt.want.deps {
				found := false
				for _, res := range dag.ListDependencies() {
					if res.Source.Id() == dep.Source && res.Destination.Id() == dep.Destination {
						found = true
					}
				}
				assert.True(found, "did not find resource: %s -> %s", dep.Source, dep.Destination)
			}
			assert.Len(dag.ListDependencies(), len(tt.want.deps))
			assert.Len(dag.ListResources(), len(tt.want.resourceIds))
		})

	}
}
