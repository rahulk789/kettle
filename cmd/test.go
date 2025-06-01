package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/containerd/containerd/api/services/containers/v1"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	socketPath string
	shimPath   string
	rootDir    string
)

// SimpleContainersServer implements only the Create method of containers.ContainersServer
type SimpleContainersServer struct {
	containers.UnimplementedContainersServer
	containers map[string]*containers.Container
}

// NewSimpleContainersServer creates a new SimpleContainersServer
func NewSimpleContainersServer() *SimpleContainersServer {
	return &SimpleContainersServer{
		containers: make(map[string]*containers.Container),
	}
}

// Create implements the Create method of the containers.ContainersServer interface
func (s *SimpleContainersServer) Create(ctx context.Context, req *containers.CreateContainerRequest) (*containers.CreateContainerResponse, error) {
	log.Printf("Received Create request for container ID: %s", req.Container.ID)

	if req.Container.ID == "" {
		return nil, errdefs.ErrInvalidArgument
	}

	namespace, ok := namespaces.Namespace(ctx)
	if !ok {
		namespace = "default"
	}

	// Check if container already exists
	key := fmt.Sprintf("%s/%s", namespace, req.Container.ID)
	if _, exists := s.containers[key]; exists {
		return nil, errdefs.ErrAlreadyExists
	}

	// Create container in our internal map
	container := &containers.Container{
		ID:         req.Container.ID,
		Labels:     req.Container.Labels,
		Image:      req.Container.Image,
		Runtime:    req.Container.Runtime,
		Spec:       req.Container.Spec,
		Extensions: req.Container.Extensions,
		CreatedAt:  timestamppb.New(time.Now()),
		UpdatedAt:  timestamppb.New(time.Now()),
	}
	s.containers[key] = container

	// Start the shim process
	if err := startShim(ctx, container.ID, namespace); err != nil {
		delete(s.containers, key)
		return nil, fmt.Errorf("failed to start shim: %w", err)
	}

	return &containers.CreateContainerResponse{Container: container}, nil
}

// startShim starts a new containerd shim process for the given container ID
func startShim(ctx context.Context, id, namespace string) error {
	log.Printf("Starting shim for container %s in namespace %s", id, namespace)

	// Create a directory for the container
	containerDir := filepath.Join(rootDir, namespace, id)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("failed to create container directory: %w", err)
	}

	// Shim socket path
	shimSocketPath := filepath.Join(containerDir, "shim.sock")

	// Prepare the shim command
	cmd := exec.CommandContext(ctx, shimPath,
		"--namespace", namespace,
		"--id", id,
		"--address", shimSocketPath,
	)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the shim process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shim process: %w", err)
	}

	// Don't wait for the process as it should run in the background
	log.Printf("Shim process started with PID %d", cmd.Process.Pid)

	return nil
}

func startServer() error {
	// Create a new gRPC server
	server := grpc.NewServer()

	// Register the containers service
	containers.RegisterContainersServer(server, NewSimpleContainersServer())

	// Create directory for the socket if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket file if it exists
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Listen on the Unix socket
	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	log.Printf("Server listening on %s", socketPath)
	return server.Serve(lis)
}

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "mini-containerd",
		Short: "A minimal containerd gRPC server with create container functionality",
		Run: func(cmd *cobra.Command, args []string) {
			if err := startServer(); err != nil {
				log.Fatalf("Failed to start server: %v", err)
			}
		},
	}

	// Define flags
	rootCmd.PersistentFlags().StringVar(&socketPath, "socket", "/run/mini-containerd/mini-containerd.sock", "Path to the UNIX socket")
	rootCmd.PersistentFlags().StringVar(&shimPath, "shim", "/usr/local/bin/containerd-shim-runc-v2", "Path to the containerd shim binary")
	rootCmd.PersistentFlags().StringVar(&rootDir, "root", "/var/lib/mini-containerd", "Root directory for container data")

	// Create a client command for testing
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "Test client for the mini-containerd server",
		Run: func(cmd *cobra.Command, args []string) {
			testClient(socketPath)
		},
	}
	rootCmd.AddCommand(clientCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// testClient is a simple function to test the server
func testClient(socketPath string) {
	ctx := namespaces.WithNamespace(context.Background(), "default")

	// Connect to the server
	conn, err := grpc.Dial(
		fmt.Sprintf("unix://%s", socketPath),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Create a client
	client := containers.NewContainersClient(conn)

	// Create a container
	containerID := "test-container-" + fmt.Sprintf("%d", time.Now().Unix())

	req := &containers.CreateContainerRequest{
		Container: &containers.Container{
			ID:      containerID,
			Labels:  map[string]string{"app": "test"},
			Image:   "docker.io/library/alpine:latest",
			Runtime: &containers.Container_Runtime{Name: "io.containerd.runc.v2"},
		},
	}

	resp, err := client.Create(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	log.Printf("Container created successfully: %s", resp.Container.ID)
}
