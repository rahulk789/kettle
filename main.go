/*
Copyright © 2024 Stingless <stingless.tech.org>
*/
package main

import (
	"kettle/cmd"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	// unix socket
	PROTOCOL = "unix"
	SOCKET   = "/run/kettle/kettle.sock"
)

func main() {
	cmd.Execute()
	ln, err := net.Listen(PROTOCOL, SOCKET)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Remove(SOCKET)
		os.Exit(1)
	}()

	srv := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, health.NewServer())
	reflection.Register(srv)

	log.Printf("grpc ran on unix socket protocol %s", SOCKET)
	log.Fatal(srv.Serve(ln))
}
