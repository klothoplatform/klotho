package knowledgebase

import (
	"fmt"
	"path"

	docker "github.com/klothoplatform/klotho/pkg/provider/docker/resources"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

var LambdaKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.LambdaPermission, *resources.LambdaFunction]{
		Configure: func(permission *resources.LambdaPermission, function *resources.LambdaFunction, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if permission.Function != nil && permission.Function != function {
				return fmt.Errorf("cannot configure edge %s -> %s, permission already tied to function %s", permission.Id(), function.Id(), permission.Function.Id())
			}
			permission.Function = function
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.Subnet]{
		Configure: func(lambda *resources.LambdaFunction, subnet *resources.Subnet, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if lambda.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> subnet %s, missing role", lambda.Id(), subnet.Id())
			}
			lambda.Role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.RdsInstance]{
		Configure: func(lambda *resources.LambdaFunction, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if instance.SubnetGroup == nil || len(instance.SecurityGroups) == 0 {
				return fmt.Errorf("rds instance %s is not fully operational yet", instance.Id())
			}
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = instance.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = instance.SecurityGroups
			}
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: instance.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.RdsProxy]{
		Configure: func(lambda *resources.LambdaFunction, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(proxy.Subnets) == 0 || len(proxy.SecurityGroups) == 0 {
				return fmt.Errorf("proxy %s is not fully operational yet", proxy.Id())
			}
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = proxy.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = proxy.SecurityGroups
			}
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: proxy.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.EcrImage]{},
	knowledgebase.EdgeBuilder[*resources.EcrImage, *docker.DockerImage]{
		Configure: func(ecrImage *resources.EcrImage, dockerImage *docker.DockerImage, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			// configures the ecr image to build from the docker image
			dockerImage.CreatesDockerfile = true
			dockerfile := dockerImage.Dockerfile()
			ecrImage.Dockerfile = dockerfile.Path()
			ecrImage.Context = path.Dir(dockerfile.Path())
			return nil
		},
		Reuse: knowledgebase.Upstream,
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.LogGroup]{
		Configure: func(function *resources.LambdaFunction, logGroup *resources.LogGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			logGroup.LogGroupName = fmt.Sprintf("/aws/lambda/%s", function.Name)
			logGroup.RetentionInDays = 5
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.DynamodbTable]{
		Configure: func(lambda *resources.LambdaFunction, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if lambda.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> dynamo table %s, missing role", lambda.Id(), table.Id())
			}
			dag.AddDependency(lambda.Role, table)
			// TODO: remove
			if lambda.EnvironmentVariables == nil {
				lambda.EnvironmentVariables = map[string]core.IaCValue{}
			}
			lambda.EnvironmentVariables["TableName"] = core.IaCValue{ResourceId: table.Id(), Property: string(core.KV_DYNAMODB_TABLE_NAME)}
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: table.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.ElasticacheCluster]{
		Configure: func(lambda *resources.LambdaFunction, cluster *resources.ElasticacheCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if cluster.SubnetGroup == nil || len(cluster.SecurityGroups) == 0 {
				return fmt.Errorf("elasticache cluster %s is not fully operational yet", cluster.Id())
			}
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = cluster.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = cluster.SecurityGroups
			}
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: cluster.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.S3Bucket]{
		Configure: func(lambda *resources.LambdaFunction, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if lambda.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> s3 bucket %s, missing role", lambda.Id(), bucket.Id())
			}
			dag.AddDependency(lambda.Role, bucket)
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: bucket.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.Secret]{
		Configure: func(lambda *resources.LambdaFunction, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if lambda.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> secret %s, missing role", lambda.Id(), secret.Id())
			}
			dag.AddDependency(lambda.Role, secret)
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{ResourceId: secret.Id(), Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.LambdaFunction]{
		Configure: func(source, destination *resources.LambdaFunction, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Refs:    core.BaseConstructSetOf(source, destination),
				Name:    fmt.Sprintf("%s-InvocationPolicy", destination.Id().Name),
			})
			dag.AddDependency(policy, destination)
			if err != nil {
				return err
			}
			if source.Role == nil {
				return fmt.Errorf("cannot configure lambda %s -> lambda %s, missing role", source.Id(), destination.Id())
			}
			attachment := &resources.RolePolicyAttachment{
				Name:          fmt.Sprintf("%s-%s", source.Role.Name, policy.Name),
				ConstructRefs: source.ConstructRefs.CloneWith(destination.ConstructRefs),
				Policy:        policy,
				Role:          source.Role,
			}
			dag.AddDependenciesReflect(attachment)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *kubernetes.Pod]{
		Configure: func(lambda *resources.LambdaFunction, destination *kubernetes.Pod, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			privateDnsNamespace, err := core.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    core.BaseConstructSetOf(destination, lambda),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(destination, privateDnsNamespace)
			policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    core.BaseConstructSetOf(destination, lambda),
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
			cmController, err := cluster.InstallCloudMapController(core.BaseConstructSetOf(destination, lambda), dag)
			dag.AddDependency(destination, cmController)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *kubernetes.Deployment]{
		Configure: func(lambda *resources.LambdaFunction, destination *kubernetes.Deployment, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			privateDnsNamespace, err := core.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    core.BaseConstructSetOf(destination, lambda),
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(destination, privateDnsNamespace)
			policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    core.BaseConstructSetOf(destination, lambda),
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
			cmController, err := cluster.InstallCloudMapController(core.BaseConstructSetOf(destination, lambda), dag)
			dag.AddDependency(destination, cmController)
			return err
		},
	},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.LambdaFunction]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.LambdaFunction]{},
)
