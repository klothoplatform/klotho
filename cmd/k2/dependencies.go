package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/klothoplatform/klotho/pkg/multierr"
	pulumi "github.com/pulumi/pulumi/sdk/v3/go/auto"
)

type (
	CliDependency       string
	CliDependencyConfig struct {
		Dependency CliDependency
		Optional   bool
	}
)

const (
	CliDependencyDocker CliDependency = "docker"
	CliDependencyPulumi CliDependency = "pulumi"
)

// InstallDependencies installs the dependencies specified in the configs
func InstallDependencies(configs []CliDependencyConfig) error {
	var err multierr.Error
	for _, config := range configs {
		switch config.Dependency {
		case CliDependencyDocker:
			if isDockerInstalled() {
				continue
			}
			err.Append(installDocker())
		case CliDependencyPulumi:
			if isPulumiInstalled() {
				continue
			}
			err.Append(installPulumi())
		}
	}
	return err
}

func installDocker() error {
	// Install docker
	installUrl := ""
	switch runtime.GOOS {
	case "darwin":
		installUrl = "https://docs.docker.com/desktop/install/mac-install/"
	case "linux":
		installUrl = "https://docs.docker.com/desktop/install/linux-install/"
	case "windows":
		installUrl = "https://docs.docker.com/desktop/install/windows-install/"
	default:
		return errors.New("unsupported OS")
	}
	return fmt.Errorf("install docker from %s", installUrl)
}

func installPulumi() error {
	// Install pulumi
	ctx := context.Background()
	_, err := pulumi.InstallPulumiCommand(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to install pulumi: %w", err)
	}
	return nil
}

func isDockerInstalled() bool {
	// Check if docker is installed
	cmd := exec.Command("docker", "--version")
	err := cmd.Run()
	return err == nil
}

func isPulumiInstalled() bool {
	_, err := pulumi.NewPulumiCommand(nil)
	return err == nil
}
