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

func Test_GenerateExecUnitResources(t *testing.T) {
	unit := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	bucket := resources.NewS3Bucket(fs, "test")
	policy1 := &resources.IamPolicy{Name: "policy1"}
	policy2 := &resources.IamPolicy{Name: "policy2"}
	cluster := resources.NewEksCluster("test", resources.DEFAULT_CLUSTER_NAME, nil, nil, nil)
	chart := &kubernetes.HelmChart{
		Name:           "chart",
		ConstructRefs:  []core.AnnotationKey{unit.Provenance()},
		ExecutionUnits: []*kubernetes.HelmExecUnit{{Name: unit.ID}},
		ProviderValues: []kubernetes.HelmChartValue{
			{
				ExecUnitName: unit.ID,
				Type:         string(kubernetes.ServiceAccountAnnotationTransformation),
				Key:          "sa",
			},
			{
				ExecUnitName: unit.ID,
				Type:         string(kubernetes.ImageTransformation),
				Key:          "image",
			},
		},
		Values: make(map[string]core.IaCValue),
	}

	cases := []struct {
		name              string
		existingResources []core.Resource
		cfg               config.Application
		want              coretesting.ResourcesExpectation
		wantErr           bool
	}{
		{
			name: "generate lambda",
			cfg: config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: Lambda},
				},
			},
			existingResources: []core.Resource{bucket, policy1, policy2},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:test-test",
					"aws:ecr_repo:test",
					"aws:iam_policy:policy1",
					"aws:iam_policy:policy2",
					"aws:iam_role:test-test-ExecutionRole",
					"aws:lambda_function:test_test",
					"aws:log_group:test_awslambdatest_test",
					"aws:s3_bucket:test-test",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:ecr_image:test-test", Destination: "aws:ecr_repo:test"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy1"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy2"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:s3_bucket:test-test"},
					{Source: "aws:lambda_function:test_test", Destination: "aws:ecr_image:test-test"},
					{Source: "aws:lambda_function:test_test", Destination: "aws:iam_role:test-test-ExecutionRole"},
					{Source: "aws:lambda_function:test_test", Destination: "aws:log_group:test_awslambdatest_test"},
				},
			},
		},

		{
			name: "generate kubernetes",
			cfg: config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: Kubernetes},
				},
			},
			existingResources: []core.Resource{bucket, policy1, policy2, cluster, chart},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:test-test",
					"aws:ecr_repo:test",
					"aws:eks_cluster:test-eks-cluster",
					"aws:eks_provider:UNIMPLEMENTED-eks-provider",
					"aws:iam_policy:policy1",
					"aws:iam_policy:policy2",
					"aws:iam_role:test-test-ExecutionRole",
					"aws:s3_bucket:test-test",
					"kubernetes:helm_chart:chart",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:ecr_image:test-test", Destination: "aws:ecr_repo:test"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy1"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy2"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:s3_bucket:test-test"},
					{Source: "kubernetes:helm_chart:chart", Destination: "aws:ecr_image:test-test"},
					{Source: "kubernetes:helm_chart:chart", Destination: "aws:iam_role:test-test-ExecutionRole"},
					{Source: "kubernetes:helm_chart:chart", Destination: "aws:eks_provider:UNIMPLEMENTED-eks-provider"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config: &tt.cfg,
				constructIdToResources: map[string][]core.Resource{
					fs.Id(): {bucket},
				},
				PolicyGenerator: resources.NewPolicyGenerator(),
			}
			dag := core.NewResourceGraph()

			for _, res := range tt.existingResources {
				dag.AddResource(res)
				if policy, ok := res.(*resources.IamPolicy); ok {
					aws.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), policy)
				}
			}
			result := core.NewConstructGraph()
			result.AddConstruct(unit)
			result.AddConstruct(fs)
			result.AddDependency(unit.Id(), fs.Id())

			err := aws.GenerateExecUnitResources(unit, result, dag)
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
			resources: []core.Resource{
				s3Bucket,
			},
			constructIdToResourceId: map[string][]core.Resource{
				":bucket": {s3Bucket},
			},
			execUnitResource: &resources.LambdaFunction{},
			wants: resources.EnvironmentVariables{
				"APP_NAME":           core.IaCValue{Resource: nil, Property: "test"},
				"EXECUNIT_NAME":      core.IaCValue{Resource: nil, Property: "unit"},
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
			constructIdToResourceId: make(map[string][]core.Resource),
			execUnitResource:        &resources.LambdaFunction{},
			wants: resources.EnvironmentVariables{
				"APP_NAME":      core.IaCValue{Resource: nil, Property: "test"},
				"EXECUNIT_NAME": core.IaCValue{Resource: nil, Property: "unit"},
				"TestVar":       core.IaCValue{Resource: nil, Property: "TestValue"},
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
			resources: []core.Resource{
				s3Bucket,
			},
			constructIdToResourceId: map[string][]core.Resource{
				":bucket": {s3Bucket},
			},
			execUnitResource: &kubernetes.HelmChart{
				Name: "chart",
				ProviderValues: []kubernetes.HelmChartValue{
					{
						EnvironmentVariable: core.GenerateBucketEnvVar(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}),
						Key:                 "BUCKETBUCKETNAME",
					},
				},
				Values: make(map[string]core.IaCValue),
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
				Config:                 &config.Application{AppName: "test"},
				constructIdToResources: tt.constructIdToResourceId,
			}
			aws.constructIdToResources[":unit"] = []core.Resource{tt.execUnitResource}

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
				wantAsMap := map[string]core.IaCValue{}
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
			cfg:  config.ExecutionUnit{Type: Kubernetes, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{NodeType: string(resources.Fargate)})},
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
			cfg:  config.ExecutionUnit{Type: Kubernetes, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{NodeType: string(resources.Node)})},
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
