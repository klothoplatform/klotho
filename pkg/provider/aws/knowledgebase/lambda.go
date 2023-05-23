package knowledgebase

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var LambdaKB = knowledgebase.EdgeKB{
	knowledgebase.NewEdge[*resources.LambdaPermission, *resources.LambdaFunction](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			permission := source.(*resources.LambdaPermission)
			function := dest.(*resources.LambdaFunction)
			if permission.Function != nil && permission.Function != function {
				return fmt.Errorf("cannot configure edge %s -> %s, permission already tied to function %s", permission.Id(), function.Id(), permission.Function.Id())
			}
			permission.Function = function
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.LambdaFunction, *resources.Subnet](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			lambda.Role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"})
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.Vpc{})},
	},
	knowledgebase.NewEdge[*resources.LambdaFunction, *resources.SecurityGroup](): {},
	knowledgebase.NewEdge[*resources.LambdaFunction, *resources.RdsInstance](): {
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			instance := dest.(*resources.RdsInstance)
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = instance.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = instance.SecurityGroups
			}
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			instance := dest.(*resources.RdsInstance)
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable to expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id(), instance.Id(), lambda.Id())
			}
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), core.DedupeAnnotationKeys(append(lambda.ConstructsRef, instance.ConstructsRef...)), instance.GetConnectionPolicyDocument())
			lambda.Role.InlinePolicies = append(lambda.Role.InlinePolicies, inlinePol)

			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: instance, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.LambdaFunction, *resources.RdsProxy](): {
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
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
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			proxy := dest.(*resources.RdsProxy)
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable to expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id().String(), proxy.Id().String(), lambda.Id().String())
			}

			upstreamResources := dag.GetUpstreamResources(proxy)
			for _, res := range upstreamResources {
				if tg, ok := res.(*resources.RdsProxyTargetGroup); ok {
					for _, tgUpstream := range dag.GetDownstreamResources(tg) {
						if instance, ok := tgUpstream.(*resources.RdsInstance); ok {
							inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name),
								core.DedupeAnnotationKeys(append(lambda.ConstructsRef, instance.ConstructsRef...)), instance.GetConnectionPolicyDocument())
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
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	knowledgebase.NewEdge[*resources.LambdaFunction, *resources.EcrImage](): {},
	knowledgebase.NewEdge[*resources.LambdaFunction, *resources.LogGroup](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			logGroup := dest.(*resources.LogGroup)
			function := source.(*resources.LambdaFunction)
			logGroup.LogGroupName = fmt.Sprintf("/aws/lambda/%s", function.Name)
			logGroup.RetentionInDays = 5
			return nil
		},
	},
}
