package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_EcsServiceCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EcsServiceCreateParams, *EcsService]{
		{
			Name: "nil ecs service",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:my-app-service",
					"aws:ecr_repo:my-app",
					"aws:ecs_cluster:my-app-service-ExecutionRole",
					"aws:ecs_service:my-app-service",
					"aws:ecs_task_definition:my-app-service",
					"aws:iam_role:my-app-service-ExecutionRole",
					"aws:log_group:my-app-service-LogGroup",
					"aws:region:region",
					"aws:route_table:my_app_",
					"aws:security_group:my_app:my-app",
					"aws:subnet_:my_app:my_app_0",
					"aws:subnet_:my_app:my_app_1",
					"aws:vpc:my_app",
					"aws:availability_zones:AvailabilityZones",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-service", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:ecs_service:my-app-service", Destination: "aws:ecs_cluster:my-app-service-ExecutionRole"},
					{Source: "aws:ecs_service:my-app-service", Destination: "aws:ecs_task_definition:my-app-service"},
					{Source: "aws:ecs_task_definition:my-app-service", Destination: "aws:iam_role:my-app-service-ExecutionRole"},
					{Source: "aws:ecs_task_definition:my-app-service", Destination: "aws:log_group:my-app-service-LogGroup"},
					{Source: "aws:ecs_task_definition:my-app-service", Destination: "aws:region:region"},
					{Source: "aws:ecs_service:my-app-service", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:ecs_service:my-app-service", Destination: "aws:subnet_:my_app:my_app_0"},
					{Source: "aws:ecs_service:my-app-service", Destination: "aws:subnet_:my_app:my_app_1"},
					{Source: "aws:ecs_task_definition:my-app-service", Destination: "aws:ecr_image:my-app-service"},
					{Source: "aws:route_table:my_app_", Destination: "aws:subnet_:my_app:my_app_0"},
					{Source: "aws:route_table:my_app_", Destination: "aws:subnet_:my_app:my_app_1"},
					{Source: "aws:route_table:my_app_", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_:my_app:my_app_0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_:my_app:my_app_0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_:my_app:my_app_1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_:my_app:my_app_1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, service *EcsService) {
				assert.Equal(service.Name, "my-app-service")
				assert.NotNil(service.TaskDefinition)
				assert.NotNil(service.Cluster)
				assert.NotZero(service.LaunchType)
				assert.Equal(service.LaunchType, LAUNCH_TYPE_FARGATE)
				assert.Len(service.SecurityGroups, 1)
				assert.Len(service.Subnets, 2)
				assert.Equal(service.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing ecs service",
			Existing: &EcsService{Name: "my-app-service", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsServiceCreateParams{
				AppName:    "my-app",
				Refs:       core.BaseConstructSetOf(eu),
				Name:       "service",
				LaunchType: LAUNCH_TYPE_FARGATE,
			}
			tt.Run(t)
		})
	}
}

func Test_EcsTaskDefinitionCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EcsTaskDefinitionCreateParams, *EcsTaskDefinition]{
		{
			Name: "nil ecs task definition",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:my-app-task-definition",
					"aws:ecr_repo:my-app",
					"aws:ecs_task_definition:my-app-task-definition",
					"aws:iam_role:my-app-task-definition-ExecutionRole",
					"aws:log_group:my-app-task-definition-LogGroup",
					"aws:region:region",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-task-definition", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:ecs_task_definition:my-app-task-definition", Destination: "aws:ecr_image:my-app-task-definition"},
					{Source: "aws:ecs_task_definition:my-app-task-definition", Destination: "aws:iam_role:my-app-task-definition-ExecutionRole"},
					{Source: "aws:ecs_task_definition:my-app-task-definition", Destination: "aws:log_group:my-app-task-definition-LogGroup"},
					{Source: "aws:ecs_task_definition:my-app-task-definition", Destination: "aws:region:region"},
				},
			},
			Check: func(assert *assert.Assertions, taskDef *EcsTaskDefinition) {
				assert.Equal(taskDef.Name, "my-app-task-definition")
				assert.NotNil(taskDef.LogGroup)
				assert.NotNil(taskDef.Region)
				assert.NotNil(taskDef.ExecutionRole)
				assert.NotNil(taskDef.Image)
				assert.Equal(taskDef.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing ecs task defintion",
			Existing: &EcsTaskDefinition{Name: "my-app-task-definition", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsTaskDefinitionCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "task-definition",
			}
			tt.Run(t)
		})
	}
}

func Test_EcsCluster(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
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
				assert.Equal(cluster.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing ecs cluster",
			Existing: &EcsCluster{Name: "my-app-cluster", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecs_cluster:my-app-cluster",
				},
			},
			Check: func(assert *assert.Assertions, cluster *EcsCluster) {
				assert.Equal(cluster.Name, "my-app-cluster")
				assert.Equal(cluster.ConstructsRef, initialRefs.CloneWith(core.BaseConstructSetOf(eu)))
			}},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsClusterCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "cluster",
			}
			tt.Run(t)
		})
	}
}
