package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	errors2 "github.com/klothoplatform/klotho/pkg/errors"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upConfig struct {
	inputPath  string
	outputPath string
	region     string
}

func newUpCmd() *cobra.Command {
	var upCommand = &cobra.Command{
		Use:   "up",
		Short: "Run the up command",
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Println("Invalid file path")
				os.Exit(1)
			}
			absolutePath, err := filepath.Abs(filePath)
			if err != nil {
				fmt.Println("couldn't convert to absolute path")
				os.Exit(1)
			}
			upConfig.inputPath = absolutePath

			if upConfig.outputPath == "" {
				upConfig.outputPath = filepath.Join(filepath.Dir(absolutePath), ".k2")
			}

			err = updCmd(upConfig)
			if err != nil {
				zap.S().Errorf("error running up command: %v", err)
				os.Exit(1)
			}
		},
	}
	flags := upCommand.Flags()
	flags.StringVarP(&upConfig.outputPath, "output", "o", "", "Output directory")
	flags.StringVarP(&upConfig.region, "region", "r", "us-west-2", "AWS region")
	return upCommand
}

func updCmd(args struct {
	inputPath  string
	outputPath string
	region     string
}) error {
	cmd, addr := startPythonClient()
	defer func() {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				zap.L().Warn("failed to kill Python client", zap.Error(err))
			}
		}
	}()

	// Wait for the server to be ready
	log.Print("Waiting for Python server to start")
	select {
	case <-addr.HasAddr:
	case <-time.After(30 * time.Second):
		return errors.New("timeout waiting for Python server to start")
	}

	// Connect to the Python server
	conn, err := grpc.NewClient(addr.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return errors2.WrapErrf(err, "failed to connect to Python server")
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			zap.L().Error("failed to close connection", zap.Error(err))
		}
	}(conn)

	client := pb.NewKlothoServiceClient(conn)

	// Send IR Request
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &pb.IRRequest{Filename: args.inputPath}
	res, err := client.SendIR(ctx, req)
	if err != nil {
		log.Fatalf("could not execute script: %v", err)
	}

	ir, err := model.ParseIRFile([]byte(res.GetYamlPayload()))
	if err != nil {
		return errors2.WrapErrf(err, "error parsing IR file")
	}

	// Take the IR -- generate and save a state file and stored in the
	// output directory, the path should include the environment name and
	// the project URN

	appUrn, err := model.ParseURN(ir.AppURN)
	if err != nil {
		return errors2.WrapErrf(err, "error parsing app URN")
	}

	appUrnPath, err := model.UrnPath(*appUrn)
	if err != nil {
		return errors2.WrapErrf(err, "error getting URN path")
	}
	appDir := filepath.Join(args.outputPath, appUrnPath)

	// Create the app state directory
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return errors2.WrapErrf(err, "error creating app directory")
	}

	stateFile := filepath.Join(appDir, "state.yaml")

	// Create a new state manager
	sm := model.NewStateManager(stateFile)

	// Initialize the state if it doesn't exist
	if !sm.CheckStateFileExists() {
		sm.InitState(ir)
		// Save the state
		if err = sm.SaveState(); err != nil {
			return errors2.WrapErrf(err, "error saving state")
		}
	} else {
		// Load the state
		if err = sm.LoadState(); err != nil {
			return errors2.WrapErrf(err, "error loading state")
		}
	}

	o := orchestration.NewUpOrchestrator(sm, client, appDir)

	err = o.RunUpCommand(ir, commonCfg.dryRun)
	if err != nil {
		return errors2.WrapErrf(err, "error running up command")
	}

	return nil
}
