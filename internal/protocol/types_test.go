package protocol

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestMessage_RoundTrip(t *testing.T) {
	original := Message{
		Type: MsgPlayerInput,
		Payload: PlayerInput{
			Direction: DirUp,
		},
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(&original); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Message
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Type != MsgPlayerInput {
		t.Errorf("expected type %v, got %v", MsgPlayerInput, decoded.Type)
	}
}

func TestDirection_Values(t *testing.T) {
	if DirUp == DirDown {
		t.Error("directions should be distinct")
	}
	if DirLeft == DirRight {
		t.Error("directions should be distinct")
	}
}
