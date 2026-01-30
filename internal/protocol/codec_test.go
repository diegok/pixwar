package protocol

import (
	"bytes"
	"testing"
)

func TestCodec_EncodeDecodeMessage(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf)

	original := Message{
		Type: MsgJoinRequest,
		Payload: JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}

	if err := codec.Encode(&original); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := codec.Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Type != MsgJoinRequest {
		t.Errorf("expected type %v, got %v", MsgJoinRequest, decoded.Type)
	}

	payload, ok := decoded.Payload.(JoinRequest)
	if !ok {
		t.Fatalf("payload is not JoinRequest")
	}
	if payload.PlayerName != "TestPlayer" {
		t.Errorf("expected name TestPlayer, got %s", payload.PlayerName)
	}
}

func TestCodec_MultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf)

	msgs := []Message{
		{Type: MsgPlayerInput, Payload: PlayerInput{Direction: DirUp}},
		{Type: MsgPlayerInput, Payload: PlayerInput{Direction: DirLeft}},
		{Type: MsgPlayerInput, Payload: PlayerInput{Direction: DirDown}},
	}

	for _, msg := range msgs {
		if err := codec.Encode(&msg); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}

	for i, expected := range msgs {
		decoded, err := codec.Decode()
		if err != nil {
			t.Fatalf("decode %d failed: %v", i, err)
		}
		if decoded.Type != expected.Type {
			t.Errorf("message %d: expected type %v, got %v", i, expected.Type, decoded.Type)
		}
	}
}
