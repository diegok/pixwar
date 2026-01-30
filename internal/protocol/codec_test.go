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

func TestNewEncoder_EncodeOnly(t *testing.T) {
	var buf bytes.Buffer
	codec := NewEncoder(&buf)

	msg := Message{
		Type:    MsgPlayerInput,
		Payload: PlayerInput{Direction: DirUp},
	}

	if err := codec.Encode(&msg); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty buffer after encode")
	}
}

func TestNewDecoder_DecodeOnly(t *testing.T) {
	// First encode a message to a buffer
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	original := Message{
		Type:    MsgPlayerInput,
		Payload: PlayerInput{Direction: DirRight},
	}
	encoder.Encode(&original)

	// Now decode with decoder-only codec
	decoder := NewDecoder(&buf)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.Type != MsgPlayerInput {
		t.Errorf("expected type %v, got %v", MsgPlayerInput, decoded.Type)
	}
}

func TestCodec_DecodeEmptyBuffer_ReturnsError(t *testing.T) {
	var buf bytes.Buffer
	codec := NewDecoder(&buf)

	_, err := codec.Decode()
	if err == nil {
		t.Error("expected error when decoding from empty buffer")
	}
}

func TestCodec_GameState_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf)

	original := Message{
		Type: MsgGameState,
		Payload: GameState{
			Tick:        100,
			BoardWidth:  80,
			BoardHeight: 40,
			TimeLeft:    120,
			Players: []PlayerState{
				{ID: 1, Name: "Alice", Position: Position{X: 10, Y: 5}, Alive: true},
			},
		},
	}

	if err := codec.Encode(&original); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := codec.Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Type != MsgGameState {
		t.Errorf("expected type %v, got %v", MsgGameState, decoded.Type)
	}

	state, ok := decoded.Payload.(GameState)
	if !ok {
		t.Fatal("payload is not GameState")
	}
	if state.Tick != 100 {
		t.Errorf("expected tick 100, got %d", state.Tick)
	}
	if len(state.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(state.Players))
	}
}

func TestCodec_LobbyState_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf)

	original := Message{
		Type: MsgLobbyState,
		Payload: LobbyState{
			Players: []LobbyPlayer{
				{ID: 1, Name: "Host", Color: 0, Ready: true},
				{ID: 2, Name: "Guest", Color: 1, Ready: false},
			},
			IsHost:   true,
			CanStart: true,
		},
	}

	if err := codec.Encode(&original); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := codec.Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	lobby, ok := decoded.Payload.(LobbyState)
	if !ok {
		t.Fatal("payload is not LobbyState")
	}
	if len(lobby.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(lobby.Players))
	}
	if !lobby.IsHost {
		t.Error("expected IsHost to be true")
	}
}
