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
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecs_service:my-app-service",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, group *EcsService) {
				assert.Equal(group.Name, "my-app-service")
				assert.Equal(group.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing profile",
			Existing: &EcsService{Name: "my-app-service", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsServiceCreateParams{
				AppName:          "my-app",
				Refs:             core.BaseConstructSetOf(eu),
				Name:             "service",
				LaunchType:       "t3.medium",
				NetworkPlacement: "private",
			}
			tt.Run(t)
		})
	}
}

func Test_EcsServiceMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*EcsService]{
		{
			Name:     "only cluster",
			Resource: &EcsService{Name: "profile", LaunchType: LAUNCH_TYPE_FARGATE},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:ecr_image:my-app-my-app-profile",
					"aws:ecs_cluster:my-app-profile-cluster",
					"aws:ecs_service:profile",
					"aws:ecs_task_definition:my-app-profile",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-my-app-profile-ExecutionRole",
					"aws:internet_gateway:my_app_igw",
					"aws:log_group:my-app-my-app-profile-LogGroup",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:region:region",
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
					{Source: "aws:ecs_service:profile", Destination: "aws:ecs_cluster:my-app-profile-cluster"},
					{Source: "aws:ecs_service:profile", Destination: "aws:ecs_task_definition:my-app-profile"},
					{Source: "aws:ecs_service:profile", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:ecs_service:profile", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:ecs_service:profile", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:ecs_task_definition:my-app-profile", Destination: "aws:ecr_image:my-app-my-app-profile"},
					{Source: "aws:ecs_task_definition:my-app-profile", Destination: "aws:iam_role:my-app-my-app-profile-ExecutionRole"},
					{Source: "aws:ecs_task_definition:my-app-profile", Destination: "aws:log_group:my-app-my-app-profile-LogGroup"},
					{Source: "aws:ecs_task_definition:my-app-profile", Destination: "aws:region:region"},
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
				},
			},
			Check: func(assert *assert.Assertions, service *EcsService) {
				assert.NotNil(service.Cluster)
				assert.Len(service.Subnets, 2)
				assert.Len(service.SecurityGroups, 1)
				assert.NotNil(service.TaskDefinition)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
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
			Name: "nil task definition",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecs_task_definition:my-app-td",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, td *EcsTaskDefinition) {
				assert.Equal(td.Name, "my-app-td")
				assert.Equal(td.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing profile",
			Existing: &EcsTaskDefinition{Name: "my-app-td", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EcsTaskDefinitionCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "td",
			}
			tt.Run(t)
		})
	}
}

func Test_EcsTaskDefinitionMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*EcsTaskDefinition]{
		{
			Name:     "only task definition",
			Resource: &EcsTaskDefinition{Name: "td"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:my-app-td",
					"aws:ecs_task_definition:td",
					"aws:iam_role:my-app-td-ExecutionRole",
					"aws:log_group:my-app-td-LogGroup",
					"aws:region:region",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecs_task_definition:td", Destination: "aws:ecr_image:my-app-td"},
					{Source: "aws:ecs_task_definition:td", Destination: "aws:iam_role:my-app-td-ExecutionRole"},
					{Source: "aws:ecs_task_definition:td", Destination: "aws:log_group:my-app-td-LogGroup"},
					{Source: "aws:ecs_task_definition:td", Destination: "aws:region:region"},
				},
			},
			Check: func(assert *assert.Assertions, td *EcsTaskDefinition) {
				assert.NotNil(td.Image)
				assert.NotNil(td.LogGroup)
				assert.NotNil(td.Region)
				assert.NotNil(td.ExecutionRole)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
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
