package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var RdsKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.RdsProxyTargetGroup, *resources.RdsInstance]{
		Expand: func(targetGroup *resources.RdsProxyTargetGroup, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if targetGroup.Name == "" {
				targetGroup, err := core.CreateResource[*resources.RdsProxyTargetGroup](dag, resources.RdsProxyTargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    instance.ConstructsRef.Clone(),
					Name:    instance.Name,
				})
				if err != nil {
					return err
				}
				targetGroup.RdsInstance = instance
				dag.AddDependency(targetGroup, instance)
			}
			targetGroup.ConstructsRef.AddAll(data.Source.KlothoConstructRef())
			for _, res := range dag.GetUpstreamResources(data.Source) {
				if role, ok := res.(*resources.IamRole); ok {
					dag.AddDependency(role, instance)
				}
			}
			return nil
		},
		Configure: func(targetGroup *resources.RdsProxyTargetGroup, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
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
	knowledgebase.EdgeBuilder[*resources.RdsProxyTargetGroup, *resources.RdsProxy]{
		ReverseDirection: true,
		Expand: func(targetGroup *resources.RdsProxyTargetGroup, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			var err error
			destination := data.Destination.(*resources.RdsInstance)
			if proxy.Name == "" {
				proxy, err = core.CreateResource[*resources.RdsProxy](dag, resources.RdsProxyCreateParams{
					AppName: data.AppName,
					Refs:    data.Source.KlothoConstructRef().CloneWith(destination.KlothoConstructRef()),
					Name:    fmt.Sprintf("%s-proxy", strings.TrimPrefix(destination.Name, fmt.Sprintf("%s-", data.AppName))),
				})
				if err != nil {
					return err
				}
				dag.AddDependencyWithData(data.Source, proxy, data)
			}
			proxy.ConstructsRef.AddAll(data.Source.KlothoConstructRef())
			proxy.ConstructsRef.AddAll(destination.ConstructsRef)

			if targetGroup.Name == "" {
				targetGroup, err = core.CreateResource[*resources.RdsProxyTargetGroup](dag, resources.RdsProxyTargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    proxy.ConstructsRef.Clone(),
					Name:    destination.Name,
				})
				if err != nil {
					return err
				}
			}
			targetGroup.ConstructsRef.AddAll(proxy.ConstructsRef)
			targetGroup.RdsProxy = proxy
			dag.AddDependency(targetGroup, proxy)

			secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
				AppName: data.AppName,
				Refs:    proxy.KlothoConstructRef().Clone(),
				Name:    fmt.Sprintf("%s-credentials", strings.TrimPrefix(proxy.Name, fmt.Sprintf("%s-", data.AppName))),
			})
			if err != nil {
				return err
			}

			proxy.Auths = append(proxy.Auths, &resources.ProxyAuth{
				AuthScheme: "SECRETS",
				IamAuth:    "DISABLED",
				SecretArn:  core.IaCValue{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE},
			})
			dag.AddDependency(proxy, secretVersion.Secret)

			secretPolicy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    fmt.Sprintf("%s-ormsecretpolicy", proxy.Name),
				Refs:    proxy.ConstructsRef.Clone(),
			})
			if err != nil {
				return err
			}
			dag.AddDependency(secretPolicy, secretVersion.Secret)
			dag.AddDependency(proxy.Role, secretPolicy)
			dag.AddDependenciesReflect(proxy)
			return nil
		},
		Configure: func(targetGroup *resources.RdsProxyTargetGroup, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
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
						secretVersion.Type = "string"
					}
				}
			}
			return nil
		},
		ValidDestinations: []core.Resource{&resources.RdsInstance{}},
	},
	knowledgebase.EdgeBuilder[*resources.RdsSubnetGroup, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.RdsInstance, *resources.RdsSubnetGroup]{},
	knowledgebase.EdgeBuilder[*resources.RdsInstance, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.RdsProxy, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.RdsProxy, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.RdsProxy, *resources.Secret]{},
)
