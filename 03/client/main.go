package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	addr := flag.String("addr", "", "")
	flag.Parse()

	if addr == nil || *addr == "" {
		fmt.Fprintf(os.Stderr, "addr must be set\n")
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", *addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect server: %s\n: %s", *addr, err)
		os.Exit(1)
	}
	defer conn.Close()

	chat := Chat{
		server: conn,
	}

	if err := chat.Listen(); err != nil {
		return
	}
}

type Chat struct {
	server   net.Conn
	nickname string
}

func (c *Chat) Listen() error {
	w := make(chan []byte, 1)
	r := make(chan []byte, 1)

	waitGreeting := make(chan struct{})
	go func() {
		<-waitGreeting
		c.listenInput(w)
	}()

	go c.listenMessages(r)

	for i := 0; ; i++ {
		select {
		case msg, ok := <-r:
			if !ok {
				return fmt.Errorf("server closed")
			}

			fmt.Println(string(msg))

			if i == 0 {
				close(waitGreeting)
			}
		case msg := <-w:
			_, err := c.server.Write(msg)
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
		}
	}
}

func (c *Chat) listenInput(ch chan<- []byte) {
	for {
		b, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			fmt.Printf("failed to read from stdin: %s", err)
			continue
		}

		b = bytes.TrimSpace(b)
		b = append(b, '\n')
		ch <- b
	}
}

func (c *Chat) listenMessages(ch chan<- []byte) {
	defer close(ch)

	for {
		b, err := bufio.NewReader(c.server).ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Printf("failed to read message from server: %s\n", err)
			continue
		}

		ch <- bytes.TrimSpace(b)
	}
}
