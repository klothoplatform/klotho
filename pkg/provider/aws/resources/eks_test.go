package resources

import (
	"fmt"
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
	subnet := NewSubnet("test-subnet", NewVpc(appName), "", PrivateSubnet, core.IaCValue{})
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
					"aws:iam_role:test-app-test-cluster-NodeGroupRole",
					"aws:iam_role:test-app-test-cluster-k8sAdmin",
					"aws:region:region",
					"aws:vpc_subnet:test_app_test_subnet",
					"kubernetes:helm_chart:test-cluster-cert-manager",
					"kubernetes:helm_chart:test-cluster-metrics-server",
					"kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns",
					"kubernetes:manifest:test-app-test-cluster-aws-observability-config-map",
					"kubernetes:manifest:test-app-test-cluster-aws-observability-ns",
					"kubernetes:manifest:test-app-test-cluster-fluent-bit",
					"kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-k8sAdmin"},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: "aws:vpc_subnet:test_app_test_subnet"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-FargateExecutionRole"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:vpc_subnet:test_app_test_subnet"},
					{Source: "aws:eks_node_group:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_node_group:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-NodeGroupRole"},
					{Source: "aws:eks_node_group:test-app-test-cluster", Destination: "aws:vpc_subnet:test_app_test_subnet"},
					{Source: "kubernetes:helm_chart:test-cluster-cert-manager", Destination: "aws:eks_node_group:test-app-test-cluster"},
					{Source: "kubernetes:helm_chart:test-cluster-metrics-server", Destination: "aws:eks_node_group:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-config-map", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-config-map", Destination: "kubernetes:manifest:test-app-test-cluster-aws-observability-ns"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-ns", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit", Destination: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map", Destination: "aws:region:region"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			CreateEksCluster(appName, clusterName, []*Subnet{subnet}, nil, eus, dag)
			for _, r := range dag.ListResources() {
				if cluster, ok := r.(*EksCluster); ok {
					assert.Len(cluster.Manifests, 4)
				}
				if r != subnet && r.Id() != "aws:region:region" { // ignore input resources
					assert.ElementsMatch(sources, r.KlothoConstructRef(), "not matching refs in %s", r.Id())
				}
			}
			fmt.Println(coretesting.ResoucesFromDAG(dag).GoString())
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
