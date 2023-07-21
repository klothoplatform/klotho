package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	ECS_TASK_DEFINITION_TYPE = "ecs_task_definition"
	ECS_SERVICE_TYPE         = "ecs_service"
	ECS_CLUSTER_TYPE         = "ecs_cluster"

	ECS_NETWORK_MODE_AWSVPC = "awsvpc"
	ECS_NETWORK_MODE_HOST   = "host"

	LAUNCH_TYPE_FARGATE            = "FARGATE"
	REQUIRES_COMPATIBILITY_FARGATE = "FARGATE"
)

type (
	EcsTaskDefinition struct {
		Name                    string
		ConstructRefs           core.BaseConstructSet `yaml:"-"`
		Image                   *EcrImage
		EnvironmentVariables    map[string]core.IaCValue
		Cpu                     string
		Memory                  string
		LogGroup                *LogGroup
		ExecutionRole           *IamRole
		Region                  *Region
		NetworkMode             string
		PortMappings            []PortMapping
		RequiresCompatibilities []string
	}

	PortMapping struct {
		ContainerPort int
		HostPort      int
		Protocol      string
	}

	EcsCluster struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		//TODO: add support for cluster configuration
	}

	EcsService struct {
		Name                     string
		ConstructRefs            core.BaseConstructSet `yaml:"-"`
		AssignPublicIp           bool
		Cluster                  *EcsCluster
		DeploymentCircuitBreaker *EcsServiceDeploymentCircuitBreaker
		DesiredCount             int
		ForceNewDeployment       bool
		LaunchType               string
		LoadBalancers            []EcsServiceLoadBalancerConfig
		SecurityGroups           []*SecurityGroup
		Subnets                  []*Subnet
		TaskDefinition           *EcsTaskDefinition
	}

	EcsServiceDeploymentCircuitBreaker struct {
		Enable   bool
		Rollback bool
	}

	EcsServiceLoadBalancerConfig struct {
		TargetGroupArn core.IaCValue
		ContainerName  string
		ContainerPort  int
	}

	EcsServiceCreateParams struct {
		AppName          string
		Refs             core.BaseConstructSet `yaml:"-"`
		Name             string
		LaunchType       string
		NetworkPlacement string
	}

	EcsServiceConfigureParams struct {
		DesiredCount             int
		ForceNewDeployment       bool
		DeploymentCircuitBreaker *EcsServiceDeploymentCircuitBreaker
		AssignPublicIp           bool
	}

	EcsTaskDefinitionCreateParams struct {
		AppName string
		Refs    core.BaseConstructSet
		Name    string
	}

	EcsTaskDefinitionConfigureParams struct {
		Cpu                  int
		Memory               int
		EnvironmentVariables core.EnvironmentVariables
		PortMappings         []PortMapping
	}

	EcsClusterCreateParams struct {
		AppName string
		Refs    core.BaseConstructSet
		Name    string
	}
)

func (td *EcsTaskDefinition) Create(dag *core.ResourceGraph, params EcsTaskDefinitionCreateParams) error {

	name := aws.EcsTaskDefinitionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	td.Name = name
	td.ConstructRefs = params.Refs.Clone()

	existingTaskDefinition := dag.GetResource(td.Id())
	if existingTaskDefinition != nil {
		return fmt.Errorf("task definition with name %s already exists", name)
	}

	dag.AddResource(td)
	return nil
}

func (td *EcsTaskDefinition) BaseConstructRefs() core.BaseConstructSet {
	return td.ConstructRefs
}

func (td *EcsTaskDefinition) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECS_TASK_DEFINITION_TYPE,
		Name:     td.Name,
	}
}

func (td *EcsTaskDefinition) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (s *EcsService) Create(dag *core.ResourceGraph, params EcsServiceCreateParams) error {
	name := aws.EcsServiceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	s.Name = name
	s.ConstructRefs = params.Refs.Clone()
	s.LaunchType = params.LaunchType

	existingService := dag.GetResource(s.Id())
	if existingService != nil {
		return fmt.Errorf("service with name %s already exists", name)
	}
	dag.AddResource(s)
	return nil
}

func (s *EcsService) Configure(params EcsServiceConfigureParams) error {
	s.DesiredCount = config.ValueOrDefault(params.DesiredCount, 1)
	s.ForceNewDeployment = config.ValueOrDefault(params.ForceNewDeployment, true)
	s.DeploymentCircuitBreaker = params.DeploymentCircuitBreaker
	s.AssignPublicIp = params.AssignPublicIp
	return nil
}

func (s *EcsService) BaseConstructRefs() core.BaseConstructSet {
	return s.ConstructRefs
}

func (s *EcsService) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECS_SERVICE_TYPE,
		Name:     s.Name,
	}
}

func (td *EcsService) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

func (c *EcsCluster) Create(dag *core.ResourceGraph, params EcsClusterCreateParams) error {
	name := aws.EcsClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	c.Name = name
	c.ConstructRefs = params.Refs.Clone()

	if existingCluster, ok := core.GetResource[*EcsCluster](dag, c.Id()); ok {
		existingCluster.ConstructRefs.AddAll(params.Refs)
	}
	dag.AddResource(c)
	return nil
}

func (c *EcsCluster) BaseConstructRefs() core.BaseConstructSet {
	return c.ConstructRefs
}

func (c *EcsCluster) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECS_CLUSTER_TYPE,
		Name:     c.Name,
	}
}

func (c *EcsCluster) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
