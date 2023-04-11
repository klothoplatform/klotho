package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_CreateEksCluster(t *testing.T) {
	cfg := config.Application{AppName: "test-app", ExecutionUnits: make(map[string]*config.ExecutionUnit)}
	clusterName := "test-cluster"
	eus := []*core.ExecutionUnit{{AnnotationKey: core.AnnotationKey{ID: "test"}}}
	for _, eu := range eus {
		cfg.ExecutionUnits[eu.ID] = &config.ExecutionUnit{
			Type:             "kubernetes",
			NetworkPlacement: "private",
			InfraParams:      config.InfraParams{"instance_type": "t3.medium"},
		}
	}
	sources := []core.AnnotationKey{{ID: "test"}}
	subnet := NewSubnet("test-subnet", NewVpc(cfg.AppName), "", PrivateSubnet, core.IaCValue{})
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_cluster:test-app-test-cluster",
					"aws:eks_fargate_profile:test-app-test-cluster",
					"aws:eks_node_group:private_t3_medium",
					"aws:iam_role:test-app-test-cluster-FargateExecutionRole",
					"aws:iam_role:test-app-test-cluster-k8sAdmin",
					"aws:iam_role:test-app-test-cluster.private_t3_medium",
					subnet.Id(),
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-k8sAdmin"},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: subnet.Id()},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-FargateExecutionRole"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: subnet.Id()},
					{Source: "aws:eks_node_group:private_t3_medium", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_node_group:private_t3_medium", Destination: "aws:iam_role:test-app-test-cluster.private_t3_medium"},
					{Source: "aws:eks_node_group:private_t3_medium", Destination: subnet.Id()},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			CreateEksCluster(&cfg, clusterName, []*Subnet{subnet}, nil, eus, dag)
			for _, r := range dag.ListResources() {
				if r != subnet { // ignore input resources
					assert.ElementsMatch(sources, r.KlothoConstructRef(), "not matching refs in %s", r.Id())
				}
			}

			tt.want.Assert(t, dag)
		})
	}
}

func Test_createClusterAdminRole(t *testing.T) {
	appName := "test-app"
	eus := []core.AnnotationKey{{ID: "test"}}
	assert := assert.New(t)
	role := createClusterAdminRole(appName, "test", eus)
	assert.ElementsMatch(role.AwsManagedPolicies, []string{"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"})
}

func Test_createPodExecutionRole(t *testing.T) {
	appName := "test-app"
	eus := []core.AnnotationKey{{ID: "test"}}
	assert := assert.New(t)
	role := createPodExecutionRole(appName, "test", eus)
	assert.ElementsMatch(role.InlinePolicy.Statement, []StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"logs:CreateLogStream",
				"logs:CreateLogGroup",
				"logs:DescribeLogStreams",
				"logs:PutLogEvents",
			},
			Resource: []core.IaCValue{{Property: "*"}},
		},
	})
}

func Test_createNodeRole(t *testing.T) {
	appName := "test-app"
	eus := []core.AnnotationKey{{ID: "test"}}
	assert := assert.New(t)
	role := createNodeRole(appName, "test", eus)
	assert.ElementsMatch(role.AwsManagedPolicies, []string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		"arn:aws:iam::aws:policy/AWSCloudMapFullAccess",
		"arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy",
	})
}

func TestNodeGroupNameFromConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ExecutionUnit
		want string
	}{
		{
			name: "simple config",
			cfg: config.ExecutionUnit{
				NetworkPlacement: "public",
				InfraParams:      config.InfraParams{"instance_type": "test"},
			},
			want: "public_test",
		},
		{
			name: "translate config",
			cfg: config.ExecutionUnit{
				NetworkPlacement: "private",
				InfraParams:      config.InfraParams{"instance_type": "t3.medium"},
			},
			want: "private_t3_medium",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			assert.Equal(tt.want, NodeGroupNameFromConfig(tt.cfg))
		})
	}
}
