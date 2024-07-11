package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/klothoplatform/klotho/pkg/k2/language_host"

	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/engine/debug"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upConfig struct {
	outputPath string
	region     string
	debugMode  string
	debugPort  int
}

func newUpCmd() *cobra.Command {
	var upCommand = &cobra.Command{
		Use:   "up",
		Short: "Run the up command",
		RunE:  up,
	}
	flags := upCommand.Flags()
	flags.StringVarP(&upConfig.outputPath, "output", "o", "", "Output directory")
	flags.StringVarP(&upConfig.region, "region", "r", "us-west-2", "AWS region")
	flags.StringVarP(&upConfig.debugMode, "debug", "d", "", "Debug mode")
	flags.IntVarP(&upConfig.debugPort, "debug-port", "p", 5678, "Language Host Debug port")
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

	if upConfig.outputPath == "" {
		upConfig.outputPath = filepath.Join(filepath.Dir(absolutePath), ".k2")
	}

	if debugDir := debug.GetDebugDir(ctx); debugDir == "" {
		ctx = debug.WithDebugDir(ctx, upConfig.outputPath)
		cmd.SetContext(ctx)
	}

	langHost, addr, err := language_host.StartPythonClient(ctx, language_host.DebugConfig{
		Enabled: upConfig.debugMode != "",
		Port:    upConfig.debugPort,
		Mode:    upConfig.debugMode,
	})
	if err != nil {
		return err
	}

	defer func() {
		if err := langHost.Process.Kill(); err != nil {
			zap.L().Warn("failed to kill Python client", zap.Error(err))
		}
	}()

	log := logging.GetLogger(cmd.Context()).Sugar()

	log.Debug("Waiting for Python server to start")
	if upConfig.debugMode != "" {
		// Don't add a timeout in case there are breakpoints in the language host before an address is printed
		<-addr.HasAddr
	} else {
		select {
		case <-addr.HasAddr:
		case <-time.After(30 * time.Second):
			return errors.New("timeout waiting for Python server to start")
		}
	}
	conn, err := grpc.NewClient(addr.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to Python server: %w", err)
	}

	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		if err != nil {
			zap.L().Error("failed to close connection", zap.Error(err))
		}
	}(conn)

	client := pb.NewKlothoServiceClient(conn)

	// make sure the ctx used later doesn't have the timeout (which is only for the IR request)
	irCtx := ctx
	if upConfig.debugMode == "" {
		var cancel context.CancelFunc
		irCtx, cancel = context.WithTimeout(irCtx, time.Second*10)
		defer cancel()
	}

	req := &pb.IRRequest{Filename: inputPath}
	res, err := client.SendIR(irCtx, req)
	if err != nil {
		return fmt.Errorf("error sending IR request: %w", err)
	}

	ir, err := model.ParseIRFile([]byte(res.GetYamlPayload()))
	if err != nil {
		return fmt.Errorf("error parsing IR file: %w", err)
	}

	// Take the IR -- generate and save a state file and stored in the
	// output directory, the path should include the environment name and
	// the project URN
	appUrn, err := model.ParseURN(ir.AppURN)
	if err != nil {
		return fmt.Errorf("error parsing app URN: %w", err)
	}

	appUrnPath, err := model.UrnPath(*appUrn)
	if err != nil {
		return fmt.Errorf("error getting URN path: %w", err)
	}
	appDir := filepath.Join(upConfig.outputPath, appUrnPath)

	// Create the app state directory
	if err = os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("error creating app directory: %w", err)
	}

	stateFile := filepath.Join(appDir, "state.yaml")

	sm := model.NewStateManager(afero.NewOsFs(), stateFile)

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

	o, err := orchestration.NewUpOrchestrator(sm, client, appDir)
	if err != nil {
		return fmt.Errorf("error creating up orchestrator: %w", err)
	}

	err = o.RunUpCommand(ctx, ir, commonCfg.dryRun, 5)
	if err != nil {
		return fmt.Errorf("error running up command: %w", err)
	}

	return nil
}
