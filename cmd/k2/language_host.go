package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
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

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start Python client: %v", err)
	}
	log.Println("Python client started")

	go func() {
		cmd.Wait()
		log.Println("Python client exited")
	}()
	return cmd
}
