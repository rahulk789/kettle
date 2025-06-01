package client

import (
	"context"
	"net"

	"github.com/containerd/containerd/api/runtime/task/v2"
	"github.com/containerd/ttrpc"
)

func getTTRPCTaskClient(ctx context.Context) (task.TaskService, error) {
	socketPath := "/run/kettle/kettle.sock.ttrpc"
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	ttrpcClient := ttrpc.NewClient(conn)
	return task.NewTaskClient(ttrpcClient), nil
}
