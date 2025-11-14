package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

const listenerFDEnv = "LISTENER_FD"

// This is a simplified example of a Go application that supports graceful
// restarts by inheriting the listener from the parent process.
// In a real application, you would integrate this logic with your server setup.
// TODO: integrate this with proper tests for a "non-stop" grpc server.
func main() {
	// 1. Check if a listener is passed from the parent process
	var listener net.Listener
	var err error

	// File descriptor 3 is commonly used for the inherited listener in this pattern
	// (0: stdin, 1: stdout, 2: stderr, 3: inherited listener)
	if os.Getenv(listenerFDEnv) != "" {
		fmt.Println("Inheriting listener from parent process")
		// The file descriptor is passed as an *os.File
		file := os.NewFile(3, "listener")
		listener, err = net.FileListener(file)
		if err != nil {
			fmt.Printf("net.FileListener error: %v\n", err)
			return
		}
	} else {
		fmt.Println("Starting new listener")
		// Initial start, create a new listener
		listener, err = net.Listen("tcp", ":8080")
		if err != nil {
			fmt.Printf("net.Listen error: %v\n", err)
			return
		}
	}

	// 2. Setup graceful shutdown for the current server
	server := &http.Server{Addr: ":8080"}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server Serve error: %v\n", err)
		}
	}()

	// 3. Handle signals for restart and termination
	signalCh := make(chan os.Signal, 1)
	// SIGUSR2 is the common signal for graceful restart
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)

	for {
		sig := <-signalCh
		fmt.Printf("Received signal: %s\n", sig)

		switch sig {
		case syscall.SIGUSR2:
			// Restart signal received, spawn the new process
			fmt.Println("Spawning new process")
			if err := spawnChild(listener); err != nil {
				fmt.Printf("Failed to spawn child: %v\n", err)
			}
			// Continue to the termination logic for the current process
			fallthrough

		case syscall.SIGINT, syscall.SIGTERM:
			// Termination signal received, shut down gracefully
			fmt.Println("Initiating graceful shutdown")
			if err := server.Shutdown(context.Background()); err != nil {
				fmt.Printf("HTTP server Shutdown error: %v\n", err)
			}
			fmt.Println("Server gracefully stopped")
			return
		}
	}
}

// Helper function to spawn a child process with the inherited listener
func spawnChild(listener net.Listener) error {
	// Get the underlying file from the listener
	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		return err
	}
	//nolint:errcheck // file will be closed when process exits
	defer file.Close()

	// Setup the command for the new process, passing the file descriptor
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Mark file descriptor 3 for the child to inherit
	cmd.ExtraFiles = []*os.File{file}
	// Set an environment variable so the child knows to use the inherited FD
	cmd.Env = append(os.Environ(), listenerFDEnv+"=3")

	return cmd.Start()
}
