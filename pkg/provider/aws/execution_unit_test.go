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
	"github.com/stretchr/testify/assert"
)

func Test_GenerateExecUnitResources(t *testing.T) {
	unit := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	repo := ecr.NewEcrRepository("test", core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability})
	image := ecr.NewEcrImage(&core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}, "test", repo)
	role := iam.NewIamRole("test", "test-ExecutionRole", core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}, iam.LAMBDA_ASSUMER_ROLE_POLICY)
	lambda := lambda.NewLambdaFunction(unit, "test", role)
	logGroup := cloudwatch.NewLogGroup("test", fmt.Sprintf("/aws/lambda/%s", lambda.Name), unit.Provenance(), 5)

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
		{
			name: "ecr repo already exists",
			cfg: config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: "lambda"},
				},
			},
			existingResources: []core.Resource{ecr.NewEcrRepository("test", core.AnnotationKey{ID: "test2", Capability: annotation.ExecutionUnitCapability})},
			want: testResult{
				nodes: []core.Resource{
					repo, image, role, lambda,
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
			}
			dag := core.NewResourceGraph()

			for _, res := range tt.existingResources {
				dag.AddResource(res)
			}

			err := aws.GenerateExecUnitResources(unit, dag)
			if tt.want.err {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			for _, node := range tt.want.nodes {
				found := false
				for _, res := range dag.ListConstructs() {
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

			for _, res := range dag.ListConstructs() {
				if repo, ok := res.(*ecr.EcrRepository); ok {
					if len(tt.existingResources) != 0 {
						assert.Len(repo.ConstructsRef, 2)
					} else {
						assert.Len(repo.ConstructsRef, 1)
					}
				}
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
