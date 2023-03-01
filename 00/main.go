package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os/signal"
	"syscall"
)

const defaultPort = "8080"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	port := flag.String("port", defaultPort, "port to listen on")
	flag.Parse()

	if err := listen(ctx, *port); err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}
}

func listen(ctx context.Context, port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	go func() {
		<-ctx.Done()
		if err := ln.Close(); err != nil {
			log.Printf("failed to close listener: %v\n", err)
		}
	}()

	log.Printf("listening on port %s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			log.Printf("failed to accept: %v\n", err)
			break
		}

		go handleConnection(conn)
	}

	if err := ln.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %v", err)
	}

	return nil
}

func handleConnection(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection: %v\n", err)
		}
	}()

	_, err := io.Copy(conn, conn)
	if err != nil {
		log.Printf("failed to copy: %v\n", err)
		return
	}
}
