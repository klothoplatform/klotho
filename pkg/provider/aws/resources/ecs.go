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
		EnvironmentVariables    map[string]*AwsResourceValue
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
		TargetGroupArn *AwsResourceValue
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

	existingTaskDefinition := dag.GetResource(td.Id())
	if existingTaskDefinition != nil {
		return fmt.Errorf("task definition with name %s already exists", name)
	}

	dag.AddResource(td)
	return nil
}

func (td *EcsTaskDefinition) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if td.Region == nil {
		td.Region = NewRegion()
		dag.AddDependenciesReflect(td)
	}

	if td.LogGroup == nil {
		logGroups := core.GetDownstreamResourcesOfType[*LogGroup](dag, td)
		if len(logGroups) == 0 {
			err := dag.CreateDependencies(td, map[string]any{
				"LogGroup": CloudwatchLogGroupCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-LogGroup", td.Name),
					Refs:    core.BaseConstructSetOf(td),
				},
			})
			if err != nil {
				return err
			}
		} else if len(logGroups) == 1 {
			td.LogGroup = logGroups[0]
			dag.AddDependenciesReflect(td)
		} else {
			return fmt.Errorf("task definition %s has more than one log group downstream", td.Id())
		}
	}
	if td.ExecutionRole == nil {
		roles := core.GetDownstreamResourcesOfType[*IamRole](dag, td)
		if len(roles) == 0 {
			err := dag.CreateDependencies(td, map[string]any{
				"ExecutionRole": RoleCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-ExecutionRole", td.Name),
					Refs:    core.BaseConstructSetOf(td),
				},
			})
			if err != nil {
				return err
			}
		} else if len(roles) == 1 {
			td.ExecutionRole = roles[0]
			dag.AddDependenciesReflect(td)
		} else {
			return fmt.Errorf("task definition %s has more than one execution role downstream", td.Id())
		}
	}

	if td.Image == nil {
		images := core.GetDownstreamResourcesOfType[*EcrImage](dag, td)
		if len(images) == 0 {
			err := dag.CreateDependencies(td, map[string]any{
				"Image": ImageCreateParams{
					AppName: appName,
					Name:    td.Name,
					Refs:    core.BaseConstructSetOf(td),
				},
			})
			if err != nil {
				return err
			}
		} else if len(images) == 1 {
			td.Image = images[0]
			dag.AddDependenciesReflect(td)
		} else {
			return fmt.Errorf("task definition %s has more than one image downstream", td.Id())
		}
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
		td.EnvironmentVariables = make(map[string]*AwsResourceValue)
	}
	for _, env := range params.EnvironmentVariables {
		td.EnvironmentVariables[env.GetName()] = &AwsResourceValue{PropertyVal: env.GetValue()}
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

func (td *EcsTaskDefinition) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
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
	dag.AddResource(s)
	return nil
}

func (service *EcsService) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if service.Cluster == nil {
		clusters := core.GetDownstreamResourcesOfType[*EcsCluster](dag, service)
		if len(clusters) == 0 {
			err := dag.CreateDependencies(service, map[string]any{
				"Cluster": EcsClusterCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-cluster", service.Name),
					Refs:    core.BaseConstructSetOf(service),
				},
			})
			if err != nil {
				return err
			}
		} else if len(clusters) > 1 {
			return fmt.Errorf("service %s has more than one cluster downstream", service.Id())
		} else {
			service.Cluster = clusters[0]
			dag.AddDependenciesReflect(service)
		}
	}

	if service.TaskDefinition == nil {
		taskDefinitions := core.GetDownstreamResourcesOfType[*EcsTaskDefinition](dag, service)
		if len(taskDefinitions) == 0 {
			err := dag.CreateDependencies(service, map[string]any{
				"TaskDefinition": EcsTaskDefinitionCreateParams{
					AppName: appName,
					Name:    service.Name,
					Refs:    core.BaseConstructSetOf(service),
				},
			})
			if err != nil {
				return err
			}
		} else if len(taskDefinitions) > 1 {
			return fmt.Errorf("service %s has more than one task definition downstream", service.Id())
		} else {
			service.TaskDefinition = taskDefinitions[0]
			dag.AddDependenciesReflect(service)
		}
	}

	if service.LaunchType == LAUNCH_TYPE_FARGATE {
		if service.Subnets == nil {
			subnets, err := getSubnetsOperational(dag, service, appName)
			if err != nil {
				return err
			}
			for _, subnet := range subnets {
				if subnet.Type == PrivateSubnet {
					service.Subnets = append(service.Subnets, subnet)
				}
			}
		}

		if service.SecurityGroups == nil {
			sgs, err := getSecurityGroupsOperational(dag, service, appName)
			if err != nil {
				return err
			}
			service.SecurityGroups = sgs
		}
		dag.AddDependenciesReflect(service)
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

func (c *EcsCluster) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
