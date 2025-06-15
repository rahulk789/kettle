package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	containerTask "kettle/api/kettle"
	shimTask "kettle/api/shim"
)

type ContainerTaskServiceImpl struct {
	containerTask.UnimplementedContainersServer
}

func (s *ContainerTaskServiceImpl) Create(ctx context.Context, req *containerTask.CreateContainerRequest) (*containerTask.CreateContainerResponse, error) {
	fmt.Println("function create called on grpc")
	err := createContainer(req.Container.Bundle, req.Container.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	return &containerTask.CreateContainerResponse{}, nil
}

func (s *ContainerTaskServiceImpl) Start(ctx context.Context, req *containerTask.StartRequest) (*containerTask.StartResponse, error) {
	fmt.Println("function start called on grpc")
	pid, err := runShim(req.ContainerId)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	startReq := shimTask.StartRequest{
		ContainerId: req.ContainerId,
	}
	TaskServiceImpl.Start(TaskServiceImpl{}, ctx, &startReq)
	return &containerTask.StartResponse{Pid: pid}, nil
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

func createBundle(bundlePath string) error {
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return fmt.Errorf("failed to create bundle directory: %w", err)
	}
	cmd := exec.Command("runc", "spec")
	cmd.Dir = bundlePath // Set working directory to bundle path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate default spec: %w", err)
	}
	if err := os.MkdirAll(bundlePath+"/rootfs", 0755); err != nil {
		return fmt.Errorf("failed to create bundle rootfs directory: %w", err)
	}
	fmt.Println("Default runc spec created at:", bundlePath+"/config.json")

	return nil
}
