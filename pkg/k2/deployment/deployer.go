package deployment

import (
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

type (
	Deployer struct {
		StateManager *model.StateManager
	}

	UpRequest struct {
		StackReferences []pulumi.StackReference
		DryRun          bool
	}

	DownRequest struct {
		StackReferences []pulumi.StackReference
		DryRun          bool
	}
)

func (d *Deployer) RunApplicationDownCommand(ref pulumi.StackReference) error {

	return pulumi.RunStackDown(ref)

}

func (d *Deployer) RunStackUpCommand(ref pulumi.StackReference) (auto.UpResult, pulumi.StackState, error) {
	return pulumi.RunStackUp(ref)
}

func (d *Deployer) RunStackPreviewCommand(ref pulumi.StackReference) (auto.PreviewResult, error) {
	return pulumi.RunStackPreview(ref)
}
