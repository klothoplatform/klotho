package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_EksClusterCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
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
				assert.Equal(cluster.ConstructRefs, construct.BaseConstructSetOf(eu))
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
				assert.Equal(cluster.ConstructRefs, construct.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksClusterCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
				Name:    "cluster",
			}
			tt.Run(t)
		})
	}
}

func Test_EksFargateProfileCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
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
				assert.Equal(profile.ConstructRefs, construct.BaseConstructSetOf(eu))
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
				assert.Equal(profile.ConstructRefs, construct.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksFargateProfileCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}

func Test_EksNodeGroupCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
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
				assert.Equal(group.ConstructRefs, construct.BaseConstructSetOf(eu))
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
				assert.Equal(group.ConstructRefs, construct.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EksNodeGroupCreateParams{
				AppName:      "my-app",
				Refs:         construct.BaseConstructSetOf(eu),
				InstanceType: "t3.medium",
				NetworkType:  "private",
			}
			tt.Run(t)
		})
	}
}
