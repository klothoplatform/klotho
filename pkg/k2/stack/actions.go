package stack

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/tui"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

type Reference struct {
	ConstructURN model.URN
	Name         string
	IacDirectory string
	AwsRegion    string
}

func Initialize(ctx context.Context, fs afero.Fs, projectName string, stackName string, stackDirectory string) (StackInterface, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("Failed to get user home directory: %w", err)
	}
	pulumiHomeDir := filepath.Join(homeDir, ".k2", "pulumi")

	if exists, err := afero.DirExists(fs, pulumiHomeDir); !exists || err != nil {
		if err := fs.MkdirAll(pulumiHomeDir, 0755); err != nil {
			return nil, fmt.Errorf("Failed to create pulumi home directory: %w", err)
		}
	}

	stateDir := filepath.Join(pulumiHomeDir, "state")
	if exists, err := afero.DirExists(fs, stateDir); !exists || err != nil {
		if err := fs.MkdirAll(stateDir, 0755); err != nil {
			return nil, fmt.Errorf("Failed to create stack state directory: %w", err)
		}
	}

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
	stack, err := auto.UpsertStackLocalSource(ctx, stackName, stackDirectory, proj, envvars, auto.PulumiHome(pulumiHomeDir), secretsProvider)
	if err != nil {
		return nil, fmt.Errorf("Failed to create or select stack: %w", err)
	}
	return &stack, nil
}

func RunUp(ctx context.Context, fs afero.Fs, stackReference Reference) (*auto.UpResult, *State, error) {
	log := logging.GetLogger(ctx).Named("pulumi.up").Sugar()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory

	s, err := Initialize(ctx, fs, "myproject", stackName, stackDirectory)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create or select stack: %w", err)
	}
	log.Debugf("Created/Selected stack %q", stackName)

	err = InstallDependencies(ctx, stackDirectory)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to install dependencies: %w", err)
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to set stack configuration: %w", err)
	}

	log.Debug("Starting update")

	upResult, err := s.Up(
		ctx,
		optup.ProgressStreams(logging.NewLoggerWriter(log.Desugar(), zap.InfoLevel)),
		optup.EventStreams(Events(ctx, "Deploying")),
		optup.Refresh(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to update stack: %w", err)
	}

	log.Infof("Successfully deployed stack %s", stackName)

	stackState, err := GetState(ctx, s)
	return &upResult, &stackState, err
}

func RunPreview(ctx context.Context, fs afero.Fs, stackReference Reference) (*auto.PreviewResult, error) {
	log := logging.GetLogger(ctx).Named("pulumi.preview").Sugar()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory

	s, err := Initialize(ctx, fs, "myproject", stackName, stackDirectory)
	if err != nil {
		return nil, fmt.Errorf("Failed to create or select stack: %w", err)
	}
	log.Infof("Created/Selected stack %q", stackName)

	err = InstallDependencies(ctx, stackDirectory)
	if err != nil {
		return nil, fmt.Errorf("Failed to install dependencies: %w", err)
	}

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return nil, fmt.Errorf("Failed to set stack configuration: %w", err)
	}

	log.Debug("Starting preview")

	previewResult, err := s.Preview(
		ctx,
		optpreview.ProgressStreams(logging.NewLoggerWriter(log.Desugar(), zap.InfoLevel)),
		optpreview.EventStreams(Events(ctx, "Previewing")),
		optpreview.Refresh(),
	)

		str := err.Error()
		// Use the first line only, the rest of it is redundant with the first line or the live logging already shown
		firstLine := strings.Split(str, "\n")[0]
		
		if auto.IsCompilationError(err) || auto.IsRuntimeError(err) || auto.IsCreateStack409Error(err) {
			return nil, fmt.Errorf("Failed to preview stack: %s", firstLine)
		}

		log.Warnf("Failed to preview stack %s: %s", stackName, firstLine)

		// Don't return an error for preview failures so that futher previewing can proceed
		return nil, nil
	}

	log.Infof("Successfully previewed stack %s", stackName)

	return &previewResult, nil
}

func RunDown(ctx context.Context, fs afero.Fs, stackReference Reference) error {
	log := logging.GetLogger(ctx).Named("pulumi.destroy").Sugar()

	stackName := stackReference.Name
	stackDirectory := stackReference.IacDirectory
	s, err := Initialize(ctx, fs, "myproject", stackName, stackDirectory)
	if err != nil {
		return fmt.Errorf("Failed to create or select stack: %w", err)
	}

	log.Debugf("Created/Selected stack %q", stackName)

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: stackReference.AwsRegion})
	if err != nil {
		return fmt.Errorf("Failed to set stack configuration: %w", err)
	}

	log.Debug("Starting destroy")

	// wire up our destroy to stream progress to stdout
	stdoutStreamer := optdestroy.ProgressStreams(logging.NewLoggerWriter(log.Desugar(), zap.InfoLevel))
	refresh := optdestroy.Refresh()
	eventStream := optdestroy.EventStreams(Events(ctx, "Destroying"))

	// run the destroy to remove our resources
	_, err = s.Destroy(ctx, stdoutStreamer, eventStream, refresh)
	if err != nil {
		return fmt.Errorf("Failed to destroy stack: %w", err)
	}

	log.Infof("Successfully destroyed stack %s", stackName)

	log.Infof("Removing stack %s", stackName)
	err = s.Workspace().RemoveStack(ctx, stackName)
	if err != nil {
		return fmt.Errorf("Failed to remove stack: %w", err)
	}
	return nil
}

func InstallDependencies(ctx context.Context, stackDirectory string) error {
	prog := tui.GetProgress(ctx)
	log := logging.GetLogger(ctx).Named("npm").Sugar()

	log.Debugf("Installing pulumi dependencies in %s", stackDirectory)
	prog.UpdateIndeterminate("Installing pulumi packages")
	npmCmd := logging.Command(
		ctx,
		logging.CommandLogger{
			RootLogger:  log.Desugar(),
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
