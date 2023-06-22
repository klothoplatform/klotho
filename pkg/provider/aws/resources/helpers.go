package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

func getSingleUpstreamVpc(dag *core.ResourceGraph, resource core.Resource) (vpc *Vpc, err error) {
	vpcs := core.GetAllDownstreamResourcesOfType[*Vpc](dag, resource)
	if len(vpcs) > 1 {
		return nil, fmt.Errorf("resource %s has more than one vpc downstream", resource.Id())
	} else if len(vpcs) == 1 {
		return vpcs[0], nil
	}
	return nil, nil
}

func getSecurityGroupsOperational(dag *core.ResourceGraph, resource core.Resource, appName string) ([]*SecurityGroup, error) {
	vpc, err := getSingleUpstreamVpc(dag, resource)
	if err != nil {
		return nil, err
	}
	sgs := core.GetAllDownstreamResourcesOfType[*SecurityGroup](dag, resource)
	if len(sgs) == 0 {
		securityGroup, err := core.CreateResource[*SecurityGroup](dag, SecurityGroupCreateParams{
			AppName: appName,
			Refs:    core.BaseConstructSetOf(resource),
		})
		if vpc != nil {
			dag.AddDependency(securityGroup, vpc)
		}
		if err != nil {
			return nil, err
		}
		err = securityGroup.MakeOperational(dag, appName)
		if err != nil {
			return nil, err
		}
		return []*SecurityGroup{securityGroup}, nil
	} else {
		securityGroups := []*SecurityGroup{}
		for _, sg := range sgs {
			if sg.Vpc != vpc {
				return nil, fmt.Errorf("resource %s has security groups from multiple vpcs downstream", resource.Id())
			}
			securityGroups = append(securityGroups, sg)
		}
		return securityGroups, nil
	}
}

func getSubnetsOperational(dag *core.ResourceGraph, resource core.Resource, appName string) ([]*Subnet, error) {
	vpc, err := getSingleUpstreamVpc(dag, resource)
	if err != nil {
		return nil, err
	}
	downstreamSubnets := core.GetAllDownstreamResourcesOfType[*Subnet](dag, resource)
	if len(downstreamSubnets) == 0 {
		if vpc != nil {
			subnets := vpc.GetVpcSubnets(dag)
			if len(subnets) == 0 {
				return createSubnets(dag, appName, resource, vpc)
			}
			return subnets, nil
		} else {
			return createSubnets(dag, appName, resource, nil)
		}
	}
	subnets := []*Subnet{}
	for _, subnet := range downstreamSubnets {
		if vpc != nil && subnet.Vpc != vpc {
			return nil, fmt.Errorf("resource %s has subnets from multiple vpcs downstream", resource.Id())
		}
		subnets = append(subnets, subnet)
	}
	return subnets, nil
}
