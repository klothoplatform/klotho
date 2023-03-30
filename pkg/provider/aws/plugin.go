package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

func (a *AWS) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (Links []core.CloudResourceLink, err error) {
	log := zap.S()

	constructIds, err := result.TopologicalSort()
	if err != nil {
		return
	}
	// We want to reverse the list so that we start at the leaf nodes. This allows us to check downstream dependencies each time and process them.
	reverseInPlace(constructIds)
	for _, id := range constructIds {
		construct := result.GetConstruct(id)
		log.Debugf("Converting construct with id, %s, to aws resources", construct.Id())
		switch construct := construct.(type) {
		case *core.ExecutionUnit:
			err = a.GenerateExecUnitResources(construct, result, dag)
			if err != nil {
				return
			}
		case *core.Fs:
			err = a.GenerateFsResources(construct, result, dag)
			if err != nil {
				return
			}
		default:
			log.Warnf("Unsupported resource %s", construct.Id())
		}
	}

	err = a.convertExecUnitParams(result, dag)
	if err != nil {
		return
	}
	return
}

func reverseInPlace[A any](a []A) {
	// taken from https://github.com/golang/go/wiki/SliceTricks/33793edcc2c7aee6448ed1dd0c36524eddfdf1e2#reversing
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}
