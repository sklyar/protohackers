package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

const greetingPhrase = "Welcome to budgetchat! What shall I call you?"

type EventType int

const (
	EventTypeMessage EventType = iota
	EventTypeJoin
	EventTypeLeave
	EventTypePresence
)

type Event struct {
	From string
	Type EventType

	Message string
}

type Hub struct {
	clients map[string]*Client
	mu      sync.RWMutex

	events chan Event
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
		events:  make(chan Event, 100),
	}
}

func (h *Hub) Register(client *Client) error {
	if err := client.Send(greetingPhrase); err != nil {
		return fmt.Errorf("failed to send greeting: %w", err)
	}

	name, err := client.Receive()
	if err != nil {
		return fmt.Errorf("failed to read name: %w", err)
	}

	if err := validateName(name); err != nil {
		defer client.Close()

		if err := client.Send(fmt.Sprintf("failed to register: %s", err)); err != nil {
			return fmt.Errorf("failed to send error message: %w", err)
		}
		return err
	}

	h.mu.Lock()
	h.clients[name] = client
	h.mu.Unlock()

	h.events <- Event{
		From: name,
		Type: EventTypeJoin,
	}

	h.events <- Event{
		From: name,
		Type: EventTypePresence,
	}

	go h.handleClient(name, client)

	return nil
}

func (h *Hub) Run() {
	for event := range h.events {
		switch event.Type {
		case EventTypeMessage:
			msg := fmt.Sprintf("[%s] %s", event.From, event.Message)
			h.broadcast(event.From, msg)

			fmt.Println(msg)
		case EventTypeJoin:
			msg := fmt.Sprintf("* %s has entered the room", event.From)
			h.broadcast(event.From, msg)

			fmt.Println(msg)
		case EventTypeLeave:
			msg := fmt.Sprintf("* %s has left the room", event.From)
			h.broadcast(event.From, msg)
		case EventTypePresence:
			h.mu.RLock()
			client := h.clients[event.From]
			h.mu.RUnlock()

			names := h.allClientNames()
			for i, name := range names {
				if name == event.From {
					names = append(names[:i], names[i+1:]...)
					break
				}
			}

			msg := fmt.Sprintf("* The room contains: %s", strings.Join(names, ", "))
			if err := client.Send(msg); err != nil {
				fmt.Printf("failed to send message: %s\n", err)
			}

			fmt.Println(msg)
		}
	}
}

func (h *Hub) allClientNames() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	names := make([]string, 0, len(h.clients))
	for name := range h.clients {
		names = append(names, name)
	}

	return names
}

func (h *Hub) handleClient(name string, client *Client) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, name)
		h.mu.Unlock()

		h.events <- Event{
			From: name,
			Type: EventTypeLeave,
		}
	}()

	for {
		msg, err := client.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Printf("failed to read message: %s\n", err)
			return
		}

		h.events <- Event{
			From:    name,
			Type:    EventTypeMessage,
			Message: msg,
		}
	}
}

func (h *Hub) broadcast(from string, msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for name, client := range h.clients {
		if name == from {
			continue
		}

		if err := client.Send(msg); err != nil {
			fmt.Printf("failed to send message: %s\n", err)
		}
	}
}

func validateName(name string) error {
	if len(name) == 0 {
		return errors.New("name must contains at least 1 character")
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return errors.New("name must contain only alphanumeric characters")
		}
	}

	return nil
}
