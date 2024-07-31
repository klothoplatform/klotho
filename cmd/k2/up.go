package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/k2/language_host"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"github.com/klothoplatform/klotho/pkg/engine/debug"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var upConfig struct {
	stateDir  string
	region    string
	debugMode string
	debugPort int
}

func newUpCmd() *cobra.Command {
	var upCommand = &cobra.Command{
		Use:   "up",
		Short: "Run the up command",
		RunE:  up,
	}
	flags := upCommand.Flags()
	flags.StringVar(&upConfig.stateDir, "state-directory", "", "State directory")
	flags.StringVarP(&upConfig.region, "region", "r", "us-west-2", "AWS region")
	flags.StringVar(&upConfig.debugMode, "debug", "", "Debug mode")
	flags.IntVar(&upConfig.debugPort, "debug-port", 5678, "Language Host Debug port")
	return upCommand
}

func up(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	inputPath := absolutePath
	ctx := cmd.Context()

	if upConfig.stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		upConfig.stateDir = filepath.Join(homeDir, ".k2")
	}

	if debugDir := debug.GetDebugDir(ctx); debugDir == "" {
		ctx = debug.WithDebugDir(ctx, upConfig.stateDir)
		cmd.SetContext(ctx)
	}

	log := logging.GetLogger(ctx).Sugar()

	var langHost language_host.LanguageHost
	err = langHost.Start(ctx, language_host.DebugConfig{
		Port: upConfig.debugPort,
		Mode: upConfig.debugMode,
	}, filepath.Dir(inputPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := langHost.Close(); err != nil {
			log.Warnf("Error closing language host", zap.Error(err))
		}
	}()

	ir, err := langHost.GetIR(ctx, &pb.IRRequest{Filename: inputPath})
	if err != nil {
		return fmt.Errorf("error getting IR: %w", err)
	}

	// Take the IR -- generate and save a state file and stored in the
	// output directory, the path should include the environment name and
	// the project URN
	appUrnPath, err := model.UrnPath(ir.AppURN)
	if err != nil {
		return fmt.Errorf("error getting URN path: %w", err)
	}
	appDir := filepath.Join(upConfig.stateDir, appUrnPath)

	// Create the app state directory
	if err = os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("error creating app directory: %w", err)
	}

	stateFile := filepath.Join(appDir, "state.yaml")

	osfs := afero.NewOsFs()

	sm := model.NewStateManager(osfs, stateFile)

	if !sm.CheckStateFileExists() {
		sm.InitState(ir)
		if err = sm.SaveState(); err != nil {
			return fmt.Errorf("error saving state: %w", err)
		}
	} else {
		if err = sm.LoadState(); err != nil {
			return fmt.Errorf("error loading state: %w", err)
		}
	}

	o, err := orchestration.NewUpOrchestrator(sm, langHost.NewClient(), osfs, appDir)
	if err != nil {
		return fmt.Errorf("error creating up orchestrator: %w", err)
	}

	err = o.RunUpCommand(ctx, ir, model.DryRun(commonCfg.dryRun), semaphore.NewWeighted(5))
	if err != nil {
		return fmt.Errorf("error running up command: %w", err)
	}

	return nil
}
