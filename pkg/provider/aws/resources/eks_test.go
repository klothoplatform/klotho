package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_EksClusterCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EksClusterCreateParams, *EksCluster]{
		{
			Name: "nil cluster",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_addon:my-app-cluster-addon-vpc-cni",
					"aws:eks_cluster:my-app-cluster",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-cluster"},
				},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.Equal(cluster.Name, "my-app-cluster")
				assert.Equal(cluster.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing cluster",
			Existing: &EksCluster{Name: "my-app-cluster", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_cluster:my-app-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.Equal(cluster.Name, "my-app-cluster")
				assert.Equal(cluster.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksClusterCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "cluster",
			}
			tt.Run(t)
		})
	}
}

func Test_EksClusterMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*EksCluster]{
		{
			Name:     "only cluster",
			Resource: &EksCluster{Name: "my_app"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_cluster:my_app",
					"aws:eks_node_group:my_app_private_t3_medium",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-my_app-ClusterAdmin",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
					"kubernetes:helm_chart:my_app-cert-manager",
					"kubernetes:helm_chart:my_app-metrics-server",
					"kubernetes:manifest:my_app-awmazon-cloudwatch-ns",
					"kubernetes:manifest:my_app-fluent-bit",
					"kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:my_app", Destination: "aws:iam_role:my-app-my_app-ClusterAdmin"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:my_app"},
					{Source: "aws:eks_node_group:my_app_private_t3_medium", Destination: "aws:eks_cluster:my_app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "kubernetes:helm_chart:my_app-cert-manager", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:helm_chart:my_app-metrics-server", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns"},
				},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.NotNil(cluster.Vpc)
				assert.Len(cluster.Subnets, 4)
				assert.Len(cluster.SecurityGroups, 1)
				assert.NotNil(cluster.ClusterRole)
			},
		},
		{
			Name:     "cluster has upstream role",
			Resource: &EksCluster{Name: "my_app"},
			Existing: []core.Resource{&IamRole{Name: "ClusterAdmin"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:eks_cluster:my_app", Destination: "aws:iam_role:ClusterAdmin"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_cluster:my_app",
					"aws:eks_node_group:my_app_private_t3_medium",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:ClusterAdmin",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
					"kubernetes:helm_chart:my_app-cert-manager",
					"kubernetes:helm_chart:my_app-metrics-server",
					"kubernetes:manifest:my_app-awmazon-cloudwatch-ns",
					"kubernetes:manifest:my_app-fluent-bit",
					"kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:my_app", Destination: "aws:iam_role:ClusterAdmin"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:my_app"},
					{Source: "aws:eks_node_group:my_app_private_t3_medium", Destination: "aws:eks_cluster:my_app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "kubernetes:helm_chart:my_app-cert-manager", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:helm_chart:my_app-metrics-server", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns"},
				},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.NotNil(cluster.Vpc)
				assert.Len(cluster.Subnets, 4)
				assert.Len(cluster.SecurityGroups, 1)
				assert.Equal(cluster.ClusterRole.Name, "ClusterAdmin")
			},
		},
		{
			Name:     "cluster has upstream vpc",
			Resource: &EksCluster{Name: "my_app"},
			Existing: []core.Resource{&Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_cluster:my_app",
					"aws:eks_node_group:my_app_private_t3_medium",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-my_app-ClusterAdmin",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:test:my-app",
					"aws:subnet_private:test:my_app_private0",
					"aws:subnet_private:test:my_app_private1",
					"aws:subnet_public:test:my_app_public0",
					"aws:subnet_public:test:my_app_public1",
					"aws:vpc:test",
					"kubernetes:helm_chart:my_app-cert-manager",
					"kubernetes:helm_chart:my_app-metrics-server",
					"kubernetes:manifest:my_app-awmazon-cloudwatch-ns",
					"kubernetes:manifest:my_app-fluent-bit",
					"kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:my_app", Destination: "aws:iam_role:my-app-my_app-ClusterAdmin"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:security_group:test:my-app"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:test:my_app_private0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:test:my_app_private1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:test:my_app_public0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:test"},
					{Source: "aws:eks_node_group:my_app_private_t3_medium", Destination: "aws:eks_cluster:my_app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:test:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:test:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:test:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:test"},
					{Source: "aws:security_group:test:my-app", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_private:test:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:test:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:test:my_app_private0", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:test:my_app_public0", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:vpc:test"},
					{Source: "kubernetes:helm_chart:my_app-cert-manager", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:helm_chart:my_app-metrics-server", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns"},
				},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.Equal(cluster.Vpc.Name, "test")
				assert.Len(cluster.Subnets, 4)
				assert.Len(cluster.SecurityGroups, 1)
				assert.Equal(cluster.ClusterRole.Name, "my-app-my_app-ClusterAdmin")
			},
		},
		{
			Name:     "cluster has upstream subnet",
			Resource: &EksCluster{Name: "my_app"},
			Existing: []core.Resource{&Subnet{Name: "test", Type: PrivateSubnet, AvailabilityZone: core.IaCValue{Property: "1"}, Vpc: &Vpc{Name: "test"}}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:subnet_private:test:test", Destination: "aws:vpc:test"},
				{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:test:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_cluster:my_app",
					"aws:eks_node_group:my_app_private_t3_medium",
					"aws:iam_role:my-app-my_app-ClusterAdmin",
					"aws:security_group:test:my-app",
					"aws:subnet_private:test:test",
					"aws:vpc:test",
					"kubernetes:helm_chart:my_app-cert-manager",
					"kubernetes:helm_chart:my_app-metrics-server",
					"kubernetes:manifest:my_app-awmazon-cloudwatch-ns",
					"kubernetes:manifest:my_app-fluent-bit",
					"kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:my_app", Destination: "aws:iam_role:my-app-my_app-ClusterAdmin"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:security_group:test:my-app"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:test:test"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:test"},
					{Source: "aws:eks_node_group:my_app_private_t3_medium", Destination: "aws:eks_cluster:my_app"},
					{Source: "aws:security_group:test:my-app", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_private:test:test", Destination: "aws:vpc:test"},
					{Source: "kubernetes:helm_chart:my_app-cert-manager", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:helm_chart:my_app-metrics-server", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns"},
				},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.Equal(cluster.Vpc.Name, "test")
				assert.Len(cluster.Subnets, 1)
				assert.Len(cluster.SecurityGroups, 1)
				assert.Equal(cluster.ClusterRole.Name, "my-app-my_app-ClusterAdmin")
			},
		},
		{
			Name:     "cluster has upstream security group",
			Resource: &EksCluster{Name: "my_app"},
			Existing: []core.Resource{&SecurityGroup{Name: "test", Vpc: &Vpc{Name: "test"}}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:security_group:test:test", Destination: "aws:vpc:test"},
				{Source: "aws:eks_cluster:my_app", Destination: "aws:security_group:test:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_cluster:my_app",
					"aws:eks_node_group:my_app_private_t3_medium",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-my_app-ClusterAdmin",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:test:test",
					"aws:subnet_private:test:my_app_private0",
					"aws:subnet_private:test:my_app_private1",
					"aws:subnet_public:test:my_app_public0",
					"aws:subnet_public:test:my_app_public1",
					"aws:vpc:test",
					"kubernetes:helm_chart:my_app-cert-manager",
					"kubernetes:helm_chart:my_app-metrics-server",
					"kubernetes:manifest:my_app-awmazon-cloudwatch-ns",
					"kubernetes:manifest:my_app-fluent-bit",
					"kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:my_app", Destination: "aws:iam_role:my-app-my_app-ClusterAdmin"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:security_group:test:test"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:test:my_app_private0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_private:test:my_app_private1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:test:my_app_public0"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:test"},
					{Source: "aws:eks_node_group:my_app_private_t3_medium", Destination: "aws:eks_cluster:my_app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:test:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:test:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:test:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:test"},
					{Source: "aws:security_group:test:test", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_private:test:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:test:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:test:my_app_private0", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:test:my_app_public0", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:vpc:test"},
					{Source: "kubernetes:helm_chart:my_app-cert-manager", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:helm_chart:my_app-metrics-server", Destination: "aws:eks_node_group:my_app_private_t3_medium"},
					{Source: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit", Destination: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:my_app"},
					{Source: "kubernetes:manifest:my_app-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:my_app-awmazon-cloudwatch-ns"},
				},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.Equal(cluster.Vpc.Name, "test")
				assert.Len(cluster.Subnets, 4)
				assert.Len(cluster.SecurityGroups, 1)
				assert.Equal(cluster.ClusterRole.Name, "my-app-my_app-ClusterAdmin")
			},
		},
		{
			Name:     "multiple vpcs error",
			Resource: &EksCluster{Name: "my_app"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:test-down"},
				{Source: "aws:eks_cluster:my_app", Destination: "aws:vpc:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_EksFargateProfileCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EksFargateProfileCreateParams, *EksFargateProfile]{
		{
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_fargate_profile:my-app_profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, profile *EksFargateProfile) {
				assert.Equal(profile.Name, "my-app_profile")
				assert.Equal(profile.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing profile",
			Existing: &EksFargateProfile{Name: "my-app_profile", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_fargate_profile:my-app_profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, profile *EksFargateProfile) {
				assert.Equal(profile.Name, "my-app_profile")
				assert.Equal(profile.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksFargateProfileCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}

func Test_EksFargateProfileMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*EksFargateProfile]{
		{
			Name:     "only cluster",
			Resource: &EksFargateProfile{Name: "profile"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:my-app-eks-cluster-addon-vpc-cni",
					"aws:eks_cluster:my-app-eks-cluster",
					"aws:eks_fargate_profile:profile",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-profile-PodExecutionRole",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-eks-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-eks-cluster"},
					{Source: "aws:eks_fargate_profile:profile", Destination: "aws:eks_cluster:my-app-eks-cluster"},
					{Source: "aws:eks_fargate_profile:profile", Destination: "aws:iam_role:my-app-profile-PodExecutionRole"},
					{Source: "aws:eks_fargate_profile:profile", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_fargate_profile:profile", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, profile *EksFargateProfile) {
				assert.NotNil(profile.Cluster)
				assert.Len(profile.Subnets, 2)
				assert.NotNil(profile.PodExecutionRole)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_EksFargateProfileConfigure(t *testing.T) {
	cases := []coretesting.ConfigureCase[EksFargateProfileConfigureParams, *EksFargateProfile]{
		{
			Name:   "nil namespace",
			Params: EksFargateProfileConfigureParams{},
			Want:   &EksFargateProfile{Selectors: []*FargateProfileSelector{{Namespace: "default", Labels: map[string]string{"klotho-fargate-enabled": "true"}}}},
		},
		{
			Name: "existing namespace",
			Params: EksFargateProfileConfigureParams{
				Namespace: "unit1",
			},
			Want: &EksFargateProfile{Selectors: []*FargateProfileSelector{{Namespace: "unit1", Labels: map[string]string{"klotho-fargate-enabled": "true"}}}},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_EksNodeGroupCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EksNodeGroupCreateParams, *EksNodeGroup]{
		{
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_node_group:my-app_private_t3_medium",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, group *EksNodeGroup) {
				assert.Equal(group.Name, "my-app_private_t3_medium")
				assert.Equal(group.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing profile",
			Existing: &EksNodeGroup{Name: "my-app_private_t3_medium", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_node_group:my-app_private_t3_medium",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, group *EksNodeGroup) {
				assert.Equal(group.Name, "my-app_private_t3_medium")
				assert.Equal(group.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksNodeGroupCreateParams{
				AppName:      "my-app",
				Refs:         core.BaseConstructSetOf(eu),
				InstanceType: "t3.medium",
				NetworkType:  "private",
			}
			tt.Run(t)
		})
	}
}

func Test_EksNodeGroupMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*EksNodeGroup]{
		{
			Name:     "only cluster",
			Resource: &EksNodeGroup{Name: "profile"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:my-app-eks-cluster-addon-vpc-cni",
					"aws:eks_cluster:my-app-eks-cluster",
					"aws:eks_node_group:my-app-eks-cluster-profile",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-my-app-eks-cluster-profile-NodeRole",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-eks-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-eks-cluster"},
					{Source: "aws:eks_node_group:my-app-eks-cluster-profile", Destination: "aws:eks_cluster:my-app-eks-cluster"},
					{Source: "aws:eks_node_group:my-app-eks-cluster-profile", Destination: "aws:iam_role:my-app-my-app-eks-cluster-profile-NodeRole"},
					{Source: "aws:eks_node_group:my-app-eks-cluster-profile", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_node_group:my-app-eks-cluster-profile", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, group *EksNodeGroup) {
				assert.NotNil(group.Cluster)
				assert.Len(group.Subnets, 2)
				assert.NotNil(group.NodeRole)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
