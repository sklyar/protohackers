package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
)

func main() {
	addr := flag.String("addr", "", "")
	flag.Parse()

	if addr == nil || *addr == "" {
		slog.Error("addr must be set")
		os.Exit(1)
	}

	conn, err := net.Dial("udp", *addr)
	if err != nil {
		slog.Error("failed to connect server", err)
		os.Exit(1)
	}
	defer conn.Close()

	b, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
	if err != nil {
		slog.Error("failed to read from stdin", err)
		os.Exit(1)
	}

	b = b[:len(b)-1]

	if _, err := conn.Write(b); err != nil {
		slog.Error("failed to write to conn", err)
		os.Exit(1)
	}

	b = make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		slog.Error("failed to read from conn", err)
		os.Exit(1)
	}

	fmt.Printf("> %s\n", b[:n])
}
