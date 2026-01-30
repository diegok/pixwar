package client

import (
	"net"
	"testing"
	"time"

	"github.com/diegok/pixwar/internal/protocol"
	"github.com/diegok/pixwar/internal/server"
)

func TestNewClient(t *testing.T) {
	c := NewClient("TestPlayer", 80, 24)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}

	if c.Name != "TestPlayer" {
		t.Errorf("expected name 'TestPlayer', got '%s'", c.Name)
	}

	if c.Width != 80 {
		t.Errorf("expected width 80, got %d", c.Width)
	}

	if c.Height != 24 {
		t.Errorf("expected height 24, got %d", c.Height)
	}

	if c.PlayerID != -1 {
		t.Errorf("expected initial PlayerID -1, got %d", c.PlayerID)
	}

	if c.connected {
		t.Error("client should not be connected initially")
	}

	if c.GameState == nil {
		t.Error("GameState channel is nil")
	}

	if c.LobbyState == nil {
		t.Error("LobbyState channel is nil")
	}

	if c.GameOver == nil {
		t.Error("GameOver channel is nil")
	}

	if c.GameStart == nil {
		t.Error("GameStart channel is nil")
	}

	if c.Error == nil {
		t.Error("Error channel is nil")
	}
}

func TestClientConnect(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	// Create and connect client
	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	if !c.IsConnected() {
		t.Error("client should be connected")
	}

	if c.PlayerID <= 0 {
		t.Errorf("expected positive PlayerID, got %d", c.PlayerID)
	}
}

func TestClientConnectRejected(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	// Create client with small terminal
	c := NewClient("TestPlayer", 20, 10)
	err = c.Connect(s.Addr())
	if err == nil {
		c.Close()
		t.Fatal("expected Connect to fail for small terminal")
	}

	if c.IsConnected() {
		t.Error("client should not be connected after rejection")
	}
}

func TestClientConnectInvalidAddress(t *testing.T) {
	c := NewClient("TestPlayer", 80, 24)
	err := c.Connect("invalid:address:format")
	if err == nil {
		c.Close()
		t.Fatal("expected Connect to fail for invalid address")
	}
}

func TestClientConnectNoServer(t *testing.T) {
	c := NewClient("TestPlayer", 80, 24)
	// Use a port that's unlikely to have a server
	err := c.Connect("127.0.0.1:59999")
	if err == nil {
		c.Close()
		t.Fatal("expected Connect to fail when no server")
	}
}

func TestClientDoubleConnect(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("first Connect failed: %v", err)
	}
	defer c.Close()

	// Second connect should fail
	err = c.Connect(s.Addr())
	if err == nil {
		t.Fatal("expected second Connect to fail")
	}
}

func TestClientSendInput(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	// Send input
	err = c.SendInput(protocol.DirRight)
	if err != nil {
		t.Errorf("SendInput failed: %v", err)
	}

	err = c.SendInput(protocol.DirUp)
	if err != nil {
		t.Errorf("SendInput failed: %v", err)
	}
}

func TestClientSendInputNotConnected(t *testing.T) {
	c := NewClient("TestPlayer", 80, 24)
	err := c.SendInput(protocol.DirRight)
	if err == nil {
		t.Fatal("expected SendInput to fail when not connected")
	}
}

func TestClientReceiveLobbyState(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	// Should receive lobby state after connecting
	select {
	case state := <-c.LobbyState:
		if len(state.Players) != 1 {
			t.Errorf("expected 1 player in lobby, got %d", len(state.Players))
		}
		if state.Players[0].Name != "TestPlayer" {
			t.Errorf("expected player name 'TestPlayer', got '%s'", state.Players[0].Name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for lobby state")
	}
}

func TestClientReceiveGameStart(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	// Drain lobby state
	select {
	case <-c.LobbyState:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for lobby state")
	}

	// Start game
	s.StartGame()

	// Should receive game start
	select {
	case <-c.GameStart:
		// Good
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for game start")
	}
}

func TestClientReceiveGameState(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	// Drain lobby state
	select {
	case <-c.LobbyState:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for lobby state")
	}

	// Start game
	s.StartGame()

	// Drain game start
	select {
	case <-c.GameStart:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for game start")
	}

	// Should receive game state
	select {
	case state := <-c.GameState:
		if len(state.Players) != 1 {
			t.Errorf("expected 1 player in game, got %d", len(state.Players))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for game state")
	}
}

func TestClientClose(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	c.Close()

	if c.IsConnected() {
		t.Error("client should not be connected after Close")
	}

	// Double close should not panic
	c.Close()
}

func TestClientCloseNotConnected(t *testing.T) {
	c := NewClient("TestPlayer", 80, 24)
	// Should not panic
	c.Close()
}

func TestClientIsConnected(t *testing.T) {
	c := NewClient("TestPlayer", 80, 24)

	if c.IsConnected() {
		t.Error("new client should not be connected")
	}

	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !c.IsConnected() {
		t.Error("client should be connected after Connect")
	}

	c.Close()

	if c.IsConnected() {
		t.Error("client should not be connected after Close")
	}
}

func TestClientReceiveError(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Drain initial lobby state
	select {
	case <-c.LobbyState:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for lobby state")
	}

	// Stop server to force error
	s.Stop()

	// Should receive error
	select {
	case err := <-c.Error:
		if err == nil {
			t.Error("expected non-nil error")
		}
	case <-time.After(2 * time.Second):
		// May not get error if connection closes cleanly
	}

	// Client should disconnect
	time.Sleep(100 * time.Millisecond)
	if c.IsConnected() {
		t.Error("client should disconnect after server stops")
	}
}

func TestMultipleClientsConnect(t *testing.T) {
	// Start server
	s := server.NewServer(0)
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.Stop()

	// Connect first client
	c1 := NewClient("Player1", 80, 24)
	err = c1.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect client 1 failed: %v", err)
	}
	defer c1.Close()

	// Drain lobby state for client 1
	select {
	case <-c1.LobbyState:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for client 1 lobby state")
	}

	// Connect second client
	c2 := NewClient("Player2", 80, 24)
	err = c2.Connect(s.Addr())
	if err != nil {
		t.Fatalf("Connect client 2 failed: %v", err)
	}
	defer c2.Close()

	// Both clients should have unique player IDs
	if c1.PlayerID == c2.PlayerID {
		t.Errorf("clients have same PlayerID: %d", c1.PlayerID)
	}

	// Client 2 should receive lobby state with both players
	select {
	case state := <-c2.LobbyState:
		if len(state.Players) != 2 {
			t.Errorf("expected 2 players in lobby, got %d", len(state.Players))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for client 2 lobby state")
	}
}

// mockServer creates a mock server for testing handshake edge cases
func mockServer(t *testing.T, handler func(net.Conn)) string {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		handler(conn)
	}()

	return listener.Addr().String()
}

func TestClientConnectInvalidResponse(t *testing.T) {
	addr := mockServer(t, func(conn net.Conn) {
		defer conn.Close()

		codec := protocol.NewCodec(conn)

		// Read join request
		_, err := codec.Decode()
		if err != nil {
			return
		}

		// Send wrong message type
		msg := &protocol.Message{
			Type:    protocol.MsgGameState,
			Payload: protocol.GameState{},
		}
		codec.Encode(msg)
	})

	c := NewClient("TestPlayer", 80, 24)
	err := c.Connect(addr)
	if err == nil {
		c.Close()
		t.Fatal("expected Connect to fail for invalid response type")
	}
}
