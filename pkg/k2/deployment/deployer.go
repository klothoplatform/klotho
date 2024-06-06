package deployment

import (
	"context"
	"fmt"
	"os"
	"time"

	errors2 "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"go.uber.org/zap"
)

type (
	Deployer struct {
		StateManager *model.StateManager
	}

	StackReference struct {
		ConstructURN model.URN
		Name         string
		IacDirectory string
		AwsRegion    string
	}
)

func (d *Deployer) RunApplicationUpCommand(stackReferences []StackReference) error {
	//todo, this needs to take into account dependency order
	sm := d.StateManager
	defer sm.SaveState()
	for _, stackReference := range stackReferences {
		name := stackReference.Name
		now := time.Now().String()
		if sm.GetState().Constructs[name].Status == model.New {
			sm.UpdateResourceState(name, model.Creating, now)
			if err := d.runStackUp(stackReference); err != nil {
				sm.UpdateResourceState(stackReference.ConstructURN.String(), model.Creating, time.Now().String())
				return err
			}

		}
		sm.UpdateResourceState(name, model.Created, now)
	}

	return nil
}

func (d *Deployer) runStackUp(stackReference StackReference) error {
	ctx := context.Background()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory

	s, err := auto.UpsertStackLocalSource(ctx, stackName, stackDirectory)
	if err != nil {
		zap.S().Errorf("Failed to create or select stack: %v\n", err)
		os.Exit(1)
	}

	zap.S().Infof("Created/Selected stack %q\n", stackName)

	err = s.Workspace().SetEnvVars(map[string]string{
		"PULUMI_CONFIG_PASSPHRASE": "",
	})
	if err != nil {
		zap.S().Errorf("Failed to set environment variables: %v\n", err)
		return errors2.WrapErrf(err, "Failed to set environment variables")
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	zap.S().Info("Successfully set config")
	zap.S().Info("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		zap.S().Errorf("Failed to refresh stack: %v\n", err)
		return errors2.WrapErrf(err, "Failed to refresh stack")
	}

	zap.S().Info("Refresh succeeded!")

	zap.S().Info("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	// run the update to deploy our fargate web service
	res, err := s.Up(ctx, stdoutStreamer)
	if err != nil {
		zap.S().Errorf("Failed to update stack: %v\n\n", err)
		return errors2.WrapErrf(err, "Failed to update stack")
	}

	zap.S().Infof("Successfully deployed stack %s", stackName)

	zap.S().Info("Stack outputs:")
	for key, value := range res.Outputs {
		zap.S().Infof("%s=%s", key, RenderOutputValue(value))
	}
	return nil
}

func (d *Deployer) RunApplicationDownCommand(stackReferences []StackReference) error {
	for _, stackReference := range stackReferences {
		if err := d.runStackDown(stackReference); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) runStackDown(stackReference StackReference) error {
	ctx := context.Background()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory
	s, err := auto.UpsertStackLocalSource(ctx, stackName, stackDirectory)
	if err != nil {
		zap.S().Errorf("Failed to create or select stack: %v\n", err)
		os.Exit(1)
	}

	zap.S().Infof("Created/Selected stack %q\n", stackName)

	err = s.Workspace().SetEnvVars(map[string]string{
		"PULUMI_CONFIG_PASSPHRASE": "",
	})
	if err != nil {
		zap.S().Errorf("Failed to set environment variables: %v\n", err)
		return errors2.WrapErrf(err, "Failed to set environment variables")
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	zap.S().Info("Successfully set config")
	zap.S().Info("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		zap.S().Errorf("Failed to refresh stack: %v\n", err)
		return errors2.WrapErrf(err, "Failed to refresh stack")
	}

	zap.S().Info("Refresh succeeded!")

	zap.S().Info("Starting destroy")

	// wire up our destroy to stream progress to stdout
	stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

	// run the destroy to remove our resources
	_, err = s.Destroy(ctx, stdoutStreamer)
	if err != nil {
		zap.S().Errorf("Failed to destroy stack: %v\n\n", err)
		return errors2.WrapErrf(err, "Failed to destroy stack")
	}

	zap.S().Infof("Successfully destroyed stack %s", stackName)

	zap.S().Info("Removing stack %s", stackName)
	err = s.Workspace().RemoveStack(ctx, stackName)
	if err != nil {
		zap.S().Errorf("Failed to remove stack: %v\n", err)
		return errors2.WrapErrf(err, "Failed to remove stack")
	}
	return nil
}

func RenderOutputValue(output auto.OutputValue) string {
	if output.Secret {
		return "[secret]"
	}

	return fmt.Sprintf("%s", output.Value)
}
