package deployment

import (
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

type (
	Deployer struct {
		StateManager       *model.StateManager
		LanguageHostClient pb.KlothoServiceClient
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

func (d *Deployer) RunApplicationDownCommand(req DownRequest) error {
	for _, stackReference := range req.StackReferences {
		if err := pulumi.RunStackDown(stackReference, req.DryRun); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) RunStackUpCommand(ref pulumi.StackReference) (auto.UpResult, pulumi.StackState, error) {
	return pulumi.RunStackUp(ref)
}

func (d *Deployer) RunStackPreviewCommand(ref pulumi.StackReference) (auto.PreviewResult, error) {
	return pulumi.RunStackPreview(ref)
}
