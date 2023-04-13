package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

func (a *AWS) GenerateOrmResources(construct *core.Orm, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	vpc := resources.CreateNetwork(a.Config, dag)
	securityGroups := []*resources.SecurityGroup{resources.GetSecurityGroup(a.Config, dag)}
	privateSubnets := vpc.GetPrivateSubnets(dag)
	instance, proxy, err := resources.CreateRdsInstance(a.Config, construct, true, privateSubnets, securityGroups, dag)
	if err != nil {
		return err
	}
	a.MapResourceDirectlyToConstruct(instance, construct)
	if proxy != nil {
		a.MapResourceDirectlyToConstruct(proxy, construct)
	}

	policy := instance.GetConnectionPolicy(dag)
	if policy == nil {
		return errors.Errorf("unable to find connection policy for instance %s", instance.Id())
	}
	upstreamResources := result.GetUpstreamConstructs(construct)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			a.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), policy)
		}
	}
	return nil
}
