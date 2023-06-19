package knowledgebase

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var LbKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.Listener, *resources.TargetGroup]{
		ValidDestinations: []core.Resource{&resources.Ec2Instance{}, &resources.EcsService{}},
	},
	knowledgebase.EdgeBuilder[*resources.Listener, *resources.LoadBalancer]{
		ValidDestinations: []core.Resource{&resources.TargetGroup{}, &resources.Ec2Instance{}, &resources.EcsService{}},
		ReverseDirection:  true,
	},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.Ec2Instance]{
		Configure: func(source *resources.TargetGroup, destination *resources.Ec2Instance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			target := &resources.Target{
				Id:   &resources.AwsResourceValue{ResourceVal: destination, PropertyVal: resources.ID_IAC_VALUE},
				Port: 3000,
			}
			source.AddTarget(target)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.EcsService]{},
)
