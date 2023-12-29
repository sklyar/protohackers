package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
)

type command uint8

const (
	commandInsert command = iota
	commandRetrieve
	commandVersion
)

const version = "1.0.0"

func determineCommand(msg string) command {
	if msg == "version" {
		return commandVersion
	}

	if strings.Contains(msg, "=") {
		return commandInsert
	}

	return commandRetrieve
}

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		fmt.Fprintf(os.Stderr, "ADDR environment variable must be set\n")
		os.Exit(1)
	}

	s, err := NewServer(addr)
	if err != nil {
		slog.Error("failed to create server", err)
		os.Exit(1)
	}

	slog.Info("starting server", slog.String("addr", addr))

	if err := s.Serve(); err != nil {
		slog.Error("failed to run server", err)
		os.Exit(1)
	}
}

type Server struct {
	ln *net.UDPConn

	store map[string]string
	mu    sync.RWMutex
}

func NewServer(addr string) (*Server, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	ln, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP address: %w", err)
	}

	return &Server{
		ln:    ln,
		store: make(map[string]string),
	}, nil
}

func (s *Server) Serve() error {
	for {
		buf := make([]byte, 1000)
		n, addr, err := s.ln.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("failed to read from UDP: %w", err)
		}

		s.handleConn(addr, string(buf[:n]))
	}
}

func (s *Server) handleConn(addr *net.UDPAddr, msg string) {
	cmd := determineCommand(msg)
	switch cmd {
	case commandInsert:
		s.handleInsert(msg)
	case commandRetrieve:
		s.handleRetrieve(addr, msg)
	case commandVersion:
		s.handleVersion(addr)
	}

	slog.Info("received message", slog.String("msg", msg), slog.String("addr", addr.String()))
}

func (s *Server) handleInsert(msg string) {
	parts := strings.SplitN(msg, "=", 2)
	if len(parts) != 2 {
		slog.Error("bad message", slog.String("msg", msg))
		return
	}

	key, value := parts[0], parts[1]
	if key == "version" {
		return
	}

	s.mu.Lock()
	s.store[key] = value
	s.mu.Unlock()
}

func (s *Server) handleRetrieve(addr *net.UDPAddr, key string) {
	s.mu.RLock()
	value := s.store[key]
	s.mu.RUnlock()

	_, err := s.ln.WriteToUDP(newResponse(key, value), addr)
	if err != nil {
		slog.Error("failed to write", err)
		return
	}
}

func (s *Server) handleVersion(addr *net.UDPAddr) {
	_, err := s.ln.WriteToUDP(newResponse("version", version), addr)
	if err != nil {
		slog.Error("failed to write", err)
		return
	}
}

func newResponse(key, value string) []byte {
	return []byte(fmt.Sprintf("%s=%s", key, value))
}
