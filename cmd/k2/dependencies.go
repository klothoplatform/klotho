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
	pulumiHome, err := pulumiHome()
	if err != nil {
		return err
	}

	_, err = pulumi.InstallPulumiCommand(ctx, &pulumi.PulumiCommandOptions{
		Root: pulumiHome,
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
	pulumiHome, err := pulumiHome()
	if err != nil {
		return false
	}
	_, err = pulumi.NewPulumiCommand(&pulumi.PulumiCommandOptions{
		Root: pulumiHome,
	})
	return err == nil
}

func pulumiHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".k2", "pulumi"), nil
}
