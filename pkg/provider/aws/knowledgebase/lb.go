package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var LbKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.Listener, *resources.TargetGroup]{
		Configure: func(listener *resources.Listener, tg *resources.TargetGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			listener.Protocol = tg.Protocol
			listener.DefaultActions = []*resources.LBAction{{TargetGroupArn: &resources.AwsResourceValue{ResourceVal: tg, PropertyVal: resources.ARN_IAC_VALUE}, Type: "forward"}}
			if listener.LoadBalancer == nil || len(listener.LoadBalancer.Subnets) == 0 {
				return fmt.Errorf("cannot configure targetGroup's Vpc %s, missing load balancer vpc for listener %s", tg.Id(), listener.Id())
			}
			tg.Vpc = listener.LoadBalancer.Subnets[0].Vpc
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.Listener]{
		Configure: func(loadBalancer *resources.LoadBalancer, listener *resources.Listener, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if listener.LoadBalancer != loadBalancer {
				return fmt.Errorf("listener %s does not belong to load balancer %s", listener.Id(), loadBalancer.Id())
			}
			loadBalancer.Type = "network"
			loadBalancer.Scheme = "internal"
			listener.Port = 80
			return nil
		},
		DeploymentOrderReversed: true,
	},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.Ec2Instance]{
		Configure: func(targetGroup *resources.TargetGroup, instance *resources.Ec2Instance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			targetGroup.Port = 3000
			targetGroup.Protocol = "HTTPS"
			targetGroup.TargetType = "instance"
			target := &resources.Target{
				Id:   &resources.AwsResourceValue{ResourceVal: instance, PropertyVal: resources.ID_IAC_VALUE},
				Port: 3000,
			}
			targetGroup.AddTarget(target)
			dag.AddDependency(targetGroup, instance)
			return nil
		},
	},
)
