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
		Expand: func(source *resources.Listener, destination *resources.TargetGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if data.Source.Id().Type != resources.API_GATEWAY_INTEGRATION_TYPE {
				src := data.Source.Id().Name
				dst := data.Destination.Id().Name
				if source.Name == "" || source == nil {
					var err error
					source, err = core.CreateResource[*resources.Listener](dag, resources.ListenerCreateParams{
						AppName: data.AppName,
						Refs:    core.BaseConstructSetOf(data.Source, data.Destination),
						Name:    src,
					})
					if err != nil {
						return err
					}
				}
				if destination.Name == "" || destination == nil {
					var err error
					destination, err = core.CreateResource[*resources.TargetGroup](dag, resources.TargetGroupCreateParams{
						AppName: data.AppName,
						Refs:    core.BaseConstructSetOf(data.Source, data.Destination),
						Name:    dst,
					})
					if err != nil {
						return err
					}
				}
				dag.AddDependency(source, destination)
				if source.LoadBalancer != nil && len(source.LoadBalancer.Subnets) > 0 && source.LoadBalancer.Subnets[0].Vpc != nil {
					destination.Vpc = source.LoadBalancer.Subnets[0].Vpc
				} else {
					return fmt.Errorf("could not set target groups vpc")
				}
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Listener, *resources.LoadBalancer]{
		Expand: func(source *resources.Listener, destination *resources.LoadBalancer, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if data.Source.Id().Type != resources.API_GATEWAY_INTEGRATION_TYPE {
				src := data.Source.Id().Name
				dst := data.Destination.Id().Name
				if source.Name == "" || source == nil {
					var err error
					source, err = core.CreateResource[*resources.Listener](dag, resources.ListenerCreateParams{
						AppName: data.AppName,
						Refs:    core.BaseConstructSetOf(data.Source, data.Destination),
						Name:    src,
					})
					if err != nil {
						return err
					}
				}
				if destination.Name == "" || destination == nil {
					var err error
					destination, err = core.CreateResource[*resources.LoadBalancer](dag, resources.LoadBalancerCreateParams{
						AppName: data.AppName,
						Refs:    core.BaseConstructSetOf(data.Source, data.Destination),
						Name:    dst,
					})
					if err != nil {
						return err
					}
				}
				dag.AddDependency(source, destination)
			}
			return nil
		},
		ReverseDirection: true,
	},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.Ec2Instance]{
		Expand: func(source *resources.TargetGroup, destination *resources.Ec2Instance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if data.Source.Id().Type != resources.API_GATEWAY_INTEGRATION_TYPE {
				dst := data.Destination.Id().Name
				if source.Name == "" || source == nil {
					var err error
					source, err = core.CreateResource[*resources.TargetGroup](dag, resources.TargetGroupCreateParams{
						AppName: data.AppName,
						Refs:    core.BaseConstructSetOf(data.Source, data.Destination),
						Name:    dst,
					})
					if err != nil {
						return err
					}
				}
				dag.AddDependency(source, destination)
			}
			return nil
		},
		Configure: func(source *resources.TargetGroup, destination *resources.Ec2Instance, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			target := &resources.Target{
				Id:   &resources.AwsResourceValue{ResourceVal: destination, PropertyVal: resources.ID_IAC_VALUE},
				Port: 3000,
			}
			source.AddTarget(target)
			dag.AddDependency(source, destination)
			return nil
		},
	},
)
