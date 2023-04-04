package aws

import (
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
	cases := []struct {
		name                   string
		gw                     *core.Gateway
		units                  []*core.ExecutionUnit
		constructIdToResources map[string][]core.Resource
		cfg                    config.Application
		want                   DagExpectation
		wantErr                bool
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
			want: DagExpectation{
				Nodes: []string{
					"aws:account_id:AccountId",
					"aws:api_deployment:test-test",
					"aws:api_integration:test-test-GET",
					"aws:api_method:test-test-GET",
					"aws:api_stage:test-test-stage",
					"aws:lambda_function:test_test",
					"aws:lambda_permission:test_test_awsapi_methodtest_test_GET",
					"aws:rest_api:test-test",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_method:test-test-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_stage:test-test-stage", Destination: "aws:api_deployment:test-test"},
					{Source: "aws:api_stage:test-test-stage", Destination: "aws:rest_api:test-test"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_GET", Destination: "aws:account_id:AccountId"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_GET", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_GET", Destination: "aws:lambda_function:test_test"},
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
			want: DagExpectation{
				Nodes: []string{
					"aws:account_id:AccountId",
					"aws:api_deployment:test-test",
					"aws:api_integration:test-test-GET",
					"aws:api_integration:test-test-test/-GET",
					"aws:api_integration:test-test-test/-id-/-GET",
					"aws:api_integration:test-test-test/-POST",
					"aws:api_method:test-test-GET",
					"aws:api_method:test-test-test/-GET",
					"aws:api_method:test-test-test/-id-/-GET",
					"aws:api_method:test-test-test/-POST",
					"aws:api_resource:test-test-test/-id-/",
					"aws:api_resource:test-test-test/",
					"aws:api_stage:test-test-stage",
					"aws:lambda_function:test_test",
					"aws:lambda_function:test_test2",
					"aws:lambda_permission:test_test_awsapi_methodtest_test_GET",
					"aws:lambda_permission:test_test_awsapi_methodtest_test_test_POST",
					"aws:lambda_permission:test_test2_awsapi_methodtest_test_test_GET",
					"aws:lambda_permission:test_test2_awsapi_methodtest_test_test_id__GET",
					"aws:rest_api:test-test",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-id-/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-POST"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-id-/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-POST"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:api_method:test-test-test/-GET"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:lambda_function:test_test2"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:api_method:test-test-test/-id-/-GET"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:api_resource:test-test-test/-id-/"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:lambda_function:test_test2"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:api_method:test-test-test/-POST"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_method:test-test-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_method:test-test-test/-GET", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_method:test-test-test/-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_method:test-test-test/-id-/-GET", Destination: "aws:api_resource:test-test-test/-id-/"},
					{Source: "aws:api_method:test-test-test/-POST", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_method:test-test-test/-POST", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_resource:test-test-test/-id-/", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_resource:test-test-test/-id-/", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_resource:test-test-test/", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_stage:test-test-stage", Destination: "aws:api_deployment:test-test"},
					{Source: "aws:api_stage:test-test-stage", Destination: "aws:rest_api:test-test"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_GET", Destination: "aws:account_id:AccountId"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_GET", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_GET", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_test_POST", Destination: "aws:account_id:AccountId"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_test_POST", Destination: "aws:api_method:test-test-test/-POST"},
					{Source: "aws:lambda_permission:test_test_awsapi_methodtest_test_test_POST", Destination: "aws:lambda_function:test_test"},
					{Source: "aws:lambda_permission:test_test2_awsapi_methodtest_test_test_GET", Destination: "aws:account_id:AccountId"},
					{Source: "aws:lambda_permission:test_test2_awsapi_methodtest_test_test_GET", Destination: "aws:api_method:test-test-test/-GET"},
					{Source: "aws:lambda_permission:test_test2_awsapi_methodtest_test_test_GET", Destination: "aws:lambda_function:test_test2"},
					{Source: "aws:lambda_permission:test_test2_awsapi_methodtest_test_test_id__GET", Destination: "aws:account_id:AccountId"},
					{Source: "aws:lambda_permission:test_test2_awsapi_methodtest_test_test_id__GET", Destination: "aws:api_method:test-test-test/-id-/-GET"},
					{Source: "aws:lambda_permission:test_test2_awsapi_methodtest_test_test_id__GET", Destination: "aws:lambda_function:test_test2"},
					{Source: "aws:api_method:test-test-test/-id-/-GET", Destination: "aws:rest_api:test-test"},
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
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)
		})

	}
}
