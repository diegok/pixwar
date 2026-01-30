package server

import (
	"net"
	"testing"
	"time"

	"github.com/diegok/pixwar/internal/config"
	"github.com/diegok/pixwar/internal/protocol"
)

func TestNewServer(t *testing.T) {
	s := NewServer(0) // Port 0 means any available port
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.cfg == nil {
		t.Error("cfg is nil")
	}
	if s.cfg.Port != 0 {
		t.Errorf("expected port 0, got %d", s.cfg.Port)
	}
	if !s.inLobby {
		t.Error("server should start in lobby")
	}
}

func TestNewServerWithConfig(t *testing.T) {
	cfg := &config.Config{
		Port:            8080,
		GameDuration:    10,
		Threshold:       80,
		PowerupsEnabled: true,
	}
	s := NewServerWithConfig(cfg)
	if s == nil {
		t.Fatal("NewServerWithConfig returned nil")
	}
	if s.cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", s.cfg.Port)
	}
	if s.cfg.GameDuration != 10 {
		t.Errorf("expected duration 10, got %d", s.cfg.GameDuration)
	}
}

func TestServerStartAndStop(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	addr := s.Addr()
	if addr == "" {
		t.Error("Addr returned empty string")
	}

	// Verify we can connect
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	conn.Close()

	s.Stop()

	// Give server time to close
	time.Sleep(100 * time.Millisecond)

	// Verify we can't connect anymore
	conn, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Error("expected connection to fail after stop")
	}
}

func TestJoinHandshake(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	// Connect
	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	codec := protocol.NewCodec(conn)

	// Send join request
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}
	if err := codec.Encode(joinReq); err != nil {
		t.Fatalf("failed to send join request: %v", err)
	}

	// Receive join response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msg, err := codec.Decode()
	if err != nil {
		t.Fatalf("failed to receive response: %v", err)
	}

	if msg.Type != protocol.MsgJoinResponse {
		t.Errorf("expected MsgJoinResponse, got %v", msg.Type)
	}

	response, ok := msg.Payload.(protocol.JoinResponse)
	if !ok {
		t.Fatal("failed to cast payload to JoinResponse")
	}

	if !response.Accepted {
		t.Errorf("join was not accepted: %s", response.Reason)
	}

	if response.PlayerID <= 0 {
		t.Errorf("expected positive player ID, got %d", response.PlayerID)
	}
}

func TestJoinRejectedSmallTerminal(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	codec := protocol.NewCodec(conn)

	// Send join request with small terminal
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  20, // Too small
			TerminalHeight: 10, // Too small
		},
	}
	if err := codec.Encode(joinReq); err != nil {
		t.Fatalf("failed to send join request: %v", err)
	}

	// Receive join response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msg, err := codec.Decode()
	if err != nil {
		t.Fatalf("failed to receive response: %v", err)
	}

	if msg.Type != protocol.MsgJoinResponse {
		t.Errorf("expected MsgJoinResponse, got %v", msg.Type)
	}

	response, ok := msg.Payload.(protocol.JoinResponse)
	if !ok {
		t.Fatal("failed to cast payload to JoinResponse")
	}

	if response.Accepted {
		t.Error("join should have been rejected for small terminal")
	}
}

func TestLobbyStateBroadcast(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	codec := protocol.NewCodec(conn)

	// Send join request
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}
	if err := codec.Encode(joinReq); err != nil {
		t.Fatalf("failed to send join request: %v", err)
	}

	// Receive join response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msg, err := codec.Decode()
	if err != nil {
		t.Fatalf("failed to receive join response: %v", err)
	}

	if msg.Type != protocol.MsgJoinResponse {
		t.Errorf("expected MsgJoinResponse, got %v", msg.Type)
	}

	// Receive lobby state
	msg, err = codec.Decode()
	if err != nil {
		t.Fatalf("failed to receive lobby state: %v", err)
	}

	if msg.Type != protocol.MsgLobbyState {
		t.Errorf("expected MsgLobbyState, got %v", msg.Type)
	}

	lobbyState, ok := msg.Payload.(protocol.LobbyState)
	if !ok {
		t.Fatal("failed to cast payload to LobbyState")
	}

	if len(lobbyState.Players) != 1 {
		t.Errorf("expected 1 player in lobby, got %d", len(lobbyState.Players))
	}

	if lobbyState.Players[0].Name != "TestPlayer" {
		t.Errorf("expected player name 'TestPlayer', got '%s'", lobbyState.Players[0].Name)
	}
}

func TestStartGame(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	codec := protocol.NewCodec(conn)

	// Send join request
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}
	if err := codec.Encode(joinReq); err != nil {
		t.Fatalf("failed to send join request: %v", err)
	}

	// Receive join response and lobby state
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	codec.Decode() // join response
	codec.Decode() // lobby state

	// Start the game
	s.StartGame()

	// Receive start game message
	msg, err := codec.Decode()
	if err != nil {
		t.Fatalf("failed to receive start game message: %v", err)
	}

	if msg.Type != protocol.MsgStartGame {
		t.Errorf("expected MsgStartGame, got %v", msg.Type)
	}

	// Receive game state
	msg, err = codec.Decode()
	if err != nil {
		t.Fatalf("failed to receive game state: %v", err)
	}

	if msg.Type != protocol.MsgGameState {
		t.Errorf("expected MsgGameState, got %v", msg.Type)
	}
}

func TestClientInput(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	conn, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	codec := protocol.NewCodec(conn)

	// Join
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}
	codec.Encode(joinReq)

	// Receive join response and lobby state
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	codec.Decode() // join response
	codec.Decode() // lobby state

	// Start game
	s.StartGame()
	codec.Decode() // start game

	// Send player input
	inputMsg := &protocol.Message{
		Type: protocol.MsgPlayerInput,
		Payload: protocol.PlayerInput{
			Direction: protocol.DirRight,
		},
	}
	if err := codec.Encode(inputMsg); err != nil {
		t.Fatalf("failed to send input: %v", err)
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Verify input was processed (player direction should be Right)
	s.mu.RLock()
	if s.gameState != nil && len(s.gameState.Players) > 0 {
		player := s.gameState.Players[0]
		if player.Direction != protocol.DirRight {
			t.Errorf("expected direction Right, got %v", player.Direction)
		}
	}
	s.mu.RUnlock()
}

func TestNewClient(t *testing.T) {
	// Create a mock connection using pipes
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	c := NewClient(1, server)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}

	if c.ID != 1 {
		t.Errorf("expected ID 1, got %d", c.ID)
	}

	if c.PlayerID != -1 {
		t.Errorf("expected PlayerID -1, got %d", c.PlayerID)
	}

	if c.Codec == nil {
		t.Error("Codec is nil")
	}
}

func TestClientSendAndClose(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	c := NewClient(1, server)
	c.StartWriter()

	// Send a message
	msg := &protocol.Message{
		Type: protocol.MsgLobbyState,
		Payload: protocol.LobbyState{
			Players:  []protocol.LobbyPlayer{},
			IsHost:   true,
			CanStart: false,
		},
	}
	c.Send(msg)

	// Read message on client side
	clientCodec := protocol.NewCodec(client)
	client.SetReadDeadline(time.Now().Add(time.Second))
	received, err := clientCodec.Decode()
	if err != nil {
		t.Fatalf("failed to receive message: %v", err)
	}

	if received.Type != protocol.MsgLobbyState {
		t.Errorf("expected MsgLobbyState, got %v", received.Type)
	}

	// Close client
	c.Close()

	// Verify double close doesn't panic
	c.Close()
}

func TestMultipleClients(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	// Connect two clients sequentially to avoid race conditions
	conn1, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	codec1 := protocol.NewCodec(conn1)

	// Join with client 1 first and wait for it to complete
	joinReq1 := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "Player1",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}
	codec1.Encode(joinReq1)

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	codec1.Decode() // join response
	codec1.Decode() // lobby state

	// Small delay before second client joins
	time.Sleep(50 * time.Millisecond)

	conn2, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect client 2: %v", err)
	}
	defer conn2.Close()

	codec2 := protocol.NewCodec(conn2)

	// Join with client 2
	joinReq2 := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "Player2",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}
	codec2.Encode(joinReq2)

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err := codec2.Decode() // join response
	if err != nil {
		t.Fatalf("failed to receive join response for client 2: %v", err)
	}

	if resp.Type != protocol.MsgJoinResponse {
		t.Errorf("expected MsgJoinResponse, got %v", resp.Type)
	}

	response, ok := resp.Payload.(protocol.JoinResponse)
	if !ok {
		t.Fatal("failed to cast payload")
	}

	if !response.Accepted {
		t.Error("client 2 join was not accepted")
	}

	// Verify server has 2 clients
	time.Sleep(100 * time.Millisecond)
	s.mu.RLock()
	clientCount := len(s.clients)
	s.mu.RUnlock()

	if clientCount != 2 {
		t.Errorf("expected 2 clients, got %d", clientCount)
	}
}

func TestBroadcast(t *testing.T) {
	s := NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	// Connect two clients
	conn1, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	codec1 := protocol.NewCodec(conn1)

	// Join client 1 first
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "Player1",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}

	codec1.Encode(joinReq)
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	codec1.Decode() // join response
	codec1.Decode() // lobby state

	// Small delay before connecting client 2
	time.Sleep(50 * time.Millisecond)

	conn2, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatalf("failed to connect client 2: %v", err)
	}
	defer conn2.Close()

	codec2 := protocol.NewCodec(conn2)

	joinReq2 := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "Player2",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	}

	codec2.Encode(joinReq2)
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	codec2.Decode() // join response
	codec2.Decode() // lobby state

	// Client 1 may have received a lobby state update when client 2 joined
	// Drain any pending messages with short timeout
	conn1.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		_, err := codec1.Decode()
		if err != nil {
			break
		}
	}

	// Reset deadline and start game
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Start game to trigger broadcast
	s.StartGame()

	// Both clients should receive start game message
	msg1, err := codec1.Decode()
	if err != nil {
		t.Fatalf("client 1 failed to receive: %v", err)
	}
	if msg1.Type != protocol.MsgStartGame {
		t.Errorf("client 1: expected MsgStartGame, got %v", msg1.Type)
	}

	msg2, err := codec2.Decode()
	if err != nil {
		t.Fatalf("client 2 failed to receive: %v", err)
	}
	if msg2.Type != protocol.MsgStartGame {
		t.Errorf("client 2: expected MsgStartGame, got %v", msg2.Type)
	}
}
