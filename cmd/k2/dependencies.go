package main

import (
	"context"
	"errors"
	"runtime"

	errors2 "github.com/klothoplatform/klotho/pkg/errors"
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
			err.Append(installDocker(config))
		case CliDependencyPulumi:
			if isPulumiInstalled() {
				continue
			}
			err.Append(installPulumi(config))
		}
	}
	return err
}

func installDocker(config CliDependencyConfig) error {
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
	return errors2.WrapErrf(errors.New("docker not installed"), "install docker from %s", installUrl)
}

func installPulumi(config CliDependencyConfig) error {
	// Install pulumi
	ctx := context.Background()
	_, err := pulumi.InstallPulumiCommand(ctx, nil)
	if err != nil {
		return errors2.WrapErrf(err, "failed to install pulumi")
	}
	return nil
}

func isDockerInstalled() bool {
	//TODO: Implement this

	// Check if docker is installed
	return true
}

func isPulumiInstalled() bool {
	_, err := pulumi.NewPulumiCommand(nil)
	return err == nil
}
