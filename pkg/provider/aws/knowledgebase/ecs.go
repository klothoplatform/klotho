package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var EcsKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.EcsCluster]{
		Expand: func(source *resources.EcsService, destination *resources.EcsCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			source.Cluster = destination
			dag.AddDependency(source, destination)
			return nil
		},
		Configure: func(service *resources.EcsService, cluster *resources.EcsCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.Cluster != nil && service.Cluster != cluster {
				return fmt.Errorf("cannot configure edge %s -> %s, service already tied to cluster %s", service.Id(), cluster.Id(), service.Cluster.Id())
			}
			service.Cluster = cluster
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.EcsTaskDefinition]{
		Configure: func(service *resources.EcsService, taskDefinition *resources.EcsTaskDefinition, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.TaskDefinition != nil && service.TaskDefinition != taskDefinition {
				return fmt.Errorf("cannot configure edge %s -> %s, service already tied to task definition %s", service.Id(), taskDefinition.Id(), service.TaskDefinition.Id())
			}
			service.TaskDefinition = taskDefinition
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsTaskDefinition, *resources.EcrImage]{},
	knowledgebase.EdgeBuilder[*resources.EcsTaskDefinition, *resources.Region]{},
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
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.DynamodbTable]{
		Expand: func(service *resources.EcsService, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(service.TaskDefinition.ExecutionRole, table)
			return nil
		},
		Configure: func(service *resources.EcsService, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				service.TaskDefinition.EnvironmentVariables[env.GetName()] = &resources.AwsResourceValue{ResourceVal: table, PropertyVal: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.ElasticacheCluster]{
		Configure: func(service *resources.EcsService, cluster *resources.ElasticacheCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				service.TaskDefinition.EnvironmentVariables[env.GetName()] = &resources.AwsResourceValue{ResourceVal: cluster, PropertyVal: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.S3Bucket]{
		Expand: func(service *resources.EcsService, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(service.TaskDefinition.ExecutionRole, bucket)
			return nil
		},
		Configure: func(service *resources.EcsService, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				service.TaskDefinition.EnvironmentVariables[env.GetName()] = &resources.AwsResourceValue{ResourceVal: bucket, PropertyVal: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.Secret]{
		Expand: func(service *resources.EcsService, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(service.TaskDefinition.ExecutionRole, secret)
			return nil
		},
		Configure: func(service *resources.EcsService, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				service.TaskDefinition.EnvironmentVariables[env.GetName()] = &resources.AwsResourceValue{ResourceVal: secret, PropertyVal: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.RdsInstance]{
		Configure: func(service *resources.EcsService, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				service.TaskDefinition.EnvironmentVariables[env.GetName()] = &resources.AwsResourceValue{ResourceVal: instance, PropertyVal: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.RdsProxy]{
		Configure: func(service *resources.EcsService, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				service.TaskDefinition.EnvironmentVariables[env.GetName()] = &resources.AwsResourceValue{ResourceVal: proxy, PropertyVal: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.TargetGroup]{
		ReverseDirection: true,
		Expand: func(source *resources.EcsService, destination *resources.TargetGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if data.Source.Id().Type != resources.API_GATEWAY_INTEGRATION_TYPE {
				dst := data.Destination.Id().Name
				if destination.Name == "" || destination == nil {
					var err error
					destination, err = core.CreateResource[*resources.TargetGroup](dag, resources.TargetGroupCreateParams{
						AppName: data.AppName,
						Refs:    core.BaseConstructSetOf(data.Source, data.Destination),
						Name:    dst,
					})
					if err != nil {
						return err
					}
				}
				dag.AddDependency(source, destination)
				if len(source.Subnets) > 0 {
					destination.Vpc = source.Subnets[0].Vpc
				}
			}
			return nil
		},
		Configure: func(service *resources.EcsService, tg *resources.TargetGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(service.TaskDefinition.PortMappings) != 1 {
				return fmt.Errorf("cannot configure edge %s -> %s, the service's task definition does not have exactly one port mapping", service.Id(), tg.Id())
			}

			service.LoadBalancers = []resources.EcsServiceLoadBalancerConfig{
				{
					TargetGroupArn: &resources.AwsResourceValue{ResourceVal: tg, PropertyVal: resources.ARN_IAC_VALUE},
					ContainerName:  service.Name,
					ContainerPort:  service.TaskDefinition.PortMappings[0].ContainerPort,
				},
			}
			tg.Port = 3000
			tg.Protocol = "TCP"
			tg.TargetType = "ip"
			return nil
		},
	},
)
