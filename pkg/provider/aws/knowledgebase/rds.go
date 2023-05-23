package knowledgebase

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var RdsKB = knowledgebase.EdgeKB{
	knowledgebase.NewEdge[*resources.RdsProxyTargetGroup, *resources.RdsInstance](): {
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
				proxyTargetGroup.RdsInstance = instance
				dag.AddDependency(proxyTargetGroup, instance)
			}
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			instance := dest.(*resources.RdsInstance)
			targetGroup := source.(*resources.RdsProxyTargetGroup)

			if targetGroup.RdsInstance == nil {
				targetGroup.RdsInstance = instance
			} else if targetGroup.RdsInstance.Name != instance.Name {
				return fmt.Errorf("target group, %s, has  Destination instance, %s, but internal property is set Destination a different instance %s", targetGroup.Name, instance.Name, targetGroup.RdsInstance.Name)
			}
			if targetGroup.RdsProxy != nil {
				secret := targetGroup.RdsProxy.Auths[0].SecretArn.Resource.(*resources.Secret)
				for _, res := range dag.GetUpstreamResources(secret) {
					if secretVersion, ok := res.(*resources.SecretVersion); ok {
						secretVersion.Path = instance.CredentialsPath
					}
				}
			}

			return nil
		},
	},
	knowledgebase.NewEdge[*resources.RdsProxyTargetGroup, *resources.RdsProxy](): {
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
				dag.AddDependencyWithData(data.Source, proxy, data)
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
				proxyTargetGroup.RdsProxy = proxy
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
			} else if targetGroup.RdsProxy.Name != proxy.Name {
				return fmt.Errorf("target group, %s, has destination proxy, %s, but internal property is set Destination a different proxy %s", targetGroup.Name, proxy.Name, targetGroup.RdsProxy.Name)
			}
			if targetGroup.RdsInstance != nil {
				secret := proxy.Auths[0].SecretArn.Resource.(*resources.Secret)
				for _, res := range dag.GetUpstreamResources(secret) {
					if secretVersion, ok := res.(*resources.SecretVersion); ok {
						secretVersion.Path = targetGroup.RdsInstance.CredentialsPath
					}
				}
			}
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	knowledgebase.NewEdge[*resources.RdsProxy, *resources.SecretVersion](): {
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
	knowledgebase.NewEdge[*resources.RdsSubnetGroup, *resources.Subnet]():      {},
	knowledgebase.NewEdge[*resources.RdsInstance, *resources.RdsSubnetGroup](): {},
	knowledgebase.NewEdge[*resources.RdsInstance, *resources.SecurityGroup]():  {},
	knowledgebase.NewEdge[*resources.RdsProxy, *resources.SecurityGroup]():     {},
	knowledgebase.NewEdge[*resources.RdsProxy, *resources.Subnet]():            {},
	knowledgebase.NewEdge[*resources.RdsProxy, *resources.Secret]():            {},
}
