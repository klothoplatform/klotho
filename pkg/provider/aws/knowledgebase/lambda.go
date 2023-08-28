package knowledgebase

import (
	"fmt"
	"path"
	"strings"

	"github.com/klothoplatform/klotho/pkg/sanitization"

	docker "github.com/klothoplatform/klotho/pkg/provider/docker/resources"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

var LambdaKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.EcrImage, *docker.DockerImage]{
		Configure: func(ecrImage *resources.EcrImage, dockerImage *docker.DockerImage, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			// configures the ecr image to build from an auto-generated dockerfile that pulls in a base image from docker hub
			dockerfile := dockerImage.Dockerfile()
			ecrImage.Dockerfile = dockerfile.Path()
			ecrImage.Context = path.Dir(dockerfile.Path())
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.LogGroup]{
		Configure: func(function *resources.LambdaFunction, logGroup *resources.LogGroup, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			logGroup.LogGroupName = fmt.Sprintf("/aws/lambda/%s", function.Name)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *kubernetes.Pod]{
		Configure: func(lambda *resources.LambdaFunction, destination *kubernetes.Pod, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			privateDnsNamespace, err := construct.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    construct.BaseConstructSetOf(destination, lambda),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(destination, privateDnsNamespace)
			policy, err := construct.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    construct.BaseConstructSetOf(destination, lambda),
			})
			if err != nil {
				return err
			}
			dag.AddDependency(policy, privateDnsNamespace)
			if lambda.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> pod %s, missing role", lambda.Id(), destination.Id())
			}
			dag.AddDependency(lambda.Role, policy)
			if err != nil {
				return err
			}
			clusterProvider := destination.Cluster
			cluster, ok := dag.GetResource(clusterProvider).(*resources.EksCluster)
			if !ok {
				return fmt.Errorf("cluster provider resource for %s, must be an eks cluster, was %T", destination.Id(), clusterProvider)
			}
			if len(lambda.Subnets) == 0 || len(lambda.SecurityGroups) == 0 {
				if cluster.Vpc == nil {
					return fmt.Errorf("cluster %s is not fully operational yet", cluster.Id())
				}
				dag.AddDependency(lambda, cluster.Vpc)
			}
			cmController, err := cluster.InstallCloudMapController(construct.BaseConstructSetOf(destination, lambda), dag)
			dag.AddDependency(destination, cmController)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *kubernetes.Deployment]{
		Configure: func(lambda *resources.LambdaFunction, destination *kubernetes.Deployment, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			privateDnsNamespace, err := construct.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    construct.BaseConstructSetOf(destination, lambda),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(destination, privateDnsNamespace)
			policy, err := construct.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    construct.BaseConstructSetOf(destination, lambda),
			})
			if err != nil {
				return err
			}
			dag.AddDependency(policy, privateDnsNamespace)
			if lambda.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> deployment %s, missing role", lambda.Id(), destination.Id())
			}
			dag.AddDependency(lambda.Role, policy)
			if err != nil {
				return err
			}
			clusterProvider := destination.Cluster
			cluster, ok := dag.GetResource(clusterProvider).(*resources.EksCluster)
			if !ok {
				return fmt.Errorf("cluster provider resource for %s, must be an eks cluster, was %T", destination.Id(), clusterProvider)
			}
			if len(lambda.Subnets) == 0 || len(lambda.SecurityGroups) == 0 {
				if cluster.Vpc == nil {
					return fmt.Errorf("cluster %s is not fully operational yet", cluster.Id())
				}
				dag.AddDependency(lambda, cluster.Vpc)
			}
			cmController, err := cluster.InstallCloudMapController(construct.BaseConstructSetOf(destination, lambda), dag)
			dag.AddDependency(destination, cmController)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.EfsAccessPoint]{
		Configure: func(lambda *resources.LambdaFunction, accessPoint *resources.EfsAccessPoint, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if lambda.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> efs access point %s, missing role", lambda.Id(), accessPoint.Id())
			}
			dag.AddDependency(lambda.Role, accessPoint)
			mountPathEnvVarName := sanitization.EnvVarKeySanitizer.Apply(strings.ToUpper(fmt.Sprintf("%s_MOUNT_PATH", accessPoint.FileSystem.Id().Name)))
			if lambda.EnvironmentVariables == nil {
				lambda.EnvironmentVariables = map[string]construct.IaCValue{}
			}
			lambda.EnvironmentVariables[mountPathEnvVarName] = construct.IaCValue{ResourceId: accessPoint.Id(), Property: resources.EFS_MOUNT_PATH_IAC_VALUE}
			lambda.EfsAccessPoint = accessPoint
			efs := accessPoint.FileSystem
			mountTarget, _ := construct.GetSingleUpstreamResourceOfType[*resources.EfsMountTarget](dag, efs)
			if mountTarget == nil {
				return fmt.Errorf("efs file system %s is not fully operational yet", efs.Id())
			}
			efsVpc, err := construct.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, mountTarget)
			if err != nil {
				return err
			}
			lambdaVpc, _ := construct.GetSingleDownstreamResourceOfType[*resources.Vpc](dag, lambda)

			if lambdaVpc != nil && efsVpc != nil && lambdaVpc != efsVpc {
				return fmt.Errorf("lambda %s and efs access point %s must be in the same vpc", lambda.Id(), accessPoint.Id())
			}

			if lambdaVpc == nil {
				dag.AddDependencyWithData(lambda, efsVpc, data)
			}

			return nil
		},
	},
)
