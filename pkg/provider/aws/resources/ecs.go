package resources

import (
	"fmt"
	"strconv"

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
		ConstructsRef           core.BaseConstructSet `yaml:"-"`
		Image                   *EcrImage
		EnvironmentVariables    EnvironmentVariables
		Cpu                     string
		Memory                  string
		LogGroup                *LogGroup
		LoggingRegion           *Region
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
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		//TODO: add support for cluster configuration
	}

	EcsService struct {
		Name                     string
		ConstructsRef            core.BaseConstructSet `yaml:"-"`
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
		TargetGroupArn core.IaCValue `yaml:"-"`
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
	td.ConstructsRef = params.Refs.Clone()
	td.Region = NewRegion()

	existingTaskDefinition := dag.GetResource(td.Id())
	if existingTaskDefinition != nil {
		return fmt.Errorf("task definition with name %s already exists", name)
	}

	logGroup, err := core.CreateResource[*LogGroup](dag, CloudwatchLogGroupCreateParams{
		AppName: params.AppName,
		Name:    fmt.Sprintf("%s-LogGroup", params.Name),
		Refs:    td.ConstructsRef.Clone(),
	})
	if err != nil {
		return err
	}
	td.LogGroup = logGroup
	subParams := map[string]any{
		"ExecutionRole": RoleCreateParams{
			AppName: params.AppName,
			Name:    fmt.Sprintf("%s-ExecutionRole", params.Name),
			Refs:    td.ConstructsRef.Clone(),
		},
		"Image": ImageCreateParams{
			AppName: params.AppName,
			Name:    params.Name,
			Refs:    td.ConstructsRef.Clone(),
		},
	}

	err = dag.CreateDependencies(td, subParams)
	if err != nil {
		return err
	}
	return nil
}

func (td *EcsTaskDefinition) Configure(params EcsTaskDefinitionConfigureParams) error {
	td.NetworkMode = ECS_NETWORK_MODE_AWSVPC
	td.RequiresCompatibilities = append(td.RequiresCompatibilities, REQUIRES_COMPATIBILITY_FARGATE)
	td.Cpu = strconv.Itoa(config.ValueOrDefault(params.Cpu, 256))
	td.Memory = strconv.Itoa(config.ValueOrDefault(params.Memory, 512))
	td.PortMappings = config.ValueOrDefault(params.PortMappings, []PortMapping{{
		ContainerPort: 3000,
		HostPort:      3000,
		Protocol:      "tcp",
	}})
	if td.EnvironmentVariables == nil {
		td.EnvironmentVariables = make(EnvironmentVariables)
	}
	for _, env := range params.EnvironmentVariables {
		td.EnvironmentVariables[env.GetName()] = core.IaCValue{Property: env.GetValue()}
	}

	return nil
}

func (td *EcsTaskDefinition) BaseConstructsRef() core.BaseConstructSet {
	return td.ConstructsRef
}

func (td *EcsTaskDefinition) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECS_TASK_DEFINITION_TYPE,
		Name:     td.Name,
	}
}

func (td *EcsTaskDefinition) DeleteCriteria() core.DeleteCriteria {
	return core.DeleteCriteria{
		RequiresNoUpstream: true,
	}
}

func (s *EcsService) Create(dag *core.ResourceGraph, params EcsServiceCreateParams) error {
	name := aws.EcsServiceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	s.Name = name
	s.ConstructsRef = params.Refs.Clone()
	s.LaunchType = params.LaunchType

	existingService := dag.GetResource(s.Id())
	if existingService != nil {
		return fmt.Errorf("service with name %s already exists", name)
	}

	subParams := map[string]any{
		"Cluster": EcsClusterCreateParams{
			AppName: params.AppName,
			Name:    fmt.Sprintf("%s-ExecutionRole", params.Name),
			Refs:    s.ConstructsRef.Clone(),
		},
		"TaskDefinition": EcsTaskDefinitionCreateParams{
			AppName: params.AppName,
			Name:    params.Name,
			Refs:    s.ConstructsRef.Clone(),
		},
	}

	if params.LaunchType == LAUNCH_TYPE_FARGATE {
		s.Subnets = make([]*Subnet, 2)
		subParams["Subnets"] = []SubnetCreateParams{
			{
				AppName: params.AppName,
				Refs:    s.ConstructsRef,
				AZ:      "0",
				Type:    params.NetworkPlacement,
			},
			{
				AppName: params.AppName,
				Refs:    s.ConstructsRef,
				AZ:      "1",
				Type:    params.NetworkPlacement,
			},
		}
		s.SecurityGroups = make([]*SecurityGroup, 1)
		subParams["SecurityGroups"] = []SecurityGroupCreateParams{
			{
				AppName: params.AppName,
				Refs:    s.ConstructsRef,
			},
		}
	}

	err := dag.CreateDependencies(s, subParams)

	if err != nil {
		return err
	}
	return nil
}

func (s *EcsService) Configure(params EcsServiceConfigureParams) error {
	s.DesiredCount = config.ValueOrDefault(params.DesiredCount, 1)
	s.ForceNewDeployment = config.ValueOrDefault(params.ForceNewDeployment, true)
	s.DeploymentCircuitBreaker = params.DeploymentCircuitBreaker
	s.AssignPublicIp = params.AssignPublicIp
	return nil
}

func (s *EcsService) BaseConstructsRef() core.BaseConstructSet {
	return s.ConstructsRef
}

func (s *EcsService) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECS_SERVICE_TYPE,
		Name:     s.Name,
	}
}

func (td *EcsService) DeleteCriteria() core.DeleteCriteria {
	return core.DeleteCriteria{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

func (c *EcsCluster) Create(dag *core.ResourceGraph, params EcsClusterCreateParams) error {
	name := aws.EcsClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	c.Name = name
	c.ConstructsRef = params.Refs.Clone()

	if existingCluster, ok := core.GetResource[*EcsCluster](dag, c.Id()); ok {
		existingCluster.ConstructsRef.AddAll(params.Refs)
	}
	dag.AddResource(c)
	return nil
}

func (c *EcsCluster) BaseConstructsRef() core.BaseConstructSet {
	return c.ConstructsRef
}

func (c *EcsCluster) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECS_CLUSTER_TYPE,
		Name:     c.Name,
	}
}

func (c *EcsCluster) DeleteCriteria() core.DeleteCriteria {
	return core.DeleteCriteria{
		RequiresNoUpstream: true,
	}
}
