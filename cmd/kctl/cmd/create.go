/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	task "kettle/api"
	"log"
	"net"

	"github.com/containerd/ttrpc"
	"github.com/spf13/cobra"
)

type TaskServiceImpl struct{}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create called")
		ctx := cmd.Context() // Get Cobra's context

		id, err := cmd.Flags().GetString("id")
		if err != nil {
			log.Fatalf("Failed to get id flag: %v", err)
		}
		if id == "" {
			log.Fatalf("Container ID is required")
		}

		client, err := getTaskClient(ctx)
		if err != nil {
			log.Fatalf("Failed to create task client: %v", err)
		}
		req := task.CreateTaskRequest{
			Id: id,
		}

		resp, _ := client.Create(ctx, &req)
		fmt.Println(resp.Pid)
		return
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.
	createCmd.PersistentFlags().String("id", "", "container id")

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
