package deployment

import (
	"context"

	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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

func (d *Deployer) RunApplicationUpCommand(req UpRequest) (err error) {
	//todo, this needs to take into account dependency order
	sm := d.StateManager
	defer func() {
		saveErr := sm.SaveState()
		if err == nil {
			err = saveErr
		}
	}()
	//TODO: execute runStackDown for removed construct stack references when we have state management
	for _, stackReference := range req.StackReferences {
		stackState, err := pulumi.RunStackUp(stackReference, req.DryRun)
		if err != nil {
			return err
		}
		err2 := d.resolveOutputValues(stackReference, stackState)
		if err2 != nil {
			return err2
		}
	}
	return nil
}

func (d *Deployer) resolveOutputValues(stackReference pulumi.StackReference, stackState pulumi.StackState) error {
	// TODO: This is a demo implementation that passes the stack outputs to the language host
	//       and gets the resolved output references back.
	//       It doesn't actually do anything with the resolved outputs yet.
	outputs := map[string]map[string]interface{}{
		stackReference.ConstructURN.String(): stackState.Outputs,
	}
	payload, err := yaml.Marshal(outputs)
	if err != nil {
		return err
	}
	resp, err := d.LanguageHostClient.RegisterConstruct(context.Background(), &pb.RegisterConstructRequest{
		YamlPayload: string(payload),
	})
	zap.S().Info(resp.GetMessage())
	var resolvedOutputs []any
	for _, o := range resp.GetResolvedOutputs() {
		if err != nil {
			return err
		}
		resolvedOutputs = append(resolvedOutputs, map[string]interface{}{
			"id":    o.GetId(),
			"value": o.GetYamlPayload(),
		})
	}
	zap.S().Infof("Resolved Outputs: %v", resolvedOutputs)
	if err != nil {
		return err
	}
	return nil
}

func (d *Deployer) RunApplicationDownCommand(req DownRequest) error {
	for _, stackReference := range req.StackReferences {
		if err := pulumi.RunStackDown(stackReference, req.DryRun); err != nil {
			return err
		}
	}
	return nil
}
