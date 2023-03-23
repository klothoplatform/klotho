package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

func (a *AWS) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (Links []core.CloudResourceLink, err error) {
	fmt.Println(dag.ListConstructs())
	fmt.Println(dag.ListDependencies())

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
	constructs := dag.ListConstructs()
	for _, c := range constructs {
		fmt.Println(c)
	}
	deps := dag.ListDependencies()
	for _, c := range deps {
		fmt.Printf("Source: %s, Dest: %s\n", c.Source.Id(), c.Destination.Id())
	}
	return
}
