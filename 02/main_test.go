package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"
)

func Test_listen(t *testing.T) {
	t.Parallel()

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

	t.Run("session with multiple messages", func(t *testing.T) {
		t.Parallel()

		port, stopServer, errs := listen()
		defer stopServer()

		go func() {
			err, ok := <-errs
			if !ok {
				return
			}
			t.Fatal(err)
		}()

		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Fatalf("failed to dial: %v", err)
		}
		defer conn.Close()

		req := []byte{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65} // I 12345 101
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}

		req = []byte{0x49, 0x00, 0x00, 0xa0, 0x00, 0x00, 0x00, 0x00, 0x05} // I 40960 5
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}

		req = []byte{0x49, 0x00, 0x00, 0x30, 0x3a, 0x00, 0x00, 0x00, 0x66} // I 12346 102
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}

		req = []byte{0x49, 0x00, 0x00, 0x30, 0x3b, 0x00, 0x00, 0x00, 0x64} // I 12347 100
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}

		req = []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00} // Q 12288 16384
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}
		want := []byte{0x00, 0x00, 0x00, 0x65} // 101
		if err := read(conn, want); err != nil {
			t.Fatal(err)
		}

		req = []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00} // Q 12288 16384
		if err := write(conn, req); err != nil {
			t.Fatal(err)
		}
		want = []byte{0x00, 0x00, 0x00, 0x65} // 101
		if err := read(conn, want); err != nil {
			t.Fatal(err)
		}
	})
}

func TestParseMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		want  Message

		wantErr bool
	}{
		{
			name:    "unknown message type",
			input:   []byte{0x45, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65}, // E 12345 101
			wantErr: true,
		},
		{
			name:  "insert message",
			input: []byte{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65}, // I 12345 101
			want:  &InsertMessage{Timestamp: 12345, Price: 101},
		},
		{
			name:  "insert message with negative price",
			input: []byte{0x49, 0x00, 0x00, 0xa0, 0x00, 0xff, 0xff, 0xff, 0xfb}, // I 40960 -5
			want:  &InsertMessage{Timestamp: 40960, Price: -5}},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalQueryMessageResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		avgPrice int32
		want     []byte
	}{
		{
			name:     "positive average price",
			avgPrice: 17,
			want:     []byte{0x00, 0x00, 0x00, 0x11},
		},
		{
			name:     "negative average price",
			avgPrice: -5,
			want:     []byte{0xff, 0xff, 0xff, 0xfb},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := marshalQueryMessageResponse(tt.avgPrice); !bytes.Equal(got, tt.want) {
				t.Errorf("Marshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientHandle(t *testing.T) {
	t.Parallel()

	type (
		insertMessage struct {
			request []byte
		}
		queryMessage struct {
			request  []byte
			response []byte
		}
	)

	tests := []struct {
		name    string
		client  *Client
		session []any
	}{
		{
			name:   "query with no assets",
			client: NewClient("127.0.0.1:1001"),
			session: []any{
				queryMessage{
					request:  []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00}, // Q 12288 16384
					response: []byte{0x00, 0x00, 0x00, 0x00},                               // 0
				},
			},
		},
		{
			name:   "query with one asset",
			client: NewClient("127.0.0.1:1001"),
			session: []any{
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65}}, // I 12345 101
				queryMessage{
					request:  []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00}, // Q 12288 16384
					response: []byte{0x00, 0x00, 0x00, 0x65},                               // 101
				},
			},
		},
		{
			name:   "correct positive average price",
			client: NewClient("127.0.0.1:1000"),
			session: []any{
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65}}, // I 12345 101
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3a, 0x00, 0x00, 0x00, 0x66}}, // I 12346 102
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3b, 0x00, 0x00, 0x00, 0x64}}, // I 12347 100
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0xa0, 0x00, 0x00, 0x00, 0x00, 0x05}}, // I 40960 5
				queryMessage{
					request:  []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00}, // Q 12288 16384
					response: []byte{0x00, 0x00, 0x00, 0x65},                               // 101
				},
			},
		},
		{
			name:   "correct negative average price",
			client: NewClient("127.0.0.1:1000"),
			session: []any{
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x39, 0xff, 0xff, 0xff, 0xff}}, // I 12345 -1
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3a, 0xff, 0xff, 0xff, 0xfe}}, // I 12346 -2
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3b, 0xff, 0xff, 0xff, 0xfd}}, // I 12347 -3
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3b, 0xff, 0xff, 0xff, 0xfc}}, // I 12347 -4
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3b, 0xff, 0xff, 0xff, 0xfb}}, // I 12347 -5
				queryMessage{
					request:  []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00}, // Q 12288 16384
					response: []byte{0xff, 0xff, 0xff, 0xfe},                               // -2
				},
			},
		},
		{
			name:   "correct average price with zero",
			client: NewClient("127.0.0.1:1000"),
			session: []any{
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x00}}, // I 12345 0
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3a, 0x00, 0x00, 0x00, 0x00}}, // I 12346 0
				queryMessage{
					request:  []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00}, // Q 12288 16384
					response: []byte{0x00, 0x00, 0x00, 0x00},                               // 0
				},
			},
		},
		{
			name:   "correct average price for range",
			client: NewClient("127.0.0.1:1000"),
			session: []any{
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65}}, // I 12345 101
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3a, 0x00, 0x00, 0x00, 0x66}}, // I 12346 102
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3b, 0x00, 0x00, 0x00, 0x64}}, // I 12347 100
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3c, 0x00, 0x00, 0x00, 0x63}}, // I 12348 99
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3d, 0x00, 0x00, 0x00, 0x62}}, // I 12349 98
				insertMessage{request: []byte{0x49, 0x00, 0x00, 0x30, 0x3e, 0x00, 0x00, 0x00, 0x61}}, // I 12350 97
				queryMessage{
					request:  []byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00}, // Q 12346 12349
					response: []byte{0x00, 0x00, 0x00, 0x63},                               // 99
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, msg := range tt.session {
				switch msg := msg.(type) {
				case insertMessage:
					var w bytes.Buffer
					tt.client.Handle(msg.request, &w)
					if w.Len() != 0 {
						t.Errorf("response = %v, want %v", w.Bytes(), []byte{})
					}
				case queryMessage:
					var w bytes.Buffer
					tt.client.Handle(msg.request, &w)
					if !bytes.Equal(w.Bytes(), msg.response) {
						t.Errorf("response = %x, want %x", w.Bytes(), msg.response)
					}
				}
			}
		})
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
	b := make([]byte, len(want))
	if _, err := conn.Read(b); err != nil {
		return fmt.Errorf("failed to read: %v", err)
	}
	if !bytes.Equal(b, want) {
		return fmt.Errorf("got %q, want %q", b, want)
	}
	return nil
}
