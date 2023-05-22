package knowledgebase

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var AwsKB = knowledgebase.EdgeKB{
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.RdsInstance{})}: knowledgebase.EdgeDetails{
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
				return fmt.Errorf("unable Destination expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id().String(), instance.Id().String(), lambda.Id().String())
			}
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), core.DedupeAnnotationKeys(append(lambda.ConstructsRef, instance.ConstructsRef...)), instance.GetConnectionPolicyDocument())
			lambda.Role.InlinePolicies = append(lambda.Role.InlinePolicies, inlinePol)

			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: instance, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.RdsProxy{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			proxy := dest.(*resources.RdsProxy)
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = proxy.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = proxy.SecurityGroups
			}
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			proxy := dest.(*resources.RdsProxy)
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable Destination expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
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
						}
					}
				}
			}

			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: proxy, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), Destination: reflect.TypeOf(&resources.RdsInstance{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			instance := dest.(*resources.RdsInstance)
			targetGroup := source.(*resources.RdsProxyTargetGroup)
			if targetGroup.Name == "" {
				proxyTargetGroup, err := core.CreateResource[*resources.RdsProxyTargetGroup](dag, resources.RdsProxyTargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    instance.ConstructsRef,
					Name:    instance.Name,
				})
				if err != nil {
					return err
				}
				dag.AddDependency(proxyTargetGroup, instance)
			}
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			instance := dest.(*resources.RdsInstance)
			targetGroup := source.(*resources.RdsProxyTargetGroup)

			if targetGroup.RdsInstance == nil {
				targetGroup.RdsInstance = instance
			} else if targetGroup.RdsInstance != instance {
				return fmt.Errorf("target group, %s, has edge Destination instance, %s, but internal property is set Destination a different instance", targetGroup.Name, instance.Name)
			}

			secret := targetGroup.RdsProxy.Auths[0].SecretArn.Resource.(*resources.Secret)
			for _, res := range dag.GetDownstreamResources(secret) {
				if secretVersion, ok := res.(*resources.SecretVersion); ok {
					secretVersion.Path = instance.CredentialsFile.Path()
				}
			}
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), Destination: reflect.TypeOf(&resources.RdsProxy{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			proxy := dest.(*resources.RdsProxy)
			targetGroup := source.(*resources.RdsProxyTargetGroup)
			destination := data.Destination.(*resources.RdsInstance)
			if proxy.Name == "" {
				var err error
				proxy, err = core.CreateResource[*resources.RdsProxy](dag, resources.RdsProxyCreateParams{
					AppName: data.AppName,
					Refs:    targetGroup.ConstructsRef,
					Name:    destination.Name,
				})
				if err != nil {
					return err
				}
			}

			if targetGroup.Name == "" {
				proxyTargetGroup, err := core.CreateResource[*resources.RdsProxyTargetGroup](dag, resources.RdsProxyTargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    proxy.ConstructsRef,
					Name:    destination.Name,
				})
				if err != nil {
					return err
				}
				dag.AddDependency(proxyTargetGroup, proxy)
			}
			secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
				AppName: data.AppName,
				Refs:    core.DedupeAnnotationKeys(append(source.KlothoConstructRef(), dest.KlothoConstructRef()...)),
				Name:    proxy.Name,
			})
			if err != nil {
				return err
			}
			proxy.Auths = append(proxy.Auths, &resources.ProxyAuth{
				AuthScheme: "SECRETS",
				IamAuth:    "DISABLED",
				SecretArn:  core.IaCValue{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE},
			})
			dag.AddDependency(proxy, secretVersion)

			secretPolicy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    fmt.Sprintf("%s-ormsecretpolicy", proxy.Name),
				Refs:    proxy.ConstructsRef,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(secretPolicy, secretVersion)
			dag.AddDependency(proxy.Role, secretPolicy)
			dag.AddDependenciesReflect(proxy)
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			proxy := dest.(*resources.RdsProxy)
			targetGroup := source.(*resources.RdsProxyTargetGroup)

			if targetGroup.RdsProxy == nil {
				targetGroup.RdsProxy = proxy
			} else if targetGroup.RdsProxy != proxy {
				return fmt.Errorf("target group, %s, has edge Destination proxy, %s, but internal property is set Destination a different proxy", targetGroup.Name, proxy.Name)
			}
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxy{}), Destination: reflect.TypeOf(&resources.SecretVersion{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			proxy := dest.(*resources.RdsProxy)
			secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
				AppName: data.AppName,
				Refs:    core.DedupeAnnotationKeys(append(source.KlothoConstructRef(), dest.KlothoConstructRef()...)),
				Name:    proxy.Name,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(proxy, secretVersion)

			secretPolicy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    fmt.Sprintf("%s-ormsecretpolicy", proxy.Name),
				Refs:    proxy.ConstructsRef,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(secretPolicy, secretVersion)
			dag.AddDependency(proxy.Role, secretPolicy)
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			secretVersion := dest.(*resources.SecretVersion)
			secretVersion.Type = "string"
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.IamPolicy{}), Destination: reflect.TypeOf(&resources.SecretVersion{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy := source.(*resources.IamPolicy)
			secretVersion := dest.(*resources.SecretVersion)
			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:GetSecretValue"}, []core.IaCValue{{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE}})
			if policy.Policy == nil {
				policy.Policy = secretPolicyDoc
			} else {
				policy.Policy.Statement = append(policy.Policy.Statement, secretPolicyDoc.Statement...)
			}
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.IamRole{}), Destination: reflect.TypeOf(&resources.IamPolicy{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy := dest.(*resources.IamPolicy)
			role := source.(*resources.IamRole)
			role.ManagedPolicies = append(role.ManagedPolicies, core.IaCValue{Resource: policy, Property: resources.ARN_IAC_VALUE})
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsInstance{}), Destination: reflect.TypeOf(&resources.RdsSubnetGroup{})}: knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsInstance{}), Destination: reflect.TypeOf(&resources.SecurityGroup{})}:  knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.IamRole{})}:     knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.EcrImage{})}:    knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.EcrImage{}), Destination: reflect.TypeOf(&resources.EcrRepository{})}:     knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.SecurityGroup{}), Destination: reflect.TypeOf(&resources.Vpc{})}:          knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.Subnet{}), Destination: reflect.TypeOf(&resources.Vpc{})}:                 knowledgebase.EdgeDetails{},
}
