package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/sanitization"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var EcsKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.EcsTaskDefinition, *resources.LogGroup]{
		Configure: func(taskDef *resources.EcsTaskDefinition, lg *resources.LogGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if taskDef.LogGroup != lg {
				return nil // this log group doesn't belong to this task definition and is configured elsewhere
			}

			// configure the task definition's log group
			lg.LogGroupName = fmt.Sprintf("/aws/ecs/%s", taskDef.Name)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.EcsService]{
		DeploymentOrderReversed: true,
		Configure: func(tg *resources.TargetGroup, service *resources.EcsService, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.TaskDefinition == nil {
				return fmt.Errorf("cannot configure edge %s -> %s, missing task definition", service.Id(), tg.Id())
			} else if len(service.TaskDefinition.PortMappings) != 1 {
				return fmt.Errorf("cannot configure edge %s -> %s, the service's task definition does not have exactly one port mapping, it has %d", service.Id(), tg.Id(), len(service.TaskDefinition.PortMappings))
			}
			service.LoadBalancers = []resources.EcsServiceLoadBalancerConfig{
				{
					TargetGroupArn: core.IaCValue{ResourceId: tg.Id(), Property: resources.ARN_IAC_VALUE},
					ContainerName:  service.Name,
					ContainerPort:  service.TaskDefinition.PortMappings[0].ContainerPort,
				},
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.EfsAccessPoint]{
		Configure: func(service *resources.EcsService, accessPoint *resources.EfsAccessPoint, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.TaskDefinition == nil {
				return fmt.Errorf("cannot configure service %s -> efs access point %s, missing task definition", service.Id(), accessPoint.Id())
			}
			taskDef := service.TaskDefinition
			if taskDef.ExecutionRole == nil {
				return fmt.Errorf("cannot configure service %s -> efs access point %s, missing execution role", service.Id(), accessPoint.Id())
			}

			efs := accessPoint.FileSystem
			mountTarget, _ := core.GetSingleUpstreamResourceOfType[*resources.EfsMountTarget](dag, efs)
			if mountTarget == nil {
				return fmt.Errorf("efs file system %s is not fully operational yet", efs.Id())
			}
			efsVpc, err := core.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, mountTarget)
			if err != nil {
				return err
			}
			serviceVpc, _ := core.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, service)

			if serviceVpc != nil && efsVpc != nil && serviceVpc != efsVpc {
				return fmt.Errorf("service %s and efs access point %s must be in the same vpc", service.Id(), accessPoint.Id())
			}

			dag.AddDependency(taskDef.ExecutionRole, accessPoint)
			mountPathEnvVarName := sanitization.EnvVarKeySanitizer.Apply(strings.ToUpper(fmt.Sprintf("%s_MOUNT_PATH", accessPoint.FileSystem.Id().Name)))
			if taskDef.EnvironmentVariables == nil {
				taskDef.EnvironmentVariables = map[string]core.IaCValue{}
			}
			taskDef.EnvironmentVariables[mountPathEnvVarName] = core.IaCValue{ResourceId: accessPoint.Id(), Property: resources.EFS_MOUNT_PATH_IAC_VALUE}

			isMissingVolume := true
			for _, volume := range taskDef.EfsVolumes {
				if volume.FileSystemId.ResourceId == accessPoint.FileSystem.Id() {
					isMissingVolume = false
					break
				}
			}
			if isMissingVolume {
				volume := &resources.EcsEfsVolume{
					FileSystemId: core.IaCValue{ResourceId: accessPoint.FileSystem.Id(), Property: resources.ID_IAC_VALUE},
					AuthorizationConfig: &resources.EcsEfsVolumeAuthorizationConfig{
						AccessPointId: core.IaCValue{ResourceId: accessPoint.Id(), Property: resources.ID_IAC_VALUE},
						Iam:           "ENABLED",
					},
					TransitEncryption: "ENABLED",
				}
				taskDef.EfsVolumes = append(taskDef.EfsVolumes, volume)
			}

			if serviceVpc == nil {
				dag.AddDependencyWithData(service, efsVpc, data)
			}

			return nil
		},
	},
)
