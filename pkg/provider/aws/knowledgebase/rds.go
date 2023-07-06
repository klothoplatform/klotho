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
		Reuse: knowledgebase.Upstream,
		Configure: func(targetGroup *resources.RdsProxyTargetGroup, instance *resources.RdsInstance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if targetGroup.RdsInstance == nil {
				targetGroup.RdsInstance = instance
			} else if targetGroup.RdsInstance.Name != instance.Name {
				return fmt.Errorf("target group, %s, has  Destination instance, %s, but internal property is set Destination a different instance %s", targetGroup.Name, instance.Name, targetGroup.RdsInstance.Name)
			}
			for _, res := range dag.GetUpstreamResources(data.Source) {
				if role, ok := res.(*resources.IamRole); ok {
					dag.AddDependency(role, instance)
				}
			}
			if targetGroup.RdsProxy == nil || len(targetGroup.RdsProxy.Auths) == 0 {
				return fmt.Errorf("proxy is not configured on target group or auths not created")
			}
			secret := targetGroup.RdsProxy.Auths[0].SecretArn.ResourceVal.(*resources.Secret)
			for _, res := range dag.GetUpstreamResources(secret) {
				if secretVersion, ok := res.(*resources.SecretVersion); ok {
					secretVersion.Path = instance.CredentialsPath
				}
			}

			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.RdsProxyTargetGroup, *resources.RdsProxy]{
		ReverseDirection: true,
		Reuse:            knowledgebase.Upstream,
		Configure: func(targetGroup *resources.RdsProxyTargetGroup, proxy *resources.RdsProxy, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if targetGroup.RdsProxy == nil {
				targetGroup.RdsProxy = proxy
			} else if targetGroup.RdsProxy.Name != proxy.Name {
				return fmt.Errorf("target group, %s, has destination proxy, %s, but internal property is set Destination a different proxy %s", targetGroup.Name, proxy.Name, targetGroup.RdsProxy.Name)
			}
			if proxy.Role == nil {
				return fmt.Errorf("proxy %s is not fully operational yet", proxy.Name)
			}
			if len(proxy.Auths) == 0 {
				secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
					AppName: data.AppName,
					Refs:    proxy.BaseConstructsRef().Clone(),
					Name:    fmt.Sprintf("%s-credentials", strings.TrimPrefix(proxy.Name, fmt.Sprintf("%s-", data.AppName))),
				})
				if err != nil {
					return err
				}
				err = secretVersion.MakeOperational(dag, data.AppName, nil)
				if err != nil {
					return err
				}

				proxy.Auths = append(proxy.Auths, &resources.ProxyAuth{
					AuthScheme: "SECRETS",
					IamAuth:    "DISABLED",
					SecretArn:  &resources.AwsResourceValue{ResourceVal: secretVersion.Secret, PropertyVal: resources.ARN_IAC_VALUE},
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
			}
			dag.AddDependenciesReflect(proxy)
			if targetGroup.RdsInstance != nil {
				secret := proxy.Auths[0].SecretArn.ResourceVal.(*resources.Secret)
				for _, res := range dag.GetUpstreamResources(secret) {
					if secretVersion, ok := res.(*resources.SecretVersion); ok {
						secretVersion.Path = targetGroup.RdsInstance.CredentialsPath
						secretVersion.Type = "string"
					}
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.RdsSubnetGroup, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.RdsInstance, *resources.RdsSubnetGroup]{},
	knowledgebase.EdgeBuilder[*resources.RdsInstance, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.RdsProxy, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.RdsProxy, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.RdsProxy, *resources.Secret]{},
)
