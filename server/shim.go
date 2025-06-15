package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	task "kettle/api/shim"
)

type TaskServiceImpl struct{}

// is run by containerd daemon to call the shim binary
func runShim(id string) (pid uint32, err error) {
	cmdDelete := exec.Command("kettle-shim", "--id", id)
	cmdDelete.Stdout = os.Stdout
	cmdDelete.Stderr = os.Stderr
	if err := cmdDelete.Run(); err != nil {
		return 0, fmt.Errorf("failed to create container shim: %w", err)
	}
	return pid, nil
}

// used by kettle shim to initialize itself
func StartShim(id string) (pid uint32, err error) {
	CreateTTRPCServer(context.TODO(), "/run/kettle/containers/+"+id+"/"+id+"ttrpc.sock")
	return pid, nil
}
func (s TaskServiceImpl) Start(ctx context.Context, req *task.StartRequest) (*task.StartResponse, error) {
	log.Printf("Received Start request for ID: %s\n", req.ContainerId)
	startContainer(req.ContainerId)

	// Call your actual function here

	return &task.StartResponse{}, nil
}

func (s TaskServiceImpl) Delete(ctx context.Context, req *task.DeleteRequest) (*task.DeleteResponse, error) {
	log.Printf("Received Delete request for ID: %s\n", req.Id)
	cmdDelete := exec.Command("runc", "delete", req.Id)
	cmdDelete.Stdout = os.Stdout
	cmdDelete.Stderr = os.Stderr
	if err := cmdDelete.Run(); err != nil {
		return nil, fmt.Errorf("failed to delete container: %w", err)
	}

	return &task.DeleteResponse{}, nil
}

func startContainer(id string) {
	cmdStart := exec.Command("runc", "start", id)
	cmdStart.Stdout = os.Stdout
	cmdStart.Stderr = os.Stderr
	if err := cmdStart.Run(); err != nil {
		return
	}
	fmt.Println("Started container:", id)
}
