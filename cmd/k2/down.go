package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/klothoplatform/klotho/pkg/engine/debug"
	"github.com/klothoplatform/klotho/pkg/k2/language_host"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var downConfig struct {
	stateDir  string
	debugMode string
	debugPort int
}

func newDownCmd() *cobra.Command {
	downCommand := &cobra.Command{
		Use:   "down",
		Short: "Run the down command",
		RunE:  down,
	}
	flags := downCommand.Flags()
	flags.StringVar(&downConfig.stateDir, "state-directory", "", "State directory")
	flags.StringVar(&downConfig.debugMode, "debug", "", "Debug mode")
	flags.IntVar(&downConfig.debugPort, "debug-port", 5678, "Language Host Debug port")
	return downCommand

}

func getProjectPath(ctx context.Context, inputPath string) (string, error) {
	langHost, addr, err := language_host.StartPythonClient(ctx, language_host.DebugConfig{
		Port: downConfig.debugPort,
		Mode: downConfig.debugMode,
	}, filepath.Dir(inputPath))
	if err != nil {
		return "", err
	}

	defer func() {
		if err := langHost.Process.Kill(); err != nil {
			zap.L().Warn("failed to kill Python client", zap.Error(err))
		}
	}()

	log := logging.GetLogger(ctx).Sugar()

	log.Debug("Waiting for Python server to start")
	if downConfig.debugMode != "" {
		// Don't add a timeout in case there are breakpoints in the language host before an address is printed
		<-addr.HasAddr
	} else {
		select {
		case <-addr.HasAddr:
		case <-time.After(30 * time.Second):
			return "", errors.New("timeout waiting for Python server to start")
		}
	}
	conn, err := grpc.NewClient(addr.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Python server: %w", err)
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
	if downConfig.debugMode == "" {
		var cancel context.CancelFunc
		irCtx, cancel = context.WithTimeout(irCtx, time.Second*10)
		defer cancel()
	}

	req := &pb.IRRequest{Filename: inputPath}
	res, err := client.SendIR(irCtx, req)
	if err != nil {
		return "", fmt.Errorf("error sending IR request: %w", err)
	}

	ir, err := model.ParseIRFile([]byte(res.GetYamlPayload()))
	if err != nil {
		return "", fmt.Errorf("error parsing IR file: %w", err)
	}

	appUrnPath, err := model.UrnPath(ir.AppURN)
	if err != nil {
		return "", fmt.Errorf("error getting URN path: %w", err)
	}
	return appUrnPath, nil
}

func down(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	var projectPath string
	switch len(args) {
	case 1:
		projectPath, err = getProjectPath(cmd.Context(), absolutePath)
		if err != nil {
			return fmt.Errorf("error getting project path: %w", err)
		}

	case 4:
		project := args[1]
		app := args[2]
		env := args[3]
		projectPath = filepath.Join(project, app, env)

	default:
		return fmt.Errorf("invalid number of arguments (%d) expected 4", len(args))
	}

	if downConfig.stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		downConfig.stateDir = filepath.Join(homeDir, ".k2")
	}

	debugDir := debug.GetDebugDir(cmd.Context())
	if debugDir == "" {
		debugDir = downConfig.stateDir
		cmd.SetContext(debug.WithDebugDir(cmd.Context(), debugDir))
	}

	stateFile := filepath.Join(downConfig.stateDir, projectPath, "state.yaml")

	osfs := afero.NewOsFs()
	sm := model.NewStateManager(osfs, stateFile)

	if !sm.CheckStateFileExists() {
		return fmt.Errorf("state file does not exist: %s", stateFile)
	}

	err = sm.LoadState()
	if err != nil {
		return fmt.Errorf("error loading state: %w", err)
	}

	var stackReferences []stack.Reference
	for name, construct := range sm.GetAllConstructs() {
		constructPath := filepath.Join(downConfig.stateDir, projectPath, name)
		stackReference := stack.Reference{
			ConstructURN: *construct.URN,
			Name:         name,
			IacDirectory: constructPath,
		}
		stackReferences = append(stackReferences, stackReference)
	}

	o := orchestration.NewDownOrchestrator(sm, osfs, downConfig.stateDir)
	err = o.RunDownCommand(
		cmd.Context(),
		orchestration.DownRequest{StackReferences: stackReferences, DryRun: model.DryRun(commonCfg.dryRun)},
		5,
	)
	if err != nil {
		return fmt.Errorf("error running down command: %w", err)
	}
	return nil
}
