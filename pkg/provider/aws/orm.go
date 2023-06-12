package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

// expandOrm takes in a single orm construct and expands the generic construct into a set of resource's based on the units configuration.
func (a *AWS) expandOrm(dag *core.ResourceGraph, orm *core.Orm) error {
	switch a.Config.GetPersistOrm(orm.Name).Type {
	case Rds_postgres:
		instance, err := core.CreateResource[*resources.RdsInstance](dag, resources.RdsInstanceCreateParams{
			AppName: a.Config.AppName,
			Refs:    core.BaseConstructSetOf(orm),
			Name:    orm.Name,
		})
		if err != nil {
			return err
		}
		a.MapResourceDirectlyToConstruct(instance, orm)
	default:
		return fmt.Errorf("unsupported orm type %s", a.Config.GetPersistOrm(orm.Name).Type)
	}
	return nil
}

func (a *AWS) getRdsConfiguration(result *core.ConstructGraph, dag *core.ResourceGraph, refs core.BaseConstructSet) (resources.RdsInstanceConfigureParams, error) {
	if len(refs) > 1 || len(refs) == 0 {
		return resources.RdsInstanceConfigureParams{}, fmt.Errorf("rds instance must only have one construct reference")
	}
	var ref core.BaseConstruct
	for r := range refs {
		ref = r
	}
	rdsConfig := resources.RdsInstanceConfigureParams{}
	construct := result.GetConstruct(ref.Id())
	if construct == nil {
		return resources.RdsInstanceConfigureParams{}, fmt.Errorf("construct with id %s does not exist", ref.Id())
	}
	orm, ok := construct.(*core.Orm)
	if !ok {
		return resources.RdsInstanceConfigureParams{}, fmt.Errorf("rds instance must only have a construct reference to an orm")
	}

	rdsConfig.DatabaseName = orm.Name
	return rdsConfig, nil
}
