package stack

import (
	"context"
	"io"
	"os"
	"path/filepath"

	errors2 "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/tui"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"go.uber.org/zap"
)

type Reference struct {
	ConstructURN model.URN
	Name         string
	IacDirectory string
	AwsRegion    string
}

func Initialize(projectName string, stackName string, stackDirectory string, ctx context.Context) (auto.Stack, error) {
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

func RunUp(ctx context.Context, stackReference Reference) (auto.UpResult, State, error) {
	log := logging.GetLogger(ctx).Sugar()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory

	s, err := Initialize("myproject", stackName, stackDirectory, ctx)
	if err != nil {
		return auto.UpResult{}, State{}, errors2.WrapErrf(err, "failed to create or select stack: %s", stackName)
	}
	log.Debugf("Created/Selected stack %q", stackName)

	err = InstallDependencies(ctx, stackDirectory)
	if err != nil {
		return auto.UpResult{}, State{}, errors2.WrapErrf(err, "Failed to install dependencies")
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return auto.UpResult{}, State{}, errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	log.Debug("Starting update")

	upResult, err := s.Up(
		ctx,
		optup.ProgressStreams(logging.NewLoggerWriter(log.Desugar().Named("pulumi.up"), zap.InfoLevel)),
		optup.EventStreams(Events(ctx, "Deploying")),
		optup.Refresh(),
	)
	if err != nil {
		return upResult, State{}, errors2.WrapErrf(err, "Failed to update stack")
	}

	log.Infof("Successfully deployed stack %s", stackName)

	stackState, err := GetState(ctx, s)
	return upResult, stackState, err
}

func RunPreview(ctx context.Context, stackReference Reference) (auto.PreviewResult, error) {
	log := logging.GetLogger(ctx).Sugar()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory

	s, err := Initialize("myproject", stackName, stackDirectory, ctx)
	if err != nil {
		return auto.PreviewResult{}, errors2.WrapErrf(err, "failed to create or select stack: %s", stackName)
	}
	log.Infof("Created/Selected stack %q", stackName)

	err = InstallDependencies(ctx, stackDirectory)
	if err != nil {
		return auto.PreviewResult{}, errors2.WrapErrf(err, "Failed to install dependencies")
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return auto.PreviewResult{}, errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	log.Debug("Starting preview")

	previewResult, err := s.Preview(
		ctx,
		optpreview.ProgressStreams(logging.NewLoggerWriter(log.Desugar().Named("pulumi.preview"), zap.InfoLevel)),
		optpreview.EventStreams(Events(ctx, "Previewing")),
		optpreview.Refresh(),
	)
	if err != nil {
		return previewResult, errors2.WrapErrf(err, "Failed to preview stack")
	}

	log.Infof("Successfully previewed stack %s", stackName)

	return previewResult, nil
}

func RunDown(ctx context.Context, stackReference Reference) error {
	log := logging.GetLogger(ctx).Sugar()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory
	s, err := Initialize("myproject", stackName, stackDirectory, ctx)
	if err != nil {
		return errors2.WrapErrf(err, "failed to create or select stack: %s", stackName)
	}

	log.Debugf("Created/Selected stack %q", stackName)

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return errors2.WrapErrf(err, "Failed to set stack configuration")
	}

	log.Debug("Starting destroy")

	// wire up our destroy to stream progress to stdout
	stdoutStreamer := optdestroy.ProgressStreams(logging.NewLoggerWriter(log.Desugar().Named("pulumi.destroy"), zap.InfoLevel))
	refresh := optdestroy.Refresh()
	eventStream := optdestroy.EventStreams(Events(ctx, "Destroying"))

	// run the destroy to remove our resources
	_, err = s.Destroy(ctx, stdoutStreamer, eventStream, refresh)
	if err != nil {
		return errors2.WrapErrf(err, "Failed to destroy stack")
	}

	log.Infof("Successfully destroyed stack %s", stackName)

	log.Infof("Removing stack %s", stackName)
	err = s.Workspace().RemoveStack(ctx, stackName)
	if err != nil {
		return errors2.WrapErrf(err, "Failed to remove stack")
	}
	return nil
}

func InstallDependencies(ctx context.Context, stackDirectory string) error {
	prog := tui.GetProgress(ctx)
	log := logging.GetLogger(ctx).Sugar()
	log.Debugf("Installing pulumi dependencies in %s", stackDirectory)
	prog.UpdateIndeterminate("Installing pulumi packages")
	npmCmd := logging.Command(
		ctx,
		logging.CommandLogger{
			RootLogger:  log.Desugar().Named("npm"),
			StdoutLevel: zap.DebugLevel,
		},
		// loglevel silly is required for the NpmProgress to capture all logs
		"npm", "install", "--loglevel", "silly", "--no-fund", "--no-audit",
	)
	npmProg := &NpmProgress{Progress: prog}
	npmCmd.Stdout = io.MultiWriter(npmCmd.Stdout, npmProg)
	npmCmd.Stderr = io.MultiWriter(npmCmd.Stderr, npmProg)
	npmCmd.Dir = stackDirectory
	return npmCmd.Run()
}
