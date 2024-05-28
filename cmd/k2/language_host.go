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
	pb.UnimplementedExampleServiceServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received SayHello request with name: %s", in.Name)
	return &pb.HelloReply{Message: "Hello, " + in.Name}, nil
}

func (s *server) GetPythonResponse(ctx context.Context, in *pb.PythonRequest) (*pb.PythonReply, error) {
	log.Printf("Received GetPythonResponse request with query: %s", in.Query)
	return &pb.PythonReply{Response: "Python Response to: " + in.Query}, nil
}

func startGRPCServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterExampleServiceServer(s, &server{})

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
func healthCheck(addr string) bool {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
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

func startPythonClient(f string) {
	cmd := exec.Command("python3", f)

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
