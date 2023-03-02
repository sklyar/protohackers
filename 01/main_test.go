package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func Test_listen(t *testing.T) {
	listen := func() (int, context.CancelFunc, chan error) {
		ctx, cancel := context.WithCancel(context.Background())
		errs := make(chan error, 1)

		ln, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		port := ln.Addr().(*net.TCPAddr).Port

		go func() {
			listener := func() (net.Listener, error) {
				return ln, nil
			}
			if err := listen(ctx, listener); err != nil && ctx.Err() == nil {
				errs <- err
			}
			close(errs)
		}()

		return port, cancel, errs
	}

	t.Run("invalid request", func(t *testing.T) {
		port, stopServer, errs := listen()
		defer stopServer()

		go func() {
			err, ok := <-errs
			if !ok {
				return
			}
			t.Fatal(err)
		}()

		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		req := []byte("failed to decode request\n")
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}
		if err := read(conn, []byte("failed to decode request\n")); err != nil {
			t.Fatal(err)
		}

		if !isClosedConn(conn) {
			t.Fatal("connection is not closed")
		}

		if err := conn.Close(); err != nil {
			t.Fatalf("failed to close connection: %v", err)
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		port, stopServer, errs := listen()
		defer stopServer()

		go func() {
			err, ok := <-errs
			if !ok {
				return
			}
			t.Fatal(err)
		}()

		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		req := []byte("{\"method\":\"unknown\"}\n")
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}
		if err := read(conn, []byte("invalid request: invalid method\n")); err != nil {
			t.Fatal(err)
		}

		if !isClosedConn(conn) {
			t.Fatal("connection is not closed")
		}

		if err := conn.Close(); err != nil {
			t.Fatalf("failed to close connection: %v", err)
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		port, stopServer, errs := listen()
		defer stopServer()

		go func() {
			err, ok := <-errs
			if !ok {
				return
			}
			t.Fatal(err)
		}()

		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		req := []byte("{\"method\":\"isPrime\"}\n")
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}
		if err := read(conn, []byte("invalid request: number is required\n")); err != nil {
			t.Fatal(err)
		}

		if !isClosedConn(conn) {
			t.Fatal("connection is not closed")
		}

		if err := conn.Close(); err != nil {
			t.Fatalf("failed to close connection: %v", err)
		}
	})

	t.Run("prime check", func(t *testing.T) {
		port, stopServer, errs := listen()
		defer stopServer()

		go func() {
			err, ok := <-errs
			if !ok {
				return
			}
			t.Fatal(err)
		}()

		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		req := []byte("{\"method\":\"isPrime\",\"number\":7}\n")
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}
		if err := read(conn, []byte("{\"method\":\"isPrime\",\"prime\":true}\n")); err != nil {
			t.Fatal(err)
		}

		req = []byte("{\"method\":\"isPrime\",\"number\":8}\n")
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}
		if err := read(conn, []byte("{\"method\":\"isPrime\",\"prime\":false}\n")); err != nil {
			t.Fatal(err)
		}
	})
}

func Test_isPrime(t *testing.T) {
	cases := []struct {
		n    int64
		want bool
	}{
		{-1, false},
		{0, false},
		{1, false},
		{2, true},
		{3, true},
		{6, false},
		{1000039, true},
		{10000019, true},
		{1000000007, true},
	}
	for _, c := range cases {
		if got := isPrime(c.n); got != c.want {
			t.Errorf("isPrime() = %v, want %v", got, c.want)
		}
	}

	// Check against each continguous sequence of primes that the primes
	// are classified as primes and the numbers in between as not.
	contiguousPrimes := [][]int64{
		{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47},
		{127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179, 181},
		{877, 881, 883, 887, 907, 911, 919, 929, 937, 941, 947, 953},
		{2089, 2099, 2111, 2113, 2129, 2131, 2137, 2141, 2143, 2153},
		{9857, 9859, 9871, 9883, 9887, 9901, 9907, 9923, 9929, 9931},
		{1000003, 1000033, 1000037},
	}
	for _, ps := range contiguousPrimes {
		for _, p := range ps {
			if !isPrime(p) {
				t.Errorf("isPrime(%d) == false, want true", p)
			}
		}
		for i := 1; i < len(ps); i++ {
			for n := ps[i-1] + 1; n < ps[i]; n++ {
				if isPrime(n) {
					t.Errorf("isPrime(%d) == true, want false", n)
				}
			}
		}
	}

	// Check that the numbers obtain by multiplying any two of the following
	// prime factors are classified as not primes
	factors := []int64{2, 3, 41, 157, 953, 2141, 9929}
	for i, f := range factors {
		for j := 0; j <= i; j++ {
			n := f * factors[j]
			if isPrime(n) {
				t.Errorf("isPrime(%d) == true, want false", n)
			}
		}
	}
}

func write(conn net.Conn, payload []byte) error {
	if err := conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		return fmt.Errorf("failed to set deadline: %v", err)
	}
	defer conn.SetDeadline(time.Time{})

	n, err := conn.Write(payload)
	if err != nil {
		return fmt.Errorf("failed to write: %v", err)
	}
	if n != len(payload) {
		return fmt.Errorf("wrote %d bytes, want %d", n, len(payload))
	}
	return nil
}

func read(conn net.Conn, want []byte) error {
	if err := conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		return fmt.Errorf("failed to set deadline: %v", err)
	}
	defer conn.SetDeadline(time.Time{})

	b, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read: %v", err)
	}

	if !bytes.Equal(b, want) {
		return fmt.Errorf("got %q, want %q", b, want)
	}
	return nil
}

func isClosedConn(conn net.Conn) bool {
	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err := conn.Read(make([]byte, 1))
	return errors.Is(err, io.EOF)
}
