package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var NetworkingKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.NatGateway, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.NatGateway, *resources.ElasticIp]{},
	knowledgebase.EdgeBuilder[*resources.RouteTable, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.RouteTable, *resources.NatGateway]{
		Configure: func(routeTable *resources.RouteTable, nat *resources.NatGateway, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					return fmt.Errorf("route table %s already has route for 0.0.0.0/0", routeTable.Name)
				}
			}
			routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", NatGatewayId: core.IaCValue{Resource: nat, Property: resources.ID_IAC_VALUE}})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.RouteTable, *resources.InternetGateway]{
		Configure: func(routeTable *resources.RouteTable, igw *resources.InternetGateway, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					return fmt.Errorf("route table %s already has route for 0.0.0.0/0", routeTable.Name)
				}
			}
			routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", GatewayId: core.IaCValue{Resource: igw, Property: resources.ID_IAC_VALUE}})
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.RouteTable, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.InternetGateway, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.Subnet, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.Subnet, *resources.AvailabilityZones]{},
	knowledgebase.EdgeBuilder[*resources.SecurityGroup, *resources.Vpc]{
		Configure: func(sg *resources.SecurityGroup, vpc *resources.Vpc, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
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
)
