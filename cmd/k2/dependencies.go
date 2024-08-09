package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/logging"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	pulumi "github.com/pulumi/pulumi/sdk/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
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

type installFunc func(ctx context.Context) error

// InstallDependencies installs the dependencies specified in the configs
func InstallDependencies(ctx context.Context, configs []CliDependencyConfig) error {
	var err error

	var installers []installFunc

	for _, config := range configs {
		switch config.Dependency {
		case CliDependencyDocker:
			if isDockerInstalled() {
				continue
			}
			installers = append(installers, installDocker)
		case CliDependencyPulumi:
			if isPulumiInstalled() {
				continue
			}
			installers = append(installers, installPulumi)
		}
	}

	log := logging.GetLogger(ctx).Sugar()
	if len(installers) > 0 {
		log.Infof("Installing CLI dependencies...")
	}

	for _, installer := range installers {
		if e := installer(ctx); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func installDocker(ctx context.Context) error {
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

func installPulumi(ctx context.Context) error {
	installDir, err := pulumiInstallDir()
	if err != nil {
		return err
	}
	_, err = auto.InstallPulumiCommand(ctx, &auto.PulumiCommandOptions{
		Root: installDir,
	})

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
	installDir, err := pulumiInstallDir()
	if err != nil {
		return false
	}
	cmd, err := auto.NewPulumiCommand(&auto.PulumiCommandOptions{
		Root: installDir,
	})
	if err != nil {
		return false
	}
	// The installed version must be the same as the current SDK version
	return cmd.Version().EQ(pulumi.Version)
}

func pulumiInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".k2", "pulumi", "versions", pulumi.Version.String()), nil
}
