package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) GenerateOrmResources(construct *core.Orm, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	instance, proxy, err := resources.CreateRdsInstance(a.Config, construct, true, dag)
	if err != nil {
		return err
	}
	a.MapResourceDirectlyToConstruct(instance, construct)
	if proxy != nil {
		a.MapResourceDirectlyToConstruct(proxy, construct)
	}
	return nil
}
