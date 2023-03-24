package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

func (a *AWS) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (Links []core.CloudResourceLink, err error) {
	log := zap.S()

	rootConstructs := result.GetRoots()
	for _, construct := range rootConstructs {
		log.Debugf("Converting construct with id, %s, to aws resources", construct.Id())
		switch construct := construct.(type) {
		case *core.Gateway:
			log.Errorf("Unsupported resource %s", construct.Id())
		case *core.ExecutionUnit:
			err = a.GenerateExecUnitResources(construct, dag)
			if err != nil {
				return
			}
		case *core.StaticUnit:
			log.Errorf("Unsupported resource %s", construct.Id())

		default:
			log.Warnf("Unsupported resource %s", construct.Id())
		}

	}
	return
}
