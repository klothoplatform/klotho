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

func Test_ExpandExecutionUnit(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}, DockerfilePath: "path"}
	cases := []struct {
		name   string
		unit   *core.ExecutionUnit
		chart  *kubernetes.HelmChart
		config *config.Application
		want   coretesting.ResourcesExpectation
	}{
		{
			name:   "single lambda exec unit",
			unit:   eu,
			config: &config.Application{AppName: "my-app", Defaults: config.Defaults{ExecutionUnit: config.KindDefaults{Type: Lambda}}},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:my-app-test",
					"aws:ecr_repo:my-app",
					"aws:iam_role:my-app-test-ExecutionRole",
					"aws:lambda_function:my-app-test",
					"aws:log_group:my-app-test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:ecr_image:my-app-test"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:iam_role:my-app-test-ExecutionRole"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:log_group:my-app-test"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.chart != nil {
				dag.AddResource(tt.chart)
			}

			aws := AWS{
				Config: tt.config,
			}
			err := aws.expandExecutionUnit(dag, tt.unit)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
			resource := aws.GetResourceTiedToConstruct(tt.unit)
			assert.NotNil(resource)
		})
	}
}

func Test_handleExecUnitProxy(t *testing.T) {
	unit1 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "unit1"}}
	unit2 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "unit2"}}
	chart := &kubernetes.HelmChart{
		Name:          "chart",
		ConstructRefs: []core.AnnotationKey{unit1.AnnotationKey, unit2.AnnotationKey},
		ExecutionUnits: []*kubernetes.HelmExecUnit{
			{Name: unit1.ID},
			{Name: unit2.ID},
		},
	}
	cfg := &config.Application{AppName: "test"}
	cases := []struct {
		name                    string
		constructs              []core.Construct
		dependencies            []graph.Edge[string]
		constructIdToResourceId map[string][]core.Resource
		existingResources       []core.Resource
		config                  config.Application
		want                    coretesting.ResourcesExpectation
		wantErr                 bool
	}{
		{
			name:       `lambda to lambda`,
			constructs: []core.Construct{unit1, unit2},
			dependencies: []graph.Edge[string]{
				{Source: unit1.Id(), Destination: unit2.Id()},
				{Source: unit2.Id(), Destination: unit1.Id()},
			},
			constructIdToResourceId: map[string][]core.Resource{
				":unit1": {resources.NewLambdaFunction(unit1, cfg, &resources.IamRole{Name: "role1"}, &resources.EcrImage{}), &resources.IamRole{Name: "role1"}},
				":unit2": {resources.NewLambdaFunction(unit2, cfg, &resources.IamRole{Name: "role2"}, &resources.EcrImage{}), &resources.IamRole{Name: "role2"}},
			},
			config: config.Application{AppName: "test", Defaults: config.Defaults{ExecutionUnit: config.KindDefaults{Type: Lambda}}},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_policy:test-unit1-invoke",
					"aws:iam_policy:test-unit2-invoke",
					"aws:iam_role:role1",
					"aws:iam_role:role2",
					"aws:lambda_function:test-unit1",
					"aws:lambda_function:test-unit2",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:iam_role:role2", Destination: "aws:iam_policy:test-unit1-invoke"},
					{Source: "aws:iam_policy:test-unit1-invoke", Destination: "aws:lambda_function:test-unit1"},
					{Source: "aws:iam_role:role1", Destination: "aws:iam_policy:test-unit2-invoke"},
					{Source: "aws:iam_policy:test-unit2-invoke", Destination: "aws:lambda_function:test-unit2"},
				},
			},
		},
		{
			name:       `k8s to k8s`,
			constructs: []core.Construct{unit1, unit2},
			dependencies: []graph.Edge[string]{
				{Source: unit1.Id(), Destination: unit2.Id()},
				{Source: unit2.Id(), Destination: unit1.Id()},
			},
			constructIdToResourceId: map[string][]core.Resource{
				":unit1": {chart, &resources.IamRole{Name: "role1"}},
				":unit2": {chart, &resources.IamRole{Name: "role2"}},
			},
			config: config.Application{AppName: "test", Defaults: config.Defaults{ExecutionUnit: config.KindDefaults{Type: kubernetes.KubernetesType}}},
			existingResources: []core.Resource{
				&resources.EksCluster{Name: "cluster", ConstructsRef: []core.AnnotationKey{unit1.AnnotationKey, unit2.AnnotationKey}},
				chart},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:private_dns_namespace:test",
					"aws:vpc:test",
					"aws:eks_cluster:cluster",
					"kubernetes:kustomize_directory:cluster-cloudmap-controller",
					"kubernetes:helm_chart:chart",
					"kubernetes:manifest:cluster-cluster-set",
					"aws:iam_policy:test-test-servicediscovery",
					"aws:iam_role:role1",
					"aws:iam_role:role2",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:private_dns_namespace:test", Destination: "aws:vpc:test"},
					{Source: "aws:eks_cluster:cluster", Destination: "aws:private_dns_namespace:test"},
					{Source: "kubernetes:kustomize_directory:cluster-cloudmap-controller", Destination: "aws:eks_cluster:cluster"},
					{Source: "kubernetes:helm_chart:chart", Destination: "kubernetes:kustomize_directory:cluster-cloudmap-controller"},
					{Source: "kubernetes:manifest:cluster-cluster-set", Destination: "kubernetes:kustomize_directory:cluster-cloudmap-controller"},
					{Source: "aws:iam_role:role1", Destination: "aws:iam_policy:test-test-servicediscovery"},
					{Source: "aws:iam_role:role2", Destination: "aws:iam_policy:test-test-servicediscovery"},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config:                 &tt.config,
				constructIdToResources: tt.constructIdToResourceId,
				PolicyGenerator:        resources.NewPolicyGenerator(),
			}

			result := core.NewConstructGraph()
			for _, construct := range tt.constructs {
				result.AddConstruct(construct)
			}
			for _, dep := range tt.dependencies {
				result.AddDependency(dep.Source, dep.Destination)
			}

			dag := core.NewResourceGraph()
			dag.AddResource(resources.NewVpc("test"))
			for id, res := range tt.constructIdToResourceId {
				for _, r := range res {
					dag.AddResource(r)
					if role, ok := r.(*resources.IamRole); ok {
						_ = aws.PolicyGenerator.AddUnitRole(id, role)
					}
				}
			}
			for _, res := range tt.existingResources {
				dag.AddDependenciesReflect(res)
			}

			err := aws.handleExecUnitProxy(result, dag)
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

func Test_convertExecUnitParams(t *testing.T) {
	s3Bucket := resources.NewS3Bucket(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}, "test-app")
	cases := []struct {
		name                    string
		construct               core.Construct
		resources               []core.Resource
		defaultType             string
		execUnitResource        core.Resource
		wants                   resources.EnvironmentVariables
		constructIdToResourceId map[string][]core.Resource
		wantErr                 bool
	}{
		{
			name: `lambda`,
			construct: &core.ExecutionUnit{
				AnnotationKey: core.AnnotationKey{ID: "unit"},
				EnvironmentVariables: core.EnvironmentVariables{
					core.GenerateBucketEnvVar(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}),
				},
			},
			defaultType: Lambda,
			resources: []core.Resource{
				s3Bucket,
			},
			constructIdToResourceId: map[string][]core.Resource{
				":bucket": {s3Bucket},
			},
			execUnitResource: &resources.LambdaFunction{},
			wants: resources.EnvironmentVariables{
				"BUCKET_BUCKET_NAME": core.IaCValue{Resource: s3Bucket, Property: "bucket_name"},
			},
		},
		{
			name: `lambda with sample key value`,
			construct: &core.ExecutionUnit{
				AnnotationKey: core.AnnotationKey{ID: "unit"},
				EnvironmentVariables: core.EnvironmentVariables{
					core.NewEnvironmentVariable("TestVar", nil, "TestValue"),
				},
			},
			defaultType:             Lambda,
			constructIdToResourceId: make(map[string][]core.Resource),
			execUnitResource:        &resources.LambdaFunction{},
			wants: resources.EnvironmentVariables{
				"TestVar": core.IaCValue{Resource: nil, Property: "TestValue"},
			},
		},
		{
			name: `kubernetes`,
			construct: &core.ExecutionUnit{
				AnnotationKey: core.AnnotationKey{ID: "unit"},
				EnvironmentVariables: core.EnvironmentVariables{
					core.GenerateBucketEnvVar(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}),
					core.NewEnvironmentVariable("TestVar", nil, "TestValue"),
				},
			},
			defaultType: kubernetes.KubernetesType,
			resources: []core.Resource{
				s3Bucket,
			},
			constructIdToResourceId: map[string][]core.Resource{
				":bucket": {s3Bucket},
			},
			execUnitResource: &kubernetes.HelmChart{
				Name:           "chart",
				ExecutionUnits: []*kubernetes.HelmExecUnit{{Name: "unit"}},
				ProviderValues: []kubernetes.HelmChartValue{
					{
						EnvironmentVariable: core.GenerateBucketEnvVar(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}),
						Key:                 "BUCKETBUCKETNAME",
					},
				},
				ConstructRefs: []core.AnnotationKey{{ID: "unit"}},
				Values:        make(map[string]any),
			},
			wants: resources.EnvironmentVariables{
				"BUCKETBUCKETNAME": core.IaCValue{Resource: s3Bucket, Property: "bucket_name"},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config:                 &config.Application{AppName: "test", Defaults: config.Defaults{ExecutionUnit: config.KindDefaults{Type: tt.defaultType}}},
				constructIdToResources: tt.constructIdToResourceId,
			}
			if _, ok := tt.execUnitResource.(*kubernetes.HelmChart); !ok {
				aws.constructIdToResources[":unit"] = []core.Resource{tt.execUnitResource}
			}

			result := core.NewConstructGraph()
			result.AddConstruct(tt.construct)

			dag := core.NewResourceGraph()
			dag.AddResource(tt.execUnitResource)
			for _, res := range tt.resources {
				dag.AddResource(res)
			}

			err := aws.convertExecUnitParams(result, dag)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			switch res := tt.execUnitResource.(type) {
			case *resources.LambdaFunction:
				assert.Equal(tt.wants, res.EnvironmentVariables)
			case *kubernetes.HelmChart:
				wantAsMap := map[string]any{}
				for key, val := range tt.wants {
					wantAsMap[key] = val
				}
				assert.Equal(wantAsMap, res.Values)
			default:
				assert.Failf(`test error`, `unrecognized test resource: %v`, res)
			}
		})

	}
}

func Test_GetAssumeRolePolicyForType(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.ExecutionUnit
		want resources.StatementEntry
	}{
		{
			name: `lambda`,
			cfg:  config.ExecutionUnit{Type: Lambda},
			want: resources.StatementEntry{
				Action: []string{"sts:AssumeRole"},
				Principal: &resources.Principal{
					Service: "lambda.amazonaws.com",
				},
				Effect: "Allow",
			},
		},
		{
			name: `ecs`,
			cfg:  config.ExecutionUnit{Type: Ecs},
			want: resources.StatementEntry{
				Action: []string{"sts:AssumeRole"},
				Principal: &resources.Principal{
					Service: "ecs-tasks.amazonaws.com",
				},
				Effect: "Allow",
			},
		},
		{
			name: `eks fargate`,
			cfg:  config.ExecutionUnit{Type: kubernetes.KubernetesType, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{NodeType: string(resources.Fargate)})},
			want: resources.StatementEntry{
				Action: []string{"sts:AssumeRole"},
				Principal: &resources.Principal{
					Service: "eks-fargate-pods.amazonaws.com",
				},
				Effect: "Allow",
			},
		},
		{
			name: `eks node`,
			cfg:  config.ExecutionUnit{Type: kubernetes.KubernetesType, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{NodeType: string(resources.Node)})},
			want: resources.StatementEntry{
				Action: []string{"sts:AssumeRole"},
				Principal: &resources.Principal{
					Service: "ec2.amazonaws.com",
				},
				Effect: "Allow",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := GetAssumeRolePolicyForType(tt.cfg)

			assert.Equal(tt.want, actual.Statement[0])
		})

	}
}
