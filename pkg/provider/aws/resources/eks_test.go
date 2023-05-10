package resources

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_EksClusterCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name    string
		cluster *EksCluster
		want    coretesting.ResourcesExpectation
	}{
		{
			name: "nil cluster",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_cluster:app-my-cluster",
					"aws:elastic_ip:app_0",
					"aws:elastic_ip:app_1",
					"aws:iam_role:app-my-cluster-ClusterAdmin",
					"aws:internet_gateway:app_igw",
					"aws:nat_gateway:app_0",
					"aws:nat_gateway:app_1",
					"aws:route_table:app_0",
					"aws:route_table:app_1",
					"aws:route_table:app_igw",
					"aws:security_group:app",
					"aws:vpc:app",
					"aws:vpc_subnet:app_private0",
					"aws:vpc_subnet:app_private1",
					"aws:vpc_subnet:app_public0",
					"aws:vpc_subnet:app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:iam_role:app-my-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:security_group:app"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private1"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:internet_gateway:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:elastic_ip:app_0"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:elastic_ip:app_1"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:route_table:app_0", Destination: "aws:nat_gateway:app_0"},
					{Source: "aws:route_table:app_0", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_1", Destination: "aws:nat_gateway:app_1"},
					{Source: "aws:route_table:app_1", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_igw", Destination: "aws:internet_gateway:app_igw"},
					{Source: "aws:route_table:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:security_group:app", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:vpc:app"},
				},
			},
		},
		{
			name:    "existing cluster",
			cluster: &EksCluster{Name: "app-my-cluster", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_cluster:app-my-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.cluster != nil {
				dag.AddResource(tt.cluster)
			}
			metadata := EksClusterCreateParams{
				AppName:     "app",
				ClusterName: "my-cluster",
				Refs:        []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			}
			cluster := &EksCluster{}
			err := cluster.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphCluster := dag.GetResourceByVertexId(cluster.Id().String()).(*EksCluster)

			assert.Equal(graphCluster.Name, "app-my-cluster")
			if tt.cluster == nil {
				assert.Equal(graphCluster.ConstructsRef, metadata.Refs)
				assert.Contains(graphCluster.SecurityGroups[0].IngressRules, SecurityGroupRule{
					Description: "Allows ingress traffic from the EKS control plane",
					FromPort:    9443,
					Protocol:    "TCP",
					ToPort:      9443,
					CidrBlocks: []core.IaCValue{
						{Property: "0.0.0.0/0"},
					},
				})
				assert.NotNil(graphCluster.Kubeconfig)
			} else {
				assert.Equal(graphCluster.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}))

			}
		})
	}
}

func Test_EksFargateProfileCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name    string
		profile *EksFargateProfile
		want    coretesting.ResourcesExpectation
	}{
		{
			name: "nil profile",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_cluster:app-my-cluster",
					"aws:eks_fargate_profile:app_my-cluster",
					"aws:elastic_ip:app_0",
					"aws:elastic_ip:app_1",
					"aws:iam_role:app-my-cluster-ClusterAdmin",
					"aws:iam_role:app_my-cluster-PodExecutionRole",
					"aws:internet_gateway:app_igw",
					"aws:nat_gateway:app_0",
					"aws:nat_gateway:app_1",
					"aws:route_table:app_0",
					"aws:route_table:app_1",
					"aws:route_table:app_igw",
					"aws:security_group:app",
					"aws:vpc:app",
					"aws:vpc_subnet:app_private0",
					"aws:vpc_subnet:app_private1",
					"aws:vpc_subnet:app_public0",
					"aws:vpc_subnet:app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:iam_role:app-my-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:security_group:app"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private1"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:eks_fargate_profile:app_my-cluster", Destination: "aws:eks_cluster:app-my-cluster"},
					{Source: "aws:eks_fargate_profile:app_my-cluster", Destination: "aws:iam_role:app_my-cluster-PodExecutionRole"},
					{Source: "aws:eks_fargate_profile:app_my-cluster", Destination: "aws:vpc_subnet:app_private0"},
					{Source: "aws:eks_fargate_profile:app_my-cluster", Destination: "aws:vpc_subnet:app_private1"},
					{Source: "aws:internet_gateway:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:elastic_ip:app_0"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:elastic_ip:app_1"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:route_table:app_0", Destination: "aws:nat_gateway:app_0"},
					{Source: "aws:route_table:app_0", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_1", Destination: "aws:nat_gateway:app_1"},
					{Source: "aws:route_table:app_1", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_igw", Destination: "aws:internet_gateway:app_igw"},
					{Source: "aws:route_table:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:security_group:app", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:vpc:app"},
				},
			},
		},
		{
			name:    "existing profile",
			profile: &EksFargateProfile{Name: "app_my-cluster", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_fargate_profile:app_my-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name: "existing profile add selector",
			profile: &EksFargateProfile{
				Name:          "app_my-cluster",
				ConstructsRef: initialRefs,
				Selectors:     []*FargateProfileSelector{{Namespace: "namespace1", Labels: map[string]string{"klotho-fargate-enabled": "true"}}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_fargate_profile:app_my-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.profile != nil {
				dag.AddResource(tt.profile)
			}
			metadata := EksFargateProfileCreateParams{
				NetworkType: "private",
				AppName:     "app",
				ClusterName: "my-cluster",
				Refs:        []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				Namespace:   "default",
			}
			profile := &EksFargateProfile{}
			err := profile.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphProfile := dag.GetResourceByVertexId(profile.Id().String()).(*EksFargateProfile)

			assert.Equal(graphProfile.Name, "app_my-cluster")
			if tt.profile == nil {
				assert.Equal(graphProfile.ConstructsRef, metadata.Refs)
				assert.ElementsMatch(graphProfile.Selectors, []*FargateProfileSelector{{Namespace: "default", Labels: map[string]string{"klotho-fargate-enabled": "true"}}})
				assert.Equal(graphProfile.PodExecutionRole.AssumeRolePolicyDoc, EKS_FARGATE_ASSUME_ROLE_POLICY)
				assert.ElementsMatch(graphProfile.PodExecutionRole.AwsManagedPolicies, []string{
					"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
					"arn:aws:iam::aws:policy/AmazonEKSFargatePodExecutionRolePolicy",
				})
				assert.Len(graphProfile.PodExecutionRole.InlinePolicies, 1)
				assert.Len(graphProfile.PodExecutionRole.InlinePolicies[0].Policy.Statement, 1)
				assert.Equal(graphProfile.PodExecutionRole.InlinePolicies[0].Policy.Statement, []StatementEntry{
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
			} else {
				if tt.profile.Selectors[0].Namespace == "default" {
					assert.ElementsMatch(graphProfile.Selectors, []*FargateProfileSelector{{Namespace: "default", Labels: map[string]string{"klotho-fargate-enabled": "true"}}})
				} else {
					assert.ElementsMatch(graphProfile.Selectors, []*FargateProfileSelector{
						{Namespace: "namespace1", Labels: map[string]string{"klotho-fargate-enabled": "true"}},
						{Namespace: "default", Labels: map[string]string{"klotho-fargate-enabled": "true"}},
					})
				}
				assert.Equal(graphProfile.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}))
			}
		})
	}
}

func Test_EksNodeGroupCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		ng   *EksNodeGroup
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil cluster",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_cluster:app-my-cluster",
					"aws:eks_node_group:app_my-cluster_private_t3_medium",
					"aws:elastic_ip:app_0",
					"aws:elastic_ip:app_1",
					"aws:iam_role:app-my-cluster-ClusterAdmin",
					"aws:iam_role:app_my-cluster_private_t3_medium-NodeRole",
					"aws:internet_gateway:app_igw",
					"aws:nat_gateway:app_0",
					"aws:nat_gateway:app_1",
					"aws:route_table:app_0",
					"aws:route_table:app_1",
					"aws:route_table:app_igw",
					"aws:security_group:app",
					"aws:vpc:app",
					"aws:vpc_subnet:app_private0",
					"aws:vpc_subnet:app_private1",
					"aws:vpc_subnet:app_public0",
					"aws:vpc_subnet:app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:iam_role:app-my-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:security_group:app"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private1"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:eks_node_group:app_my-cluster_private_t3_medium", Destination: "aws:eks_cluster:app-my-cluster"},
					{Source: "aws:eks_node_group:app_my-cluster_private_t3_medium", Destination: "aws:iam_role:app_my-cluster_private_t3_medium-NodeRole"},
					{Source: "aws:eks_node_group:app_my-cluster_private_t3_medium", Destination: "aws:vpc_subnet:app_private0"},
					{Source: "aws:eks_node_group:app_my-cluster_private_t3_medium", Destination: "aws:vpc_subnet:app_private1"},
					{Source: "aws:internet_gateway:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:elastic_ip:app_0"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:elastic_ip:app_1"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:route_table:app_0", Destination: "aws:nat_gateway:app_0"},
					{Source: "aws:route_table:app_0", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_1", Destination: "aws:nat_gateway:app_1"},
					{Source: "aws:route_table:app_1", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_igw", Destination: "aws:internet_gateway:app_igw"},
					{Source: "aws:route_table:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:security_group:app", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:vpc:app"},
				},
			},
		},
		{
			name: "existing cluster",
			ng:   &EksNodeGroup{Name: "app_my-cluster_private_t3_medium", ConstructsRef: initialRefs, DiskSize: 10},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:eks_node_group:app_my-cluster_private_t3_medium",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.ng != nil {
				dag.AddResource(tt.ng)
			}
			metadata := EksNodeGroupCreateParams{
				InstanceType: "t3.medium",
				NetworkType:  "private",
				DiskSizeGiB:  20,
				AppName:      "app",
				ClusterName:  "my-cluster",
				Refs:         []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			}
			nodeGroup := &EksNodeGroup{}
			err := nodeGroup.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphNG := dag.GetResourceByVertexId(nodeGroup.Id().String()).(*EksNodeGroup)

			assert.Equal(graphNG.Name, "app_my-cluster_private_t3_medium")
			if tt.ng == nil {
				assert.Equal(graphNG.ConstructsRef, metadata.Refs)
				assert.Equal(graphNG.DiskSize, 20)
				assert.Equal(graphNG.InstanceTypes, []string{"t3.medium"})
			} else {
				assert.Equal(graphNG.DiskSize, 20)
				assert.Equal(graphNG.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}))

			}
		})
	}
}

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
					"aws:eks_addon:test-app-test-cluster-addon-vpc-cni",
					"aws:eks_cluster:test-app-test-cluster",
					"aws:eks_fargate_profile:test-app-test-cluster",
					"aws:eks_node_group:test-cluster_private_g2",
					"aws:eks_node_group:test-cluster_private_t3_medium",
					"aws:iam_oidc_provider:test-app-test-cluster",
					"aws:iam_role:test-app-test-cluster-FargateExecutionRole",
					"aws:iam_role:test-app-test-cluster-k8sAdmin",
					"aws:iam_role:test-app-test-cluster_private_g2",
					"aws:iam_role:test-app-test-cluster_private_t3_medium",
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
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:test-app-test-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-k8sAdmin"},
					{Source: "aws:eks_cluster:test-app-test-cluster", Destination: "aws:vpc_subnet:test_app_test_subnet"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:iam_role:test-app-test-cluster-FargateExecutionRole"},
					{Source: "aws:eks_fargate_profile:test-app-test-cluster", Destination: "aws:vpc_subnet:test_app_test_subnet"},
					{Source: "aws:eks_node_group:test-cluster_private_g2", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_node_group:test-cluster_private_g2", Destination: "aws:iam_role:test-app-test-cluster_private_g2"},
					{Source: "aws:eks_node_group:test-cluster_private_g2", Destination: "aws:vpc_subnet:test_app_test_subnet"},
					{Source: "aws:eks_node_group:test-cluster_private_t3_medium", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:eks_node_group:test-cluster_private_t3_medium", Destination: "aws:iam_role:test-app-test-cluster_private_t3_medium"},
					{Source: "aws:eks_node_group:test-cluster_private_t3_medium", Destination: "aws:vpc_subnet:test_app_test_subnet"},
					{Source: "aws:iam_oidc_provider:test-app-test-cluster", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "aws:iam_oidc_provider:test-app-test-cluster", Destination: "aws:region:region"},
					{Source: "aws:vpc:test_app", Destination: "aws:region:region"},
					{Source: "aws:vpc_subnet:test_app_test_subnet", Destination: "aws:vpc:test_app"},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-cert-manager", Destination: "aws:eks_node_group:test-cluster_private_g2"},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-cert-manager", Destination: "aws:eks_node_group:test-cluster_private_t3_medium"},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-metrics-server", Destination: "aws:eks_node_group:test-cluster_private_g2"},
					{Source: "kubernetes:helm_chart:test-app-test-cluster-metrics-server", Destination: "aws:eks_node_group:test-cluster_private_t3_medium"},
					{Source: "kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-config-map", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-config-map", Destination: "kubernetes:manifest:test-app-test-cluster-aws-observability-ns"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-config-map", Destination: "aws:region:region"},
					{Source: "kubernetes:manifest:test-app-test-cluster-aws-observability-ns", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit", Destination: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-fluent-bit-cluster-info-config-map", Destination: "kubernetes:manifest:test-app-test-cluster-awmazon-cloudwatch-ns"},
					{Source: "kubernetes:manifest:test-app-test-cluster-nvidia-device-plugin", Destination: "aws:eks_cluster:test-app-test-cluster"},
					{Source: "kubernetes:manifest:test-app-test-cluster-nvidia-device-plugin", Destination: "aws:eks_node_group:test-cluster_private_g2"},
					{Source: "kubernetes:manifest:test-app-test-cluster-nvidia-device-plugin", Destination: "aws:eks_node_group:test-cluster_private_t3_medium"},
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
	_, err := cluster.InstallCloudMapController(unit1.AnnotationKey, dag)
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
	_, err = cluster.InstallCloudMapController(unit2.AnnotationKey, dag)
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
	assert.ElementsMatch(role.InlinePolicies[0].Policy.Statement, []StatementEntry{
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
		"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
	})
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
				{Name: "cluster_private_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
			},
		},
		{
			name: "network types",
			units: map[string]*config.ExecutionUnit{
				"a": {},
				"b": {NetworkPlacement: "public"},
			},
			want: []NodeGroupExpect{
				{Name: "cluster_private_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
				{Name: "cluster_public_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
			},
		},
		{
			name: "instance types",
			units: map[string]*config.ExecutionUnit{
				"a": {InfraParams: config.InfraParams{"instance_type": "c1.medium"}},
				"b": {InfraParams: config.InfraParams{"instance_type": "g2.medium"}},
			},
			want: []NodeGroupExpect{
				{Name: "cluster_private_c1_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
				{Name: "cluster_private_g2_medium", DiskSize: 20, AmiType: "AL2_x86_64_GPU"},
			},
		},
		{
			name: "disk size maximum",
			units: map[string]*config.ExecutionUnit{
				"a": {InfraParams: config.InfraParams{"disk_size_gib": 10}},
				"b": {NetworkPlacement: "public", InfraParams: config.InfraParams{"disk_size_gib": 50}},
			},
			want: []NodeGroupExpect{
				{Name: "cluster_private_t3_medium", DiskSize: 20, AmiType: "AL2_x86_64"},
				{Name: "cluster_public_t3_medium", DiskSize: 50, AmiType: "AL2_x86_64"},
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
