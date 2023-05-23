package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var NetworkingKB = knowledgebase.EdgeKB{
	//Networking Edges
	knowledgebase.NewEdge[*resources.NatGateway, *resources.Subnet](): {},
	knowledgebase.NewEdge[*resources.RouteTable, *resources.Subnet](): {},
	knowledgebase.NewEdge[*resources.RouteTable, *resources.NatGateway](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			routeTable := source.(*resources.RouteTable)
			nat := dest.(*resources.NatGateway)
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					return fmt.Errorf("route table %s already has route for 0.0.0.0/0", routeTable.Name)
				}
			}
			routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", NatGatewayId: core.IaCValue{Resource: nat, Property: resources.ID_IAC_VALUE}})
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.RouteTable, *resources.InternetGateway](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			igw := dest.(*resources.InternetGateway)
			routeTable := source.(*resources.RouteTable)
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					return fmt.Errorf("route table %s already has route for 0.0.0.0/0", routeTable.Name)
				}
			}
			routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", GatewayId: core.IaCValue{Resource: igw, Property: resources.ID_IAC_VALUE}})
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.RouteTable, *resources.Vpc]():           {},
	knowledgebase.NewEdge[*resources.InternetGateway, *resources.Vpc]():      {},
	knowledgebase.NewEdge[*resources.Subnet, *resources.Vpc]():               {},
	knowledgebase.NewEdge[*resources.Subnet, *resources.AvailabilityZones](): {},
	knowledgebase.NewEdge[*resources.SecurityGroup, *resources.Vpc](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			sg := source.(*resources.SecurityGroup)
			vpc := dest.(*resources.Vpc)
			vpcIngressRule := resources.SecurityGroupRule{
				Description: "Allow ingress traffic from ip addresses within the vpc",
				CidrBlocks: []core.IaCValue{
					{Resource: vpc, Property: resources.CIDR_BLOCK_IAC_VALUE},
				},
				FromPort: 0,
				Protocol: "-1",
				ToPort:   0,
			}
			selfIngressRule := resources.SecurityGroupRule{
				Description: "Allow ingress traffic from within the same security group",
				FromPort:    0,
				Protocol:    "-1",
				ToPort:      0,
				Self:        true,
			}
			sg.IngressRules = append(sg.IngressRules, vpcIngressRule, selfIngressRule)

			allOutboundRule := resources.SecurityGroupRule{
				Description: "Allows all outbound IPv4 traffic.",
				FromPort:    0,
				Protocol:    "-1",
				ToPort:      0,
				CidrBlocks: []core.IaCValue{
					{Property: "0.0.0.0/0"},
				},
			}
			sg.EgressRules = append(sg.EgressRules, allOutboundRule)
			return nil
		},
	},
}
