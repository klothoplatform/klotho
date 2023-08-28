package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_EcsServiceCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EcsServiceCreateParams, *EcsService]{
		{
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecs_service:my-app-service",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, group *EcsService) {
				assert.Equal(group.Name, "my-app-service")
				assert.Equal(group.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing profile",
			Existing: &EcsService{Name: "my-app-service", ConstructRefs: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsServiceCreateParams{
				AppName:          "my-app",
				Refs:             construct.BaseConstructSetOf(eu),
				Name:             "service",
				LaunchType:       "t3.medium",
				NetworkPlacement: "private",
			}
			tt.Run(t)
		})
	}
}

func Test_EcsTaskDefinitionCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EcsTaskDefinitionCreateParams, *EcsTaskDefinition]{
		{
			Name: "nil task definition",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecs_task_definition:my-app-td",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, td *EcsTaskDefinition) {
				assert.Equal(td.Name, "my-app-td")
				assert.Equal(td.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing profile",
			Existing: &EcsTaskDefinition{Name: "my-app-td", ConstructRefs: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsTaskDefinitionCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
				Name:    "td",
			}
			tt.Run(t)
		})
	}
}

func Test_EcsCluster(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EcsClusterCreateParams, *EcsCluster]{
		{
			Name: "nil ecs cluster",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecs_cluster:my-app-cluster",
				},
			},
			Check: func(assert *assert.Assertions, cluster *EcsCluster) {
				assert.Equal(cluster.Name, "my-app-cluster")
				assert.Equal(cluster.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing ecs cluster",
			Existing: &EcsCluster{Name: "my-app-cluster", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecs_cluster:my-app-cluster",
				},
			},
			Check: func(assert *assert.Assertions, cluster *EcsCluster) {
				assert.Equal(cluster.Name, "my-app-cluster")
				assert.Equal(cluster.ConstructRefs, initialRefs.CloneWith(construct.BaseConstructSetOf(eu)))
			}},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsClusterCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
				Name:    "cluster",
			}
			tt.Run(t)
		})
	}
}
