package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_CreateEksCluster(t *testing.T) {
	appName := "test-app"
	clusterName := "test-cluster"
	eus := []*core.ExecutionUnit{{AnnotationKey: core.AnnotationKey{ID: "test"}}}
	sources := []core.AnnotationKey{{ID: "test"}}
	vpc := NewVpc(appName)
	subnet := NewSubnet("test-subnet", vpc, "", PrivateSubnet, core.IaCValue{})
	region := NewRegion()
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
					"aws:eks_node_group:test-app-test-cluster",
					"aws:iam_role:test-app-test-cluster-FargateExecutionRole",
					"aws:iam_role:test-app-test-cluster-k8sAdmin",
					"aws:iam_role:test-app-test-cluster-NodeGroupRole",
					"kubernetes:kubeconfig:test-app-test-cluster-eks-kubeconfig",
					subnet.Id(),
					vpc.Id(),
					region.Id(),
				},
				Deps: []coretesting.StringDep{
					{Source: vpc.Id(), Destination: region.Id()},
					{Source: subnet.Id(), Destination: vpc.Id()},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-k8sAdmin"},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: subnet.Id()},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-FargateExecutionRole"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: subnet.Id()},
					{Source: "aws:eks_node_group:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_node_group:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-NodeGroupRole"},
					{Source: "aws:eks_node_group:test-app-test-cluster", Destination: subnet.Id()},
					{Source: "kubernetes:kubeconfig:test-app-test-cluster-eks-kubeconfig", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:kubeconfig:test-app-test-cluster-eks-kubeconfig", Destination: region.Id()},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			dag.AddResource(region)
			dag.AddResource(vpc)
			dag.AddResource(subnet)
			dag.AddDependency(subnet, vpc)
			dag.AddDependency(vpc, region)

			assert.NoError(CreateEksCluster(appName, clusterName, []*Subnet{subnet}, nil, eus, dag))
			for _, r := range dag.ListResources() {
				if r != subnet && r != vpc && r != region { // ignore input resources
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
