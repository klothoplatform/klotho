package edges

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var KnowledgeBase = core.EdgeKB{
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.RdsInstance{})}: core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.RdsProxy{})}:    core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), To: reflect.TypeOf(&resources.RdsInstance{})}: core.EdgeDetails{
		ExpansionFunc: func(from, to core.Resource, dag *core.ResourceGraph, data core.EdgeData) error {
			instance := to.(*resources.RdsInstance)
			targetGroup := from.(*resources.RdsProxyTargetGroup)
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
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	core.Edge{From: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), To: reflect.TypeOf(&resources.RdsProxy{})}: core.EdgeDetails{
		ExpansionFunc: func(from, to core.Resource, dag *core.ResourceGraph, data core.EdgeData) error {
			proxy := to.(*resources.RdsProxy)
			targetGroup := from.(*resources.RdsProxyTargetGroup)
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
				Refs:    core.DedupeAnnotationKeys(append(from.KlothoConstructRef(), to.KlothoConstructRef()...)),
				Name:    proxy.Name,
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
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	core.Edge{From: reflect.TypeOf(&resources.RdsProxy{}), To: reflect.TypeOf(&resources.SecretVersion{})}: core.EdgeDetails{
		ExpansionFunc: func(from, to core.Resource, dag *core.ResourceGraph, data core.EdgeData) error {
			proxy := to.(*resources.RdsProxy)
			secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
				AppName: data.AppName,
				Refs:    core.DedupeAnnotationKeys(append(from.KlothoConstructRef(), to.KlothoConstructRef()...)),
				Name:    proxy.Name,
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
			return nil
		},
	},
	core.Edge{From: reflect.TypeOf(&resources.IamPolicy{}), To: reflect.TypeOf(&resources.SecretVersion{})}:    core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.IamRole{}), To: reflect.TypeOf(&resources.IamPolicy{})}:          core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.RdsInstance{}), To: reflect.TypeOf(&resources.RdsSubnetGroup{})}: core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.RdsInstance{}), To: reflect.TypeOf(&resources.SecurityGroup{})}:  core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.IamRole{})}:     core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.LambdaFunction{}), To: reflect.TypeOf(&resources.EcrImage{})}:    core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.EcrImage{}), To: reflect.TypeOf(&resources.EcrRepository{})}:     core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.SecurityGroup{}), To: reflect.TypeOf(&resources.Vpc{})}:          core.EdgeDetails{},
	core.Edge{From: reflect.TypeOf(&resources.Subnet{}), To: reflect.TypeOf(&resources.Vpc{})}:                 core.EdgeDetails{},
}
