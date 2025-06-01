/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	task "kettle/api/shim"
	"log"
	"net"
	"os"
	"path/filepath"

	"os/exec"

	"github.com/containerd/ttrpc"
	"github.com/spf13/cobra"
)

type TaskServiceImpl struct{}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create called")
		createTTRPCServer(context.TODO())
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
func getTaskClient(ctx context.Context) (task.TaskService, error) {
	socketPath := "/run/kettle/kettle.sock.ttrpc"
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	ttrpcClient := ttrpc.NewClient(conn)
	return task.NewTaskClient(ttrpcClient), nil
}

func createTTRPCServer(ctx context.Context) error {
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
	fmt.Println("✅ Container created:", containerID)
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
