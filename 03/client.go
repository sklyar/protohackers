package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
)

type Client struct {
	conn net.Conn
}

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

func (c *Client) Send(msg string) error {
	b := []byte(msg)
	b = append(b, '\n')
	_, err := c.conn.Write(b)
	return err
}

func (c *Client) Receive() (string, error) {
	return c.receive()
}

func (c *Client) Close() {
	_ = c.conn.Close()
}

func (c *Client) receive() (string, error) {
	msg, err := bufio.NewReader(c.conn).ReadBytes('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read: %w", err)
	}

	msg = bytes.TrimSpace(msg)
	return string(msg), nil
}
