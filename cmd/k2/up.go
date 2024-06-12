package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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

			fmt.Println(updCmd(upConfig))
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
}) string {
	cmd := startPythonClient()
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Fatalf("failed to kill Python client: %v", err)
		}
	}()

	// Connect to the Python server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	client := pb.NewKlothoServiceClient(conn)

	// Wait for the server to be ready
	if err := waitForServer(client, 10, 1*time.Second); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	// Send IR Request
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.IRRequest{Filename: args.inputPath}
	res, err := client.SendIR(ctx, req)
	if err != nil {
		log.Fatalf("could not execute script: %v", err)
	}

	ir, err := model.ParseIRFile([]byte(res.GetYamlPayload()))
	if err != nil {
		return fmt.Sprintf("InputStatusError reading IR file: %s", err)
	}

	// Take the IR -- generate and save a state file and stored in the
	// output directory, the path should include the environment name and
	// the project URN

	appUrn, err := model.ParseURN(ir.AppURN)
	if err != nil {
		return fmt.Sprintf("InputStatusError parsing app URN: %s", err)
	}

	appUrnPath, err := model.UrnPath(*appUrn)
	if err != nil {
		return fmt.Sprintf("InputStatusError getting URN path: %s", err)
	}
	appDir := filepath.Join(args.outputPath, appUrnPath)

	// Create the app state directory
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Sprintf("InputStatusError creating app directory: %s", err)
	}

	stateFile := filepath.Join(appDir, "state.yaml")

	// Create a new state manager
	sm := model.NewStateManager(stateFile)

	// Initialize the state if it doesn't exist
	if !sm.CheckStateFileExists() {
		sm.InitState(ir)
		// Save the state
		if err = sm.SaveState(); err != nil {
			return fmt.Sprintf("InputStatusError saving state: %s", err)
		}
	} else {
		// Load the state
		if err = sm.LoadState(); err != nil {
			return fmt.Sprintf("InputStatusError loading state: %s", err)
		}
	}

	o := orchestration.NewOrchestrator(sm, client, appDir)

	err = o.RunUpCommand(ir, commonCfg.dryRun)
	if err != nil {
		zap.S().Errorf("InputStatusError running up command: %s", err)
		return fmt.Sprintf("InputStatusError running up command: %s", err)
	}

	return "success"
}
