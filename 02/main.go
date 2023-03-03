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

const (
	MessageLen        = 9
	MessageHeaderLen  = 1
	MessagePayloadLen = MessageLen - MessageHeaderLen
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	port := flag.String("port", defaultPort, "port to listen on")
	flag.Parse()

	netListener := func() (net.Listener, error) {
		return net.Listen("tcp", ":"+*port)
	}
	if err := listen(ctx, netListener); err != nil {
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

	client := NewClient(conn.RemoteAddr().String())
	log.Printf("new connection from %s\n", client.ipAddr)

	for {
		b := make([]byte, MessageLen)
		_, err := io.ReadAtLeast(conn, b, MessageLen)
		if err != nil {
			if err == io.EOF {
				log.Printf("client %s disconnected\n", client.ipAddr)
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("client %s timed out\n", client.ipAddr)
				return
			}

			log.Printf("failed to read from connection: %v\n", err)
			return
		}

		client.Handle(b, conn)
	}
}

type Client struct {
	ipAddr string
	assets map[int32]int32
}

func NewClient(ipAddr string) *Client {
	return &Client{
		ipAddr: ipAddr,
		assets: make(map[int32]int32, 50),
	}
}

func (c *Client) Handle(b []byte, w io.Writer) {
	m, err := ParseMessage(b)
	if err != nil {
		log.Printf("failed to parse message: %v\n", err)
		return
	}

	switch t := m.Type(); t {
	case MessageTypeInsert:
		im := m.(*InsertMessage)
		c.handleInsert(im)
	case MessageTypeQuery:
		qm := m.(*QueryMessage)
		c.handleQuery(qm, w)
	}
}

func (c *Client) handleInsert(m *InsertMessage) {
	log.Printf("insert message: %v\n", m)

	c.assets[m.Timestamp] = m.Price
}

func (c *Client) handleQuery(m *QueryMessage, w io.Writer) {
	log.Printf("query message: %v\n", m)

	var (
		sum   int64
		count int64
	)
	for timestamp, price := range c.assets {
		if timestamp >= m.MinTime && timestamp <= m.MaxTime {
			sum += int64(price)
			count++
		}
	}

	var avg int32
	if count > 0 {
		avg = int32(sum / count)
	}

	b := marshalQueryMessageResponse(avg)
	if _, err := w.Write(b); err != nil {
		log.Printf("failed to write to connection: %v\n", err)
		return
	}

	log.Printf("avg price: %d\n", avg)
}

func marshalQueryMessageResponse(avgPrice int32) []byte {
	b := make([]byte, 4)
	b[0] = byte(avgPrice >> 24)
	b[1] = byte(avgPrice >> 16)
	b[2] = byte(avgPrice >> 8)
	b[3] = byte(avgPrice)
	return b
}

type MessageType byte

const (
	MessageTypeInsert MessageType = 'I'
	MessageTypeQuery  MessageType = 'Q'
)

type Message interface {
	Type() MessageType

	unmarshal(b []byte) error
}

func ParseMessage(b []byte) (Message, error) {
	var m Message
	switch t := MessageType(b[0]); t {
	case MessageTypeInsert:
		m = &InsertMessage{}
	case MessageTypeQuery:
		m = &QueryMessage{}
	default:
		return nil, fmt.Errorf("unrecognized message type: %q", t)
	}

	if err := m.unmarshal(b[1:]); err != nil {
		return nil, err
	}

	return m, nil
}

type InsertMessage struct {
	Timestamp int32
	Price     int32
}

var _ Message = &InsertMessage{}

func (*InsertMessage) Type() MessageType {
	return MessageTypeInsert
}

func (m *InsertMessage) unmarshal(b []byte) error {
	if len(b) != MessagePayloadLen {
		return io.ErrUnexpectedEOF
	}

	m.Timestamp = int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3])
	m.Price = int32(b[4])<<24 | int32(b[5])<<16 | int32(b[6])<<8 | int32(b[7])

	return nil
}

type QueryMessage struct {
	MinTime int32
	MaxTime int32
}

var _ Message = &QueryMessage{}

func (*QueryMessage) Type() MessageType {
	return MessageTypeQuery
}

func (m *QueryMessage) unmarshal(b []byte) error {
	if len(b) != MessagePayloadLen {
		return io.ErrUnexpectedEOF
	}

	m.MinTime = int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3])
	m.MaxTime = int32(b[4])<<24 | int32(b[5])<<16 | int32(b[6])<<8 | int32(b[7])

	return nil
}
