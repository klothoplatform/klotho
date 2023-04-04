package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_CreateEksCluster(t *testing.T) {
	appName := "test-app"
	clusterName := "test-cluster"
	eus := []*core.ExecutionUnit{{AnnotationKey: core.AnnotationKey{ID: "test"}}}
	sources := []core.AnnotationKey{{ID: "test"}}
	subnet := NewSubnet("test-subnet", NewVpc(appName), "", PrivateSubnet)
	type stringDep struct {
		source string
		dest   string
	}
	type testResult struct {
		nodes []string
		deps  []stringDep
	}
	cases := []struct {
		name string
		want testResult
	}{
		{
			name: "happy path",
			want: testResult{
				nodes: []string{
					"aws:eks_cluster:test-app-test-cluster",
					"aws:eks_fargate_profile:test-app-test-cluster",
					"aws:eks_node_group:test-app-test-cluster",
					"aws:iam_role:test-app-test-cluster-FargateExecutionRole",
					"aws:iam_role:test-app-test-cluster-NodeGroupRole",
					"aws:iam_role:test-app-test-cluster-k8sAdmin",
					subnet.Id(),
				},
				deps: []stringDep{
					{dest: "aws:eks_cluster:test-app-test-cluster", source: "aws:eks_fargate_profile:test-app-test-cluster"},
					{dest: "aws:eks_cluster:test-app-test-cluster", source: "aws:eks_node_group:test-app-test-cluster"},
					{dest: "aws:iam_role:test-app-test-cluster-k8sAdmin", source: "aws:eks_cluster:test-app-test-cluster"},
					{dest: "aws:iam_role:test-app-test-cluster-FargateExecutionRole", source: "aws:eks_fargate_profile:test-app-test-cluster"},
					{dest: "aws:iam_role:test-app-test-cluster-NodeGroupRole", source: "aws:eks_node_group:test-app-test-cluster"},
					{dest: subnet.Id(), source: "aws:eks_node_group:test-app-test-cluster"},
					{dest: subnet.Id(), source: "aws:eks_fargate_profile:test-app-test-cluster"},
					{dest: subnet.Id(), source: "aws:eks_cluster:test-app-test-cluster"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			CreateEksCluster(appName, clusterName, []*Subnet{subnet}, nil, eus, dag)
			var res []string
			for _, r := range dag.ListResources() {
				res = append(res, r.Id())
				if r != subnet { // ignore input resources
					assert.ElementsMatch(sources, r.KlothoConstructRef(), "not matching refs in %s", r.Id())
				}
			}
			assert.ElementsMatch(tt.want.nodes, res)

			var dep []stringDep
			for _, e := range dag.ListDependencies() {
				dep = append(dep, stringDep{source: e.Source.Id(), dest: e.Destination.Id()})
			}

			assert.ElementsMatch(tt.want.deps, dep)
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
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy", "arn:aws:iam::aws:policy/AWSCloudMapFullAccess",
		"arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy",
	})
}
