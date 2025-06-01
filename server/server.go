package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	containerTask "kettle/api/kettle"
	task "kettle/api/shim"

	"github.com/containerd/ttrpc"
	"google.golang.org/grpc"
)

type TaskServiceImpl struct{}
type ContainerTaskServiceImpl struct {
	containerTask.UnimplementedContainersServer
}

func CreateTTRPCServer(ctx context.Context) error {
	socketPath := "/run/kettle/kettle.sock.ttrpc"
	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		fmt.Println("Failed to create directory:", err)
		return err
	}
	if err := os.RemoveAll(socketPath); err != nil {
		fmt.Println("Failed to remove existing socket:", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("Failed to create socket:", err)
		os.Exit(1)
	}

	server, err := ttrpc.NewServer()
	if err != nil {
		fmt.Println("Failed to create ttrpc server:", err)
		os.Exit(1)
	}
	// Register your service
	task.RegisterTaskService(server, &TaskServiceImpl{})
	fmt.Println(" ttrpc server started on", socketPath)

	if err := server.Serve(ctx, listener); err != nil {
		fmt.Println("Server stopped:", err)
	}
	return nil
}
func createBundle(bundlePath string) error {
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return fmt.Errorf("failed to create bundle directory: %w", err)
	}
	cmd := exec.Command("runc", "spec")
	cmd.Dir = bundlePath // Set working directory to bundle path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate default spec: %w", err)
	}

	fmt.Println("Default runc spec created at:", bundlePath+"/config.json")
	return nil
}

func createContainer(bundlePath, containerID string) error {
	createBundle(bundlePath)
	cmdCreate := exec.Command("runc", "create", "--bundle", bundlePath, containerID)
	cmdCreate.Stdout = os.Stdout
	cmdCreate.Stderr = os.Stderr
	if err := cmdCreate.Run(); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	fmt.Println("Container created:", containerID)
	return nil
}

func (s *TaskServiceImpl) Create(ctx context.Context, req *task.CreateTaskRequest) (*task.CreateTaskResponse, error) {
	log.Printf("Received Create request for ID: %s\n", req.Id)

	// Call your actual function here
	err := createContainer(req.Bundle, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return &task.CreateTaskResponse{}, nil
}
func (s *ContainerTaskServiceImpl) Create(ctx context.Context, req *containerTask.CreateContainerRequest) (*containerTask.CreateContainerResponse, error) {
	// Call your actual function here
	fmt.Println("function create called on grpc")
	err := createContainer(req.Container.Bundle, req.Container.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	return &containerTask.CreateContainerResponse{}, nil
}
func CreateGRPCServer(ctx context.Context) error {
	socketPath := "/run/kettle/kettle.sock"
	if err := os.MkdirAll("/run/kettle", 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	defer listener.Close()

	if err := os.Chmod(socketPath, 0666); err != nil {
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	server := grpc.NewServer()

	// Create and register your service
	containerTask.RegisterContainersServer(server, &ContainerTaskServiceImpl{}) // Note: usually ends with "Server"

	fmt.Println("gRPC server started on", socketPath)

	go func() {
		<-ctx.Done()
		fmt.Println("Shutting down gRPC server...")
		server.GracefulStop()
	}()

	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("server stopped with error: %w", err)
	}

	return nil
}
