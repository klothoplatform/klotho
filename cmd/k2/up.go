package main

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"time"

	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upConfig struct {
	inputPath  string
	outputPath string
	region     string
	debugMode  string
	debugPort  int
}

func newUpCmd() *cobra.Command {
	var upCommand = &cobra.Command{
		Use:   "up",
		Short: "Run the up command",
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				zap.L().Error("Invalid file path")
				os.Exit(1)
			}
			absolutePath, err := filepath.Abs(filePath)
			if err != nil {
				zap.L().Error("couldn't convert to absolute path")
				os.Exit(1)
			}
			upConfig.inputPath = absolutePath

			if upConfig.outputPath == "" {
				upConfig.outputPath = filepath.Join(filepath.Dir(absolutePath), ".k2")
			}

			err = updCmd(upConfig)
			if err != nil {
				zap.L().Error("error running up command", zap.Error(err))
				os.Exit(1)
			}
		},
	}
	flags := upCommand.Flags()
	flags.StringVarP(&upConfig.outputPath, "output", "o", "", "Output directory")
	flags.StringVarP(&upConfig.region, "region", "r", "us-west-2", "AWS region")
	flags.StringVarP(&upConfig.debugMode, "debug", "d", "", "Debug mode")
	flags.IntVarP(&upConfig.debugPort, "debug-port", "p", 5678, "Language Host Debug port")
	return upCommand
}

func updCmd(args struct {
	inputPath  string
	outputPath string
	region     string
	debugMode  string
	debugPort  int
}) error {
	cmd, addr := startPythonClient(DebugConfig{
		Enabled: args.debugMode != "",
		Port:    args.debugPort,
		Mode:    args.debugMode,
	})
	var err error
	defer func() {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				zap.L().Warn("failed to kill Python client", zap.Error(err))
			}
		}
	}()

	// Wait for the server to be ready
	zap.L().Info("Waiting for Python server to start")
	if args.debugMode != "" {
		// Don't add a timeout in case there are breakpoints in the language host before an address is printed
		<-addr.HasAddr
	} else {
		select {
		case <-addr.HasAddr:
		case <-time.After(30 * time.Second):
			return errors.New("timeout waiting for Python server to start")
		}
	}
	// Connect to the Python server
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

	// Send IR Request
	ctx := context.Background()
	if args.debugMode == "" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*10)
		defer cancel()
	}

	req := &pb.IRRequest{Filename: args.inputPath}
	res, err := client.SendIR(ctx, req)
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
	appDir := filepath.Join(args.outputPath, appUrnPath)

	// Create the app state directory
	if err = os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("error creating app directory: %w", err)
	}

	stateFile := filepath.Join(appDir, "state.yaml")

	// Create a new state manager
	sm := model.NewStateManager(stateFile)

	// Initialize the state if it doesn't exist
	if !sm.CheckStateFileExists() {
		sm.InitState(ir)
		// Save the state
		if err = sm.SaveState(); err != nil {
			return fmt.Errorf("error saving state: %w", err)
		}
	} else {
		// Load the state
		if err = sm.LoadState(); err != nil {
			return fmt.Errorf("error loading state: %w", err)
		}
	}

	o := orchestration.NewUpOrchestrator(sm, client, appDir)

	err = o.RunUpCommand(ir, commonCfg.dryRun)
	if err != nil {
		return fmt.Errorf("error running up command: %w", err)
	}

	return err
}
