package main

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

func Test_listen(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    []byte
	}{
		{
			name:    "empty payload",
			payload: []byte{},
			want:    []byte{},
		},
		{
			name:    "hello world",
			payload: []byte("hello world"),
			want:    []byte("hello world"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, stopServer := context.WithCancel(context.Background())
			defer stopServer()

			const serverPort = "8081"

			errs := make(chan error, 1)
			go func() {
				errs <- listen(ctx, serverPort)
			}()

			// wait for server to start
			time.Sleep(time.Millisecond)

			conn, err := net.Dial("tcp", ":"+serverPort)
			if err != nil {
				t.Fatalf("failed to dial: %v", err)
			}
			defer conn.Close()

			// write request
			n, err := conn.Write(tt.payload)
			if err != nil {
				t.Fatalf("failed to write: %v", err)
			}
			if n != len(tt.want) {
				t.Fatalf("wrote %d bytes, want %d", n, len(tt.want))
			}

			// read response
			b := make([]byte, len(tt.want))
			n, err = conn.Read(b)
			if err != nil {
				t.Fatalf("failed to read: %v", err)
			}
			if n != len(tt.want) {
				t.Fatalf("read %d bytes, want %d", n, len(tt.want))
			}

			if !bytes.Equal(b, tt.want) {
				t.Fatalf("got %q, want %q", b, tt.want)
			}

			if err := conn.Close(); err != nil {
				t.Fatalf("failed to close connection: %v", err)
			}

			// check server didn't return an error after closing connection
			if err := serverError(errs); err != nil {
				t.Fatalf("server error: %v", err)
			}

			stopServer()

			// wait for server to stop
			time.Sleep(time.Millisecond)

			if err := serverError(errs); err != nil {
				t.Fatalf("server error: %v", err)
			}
		})
	}
}

func serverError(errs <-chan error) error {
	select {
	case err := <-errs:
		return err
	default:
		return nil
	}
}
