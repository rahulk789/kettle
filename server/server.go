package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	containerTask "kettle/api/kettle"
	task "kettle/api/shim"

	"github.com/containerd/ttrpc"
	"google.golang.org/grpc"
)

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

func CreateTTRPCServer(ctx context.Context, socketPath string) error {
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
