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

func TestDirection_AllDistinct(t *testing.T) {
	directions := []Direction{DirNone, DirUp, DirDown, DirLeft, DirRight}
	names := []string{"DirNone", "DirUp", "DirDown", "DirLeft", "DirRight"}

	for i := 0; i < len(directions); i++ {
		for j := i + 1; j < len(directions); j++ {
			if directions[i] == directions[j] {
				t.Errorf("%s and %s should be distinct, both have value %d",
					names[i], names[j], directions[i])
			}
		}
	}
}

func TestGameState_RoundTrip(t *testing.T) {
	original := Message{
		Type: MsgGameState,
		Payload: GameState{
			Tick:        42,
			BoardWidth:  80,
			BoardHeight: 24,
			TimeLeft:    120,
			Players: []PlayerState{
				{
					ID:        1,
					Name:      "Player1",
					Color:     2,
					Position:  Position{X: 10, Y: 5},
					Direction: DirUp,
					Trail:     []Position{{X: 10, Y: 6}, {X: 10, Y: 7}},
					Territory: []Position{{X: 1, Y: 1}, {X: 1, Y: 2}},
					Score:     100,
					Alive:     true,
					Protected: false,
				},
				{
					ID:        2,
					Name:      "Player2",
					Color:     3,
					Position:  Position{X: 20, Y: 15},
					Direction: DirLeft,
					Trail:     []Position{},
					Territory: []Position{{X: 20, Y: 15}},
					Score:     50,
					Alive:     true,
					Protected: true,
				},
			},
			Powerups: []PowerupState{
				{Type: PowerupSpeed, Position: Position{X: 40, Y: 12}, TTL: 30},
			},
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

	if decoded.Type != MsgGameState {
		t.Errorf("expected type %v, got %v", MsgGameState, decoded.Type)
	}

	gs, ok := decoded.Payload.(GameState)
	if !ok {
		t.Fatalf("expected GameState payload, got %T", decoded.Payload)
	}

	if gs.Tick != 42 {
		t.Errorf("expected Tick 42, got %d", gs.Tick)
	}
	if gs.BoardWidth != 80 {
		t.Errorf("expected BoardWidth 80, got %d", gs.BoardWidth)
	}
	if gs.BoardHeight != 24 {
		t.Errorf("expected BoardHeight 24, got %d", gs.BoardHeight)
	}
	if len(gs.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(gs.Players))
	}
	if gs.Players[0].Name != "Player1" {
		t.Errorf("expected Player1, got %s", gs.Players[0].Name)
	}
	if gs.Players[1].Position.X != 20 || gs.Players[1].Position.Y != 15 {
		t.Errorf("expected Player2 at (20,15), got (%d,%d)",
			gs.Players[1].Position.X, gs.Players[1].Position.Y)
	}
}

func TestJoinRequest_RoundTrip(t *testing.T) {
	original := Message{
		Type: MsgJoinRequest,
		Payload: JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  120,
			TerminalHeight: 40,
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

	if decoded.Type != MsgJoinRequest {
		t.Errorf("expected type %v, got %v", MsgJoinRequest, decoded.Type)
	}

	jr, ok := decoded.Payload.(JoinRequest)
	if !ok {
		t.Fatalf("expected JoinRequest payload, got %T", decoded.Payload)
	}

	if jr.PlayerName != "TestPlayer" {
		t.Errorf("expected PlayerName 'TestPlayer', got '%s'", jr.PlayerName)
	}
	if jr.TerminalWidth != 120 {
		t.Errorf("expected TerminalWidth 120, got %d", jr.TerminalWidth)
	}
	if jr.TerminalHeight != 40 {
		t.Errorf("expected TerminalHeight 40, got %d", jr.TerminalHeight)
	}
}

func TestJoinResponse_RoundTrip(t *testing.T) {
	original := Message{
		Type: MsgJoinResponse,
		Payload: JoinResponse{
			PlayerID: 42,
			Accepted: true,
			Reason:   "",
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

	if decoded.Type != MsgJoinResponse {
		t.Errorf("expected type %v, got %v", MsgJoinResponse, decoded.Type)
	}

	jr, ok := decoded.Payload.(JoinResponse)
	if !ok {
		t.Fatalf("expected JoinResponse payload, got %T", decoded.Payload)
	}

	if jr.PlayerID != 42 {
		t.Errorf("expected PlayerID 42, got %d", jr.PlayerID)
	}
	if !jr.Accepted {
		t.Error("expected Accepted to be true")
	}

	// Test rejection case
	original2 := Message{
		Type: MsgJoinResponse,
		Payload: JoinResponse{
			PlayerID: 0,
			Accepted: false,
			Reason:   "lobby full",
		},
	}

	buf.Reset()
	enc = gob.NewEncoder(&buf)
	if err := enc.Encode(&original2); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded2 Message
	dec = gob.NewDecoder(&buf)
	if err := dec.Decode(&decoded2); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	jr2, ok := decoded2.Payload.(JoinResponse)
	if !ok {
		t.Fatalf("expected JoinResponse payload, got %T", decoded2.Payload)
	}

	if jr2.Accepted {
		t.Error("expected Accepted to be false")
	}
	if jr2.Reason != "lobby full" {
		t.Errorf("expected Reason 'lobby full', got '%s'", jr2.Reason)
	}
}

func TestLobbyState_RoundTrip(t *testing.T) {
	original := Message{
		Type: MsgLobbyState,
		Payload: LobbyState{
			IsHost:   true,
			CanStart: false,
			Players: []LobbyPlayer{
				{ID: 1, Name: "Host", Color: 1, Ready: true},
				{ID: 2, Name: "Guest1", Color: 2, Ready: false},
				{ID: 3, Name: "Guest2", Color: 3, Ready: true},
			},
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

	if decoded.Type != MsgLobbyState {
		t.Errorf("expected type %v, got %v", MsgLobbyState, decoded.Type)
	}

	ls, ok := decoded.Payload.(LobbyState)
	if !ok {
		t.Fatalf("expected LobbyState payload, got %T", decoded.Payload)
	}

	if !ls.IsHost {
		t.Error("expected IsHost to be true")
	}
	if ls.CanStart {
		t.Error("expected CanStart to be false")
	}
	if len(ls.Players) != 3 {
		t.Errorf("expected 3 players, got %d", len(ls.Players))
	}
	if ls.Players[0].Name != "Host" {
		t.Errorf("expected first player name 'Host', got '%s'", ls.Players[0].Name)
	}
	if ls.Players[1].Ready {
		t.Error("expected Guest1 to not be ready")
	}
	if !ls.Players[2].Ready {
		t.Error("expected Guest2 to be ready")
	}
}
