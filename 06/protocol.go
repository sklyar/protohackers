package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrInvalidMessage = errors.New("invalid message")

type MessageError struct {
	Msg string
}

func (e *MessageError) MarshalBinary() ([]byte, error) {
	if len(e.Msg) > 255 {
		e.Msg = e.Msg[:255]
	}

	res := make([]byte, 1+len(e.Msg))
	res[0] = byte(len(e.Msg))
	copy(res[1:], e.Msg)

	return res, nil
}

type MessagePlate struct {
	Plate     string
	Timestamp uint32
}

func (p *MessagePlate) UnmarshalBinary(b []byte) error {
	const stringSize = 1
	const timestampSize = 4
	const minMessageLen = stringSize + timestampSize

	if len(b) < minMessageLen {
		return fmt.Errorf("message too short to be valid: %w", ErrInvalidMessage)
	}

	plateLen := int(b[0])
	if len(b) < minMessageLen+plateLen {
		return fmt.Errorf("message length does not match plate length and timestamp: %w", ErrInvalidMessage)
	}

	p.Plate = string(b[stringSize : stringSize+plateLen])
	p.Timestamp = binary.BigEndian.Uint32(b[stringSize+plateLen:])

	return nil
}

type MessageTicket struct {
	Plate      string
	Road       uint16
	Mile1      uint16
	Timestamp1 uint32
	Mile2      uint16
	Timestamp2 uint32
	Speed      uint16
}

func (t *MessageTicket) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(len(t.Plate)))
	buf.WriteString(t.Plate)
	binary.Write(&buf, binary.BigEndian, t.Road)
	binary.Write(&buf, binary.BigEndian, t.Mile1)
	binary.Write(&buf, binary.BigEndian, t.Timestamp1)
	binary.Write(&buf, binary.BigEndian, t.Mile2)
	binary.Write(&buf, binary.BigEndian, t.Timestamp2)
	binary.Write(&buf, binary.BigEndian, t.Speed)
	return buf.Bytes(), nil
}

type MessageWantHeartbeat struct {
	Interval uint32
}

func (h *MessageWantHeartbeat) UnmarshalBinary(b []byte) error {
	if len(b) < 4 {
		return fmt.Errorf("invalid message length: %w", ErrInvalidMessage)
	}

	h.Interval = binary.BigEndian.Uint32(b)
	return nil
}

type MessageIAmCamera struct {
	Road  uint16
	Mile  uint16
	Limit uint16
}

func (c *MessageIAmCamera) UnmarshalBinary(b []byte) error {
	if len(b) < 6 {
		return fmt.Errorf("invalid message length: %w", ErrInvalidMessage)
	}

	c.Road = binary.BigEndian.Uint16(b)
	c.Mile = binary.BigEndian.Uint16(b[2:])
	c.Limit = binary.BigEndian.Uint16(b[4:])
	return nil
}

type MessageIAmDispatcher struct {
	NumRoads uint8
	Roads    []uint16
}

func (d *MessageIAmDispatcher) UnmarshalBinary(b []byte) error {
	if len(b) < 1 {
		return fmt.Errorf("invalid message length: %w", ErrInvalidMessage)
	}

	numRoads := int(b[0])
	if len(b) < 1+numRoads*2 {
		return fmt.Errorf("invalid message length: %w", ErrInvalidMessage)
	}

	roads := make([]uint16, numRoads)
	for i := 0; i < numRoads; i++ {
		roads[i] = binary.BigEndian.Uint16(b[1+i*2:])
	}

	d.NumRoads = uint8(numRoads)
	d.Roads = roads

	return nil
}
