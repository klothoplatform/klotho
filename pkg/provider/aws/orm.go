package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

// expandOrm takes in a single orm construct and expands the generic construct into a set of resource's based on the units configuration.
func (a *AWS) expandOrm(dag *core.ResourceGraph, orm *core.Orm) error {
	switch a.Config.GetPersistOrm(orm.ID).Type {
	case Rds_postgres:
		instance, err := core.CreateResource[*resources.RdsInstance](dag, resources.RdsInstanceCreateParams{
			AppName: a.Config.AppName,
			Refs:    []core.AnnotationKey{orm.AnnotationKey},
			Name:    orm.ID,
		})
		if err != nil {
			return err
		}
		a.MapResourceDirectlyToConstruct(instance, orm)
	default:
		return fmt.Errorf("unsupported orm type %s", a.Config.GetPersistOrm(orm.ID).Type)
	}
	return nil
}

func (a *AWS) GenerateOrmResources(construct *core.Orm, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	vpc := resources.GetVpc(a.Config, dag)
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
	policyDoc := instance.GetConnectionPolicyDocument()
	upstreamResources := result.GetUpstreamConstructs(construct)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			a.PolicyGenerator.AddInlinePolicyToUnit(unit.Id(),
				resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), []core.AnnotationKey{unit.Provenance()}, policyDoc))
		}
	}
	return nil
}
