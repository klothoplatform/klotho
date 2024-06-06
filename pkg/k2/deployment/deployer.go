package deployment

import (
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
)

type (
	Deployer struct {
		StateManager *model.StateManager
	}
)

func (d *Deployer) RunApplicationUpCommand(stackReferences []pulumi.StackReference) error {
	//todo, this needs to take into account dependency order
	sm := d.StateManager
	defer sm.SaveState()
	//TODO: execute runStackDown for removed construct stack references when we have state management
	for _, stackReference := range stackReferences {
		_, err := pulumi.RunStackUp(stackReference)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) RunApplicationDownCommand(stackReferences []pulumi.StackReference) error {
	for _, stackReference := range stackReferences {
		if err := pulumi.RunStackDown(stackReference); err != nil {
			return err
		}
	}
	return nil
}
