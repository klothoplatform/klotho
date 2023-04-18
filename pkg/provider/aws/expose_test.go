package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
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
		existingResources      []core.Resource
		existingDependencies   []graph.Edge[core.Resource]
		cfg                    config.Application
		want                   coretesting.ResourcesExpectation
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
			want: coretesting.ResourcesExpectation{
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
			want: coretesting.ResourcesExpectation{
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
		{
			name: "multiple routes and eks",
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
					resources.NewLoadBalancer(appName, unit1.ID, nil, "internal", "network", nil, nil),
				},
				"execution_unit:test2": {
					resources.NewLoadBalancer(appName, unit2.ID, nil, "internal", "network", nil, nil),
				},
			},
			existingResources: []core.Resource{
				&resources.EksCluster{
					Name:          "Cluster",
					ConstructsRef: []core.AnnotationKey{unit1.AnnotationKey, unit2.AnnotationKey},
					Subnets:       []*resources.Subnet{resources.NewSubnet("1", resources.NewVpc("test"), "", "", core.IaCValue{})},
				},
				&resources.OpenIdConnectProvider{Name: "test"},
			},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &resources.OpenIdConnectProvider{Name: "test"}, Destination: &resources.EksCluster{
					Name:          "Cluster",
					ConstructsRef: []core.AnnotationKey{unit1.AnnotationKey, unit2.AnnotationKey},
					Subnets:       []*resources.Subnet{resources.NewSubnet("1", resources.NewVpc("test"), "", "", core.IaCValue{})},
				}},
			},
			cfg: config.Application{
				AppName: appName,
				Defaults: config.Defaults{
					ExecutionUnit: config.KindDefaults{Type: kubernetes.KubernetesType},
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_deployment:test-test",
					"aws:api_integration:test-test-GET",
					"aws:api_integration:test-test-test/-GET",
					"aws:api_integration:test-test-test/-POST",
					"aws:api_integration:test-test-test/-id-/-GET",
					"aws:api_method:test-test-GET",
					"aws:api_method:test-test-test/-GET",
					"aws:api_method:test-test-test/-POST",
					"aws:api_method:test-test-test/-id-/-GET",
					"aws:api_resource:test-test-test/",
					"aws:api_resource:test-test-test/-id-/",
					"aws:api_stage:test-test-stage",
					"aws:load_balancer:test-test",
					"aws:load_balancer:test-test2",
					"aws:rest_api:test-test",
					"aws:vpc_link:aws:load_balancer:test-test",
					"aws:vpc_link:aws:load_balancer:test-test2",
					"aws:eks_cluster:Cluster",
					"aws:iam_policy:Cluster-alb-controller",
					"aws:iam_role:Cluster-alb-controller",
					"aws:region:region",
					"aws:vpc:test",
					"kubernetes:helm_chart:Cluster-alb-controller",
					"kubernetes:manifest:Cluster-alb-controller-service-account",
					"aws:iam_oidc_provider:test",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-POST"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_integration:test-test-test/-id-/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-POST"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:api_method:test-test-test/-id-/-GET"},
					{Source: "aws:api_deployment:test-test", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:api_method:test-test-GET"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:load_balancer:test-test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-GET", Destination: "aws:vpc_link:aws:load_balancer:test-test"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:api_method:test-test-test/-GET"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:load_balancer:test-test2"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-test/-GET", Destination: "aws:vpc_link:aws:load_balancer:test-test2"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:api_method:test-test-test/-POST"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:load_balancer:test-test"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-test/-POST", Destination: "aws:vpc_link:aws:load_balancer:test-test"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:api_method:test-test-test/-id-/-GET"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:api_resource:test-test-test/-id-/"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:load_balancer:test-test2"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_integration:test-test-test/-id-/-GET", Destination: "aws:vpc_link:aws:load_balancer:test-test2"},
					{Source: "aws:api_method:test-test-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_method:test-test-test/-GET", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_method:test-test-test/-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_method:test-test-test/-POST", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_method:test-test-test/-POST", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_method:test-test-test/-id-/-GET", Destination: "aws:api_resource:test-test-test/-id-/"},
					{Source: "aws:api_method:test-test-test/-id-/-GET", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_resource:test-test-test/", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_resource:test-test-test/-id-/", Destination: "aws:api_resource:test-test-test/"},
					{Source: "aws:api_resource:test-test-test/-id-/", Destination: "aws:rest_api:test-test"},
					{Source: "aws:api_stage:test-test-stage", Destination: "aws:api_deployment:test-test"},
					{Source: "aws:api_stage:test-test-stage", Destination: "aws:rest_api:test-test"},
					{Source: "aws:vpc_link:aws:load_balancer:test-test", Destination: "aws:load_balancer:test-test"},
					{Source: "aws:vpc_link:aws:load_balancer:test-test2", Destination: "aws:load_balancer:test-test2"},
					{Source: "aws:iam_oidc_provider:test", Destination: "aws:eks_cluster:Cluster"},
					{Source: "aws:iam_role:Cluster-alb-controller", Destination: "aws:iam_oidc_provider:test"},
					{Source: "aws:iam_role:Cluster-alb-controller", Destination: "aws:iam_policy:Cluster-alb-controller"},
					{Source: "kubernetes:helm_chart:Cluster-alb-controller", Destination: "aws:eks_cluster:Cluster"},
					{Source: "kubernetes:helm_chart:Cluster-alb-controller", Destination: "aws:region:region"},
					{Source: "kubernetes:helm_chart:Cluster-alb-controller", Destination: "aws:vpc:test"},
					{Source: "kubernetes:manifest:Cluster-alb-controller-service-account", Destination: "aws:eks_cluster:Cluster"},
					{Source: "kubernetes:manifest:Cluster-alb-controller-service-account", Destination: "aws:iam_role:Cluster-alb-controller"},
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
			for _, res := range tt.existingResources {
				dag.AddResource(res)
			}
			for _, dep := range tt.existingDependencies {
				dag.AddDependency(dep.Source, dep.Destination)
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
			resources, found := aws.GetResourcesDirectlyTiedToConstruct(tt.gw)
			assert.True(found)
			assert.Len(resources, 2)
		})

	}
}

func Test_ConvertPath(t *testing.T) {
	cases := []struct {
		given          string
		wantIfGreedy   string
		wantIfNoGreedy string
	}{
		{
			given:          `foo/bar`,
			wantIfGreedy:   `foo/bar`,
			wantIfNoGreedy: `foo/bar`,
		},
		{
			given:          `foo/:bar`,
			wantIfGreedy:   `foo/{bar}`,
			wantIfNoGreedy: `foo/{bar}`,
		},
		{
			given:          `foo/:bar*`,
			wantIfGreedy:   `foo/{bar+}`,
			wantIfNoGreedy: `foo/{bar}`,
		},
		{
			given:          `foo/bar*`,
			wantIfGreedy:   `foo/bar*`,
			wantIfNoGreedy: `foo/bar*`,
		},
		{
			given:          `foo//bar`,
			wantIfGreedy:   `foo/bar`,
			wantIfNoGreedy: `foo/bar`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.given, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tt.wantIfGreedy, convertPath(tt.given, true), "greedy")
			assert.Equal(tt.wantIfNoGreedy, convertPath(tt.given, false), "not greedy")
		})
	}
}
