package client

import (
	"context"
	"fmt"
	containerTask "kettle/api/kettle"
	"net"
	"time"

	"github.com/containerd/containerd/api/runtime/task/v2"
	"github.com/containerd/ttrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetTTRPCTaskClient(ctx context.Context) (task.TaskService, error) {
	socketPath := "/run/kettle/kettle.sock.ttrpc"
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	ttrpcClient := ttrpc.NewClient(conn)
	return task.NewTaskClient(ttrpcClient), nil
}

func GetGRPCTaskClient(ctx context.Context) (containerTask.ContainersClient, error) {
	socketPath := "unix:///run/kettle/kettle.sock"

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	grpcClient, err := grpc.NewClient(socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return containerTask.NewContainersClient(grpcClient), nil
}
