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
	cluster := resources.NewEksCluster("test", resources.DEFAULT_CLUSTER_NAME, nil, nil, nil, nil)
	oidc := &resources.OpenIdConnectProvider{Name: cluster.Name}
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
			{
				ExecUnitName: unit.ID,
				Type:         string(kubernetes.TargetGroupTransformation),
				Key:          "tgb",
			},
		},
		Values: make(map[string]any),
	}

	cases := []struct {
		name              string
		existingResources []core.Resource
		existingDeps      []graph.Edge[core.Resource]
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
					"aws:lambda_function:test-test",
					"aws:log_group:test-/aws/lambda/test-test",
					"aws:s3_bucket:test-test",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:ecr_image:test-test", Destination: "aws:ecr_repo:test"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy1"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy2"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:s3_bucket:test-test"},
					{Source: "aws:lambda_function:test-test", Destination: "aws:ecr_image:test-test"},
					{Source: "aws:lambda_function:test-test", Destination: "aws:iam_role:test-test-ExecutionRole"},
					{Source: "aws:lambda_function:test-test", Destination: "aws:log_group:test-/aws/lambda/test-test"},
				},
			},
		},

		{
			name: "generate kubernetes",
			cfg: config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: kubernetes.KubernetesType},
				},
			},
			existingResources: []core.Resource{bucket, policy1, policy2, cluster, chart, oidc},
			existingDeps: []graph.Edge[core.Resource]{
				{Source: oidc, Destination: cluster},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:ecr_image:test-test",
					"aws:ecr_repo:test",
					"aws:eks_cluster:test-eks-cluster",
					"aws:elastic_ip:test_public1",
					"aws:elastic_ip:test_public2",
					"aws:iam_oidc_provider:test-eks-cluster",
					"aws:iam_policy:policy1",
					"aws:iam_policy:policy2",
					"aws:iam_role:test-test-ExecutionRole",
					"aws:internet_gateway:test_igw1",
					"aws:load_balancer:test-test",
					"aws:load_balancer_listener:test-test-test",
					"aws:nat_gateway:test_public1",
					"aws:nat_gateway:test_public2",
					"aws:region:region",
					"aws:route_table:test-public",
					"aws:route_table:test_private1",
					"aws:route_table:test_private2",
					"aws:s3_bucket:test-test",
					"aws:target_group:test-test",
					"aws:vpc:test",
					"aws:security_group:test",
					"aws:vpc_endpoint:test_dynamodb",
					"aws:vpc_endpoint:test_lambda",
					"aws:vpc_endpoint:test_s3",
					"aws:vpc_endpoint:test_secretsmanager",
					"aws:vpc_endpoint:test_sns",
					"aws:vpc_endpoint:test_sqs",
					"aws:vpc_subnet:test_private1",
					"aws:vpc_subnet:test_private2",
					"aws:vpc_subnet:test_public1",
					"aws:vpc_subnet:test_public2",
					"kubernetes:helm_chart:chart",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:availability_zones:AvailabilityZones", Destination: "aws:region:region"},
					{Source: "aws:ecr_image:test-test", Destination: "aws:ecr_repo:test"},
					{Source: "aws:iam_oidc_provider:test-eks-cluster", Destination: "aws:eks_cluster:test-eks-cluster"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy1"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_policy:policy2"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:s3_bucket:test-test"},
					{Source: "aws:iam_role:test-test-ExecutionRole", Destination: "aws:iam_oidc_provider:test-eks-cluster"},
					{Source: "aws:internet_gateway:test_igw1", Destination: "aws:vpc:test"},
					{Source: "aws:load_balancer:test-test", Destination: "aws:vpc_subnet:test_private1"},
					{Source: "aws:load_balancer:test-test", Destination: "aws:vpc_subnet:test_private2"},
					{Source: "aws:load_balancer_listener:test-test-test", Destination: "aws:load_balancer:test-test"},
					{Source: "aws:load_balancer_listener:test-test-test", Destination: "aws:target_group:test-test"},
					{Source: "aws:nat_gateway:test_public1", Destination: "aws:elastic_ip:test_public1"},
					{Source: "aws:nat_gateway:test_public1", Destination: "aws:vpc_subnet:test_public1"},
					{Source: "aws:nat_gateway:test_public2", Destination: "aws:elastic_ip:test_public2"},
					{Source: "aws:nat_gateway:test_public2", Destination: "aws:vpc_subnet:test_public2"},
					{Source: "aws:route_table:test-public", Destination: "aws:internet_gateway:test_igw1"},
					{Source: "aws:route_table:test-public", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:test-public", Destination: "aws:vpc_subnet:test_public1"},
					{Source: "aws:route_table:test-public", Destination: "aws:vpc_subnet:test_public2"},
					{Source: "aws:route_table:test_private1", Destination: "aws:nat_gateway:test_public1"},
					{Source: "aws:route_table:test_private1", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:test_private1", Destination: "aws:vpc_subnet:test_private1"},
					{Source: "aws:route_table:test_private2", Destination: "aws:nat_gateway:test_public2"},
					{Source: "aws:route_table:test_private2", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:test_private2", Destination: "aws:vpc_subnet:test_private2"},
					{Source: "aws:target_group:test-test", Destination: "aws:vpc:test"},
					{Source: "aws:vpc:test", Destination: "aws:region:region"},
					{Source: "aws:security_group:test", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_endpoint:test_dynamodb", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_dynamodb", Destination: "aws:route_table:test-public"},
					{Source: "aws:vpc_endpoint:test_dynamodb", Destination: "aws:route_table:test_private1"},
					{Source: "aws:vpc_endpoint:test_dynamodb", Destination: "aws:route_table:test_private2"},
					{Source: "aws:vpc_endpoint:test_dynamodb", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_endpoint:test_lambda", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_lambda", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_endpoint:test_lambda", Destination: "aws:vpc_subnet:test_private1"},
					{Source: "aws:vpc_endpoint:test_lambda", Destination: "aws:vpc_subnet:test_private2"},
					{Source: "aws:vpc_endpoint:test_lambda", Destination: "aws:security_group:test"},
					{Source: "aws:vpc_endpoint:test_s3", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_s3", Destination: "aws:route_table:test-public"},
					{Source: "aws:vpc_endpoint:test_s3", Destination: "aws:route_table:test_private1"},
					{Source: "aws:vpc_endpoint:test_s3", Destination: "aws:route_table:test_private2"},
					{Source: "aws:vpc_endpoint:test_s3", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_endpoint:test_secretsmanager", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_secretsmanager", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_endpoint:test_secretsmanager", Destination: "aws:security_group:test"},
					{Source: "aws:vpc_endpoint:test_secretsmanager", Destination: "aws:vpc_subnet:test_private1"},
					{Source: "aws:vpc_endpoint:test_secretsmanager", Destination: "aws:vpc_subnet:test_private2"},
					{Source: "aws:vpc_endpoint:test_sns", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_sns", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_endpoint:test_sns", Destination: "aws:security_group:test"},
					{Source: "aws:vpc_endpoint:test_sns", Destination: "aws:vpc_subnet:test_private1"},
					{Source: "aws:vpc_endpoint:test_sns", Destination: "aws:vpc_subnet:test_private2"},
					{Source: "aws:vpc_endpoint:test_sqs", Destination: "aws:security_group:test"},
					{Source: "aws:vpc_endpoint:test_sqs", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_sqs", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_endpoint:test_sqs", Destination: "aws:vpc_subnet:test_private1"},
					{Source: "aws:vpc_endpoint:test_sqs", Destination: "aws:vpc_subnet:test_private2"},
					{Source: "aws:vpc_subnet:test_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_private1", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_subnet:test_private2", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_private2", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_subnet:test_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_public1", Destination: "aws:vpc:test"},
					{Source: "aws:vpc_subnet:test_public2", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_public2", Destination: "aws:vpc:test"},
					{Source: "kubernetes:helm_chart:chart", Destination: "aws:ecr_image:test-test"},
					{Source: "kubernetes:helm_chart:chart", Destination: "aws:eks_cluster:test-eks-cluster"},
					{Source: "kubernetes:helm_chart:chart", Destination: "aws:iam_role:test-test-ExecutionRole"},
					{Source: "kubernetes:helm_chart:chart", Destination: "aws:target_group:test-test"},
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
				switch res := res.(type) {
				case *resources.IamPolicy:
					aws.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), res)
				case *resources.EksCluster:
					res.Kubeconfig = &kubernetes.Kubeconfig{
						ConstructsRef: res.KlothoConstructRef(),
						Name:          "test-config",
					}
				}
			}
			for _, dep := range tt.existingDeps {
				dag.AddDependency(dep.Source, dep.Destination)
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
