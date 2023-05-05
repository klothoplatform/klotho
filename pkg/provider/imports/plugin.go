package imports

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type Plugin struct {
	Config *config.Application
}

func (p Plugin) Name() string {
	return "imports"
}

func (p Plugin) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (links []core.CloudResourceLink, err error) {
	log := zap.S()
	for resId, importId := range p.Config.Import {
		res := dag.GetResource(resId)
		if res == nil {
			log.Warnf("No resource found for import '%s'", resId)
			continue
		}
		dag.AddDependency(res, &Imported{ID: importId})
	}
	return
}
