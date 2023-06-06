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
			Refs:    core.AnnotationKeySetOf(orm.AnnotationKey),
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

func (a *AWS) getRdsConfiguration(result *core.ConstructGraph, dag *core.ResourceGraph, refs core.AnnotationKeySet) (resources.RdsInstanceConfigureParams, error) {
	ref, oneRef := refs.GetSingle()
	if !oneRef {
		return resources.RdsInstanceConfigureParams{}, fmt.Errorf("rds instance must only have one construct reference")
	}
	rdsConfig := resources.RdsInstanceConfigureParams{}
	construct := result.GetConstruct(core.ConstructId(ref).ToRid())
	if construct == nil {
		return resources.RdsInstanceConfigureParams{}, fmt.Errorf("construct with id %s does not exist", ref.ToId())
	}
	orm, ok := construct.(*core.Orm)
	if !ok {
		return resources.RdsInstanceConfigureParams{}, fmt.Errorf("rds instance must only have a construct reference to an orm")
	}

	rdsConfig.DatabaseName = orm.ID
	return rdsConfig, nil
}
