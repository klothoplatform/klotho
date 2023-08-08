package knowledgebase

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"go.uber.org/zap"
)

var NetworkingKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.NatGateway, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.NatGateway, *resources.ElasticIp]{},
	knowledgebase.EdgeBuilder[*resources.Subnet, *resources.RouteTable]{},
	knowledgebase.EdgeBuilder[*resources.RouteTable, *resources.NatGateway]{
		Configure: func(routeTable *resources.RouteTable, nat *resources.NatGateway, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			shouldAddRoute := true
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					zap.S().Warnf("route table %s already has route for 0.0.0.0/0. Not overwriting rule", routeTable.Name)
					shouldAddRoute = false
					break
				}
			}
			if shouldAddRoute {
				routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", NatGatewayId: core.IaCValue{ResourceId: nat.Id(), Property: resources.ID_IAC_VALUE}})
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.RouteTable, *resources.InternetGateway]{
		Configure: func(routeTable *resources.RouteTable, igw *resources.InternetGateway, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			shouldAddRoute := true
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					zap.S().Warnf("route table %s already has route for 0.0.0.0/0. Not overwriting rule", routeTable.Name)
					shouldAddRoute = false
					break
				}
			}
			if shouldAddRoute {
				routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", GatewayId: core.IaCValue{ResourceId: igw.Id(), Property: resources.ID_IAC_VALUE}})
			}
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
					{ResourceId: vpc.Id(), Property: resources.CIDR_BLOCK_IAC_VALUE},
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

			shouldAddVpcIngressRule := true
			shouldAddSelfIngressRule := true
			for _, rule := range sg.IngressRules {
				if rule.Equals(vpcIngressRule) {
					shouldAddVpcIngressRule = false
					continue
				}
				if rule.Equals(selfIngressRule) {
					shouldAddSelfIngressRule = false
					continue
				}
			}

			if shouldAddVpcIngressRule {
				sg.IngressRules = append(sg.IngressRules, vpcIngressRule)
			}
			if shouldAddSelfIngressRule {
				sg.IngressRules = append(sg.IngressRules, selfIngressRule)
			}

			allOutboundRule := resources.SecurityGroupRule{
				Description: "Allows all outbound IPv4 traffic.",
				FromPort:    0,
				Protocol:    "-1",
				ToPort:      0,
				CidrBlocks: []core.IaCValue{
					{Property: "0.0.0.0/0"},
				},
			}

			shouldAddAllOutboundRule := true
			for _, rule := range sg.EgressRules {
				if rule.Equals(allOutboundRule) {
					shouldAddAllOutboundRule = false
					break
				}
			}
			if shouldAddAllOutboundRule {
				sg.EgressRules = append(sg.EgressRules, allOutboundRule)
			}

			return nil
		},
	},
)
