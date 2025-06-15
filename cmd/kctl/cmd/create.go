/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	containerTask "kettle/api/kettle"
	client "kettle/client"
	"log"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

		bundle, err := cmd.Flags().GetString("bundle")
		if err != nil {
			log.Fatalf("Failed to get id flag: %v", err)
		}
		if bundle == "" {
			log.Fatalf("Container ID is required")
		}
		clientContext, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		testConnection()
		client, err := client.GetGRPCTaskClient(clientContext)
		if err != nil {
			log.Fatalf("Failed to create task client: %v", err)
		}
		container := containerTask.Container{
			ID:     id,
			Bundle: bundle,
		}

		req := containerTask.CreateContainerRequest{
			Container: &container,
		}
		resp, _ := client.Create(clientContext, &req)
		if err != nil {
			log.Fatalf("Failed to get id flag: %v", err)
		}
		fmt.Println(resp)
		return
	},
}

func testConnection() {
	socketPath := "unix:///run/kettle/kettle.sock"

	// Test 1: Check if socket file exists
	if _, err := os.Stat("/run/kettle/kettle.sock"); os.IsNotExist(err) {
		fmt.Println("Socket file doesn't exist - server definitely not running")
		return
	}

	// Test 2: Try raw connection
	conn, err := net.Dial("unix", "/run/kettle/kettle.sock")
	if err != nil {
		fmt.Printf("Raw socket connection failed: %v\n", err)
		return
	}
	conn.Close()
	fmt.Println("Raw socket connects but might be stale")

	// Test 3: gRPC connection with detailed error
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	grpcConn, err := grpc.DialContext(ctx, socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		fmt.Printf("gRPC connection failed: %v\n", err)
		return
	}
	defer grpcConn.Close()

	fmt.Println("gRPC connected successfully")
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.
	createCmd.PersistentFlags().String("id", "", "container id")
	createCmd.PersistentFlags().String("bundle", "", "bundle path")

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
