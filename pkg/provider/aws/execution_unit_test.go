package aws

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/cloudwatch"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/ecr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/iam"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/lambda"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/s3"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateExecUnitResources(t *testing.T) {
	unit := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	repo := ecr.NewEcrRepository("test", core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability})
	image := ecr.NewEcrImage(&core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}, "test", repo)
	role := iam.NewIamRole("test", "test-ExecutionRole", core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}, iam.LAMBDA_ASSUMER_ROLE_POLICY)
	lambda := lambda.NewLambdaFunction(unit, "test", role, image)
	logGroup := cloudwatch.NewLogGroup("test", fmt.Sprintf("/aws/lambda/%s", lambda.Name), unit.Provenance(), 5)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	bucket := s3.NewS3Bucket(fs, "test")

	type testResult struct {
		nodes []core.Resource
		deps  []graph.Edge[core.Resource]
		err   bool
	}
	cases := []struct {
		name              string
		existingResources []core.Resource
		cfg               config.Application
		want              testResult
	}{
		{
			name: "generate lambda",
			cfg: config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: "lambda"},
				},
			},
			existingResources: []core.Resource{bucket},
			want: testResult{
				nodes: []core.Resource{
					repo, image, role, lambda, logGroup,
				},
				deps: []graph.Edge[core.Resource]{
					{
						Source:      repo,
						Destination: image,
					},
					{
						Source:      image,
						Destination: lambda,
					},
					{
						Source:      bucket,
						Destination: role,
					},
					{
						Source:      role,
						Destination: lambda,
					},
					{
						Source:      logGroup,
						Destination: lambda,
					},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config: &tt.cfg,
				ConstructIdToResourceId: map[string]string{
					fs.Id(): bucket.Id(),
				},
				PolicyGenerator: iam.NewPolicyGenerator(),
			}
			dag := core.NewResourceGraph()

			for _, res := range tt.existingResources {
				dag.AddResource(res)
			}
			result := core.NewConstructGraph()
			result.AddConstruct(unit)
			result.AddConstruct(fs)
			result.AddDependency(unit.Id(), fs.Id())

			err := aws.GenerateExecUnitResources(unit, result, dag)
			if tt.want.err {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			for _, node := range tt.want.nodes {
				found := false
				for _, res := range dag.ListResources() {
					if res.Id() == node.Id() {
						found = true
					}
				}
				assert.True(found)
			}

			for _, dep := range tt.want.deps {
				found := false
				for _, res := range dag.ListDependencies() {
					if res.Source.Id() == dep.Source.Id() && res.Destination.Id() == dep.Destination.Id() {
						found = true
					}
				}
				assert.True(found)
			}
		})

	}
}

func Test_convertExecUnitParams(t *testing.T) {
	s3Bucket := s3.NewS3Bucket(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}, "test-app")
	cases := []struct {
		name                    string
		construct               core.Construct
		resources               []core.Resource
		testresource            core.Resource
		wants                   resources.EnvironmentVariables
		constructIdToResourceId map[string]string
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
			constructIdToResourceId: map[string]string{
				":unit":   "aws:lambda_function:",
				":bucket": "aws:s3_bucket:test-app-bucket",
			},
			testresource: &lambda.LambdaFunction{},
			wants: resources.EnvironmentVariables{
				"APP_NAME":           core.IaCValue{Resource: nil, Property: "test"},
				"EXECUNIT_NAME":      core.IaCValue{Resource: nil, Property: "unit"},
				"BUCKET_BUCKET_NAME": core.IaCValue{Resource: s3Bucket, Property: "bucket_name"},
			},
		},
		{
			name: `lambda`,
			construct: &core.ExecutionUnit{
				AnnotationKey: core.AnnotationKey{ID: "unit"},
				EnvironmentVariables: core.EnvironmentVariables{
					core.NewEnvironmentVariable("TestVar", nil, "TestValue"),
				},
			},
			constructIdToResourceId: map[string]string{
				":unit": "aws:lambda_function:",
			},
			testresource: &lambda.LambdaFunction{},
			wants: resources.EnvironmentVariables{
				"APP_NAME":      core.IaCValue{Resource: nil, Property: "test"},
				"EXECUNIT_NAME": core.IaCValue{Resource: nil, Property: "unit"},
				"TestVar":       core.IaCValue{Resource: nil, Property: "TestValue"},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config:                  &config.Application{AppName: "test"},
				ConstructIdToResourceId: tt.constructIdToResourceId,
			}

			result := core.NewConstructGraph()
			result.AddConstruct(tt.construct)

			dag := core.NewResourceGraph()
			dag.AddResource(tt.testresource)
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
			switch res := tt.testresource.(type) {
			case *lambda.LambdaFunction:
				assert.Equal(tt.wants, res.EnvironmentVariables)

			}
		})

	}
}

func Test_GetAssumeRolePolicyForType(t *testing.T) {
	cases := []struct {
		name  string
		cfg   config.ExecutionUnit
		wants string
	}{
		{
			name:  `lambda`,
			cfg:   config.ExecutionUnit{Type: Lambda},
			wants: "{\n\tVersion: '2012-10-17',\n\tStatement: [\n\t\t{\n\t\t\tAction: 'sts:AssumeRole',\n\t\t\tPrincipal: {\n\t\t\t\tService: 'lambda.amazonaws.com',\n\t\t\t},\n\t\t\tEffect: 'Allow',\n\t\t\tSid: '',\n\t\t},\n\t],\n}",
		},
		{
			name:  `ecs`,
			cfg:   config.ExecutionUnit{Type: Ecs},
			wants: "{\n\tVersion: '2012-10-17',\n\tStatement: [\n\t\t{\n\t\t\tAction: 'sts:AssumeRole',\n\t\t\tPrincipal: {\n\t\t\t\tService: 'ecs-tasks.amazonaws.com',\n\t\t\t},\n\t\t\tEffect: 'Allow',\n\t\t\tSid: '',\n\t\t},\n\t],\n}",
		},
		{
			name:  `eks fargate`,
			cfg:   config.ExecutionUnit{Type: Eks, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{NodeType: string(resources.Fargate)})},
			wants: "{\n\tVersion: '2012-10-17',\n\tStatement: [\n\t\t{\n\t\t\tAction: 'sts:AssumeRole',\n\t\t\tPrincipal: {\n\t\t\t\tService: 'eks-fargate-pods.amazonaws.com',\n\t\t\t},\n\t\t\tEffect: 'Allow',\n\t\t\tSid: '',\n\t\t},\n\t],\n}",
		},
		{
			name:  `eks node`,
			cfg:   config.ExecutionUnit{Type: Eks, InfraParams: config.ConvertToInfraParams(config.KubernetesTypeParams{NodeType: string(resources.Node)})},
			wants: "{\n\tVersion: '2012-10-17',\n\tStatement: [\n\t\t{\n\t\t\tAction: 'sts:AssumeRole',\n\t\t\tPrincipal: {\n\t\t\t\tService: 'ec2.amazonaws.com',\n\t\t\t},\n\t\t\tEffect: 'Allow',\n\t\t\tSid: '',\n\t\t},\n\t],\n}",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := GetAssumeRolePolicyForType(tt.cfg)
			assert.Equal(tt.wants, actual)
		})

	}
}
