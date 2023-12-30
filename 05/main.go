package main

import (
	"bufio"
	"log/slog"
	"net"
	"os"
	"strings"
)

const tonyAddress = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"

func main() {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		slog.Error("ADDR environment variable must be set")
		os.Exit(1)
	}

	upstreamAddr := os.Getenv("UPSTREAM_ADDR")
	if upstreamAddr == "" {
		slog.Error("UPSTREAM_ADDR environment variable must be set")
		os.Exit(1)
	}

	slog.Info(
		"starting server",
		slog.String("addr", serverAddr),
		slog.String("upstream_addr", upstreamAddr),
	)

	ln, err := net.Listen("tcp", serverAddr)
	if err != nil {
		slog.Error("failed to create listener", err)
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Error("failed to accept", err)
			continue
		}

		go handleConn(conn, upstreamAddr)
	}
}

func handleConn(conn net.Conn, upstreamAddr string) {
	defer conn.Close()

	upstreamConn, err := net.Dial("tcp", upstreamAddr)
	if err != nil {
		slog.Error("failed to connect server", err)
		os.Exit(1)
	}
	defer upstreamConn.Close()

	go proxy(upstreamConn, conn)
	proxy(conn, upstreamConn)
}

func proxy(src, dst net.Conn) {
	for {
		msg, err := bufio.NewReader(src).ReadString('\n')
		if err != nil {
			slog.Error("failed to read from client", err)
			return
		}
		slog.Info("received from client", slog.String("data", msg))

		msg = replaceAddress(msg)

		_, err = dst.Write([]byte(msg))
		if err != nil {
			slog.Error("failed to write to upstream", err)
			return
		}
	}
}

func isBogusCoinAddress(input string) bool {
	return len(input) >= 26 && len(input) <= 35 && input[0] == '7'
}

func replaceAddress(input string) string {
	words := strings.Split(strings.Trim(input, "\n"), " ")

	var isBogus bool
	for i, word := range words {
		if isBogusCoinAddress(word) {
			words[i] = tonyAddress
			isBogus = true
		}
	}

	if !isBogus {
		return input
	}

	return strings.Join(words, " ") + "\n"
}
