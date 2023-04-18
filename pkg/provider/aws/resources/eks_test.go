package resources

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_CreateEksCluster(t *testing.T) {
	cfg := config.Application{AppName: "test-app", ExecutionUnits: make(map[string]*config.ExecutionUnit)}
	clusterName := "test-cluster"
	eu1 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	cfg.ExecutionUnits[eu1.ID] = &config.ExecutionUnit{
		Type:             "kubernetes",
		NetworkPlacement: "private",
		InfraParams:      config.InfraParams{"instance_type": "t3.medium"},
	}
	eu2 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test-gpu"}}
	cfg.ExecutionUnits[eu2.ID] = &config.ExecutionUnit{
		Type:             "kubernetes",
		NetworkPlacement: "private",
		InfraParams:      config.InfraParams{"instance_type": "g2"},
	}
	eus := []*core.ExecutionUnit{eu1, eu2}
	sources := []core.AnnotationKey{{ID: "test"}, {ID: "test-gpu"}}
	vpc := NewVpc(cfg.AppName)
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
					"aws:eks_node_group:private_t3_medium",
					"aws:eks_node_group:private_g2",
					"aws:iam_role:test-app-test-cluster-FargateExecutionRole",
					"aws:iam_role:test-app-test-cluster-k8sAdmin",
					"aws:iam_role:test-app-test-cluster.private_t3_medium",
					"aws:iam_role:test-app-test-cluster.private_g2",
					"aws:region:region",
					"aws:vpc:test_app",
					"aws:vpc_subnet:test_app_test_subnet",
					"kubernetes:helm_chart:test-app-test-cluster-cert-manager",
					"kubernetes:helm_chart:test-app-test-cluster-metrics-server",
					"kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns",
					"kubernetes:manifest:test-app-test-cluster-aws-observability-config-map",
					"kubernetes:manifest:test-app-test-cluster-aws-observability-ns",
					"kubernetes:manifest:test-app-test-cluster-fluent-bit",
					"kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map",
					"kubernetes:manifest:test-app-test-cluster-nvidia-device-plugin",
					"aws:eks_addon:test-app-test-cluster-addon-vpc-cni",
					"aws:iam_oidc_provider:test-app-test-cluster",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:vpc:test_app", Destination: "aws:region:region"},
					{Source: "aws:vpc_subnet:test_app_test_subnet", Destination: "aws:vpc:test_app"},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-k8sAdmin"},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: subnet.Id()},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-FargateExecutionRole"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: subnet.Id()},
					{Source: "aws:eks_node_group:private_t3_medium", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_node_group:private_t3_medium", Destination: "aws:iam_role:test-app-test-cluster.private_t3_medium"},
					{Source: "aws:eks_node_group:private_t3_medium", Destination: subnet.Id()},
					{Source: "aws:eks_node_group:private_g2", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_node_group:private_g2", Destination: "aws:iam_role:test-app-test-cluster.private_g2"},
					{Source: "aws:eks_node_group:private_g2", Destination: subnet.Id()},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-cert-manager", Destination: "aws:eks_node_group:private_t3_medium"},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-cert-manager", Destination: "aws:eks_node_group:private_g2"},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-metrics-server", Destination: "aws:eks_node_group:private_t3_medium"},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-metrics-server", Destination: "aws:eks_node_group:private_g2"},
					{Source: "kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-config-map", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-config-map", Destination: "kubernetes:manifest:test-app-test-cluster-aws-observability-ns"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-ns", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit", Destination: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns"},
					{Source: "aws:eks_addon:test-app-test-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-nvidia-device-plugin", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-nvidia-device-plugin", Destination: "aws:eks_node_group:private_g2"},
					{Source: "kubernetes:manifest:test-app-test-cluster-nvidia-device-plugin", Destination: "aws:eks_node_group:private_t3_medium"},
					{Source: "aws:iam_oidc_provider:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:iam_oidc_provider:test-app-test-cluster", Destination: "aws:region:region"},
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

			assert.NoError(CreateEksCluster(&cfg, clusterName, []*Subnet{subnet}, nil, eus, dag))
			for _, r := range dag.ListResources() {
				if cluster, ok := r.(*EksCluster); ok {
					assert.Len(cluster.Manifests, 4)
				}
				if r != subnet && r != vpc && r != region { // ignore input resources
					assert.Subsetf(sources, r.KlothoConstructRef(), "not matching refs in %s", r.Id())
				}
			}
			tt.want.Assert(t, dag)
		})
	}
}

func Test_InstallCloudMapController(t *testing.T) {
	assert := assert.New(t)
	dag := core.NewResourceGraph()
	cluster := NewEksCluster("test", "cluster", nil, nil, nil)
	nodeGroup1 := &EksNodeGroup{
		Name:    "nodegroup1",
		Cluster: cluster,
	}
	dag.AddDependenciesReflect(nodeGroup1)

	unit1 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "1"}}
	err := cluster.InstallCloudMapController(unit1.AnnotationKey, dag)
	if !assert.NoError(err) {
		return
	}

	cloudMapController := &kubernetes.KustomizeDirectory{
		Name: fmt.Sprintf("%s-cloudmap-controller", cluster.Name),
	}

	if controller := dag.GetResource(cloudMapController.Id()); controller != nil {
		if cm, ok := controller.(*kubernetes.KustomizeDirectory); ok {
			assert.Equal(cm.ClustersProvider, core.IaCValue{
				Resource: cluster,
				Property: CLUSTER_PROVIDER_IAC_VALUE,
			})
			assert.Equal(cm.Directory, "https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release")
		} else {
			assert.NoError(errors.Errorf("Expected resource with id, %s, to be of type HelmChart, but was %s",
				controller.Id(), reflect.ValueOf(controller).Type().Name()))
		}
	}

	unit2 := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "2"}}
	err = cluster.InstallCloudMapController(unit2.AnnotationKey, dag)
	if !assert.NoError(err) {
		return
	}

	if controller := dag.GetResource(cloudMapController.Id()); controller != nil {
		assert.ElementsMatch(controller.KlothoConstructRef(), []core.AnnotationKey{unit1.AnnotationKey, unit2.AnnotationKey})
	}

}

func Test_getClustersNodeGroups(t *testing.T) {
	assert := assert.New(t)
	dag := core.NewResourceGraph()
	cluster := NewEksCluster("test", "cluster", nil, nil, nil)
	nodeGroup1 := &EksNodeGroup{
		Name:    "nodegroup1",
		Cluster: cluster,
	}
	nodeGroup2 := &EksNodeGroup{
		Name:    "nodegroup2",
		Cluster: cluster,
	}
	nodeGroup3 := &EksNodeGroup{
		Name: "nodegroup3",
	}
	dag.AddDependenciesReflect(nodeGroup1)
	dag.AddDependenciesReflect(nodeGroup2)
	dag.AddDependenciesReflect(nodeGroup3)
	assert.ElementsMatch(cluster.GetClustersNodeGroups(dag), []*EksNodeGroup{nodeGroup1, nodeGroup2})
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

func Test_createNodeGroups(t *testing.T) {
	cluster := &EksCluster{Name: "cluster"}
	subnets := []*Subnet{
		{Name: "private 1", Type: PrivateSubnet},
		{Name: "private 2", Type: PrivateSubnet},
		{Name: "public 1", Type: PublicSubnet},
		{Name: "public 2", Type: PublicSubnet},
	}
	type NodeGroupExpect struct {
		Name     string
		DiskSize int
		AmiType  string
	}
	tests := []struct {
		name  string
		units map[string]*config.ExecutionUnit
		want  []NodeGroupExpect
	}{
		{
			name: "no groups default",
			units: map[string]*config.ExecutionUnit{
				"a": {},
				"b": {},
			},
			want: []NodeGroupExpect{
				{Name: "private_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
			},
		},
		{
			name: "network types",
			units: map[string]*config.ExecutionUnit{
				"a": {},
				"b": {NetworkPlacement: "public"},
			},
			want: []NodeGroupExpect{
				{Name: "private_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
				{Name: "public_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
			},
		},
		{
			name: "instance types",
			units: map[string]*config.ExecutionUnit{
				"a": {InfraParams: config.InfraParams{"instance_type": "c1.medium"}},
				"b": {InfraParams: config.InfraParams{"instance_type": "g2.medium"}},
			},
			want: []NodeGroupExpect{
				{Name: "private_c1_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
				{Name: "private_g2_medium", DiskSize: 20, AmiType: "AL2_x86_64_GPU"},
			},
		},
		{
			name: "disk size maximum",
			units: map[string]*config.ExecutionUnit{
				"a": {InfraParams: config.InfraParams{"disk_size_gib": 10}},
				"b": {NetworkPlacement: "public", InfraParams: config.InfraParams{"disk_size_gib": 50}},
			},
			want: []NodeGroupExpect{
				{Name: "private_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
				{Name: "public_t3_medium", DiskSize: 50, AmiType: "AL2_x86_64"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			dag := core.NewResourceGraph()

			var units []*core.ExecutionUnit
			for name := range tt.units {
				units = append(units, &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: name}})
			}

			cfg := &config.Application{
				ExecutionUnits: tt.units,
			}

			groups := createNodeGroups(cfg, dag, units, cluster.Name, cluster, subnets)
			got := make([]NodeGroupExpect, len(groups))
			for i, group := range groups {
				got[i] = NodeGroupExpect{Name: group.Name, DiskSize: group.DiskSize, AmiType: group.AmiType}
			}

			assert.ElementsMatch(tt.want, got)
		})
	}
}
