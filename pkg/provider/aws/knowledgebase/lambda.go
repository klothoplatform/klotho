package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
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
			lambda.Role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"})
			return nil
		},
		ValidDestinations: []core.Resource{&resources.Vpc{}},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.RdsInstance]{
		Expand: func(lambda *resources.LambdaFunction, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = instance.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = instance.SecurityGroups
			}
			return nil
		},
		Configure: func(lambda *resources.LambdaFunction, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable to expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id(), instance.Id(), lambda.Id())
			}
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), lambda.ConstructsRef.CloneWith(instance.ConstructsRef), instance.GetConnectionPolicyDocument())
			lambda.Role.InlinePolicies = append(lambda.Role.InlinePolicies, inlinePol)

			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: instance, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.RdsProxy]{
		Expand: func(lambda *resources.LambdaFunction, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			instance := data.Destination.(*resources.RdsInstance)
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = instance.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = instance.SecurityGroups
			}
			dag.AddDependenciesReflect(lambda)
			return nil
		},
		Configure: func(lambda *resources.LambdaFunction, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable to expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id().String(), proxy.Id().String(), lambda.Id().String())
			}

			upstreamResources := dag.GetUpstreamResources(proxy)
			for _, res := range upstreamResources {
				if tg, ok := res.(*resources.RdsProxyTargetGroup); ok {
					for _, tgUpstream := range dag.GetDownstreamResources(tg) {
						if instance, ok := tgUpstream.(*resources.RdsInstance); ok {
							inlinePol := resources.NewIamInlinePolicy(
								fmt.Sprintf("%s-connectionpolicy", instance.Name),
								lambda.ConstructsRef.CloneWith(instance.ConstructsRef),
								instance.GetConnectionPolicyDocument(),
							)
							lambda.Role.InlinePolicies = append(lambda.Role.InlinePolicies, inlinePol)
							dag.AddDependency(lambda.Role, instance)
						}
					}
				}
			}
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: proxy, Property: env.GetValue()}
			}
			return nil
		},
		ValidDestinations: []core.Resource{&resources.RdsInstance{}},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.EcrImage]{},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.LogGroup]{
		Configure: func(function *resources.LambdaFunction, logGroup *resources.LogGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			logGroup.LogGroupName = fmt.Sprintf("/aws/lambda/%s", function.Name)
			logGroup.RetentionInDays = 5
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.DynamodbTable]{
		Expand: func(lambda *resources.LambdaFunction, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(lambda.Role, table)
			return nil
		},
		Configure: func(lambda *resources.LambdaFunction, table *resources.DynamodbTable, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: table, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.ElasticacheCluster]{
		Expand: func(lambda *resources.LambdaFunction, cluster *resources.ElasticacheCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = cluster.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = cluster.SecurityGroups
			}
			dag.AddDependenciesReflect(lambda)
			return nil
		},
		Configure: func(lambda *resources.LambdaFunction, cluster *resources.ElasticacheCluster, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: cluster, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.S3Bucket]{
		Expand: func(lambda *resources.LambdaFunction, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(lambda.Role, bucket)
			return nil
		},
		Configure: func(lambda *resources.LambdaFunction, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: bucket, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.Secret]{
		Expand: func(lambda *resources.LambdaFunction, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(lambda.Role, secret)
			return nil
		},
		Configure: func(lambda *resources.LambdaFunction, secret *resources.Secret, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: secret, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.LambdaFunction]{
		Expand: func(source, destination *resources.LambdaFunction, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Refs:    source.ConstructsRef.CloneWith(destination.ConstructsRef),
				Name:    fmt.Sprintf("%s-InvocationPolicy", destination.Id().Name),
			})
			dag.AddDependency(policy, destination)
			if err != nil {
				return err
			}
			attachment := &resources.RolePolicyAttachment{
				Name:          fmt.Sprintf("%s-%s", source.Role.Name, policy.Name),
				ConstructsRef: source.ConstructsRef.CloneWith(destination.ConstructsRef),
				Policy:        policy,
				Role:          source.Role,
			}
			dag.AddDependenciesReflect(attachment)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *kubernetes.HelmChart]{
		Expand: func(lambda *resources.LambdaFunction, destination *kubernetes.HelmChart, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = make([]*resources.Subnet, 2)
				subparams := map[string]any{
					"Subnets": []resources.SubnetCreateParams{
						{
							AppName: data.AppName,
							Refs:    lambda.ConstructsRef,
							AZ:      "0",
							Type:    resources.PrivateSubnet,
						},
						{
							AppName: data.AppName,
							Refs:    lambda.ConstructsRef,
							AZ:      "1",
							Type:    resources.PrivateSubnet,
						},
					},
				}
				if len(lambda.SecurityGroups) == 0 {
					lambda.SecurityGroups = make([]*resources.SecurityGroup, 1)
					subparams["SecurityGroups"] = []resources.SecurityGroupCreateParams{
						{
							AppName: data.AppName,
							Refs:    lambda.ConstructsRef,
						},
					}
				}
				err := dag.CreateDependencies(lambda, subparams)
				if err != nil {
					return err
				}
			}
			refs := core.AnnotationKeySet{}
			refs.AddAll(lambda.ConstructsRef)
			refs.AddAll(destination.ConstructRefs)
			privateDnsNamespace, err := core.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
				Refs:    refs,
				AppName: data.AppName,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(destination, privateDnsNamespace)
			policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    "servicediscovery",
				Refs:    refs,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(policy, privateDnsNamespace)
			dag.AddDependency(lambda.Role, policy)
			if err != nil {
				return err
			}
			clusterProvider := destination.ClustersProvider.Resource
			cluster, ok := clusterProvider.(*resources.EksCluster)
			if !ok {
				return fmt.Errorf("cluster provider resource for %s, must be an eks cluster, was %T", destination.Id(), clusterProvider)
			}
			cmController, err := cluster.InstallCloudMapController(refs, dag)
			dag.AddDependency(destination, cmController)
			return err
		},
	},
)
