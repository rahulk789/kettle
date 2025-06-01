/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"kettle/server"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func startShim(ctx context.Context, rootDir, id, namespace string) error {
	log.Printf("Starting shim for container %s in namespace %s", id, namespace)

	// Create a directory for the container
	containerDir := filepath.Join(rootDir, namespace, id)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("failed to create container directory: %w", err)
	}

	// Shim socket path
	shimSocketPath := filepath.Join(containerDir, "shim.sock")

	// Prepare the shim command
	cmd := exec.CommandContext(ctx, "/run/kettle/kettle.sock.ttrpc",
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cmd",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starging server")
		server.CreateGRPCServer(context.TODO())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cmd.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
