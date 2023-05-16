package edges

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var KnowledgeBase = core.EdgeKB{
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.RdsInstance{})}: core.EdgeDetails{
		Configure: func(from, to core.Resource, data core.EdgeData) error {
			lambda := from.(*resources.LambdaFunction)
			instance := to.(*resources.RdsInstance)
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable to expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id().String(), instance.Id().String(), lambda.Id().String())
			}
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), core.DedupeAnnotationKeys(append(lambda.ConstructsRef, instance.ConstructsRef...)), instance.GetConnectionPolicyDocument())
			lambda.Role.InlinePolicies = append(lambda.Role.InlinePolicies, inlinePol)

			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.RdsProxy{})}: core.EdgeDetails{
		Configure: func(from, to core.Resource, data core.EdgeData) error {
			// lambda := from.(*resources.LambdaFunction)
			// proxy := to.(*resources.RdsProxy)
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.RdsProxy{}), To: reflect.TypeOf(&resources.RdsInstance{})}: core.EdgeDetails{
		ExpansionFunc: func(from, to core.Resource, dag *core.ResourceGraph, data core.EdgeData) error {
			proxy := from.(*resources.RdsProxy)
			instance := to.(*resources.RdsInstance)
			if proxy.Name == "" {
				var err error
				proxy, err = core.CreateResource[*resources.RdsProxy](dag, resources.RdsProxyCreateParams{
					AppName: data.AppName,
					Refs:    instance.ConstructsRef,
					Name:    instance.DatabaseName,
				})
				if err != nil {
					return err
				}
			}

			proxyTargetGroup, err := core.CreateResource[*resources.RdsProxyTargetGroup](dag, resources.RdsProxyTargetGroupCreateParams{
				AppName: data.AppName,
				Refs:    instance.ConstructsRef,
				Name:    instance.DatabaseName,
			})
			if err != nil {
				return err
			}

			dag.AddDependency(proxyTargetGroup, proxy)
			dag.AddDependency(proxyTargetGroup, instance)

			secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
				AppName:    data.AppName,
				Refs:       core.DedupeAnnotationKeys(append(from.KlothoConstructRef(), to.KlothoConstructRef()...)),
				SecretName: proxy.Name,
			})

			dag.AddDependency(proxy, secretVersion)

			secretPolicy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName:    data.AppName,
				PolicyName: fmt.Sprintf("%s-ormsecretpolicy", proxy.Name),
				Refs:       proxy.ConstructsRef,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(secretPolicy, secretVersion)
			dag.AddDependency(proxy.Role, secretPolicy)
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), To: reflect.TypeOf(&resources.RdsInstance{})}: core.EdgeDetails{
		Configure: func(from, to core.Resource, data core.EdgeData) error {
			instance := to.(*resources.RdsInstance)
			targetGroup := from.(*resources.RdsProxyTargetGroup)

			if targetGroup.RdsInstance == nil {
				targetGroup.RdsInstance = instance
			} else if targetGroup.RdsInstance != instance {
				return fmt.Errorf("target group, %s, has edge to instance, %s, but internal property is set to a different instance", targetGroup.Name, instance.Name)
			}
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), To: reflect.TypeOf(&resources.RdsProxy{})}: core.EdgeDetails{
		Configure: func(from, to core.Resource, data core.EdgeData) error {
			proxy := to.(*resources.RdsProxy)
			targetGroup := from.(*resources.RdsProxyTargetGroup)

			if targetGroup.RdsProxy == nil {
				targetGroup.RdsProxy = proxy
			} else if targetGroup.RdsProxy != proxy {
				return fmt.Errorf("target group, %s, has edge to proxy, %s, but internal property is set to a different proxy", targetGroup.Name, proxy.Name)
			}
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.RdsProxy{}), To: reflect.TypeOf(&resources.SecretVersion{})}: core.EdgeDetails{
		Configure: func(from, to core.Resource, data core.EdgeData) error {
			secretVersion := to.(*resources.SecretVersion)
			secretVersion.Type = "string"
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.IamPolicy{}), To: reflect.TypeOf(&resources.SecretVersion{})}: core.EdgeDetails{
		Configure: func(from, to core.Resource, data core.EdgeData) error {
			policy := from.(*resources.IamPolicy)
			secretVersion := to.(*resources.SecretVersion)

			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:GetSecretValue"}, []core.IaCValue{{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE}})
			if policy.Policy == nil {
				policy.Policy = secretPolicyDoc
			} else {
				policy.Policy.Statement = append(policy.Policy.Statement, secretPolicyDoc.Statement...)
			}
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.IamRole{}), To: reflect.TypeOf(&resources.IamPolicy{})}: core.EdgeDetails{
		Configure: func(from, to core.Resource, data core.EdgeData) error {
			policy := to.(*resources.IamPolicy)
			role := from.(*resources.IamRole)
			role.ManagedPolicies = append(role.ManagedPolicies, core.IaCValue{Resource: policy, Property: resources.ARN_IAC_VALUE})
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.RdsInstance{}), To: reflect.TypeOf(&resources.RdsSubnetGroup{})}: core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.RdsInstance{}), To: reflect.TypeOf(&resources.SecurityGroup{})}:  core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.IamRole{})}:     core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.EcrImage{})}:    core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.EcrImage{}), To: reflect.TypeOf(&resources.EcrRepository{})}:     core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.SecurityGroup{}), To: reflect.TypeOf(&resources.Vpc{})}:          core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.Subnet{}), To: reflect.TypeOf(&resources.Vpc{})}:                 core.EdgeDetails{},
}
