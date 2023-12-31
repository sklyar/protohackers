package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestMessageError_MarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  MessageError
		want []byte
	}{
		{
			name: "short message",
			msg:  MessageError{Msg: "short message"},
			want: []byte{13, 115, 104, 111, 114, 116, 32, 109, 101, 115, 115, 97, 103, 101},
		},
		{
			name: "long message",
			msg: MessageError{
				Msg: strings.Repeat("a", 300),
			},
			want: append([]byte{255}, []byte(strings.Repeat("a", 255))...),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.msg.MarshalBinary()
			if err != nil {
				t.Errorf("MarshalBinary() error = %v", err)
				return
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("MarshalBinary() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessagePlate_UnmarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    MessagePlate
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid message with plate UN1234",
			b:    append([]byte{6}, append([]byte("UN1234"), []byte{0, 0, 0, 1}...)...),
			want: MessagePlate{
				Plate:     "UN1234",
				Timestamp: 1,
			},
			wantErr: assert.NoError,
		},
		{
			name: "short message",
			b:    []byte{0},
			want: MessagePlate{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, ErrInvalidMessage)
			},
		},
		{
			name: "invalid length",
			b:    append([]byte{10}, []byte("TOOLONG")...), // incorrect length byte
			want: MessagePlate{},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, ErrInvalidMessage)
			},
		},
		{
			name: "valid message with plate ABCD",
			b:    append([]byte{4}, append([]byte("ABCD"), []byte{0, 0, 0, 10}...)...),
			want: MessagePlate{
				Plate:     "ABCD",
				Timestamp: 10,
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got MessagePlate
			err := got.UnmarshalBinary(tt.b)
			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMessageTicket_MarshalBinary(t *testing.T) {
	t.Parallel()

	msg := MessageTicket{
		Plate:      "UN1234",
		Road:       1,
		Mile1:      2,
		Timestamp1: 3,
		Mile2:      4,
		Timestamp2: 5,
		Speed:      6,
	}

	want := []byte{
		6, 85, 78, 49, 50, 51, 52, // plate
		0, 1, // road
		0, 2, // mile1
		0, 0, 0, 3, // timestamp1
		0, 4, // mile2
		0, 0, 0, 5, // timestamp2
		0, 6, // speed
	}

	got, err := msg.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestMessageWantHeartbeat_UnmarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    MessageWantHeartbeat
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "valid message",
			b:       []byte{0, 0, 0, 1},
			want:    MessageWantHeartbeat{Interval: 1},
			wantErr: assert.NoError,
		},
		{
			name: "short message",
			b:    []byte{0, 0, 0},
			want: MessageWantHeartbeat{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, ErrInvalidMessage)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got MessageWantHeartbeat
			err := got.UnmarshalBinary(tt.b)
			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMessageIAmCamera_UnmarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    MessageIAmCamera
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid message",
			b: []byte{
				0x00, 0x42, // road
				0x00, 0x64, // mile
				0x00, 0x3c, // limit
			},
			want: MessageIAmCamera{
				Road:  66,
				Mile:  100,
				Limit: 60,
			},
			wantErr: assert.NoError,
		},
		{
			name: "short message",
			b:    []byte{0, 0, 0},
			want: MessageIAmCamera{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, ErrInvalidMessage)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got MessageIAmCamera
			err := got.UnmarshalBinary(tt.b)
			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMessageIAmDispatcher_UnmarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		b       []byte
		want    MessageIAmDispatcher
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid message",
			b: []byte{
				0x03,       // number of roads
				0x00, 0x42, // road 1
				0x01, 0x70, // road 2
				0x13, 0x88, // road 3
			},
			want: MessageIAmDispatcher{
				NumRoads: 3,
				Roads:    []uint16{66, 368, 5000},
			},
			wantErr: assert.NoError,
		},
		{
			name: "zero roads",
			b:    []byte{0x00},
			want: MessageIAmDispatcher{
				NumRoads: 0,
				Roads:    []uint16{},
			},
			wantErr: assert.NoError,
		},
		{
			name: "empty message",
			b:    []byte{},
			want: MessageIAmDispatcher{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, ErrInvalidMessage)
			},
		},
		{
			name: "invalid length",
			b:    []byte{0x01},
			want: MessageIAmDispatcher{},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, ErrInvalidMessage)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got MessageIAmDispatcher
			err := got.UnmarshalBinary(tt.b)
			tt.wantErr(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}
