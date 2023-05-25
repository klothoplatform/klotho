package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_EksClusterCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[EksClusterCreateParams, *EksCluster]{
		{
			Name: "nil lambda",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:my-app-cluster-addon-vpc-cni",
					"aws:eks_cluster:my-app-cluster",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-cluster-ClusterAdmin",
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
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-cluster"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:iam_role:my-app-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:vpc:my_app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
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
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.Equal(cluster.Name, "my-app-cluster")
				assert.NotNil(cluster.Vpc)
				assert.Len(cluster.SecurityGroups, 1)
				assert.Equal(cluster.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing lambda",
			Existing: []core.Resource{&EksCluster{Name: "my-app-cluster", ConstructsRef: initialRefs}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_cluster:my-app-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, cluster *EksCluster) {
				assert.Equal(cluster.Name, "my-app-cluster")
				expect := initialRefs.CloneWith(core.AnnotationKeySetOf(eu.AnnotationKey))
				assert.Equal(cluster.ConstructsRef, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksClusterCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(eu.AnnotationKey),
				Name:    "cluster",
			}
			tt.Run(t)
		})
	}
}

func Test_EksFargateProfileCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[EksFargateProfileCreateParams, *EksFargateProfile]{
		{
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:my-app-cluster-addon-vpc-cni",
					"aws:eks_cluster:my-app-cluster",
					"aws:eks_fargate_profile:my-app_profile_private",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-cluster-ClusterAdmin",
					"aws:iam_role:my-app-profile-PodExecutionRole",
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
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-cluster"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:iam_role:my-app-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:vpc:my_app"},
					{Source: "aws:eks_fargate_profile:my-app_profile_private", Destination: "aws:eks_cluster:my-app-cluster"},
					{Source: "aws:eks_fargate_profile:my-app_profile_private", Destination: "aws:iam_role:my-app-profile-PodExecutionRole"},
					{Source: "aws:eks_fargate_profile:my-app_profile_private", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_fargate_profile:my-app_profile_private", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
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
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, profile *EksFargateProfile) {
				assert.Equal(profile.Name, "my-app_profile_private")
				assert.NotNil(profile.Cluster)
				assert.Len(profile.Subnets, 2)
				assert.Equal(profile.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing lambda",
			Existing: []core.Resource{&EksFargateProfile{Name: "my-app_profile_private", ConstructsRef: initialRefs}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_fargate_profile:my-app_profile_private",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, profile *EksFargateProfile) {
				assert.Equal(profile.Name, "my-app_profile_private")
				expect := initialRefs.CloneWith(core.AnnotationKeySetOf(eu.AnnotationKey))
				assert.Equal(profile.ConstructsRef, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksFargateProfileCreateParams{
				AppName:     "my-app",
				Refs:        core.AnnotationKeySetOf(eu.AnnotationKey),
				Name:        "profile",
				NetworkType: PrivateSubnet,
				ClusterName: "cluster",
			}
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
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[EksNodeGroupCreateParams, *EksNodeGroup]{
		{
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:my-app-cluster-addon-vpc-cni",
					"aws:eks_cluster:my-app-cluster",
					"aws:eks_node_group:my-app_cluster_private_t3_medium",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-cluster-ClusterAdmin",
					"aws:iam_role:my-app-cluster_private_t3_medium-NodeRole",
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
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-cluster"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:iam_role:my-app-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:vpc:my_app"},
					{Source: "aws:eks_node_group:my-app_cluster_private_t3_medium", Destination: "aws:eks_cluster:my-app-cluster"},
					{Source: "aws:eks_node_group:my-app_cluster_private_t3_medium", Destination: "aws:iam_role:my-app-cluster_private_t3_medium-NodeRole"},
					{Source: "aws:eks_node_group:my-app_cluster_private_t3_medium", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_node_group:my-app_cluster_private_t3_medium", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
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
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, nodegroup *EksNodeGroup) {
				assert.Equal(nodegroup.Name, "my-app_cluster_private_t3_medium")
				assert.NotNil(nodegroup.Cluster)
				assert.Len(nodegroup.Subnets, 2)
				assert.Equal(nodegroup.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing lambda",
			Existing: []core.Resource{&EksNodeGroup{Name: "my-app_cluster_private_t3_medium", ConstructsRef: initialRefs}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_node_group:my-app_cluster_private_t3_medium",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, nodegroup *EksNodeGroup) {
				assert.Equal(nodegroup.Name, "my-app_cluster_private_t3_medium")
				expect := initialRefs.CloneWith(core.AnnotationKeySetOf(eu.AnnotationKey))
				assert.Equal(nodegroup.ConstructsRef, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksNodeGroupCreateParams{
				AppName:      "my-app",
				Refs:         core.AnnotationKeySetOf(eu.AnnotationKey),
				InstanceType: "t3.medium",
				NetworkType:  PrivateSubnet,
				ClusterName:  "cluster",
			}
			tt.Run(t)
		})
	}
}
