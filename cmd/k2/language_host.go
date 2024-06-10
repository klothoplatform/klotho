package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"go.uber.org/zap"
)

type ProgramContext struct {
	IRYaml string
}

// TODO: implement more robust context handling
// Global context for the program (spike implementation)
var programContext *ProgramContext = &ProgramContext{}

func healthCheck(client pb.KlothoServiceClient) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	return err == nil
}

func waitForServer(client pb.KlothoServiceClient, retries int, delay time.Duration) error {
	for i := 0; i < retries; i++ {
		if healthCheck(client) {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("server not available after %d retries", retries)
}

func startPythonClient() *exec.Cmd {
	cmd := exec.Command("pipenv", "run", "python", "python_language_host.py")
	cmd.Dir = "pkg/k2/language_host/python"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// spawn the python process as a subprocess of the CLI so it is guaranteed to be killed when the CLI exits
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		zap.S().Fatalf("failed to start Python client: %v", err)
	}
	zap.S().Info("Python client started")

	go func() {
		err := cmd.Wait()
		if err != nil {
			zap.S().Errorf("Python process exited with error: %v", err)
		} else {
			zap.L().Debug("Python process exited successfully")
		}
	}()
	return cmd
}
