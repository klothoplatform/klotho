package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type server struct {
	pb.UnimplementedKlothoServiceServer
}

func (s *server) SendIR(ctx context.Context, in *pb.IRRequest) (*pb.IRReply, error) {
	log.Printf("Received SendIR request with error: %v, yaml_payload: %s", in.Error, in.YamlPayload)
	return &pb.IRReply{Message: "IR received successfully"}, nil
}

func startGRPCServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterKlothoServiceServer(s, &server{})

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
func healthCheck(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func waitForServer(addr string, retries int, delay time.Duration) error {
	for i := 0; i < retries; i++ {
		if healthCheck(addr) {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("server not available after %d retries", retries)
}

func startPythonClient(infraScript string) {
	cmd := exec.Command("pipenv", "run", "python3", "python_language_host.py", infraScript)
	cmd.Dir = "pkg/k2/language_host/python" // Set the working directory to the directory containing the script

	// Wire stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start Python client: %v", err)
	}
	log.Println("Python client started")

	// Wait for the Python client to finish
	if err := cmd.Wait(); err != nil {
		log.Fatalf("Python client exited with error: %v", err)
	}
}
