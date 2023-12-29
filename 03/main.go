package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		fmt.Fprintf(os.Stderr, "ADDR environment variable must be set\n")
		os.Exit(1)
	}

	s, err := NewServer(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create server: %s\n", err)
		os.Exit(1)
	}

	if err := s.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run server: %s\n", err)
		os.Exit(1)
	}
}

type Server struct {
	listener net.Listener
	hub      *Hub
}

func NewServer(addr string) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	return &Server{
		listener: ln,
		hub:      NewHub(),
	}, nil
}

func (s *Server) Run() error {
	go s.hub.Run()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Printf("failed to accept: %s\n", err)
			continue
		}

		go s.handleConn(NewClient(conn))
	}
}

func (s *Server) handleConn(client *Client) {
	if err := s.hub.Register(client); err != nil {
		fmt.Printf("failed to register: %s\n", err)
		return
	}
}
