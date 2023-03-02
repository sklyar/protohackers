package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
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

	if err := listen(ctx, func() (net.Listener, error) {
		return net.Listen("tcp", ":"+*port)
	}); err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}
}

func listen(ctx context.Context, listener func() (net.Listener, error)) error {
	ln, err := listener()
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		<-ctx.Done()
		if err := ln.Close(); err != nil {
			log.Printf("failed to close listener: %v\n", err)
		}
	}()

	log.Printf("listening on port %d\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return err
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
		log.Printf("closing connection from %s\n", conn.RemoteAddr().String())
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection: %v\n", err)
		}
	}()

	log.Printf("new connection from %s\n", conn.RemoteAddr().String())

	write := func(b []byte) {
		b = append(b, '\n')

		log.Printf("writing to connection: %s\n", b)
		n, err := conn.Write(b)
		if err != nil {
			log.Printf("failed to write to connection: %v\n", err)
			return
		}
		if n != len(b) {
			log.Printf("failed to write all bytes to connection: %d/%d\n", n, len(b))
		}
	}

	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		b := sc.Bytes()
		req, err := parseRequest(b)
		if err != nil {
			write([]byte(err.Error()))
			break
		}

		res := response{
			Method: req.Method,
			Prime:  isPrime(int64(*req.Number)),
		}

		js, err := json.Marshal(res)
		if err != nil {
			write([]byte("failed to marshal response"))
			break
		}
		write(js)
	}
}

func isPrime(n int64) bool {
	return big.NewInt(n).ProbablyPrime(0)
}

type request struct {
	Method string   `json:"method"`
	Number *float64 `json:"number"`
}

func parseRequest(b []byte) (request, error) {
	var req request
	if err := json.Unmarshal(b, &req); err != nil {
		return request{}, fmt.Errorf("failed to decode request")
	}
	if err := req.Validate(); err != nil {
		return request{}, fmt.Errorf("invalid request: %v", err)
	}

	return req, nil
}

func (r request) Validate() error {
	if r.Method != "isPrime" {
		return fmt.Errorf("invalid method")
	}

	if r.Number == nil {
		return fmt.Errorf("number is required")
	}

	return nil
}

type response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}
