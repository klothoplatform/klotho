package pulumi

import (
	"context"
	"os"
	"path/filepath"

	errors2 "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"go.uber.org/zap"
)

type StackReference struct {
	ConstructURN model.URN
	Name         string
	IacDirectory string
	AwsRegion    string
}

func InitializeStack(projectName string, stackName string, stackDirectory string, ctx context.Context) (auto.Stack, error) {
	// PulumiHome customizes the location of $PULUMI_HOME where metadata is stored and plugins are installed.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return auto.Stack{}, errors2.WrapErrf(err, "Failed to get user home directory")

	}
	pulumiHomeDir := filepath.Join(homeDir, ".k2", "pulumi")
	ph := auto.PulumiHome(pulumiHomeDir)

	// create pulumi home directory if it does not exist
	if _, err := os.Stat(pulumiHomeDir); os.IsNotExist(err) {
		if err := os.MkdirAll(pulumiHomeDir, 0755); err != nil {
			return auto.Stack{}, errors2.WrapErrf(err, "Failed to create pulumi home directory")
		}
	}

	// create the stack directory if it does not exist
	stateDir := filepath.Join(pulumiHomeDir, "state")
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			return auto.Stack{}, errors2.WrapErrf(err, "Failed to create stack state directory")
		}
	}

	// Project provides ProjectSettings to set once the workspace is created.
	proj := auto.Project(workspace.Project{
		Name:    tokens.PackageName("myproject"),
		Runtime: workspace.NewProjectRuntimeInfo("nodejs", nil),
		Backend: &workspace.ProjectBackend{
			URL: "file://" + stateDir,
		},
	})
	secretsProvider := auto.SecretsProvider("passphrase")
	envvars := auto.EnvVars(map[string]string{
		"PULUMI_CONFIG_PASSPHRASE": "",
	})
	return auto.UpsertStackLocalSource(ctx, stackName, stackDirectory, proj, envvars, ph, secretsProvider)
}

func RunStackUp(stackReference StackReference) (auto.UpResult, StackState, error) {
	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory
	ctx := context.Background()

	s, err := InitializeStack("myproject", stackName, stackDirectory, ctx)
	if err != nil {
		return auto.UpResult{}, StackState{}, errors2.WrapErrf(err, "failed to create or select stack: %s", stackName)
	}
	zap.S().Infof("Created/Selected stack %q\n", stackName)

	err = InstallDependencies(stackDirectory)
	if err != nil {
		return auto.UpResult{}, StackState{}, errors2.WrapErrf(err, "Failed to install dependencies")
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return auto.UpResult{}, StackState{}, errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	zap.S().Info("Successfully set config")

	zap.S().Info("Starting update")

	upResult, err := s.Up(ctx, optup.ProgressStreams(os.Stdout), optup.Refresh())
	if err != nil {
		zap.S().Errorf("Failed to update stack: %v\n\n", err)
		return upResult, StackState{}, errors2.WrapErrf(err, "Failed to update stack")
	}

	zap.S().Infof("Successfully deployed stack %s", stackName)

	stackState, err := GetStackState(ctx, s)
	return upResult, stackState, err
}

func RunStackPreview(stackReference StackReference) (auto.PreviewResult, error) {
	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory
	ctx := context.Background()

	s, err := InitializeStack("myproject", stackName, stackDirectory, ctx)
	if err != nil {
		return auto.PreviewResult{}, errors2.WrapErrf(err, "failed to create or select stack: %s", stackName)
	}
	zap.S().Infof("Created/Selected stack %q\n", stackName)

	err = InstallDependencies(stackDirectory)
	if err != nil {
		return auto.PreviewResult{}, errors2.WrapErrf(err, "Failed to install dependencies")
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return auto.PreviewResult{}, errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	zap.S().Info("Successfully set config")

	zap.S().Info("Starting preview")

	previewResult, err := s.Preview(ctx, optpreview.ProgressStreams(os.Stdout), optpreview.Refresh())
	if err != nil {
		zap.S().Errorf("Failed to preview stack: %v\n\n", err)
		return previewResult, errors2.WrapErrf(err, "Failed to preview stack")
	}

	zap.S().Infof("Successfully previewed stack %s", stackName)

	return previewResult, nil
}

func RunStackDown(stackReference StackReference) error {
	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory
	ctx := context.Background()
	s, err := InitializeStack("myproject", stackName, stackDirectory, ctx)
	if err != nil {
		return errors2.WrapErrf(err, "failed to create or select stack: %s", stackName)
	}

	zap.S().Infof("Created/Selected stack %q\n", stackName)

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	zap.S().Info("Successfully set config")

	zap.S().Info("Starting destroy")

	// wire up our destroy to stream progress to stdout
	stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)
	refresh := optdestroy.Refresh()

	// run the destroy to remove our resources
	_, err = s.Destroy(ctx, stdoutStreamer, refresh)
	if err != nil {
		zap.S().Errorf("Failed to destroy stack: %v\n\n", err)
		return errors2.WrapErrf(err, "Failed to destroy stack")
	}

	zap.S().Infof("Successfully destroyed stack %s", stackName)

	zap.S().Infof("Removing stack %s", stackName)
	err = s.Workspace().RemoveStack(ctx, stackName)
	if err != nil {
		zap.S().Errorf("Failed to remove stack: %v\n", err)
		return errors2.WrapErrf(err, "Failed to remove stack")
	}
	return nil
}

func InstallDependencies(stackDirectory string) error {
	zap.S().Infof("Installing pulumi dependencies in %s", stackDirectory)
	npmCmd := logging.Command(
		context.TODO(),
		logging.CommandLogger{RootLogger: zap.L().Named("npm")},
		"npm", "install",
	)
	npmCmd.Dir = stackDirectory
	return npmCmd.Run()
}
