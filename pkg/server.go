func createTTRPCServer(ctx context.Context) error {
	socketPath := "/run/kettle/kettle.sock.ttrpc"
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
