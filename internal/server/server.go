package server

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/diegok/pixwar/internal/config"
	"github.com/diegok/pixwar/internal/game"
	"github.com/diegok/pixwar/internal/protocol"
)

const (
	TickRate      = 20
	MinTermWidth  = 40
	MinTermHeight = 20
)

// Server manages the game and client connections
type Server struct {
	cfg          *config.Config
	listener     net.Listener
	mu           sync.RWMutex
	clients      map[int]*Client
	nextID       int
	gameState    *game.GameState
	inLobby      bool
	inRematch    bool              // waiting for rematch
	rematchReady map[int]bool      // clients ready for rematch
	minWidth     int
	minHeight    int
	done         chan struct{}
	started      bool
}

// NewServer creates a new server on the specified port
func NewServer(port int) *Server {
	cfg := &config.Config{
		Port:         port,
		GameDuration: config.DefaultGameDuration,
		Threshold:    config.DefaultThreshold,
	}
	return NewServerWithConfig(cfg)
}

// NewServerWithConfig creates a new server with the specified configuration
func NewServerWithConfig(cfg *config.Config) *Server {
	return &Server{
		cfg:          cfg,
		clients:      make(map[int]*Client),
		nextID:       1,
		inLobby:      true,
		inRematch:    false,
		rematchReady: make(map[int]bool),
		minWidth:     MinTermWidth,
		minHeight:    MinTermHeight,
		done:         make(chan struct{}),
	}
}

// Start begins listening for connections
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener
	s.started = true

	go s.acceptLoop()

	return nil
}

// Addr returns the server's listening address
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// GetServerAddresses returns all IP addresses clients can use to connect
func (s *Server) GetServerAddresses() []string {
	var addresses []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{fmt.Sprintf("localhost:%d", s.cfg.Port)}
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP
		if ip.IsLoopback() || ip.To4() == nil {
			continue
		}

		addresses = append(addresses, fmt.Sprintf("%s:%d", ip.String(), s.cfg.Port))
	}

	// Always include localhost
	addresses = append(addresses, fmt.Sprintf("localhost:%d", s.cfg.Port))

	return addresses
}

// ResetForRematch enters rematch waiting state
func (s *Server) ResetForRematch() {
	s.mu.Lock()
	s.gameState = nil
	s.inLobby = false
	s.inRematch = true
	s.rematchReady = make(map[int]bool)
	s.mu.Unlock()

	// Broadcast rematch state to all clients
	s.BroadcastRematchState()
}

// SetClientRematchReady marks a client as ready for rematch
func (s *Server) SetClientRematchReady(clientID int) {
	s.mu.Lock()
	s.rematchReady[clientID] = true
	s.mu.Unlock()
	s.BroadcastRematchState()
}

// BroadcastRematchState sends the current rematch waiting state to all clients
func (s *Server) BroadcastRematchState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.inRematch {
		return
	}

	players := make([]protocol.RematchPlayer, 0, len(s.clients))
	allReady := true
	for _, client := range s.clients {
		if client.Name != "" {
			ready := s.rematchReady[client.ID]
			if !ready {
				allReady = false
			}
			players = append(players, protocol.RematchPlayer{
				ID:    client.ID,
				Name:  client.Name,
				Color: (client.ID - 1) % 8,
				Ready: ready,
			})
		}
	}

	for _, client := range s.clients {
		isHost := client.ID == 1
		rematchState := protocol.RematchState{
			Players:  players,
			IsHost:   isHost,
			AllReady: allReady,
		}
		msg := &protocol.Message{
			Type:    protocol.MsgRematchState,
			Payload: rematchState,
		}
		client.Send(msg)
	}
}

// StartGameWithCountdown broadcasts countdown then starts game
func (s *Server) StartGameWithCountdown() {
	// Broadcast countdown 3, 2, 1
	for i := 3; i >= 1; i-- {
		msg := &protocol.Message{
			Type:    protocol.MsgCountdown,
			Payload: protocol.Countdown{Seconds: i},
		}
		s.broadcast(msg)
		time.Sleep(time.Second)
	}

	// Reset rematch state and start game
	s.mu.Lock()
	s.inRematch = false
	s.inLobby = true
	s.mu.Unlock()

	s.StartGame()
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	s.mu.Lock()
	select {
	case <-s.done:
		// Already stopped
		s.mu.Unlock()
		return
	default:
		close(s.done)
	}
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	// Close all client connections
	s.mu.RLock()
	for _, client := range s.clients {
		client.Close()
	}
	s.mu.RUnlock()
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection processes a new client connection
func (s *Server) handleConnection(conn net.Conn) {
	s.mu.Lock()
	clientID := s.nextID
	s.nextID++
	client := NewClient(clientID, conn)
	s.clients[clientID] = client
	s.mu.Unlock()

	client.StartWriter()

	// Wait for join request
	msg, err := client.Codec.Decode()
	if err != nil {
		s.removeClient(clientID)
		return
	}

	if msg.Type != protocol.MsgJoinRequest {
		s.removeClient(clientID)
		return
	}

	joinReq, ok := msg.Payload.(protocol.JoinRequest)
	if !ok {
		s.removeClient(clientID)
		return
	}

	// Validate terminal size
	if joinReq.TerminalWidth < MinTermWidth || joinReq.TerminalHeight < MinTermHeight {
		response := &protocol.Message{
			Type: protocol.MsgJoinResponse,
			Payload: protocol.JoinResponse{
				Accepted: false,
				Reason:   fmt.Sprintf("terminal too small (min %dx%d)", MinTermWidth, MinTermHeight),
			},
		}
		client.SendDirect(response)
		s.removeClient(clientID)
		return
	}

	// Update client info
	client.Name = joinReq.PlayerName
	client.Width = joinReq.TerminalWidth
	client.Height = joinReq.TerminalHeight

	// Update minimum board size
	s.mu.Lock()
	if joinReq.TerminalWidth < s.minWidth || s.minWidth == MinTermWidth {
		s.minWidth = joinReq.TerminalWidth
	}
	if joinReq.TerminalHeight < s.minHeight || s.minHeight == MinTermHeight {
		s.minHeight = joinReq.TerminalHeight
	}
	s.mu.Unlock()

	// Send accept response
	response := &protocol.Message{
		Type: protocol.MsgJoinResponse,
		Payload: protocol.JoinResponse{
			PlayerID: clientID,
			Accepted: true,
		},
	}
	if err := client.SendDirect(response); err != nil {
		s.removeClient(clientID)
		return
	}

	client.PlayerID = clientID

	// Broadcast updated lobby state
	s.BroadcastLobbyState()

	// Handle messages from client
	for {
		select {
		case <-s.done:
			return
		case <-client.done:
			return
		default:
		}

		msg, err := client.Codec.Decode()
		if err != nil {
			s.removeClient(clientID)
			return
		}

		s.handleMessage(client, msg)
	}
}

// handleMessage processes a message from a client
func (s *Server) handleMessage(client *Client, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgPlayerInput:
		if !s.inLobby && !s.inRematch && s.gameState != nil {
			input, ok := msg.Payload.(protocol.PlayerInput)
			if ok {
				s.mu.Lock()
				s.gameState.ProcessInput(client.PlayerID, input.Direction)
				s.mu.Unlock()
			}
		}
	case protocol.MsgRematchReady:
		if s.inRematch {
			s.mu.Lock()
			s.rematchReady[client.ID] = true
			s.mu.Unlock()
			s.BroadcastRematchState()
		}
	}
}

// StartGame begins the game from the lobby
func (s *Server) StartGame() {
	s.mu.Lock()
	if !s.inLobby {
		s.mu.Unlock()
		return
	}

	// Calculate board size based on minimum client terminal size
	// Leave room for UI elements: 2 for borders, 16 for scoreboard (needs 15 chars + 1 gap)
	boardWidth := s.minWidth - 18
	boardHeight := s.minHeight - 4

	// Create game state
	durationSec := s.cfg.GameDuration * 60
	s.gameState = game.NewGameState(boardWidth, boardHeight, durationSec)
	s.gameState.PowerupsOn = s.cfg.PowerupsEnabled

	// Add players from connected clients (use client ID as player ID for color consistency)
	for _, client := range s.clients {
		if client.Name != "" {
			player := s.gameState.AddPlayer(client.ID, client.Name)
			if player != nil {
				client.PlayerID = player.ID
			}
		}
	}

	// Spawn players
	s.gameState.SpawnPlayers()

	s.inLobby = false
	s.mu.Unlock()

	// Broadcast game start
	startMsg := &protocol.Message{
		Type:    protocol.MsgStartGame,
		Payload: nil,
	}
	s.broadcast(startMsg)

	// Start game loop
	go s.gameLoop()
}

// gameLoop runs the main game tick loop
func (s *Server) gameLoop() {
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.gameState == nil {
				s.mu.Unlock()
				return
			}

			s.gameState.Tick()

			// Check for game over
			if s.gameState.GameOver {
				s.broadcastGameOver()
				s.mu.Unlock()
				return
			}

			// Broadcast game state
			state := s.gameState.ToProtocolState()
			s.mu.Unlock()

			msg := &protocol.Message{
				Type:    protocol.MsgGameState,
				Payload: state,
			}
			s.broadcast(msg)
		}
	}
}

// broadcast sends a message to all connected clients
func (s *Server) broadcast(msg *protocol.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		client.Send(msg)
	}
}

// BroadcastLobbyState sends the current lobby state to all clients
func (s *Server) BroadcastLobbyState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.inLobby {
		return
	}

	players := make([]protocol.LobbyPlayer, 0, len(s.clients))
	for _, client := range s.clients {
		if client.Name != "" {
			players = append(players, protocol.LobbyPlayer{
				ID:    client.ID,
				Name:  client.Name,
				Color: (client.ID - 1) % 8, // 0-based color assignment (IDs start at 1)
				Ready: true,
			})
		}
	}

	canStart := len(players) >= 1
	serverAddrs := s.GetServerAddresses()

	for _, client := range s.clients {
		// First client is the host
		isHost := client.ID == 1

		lobbyState := protocol.LobbyState{
			Players:         players,
			IsHost:          isHost,
			CanStart:        canStart,
			PowerupsEnabled: s.cfg.PowerupsEnabled,
		}

		// Only send server addresses to the host
		if isHost {
			lobbyState.ServerAddrs = serverAddrs
		}

		msg := &protocol.Message{
			Type:    protocol.MsgLobbyState,
			Payload: lobbyState,
		}
		client.Send(msg)
	}
}

// broadcastGameOver sends the game over state to all clients
func (s *Server) broadcastGameOver() {
	if s.gameState == nil {
		return
	}

	// Build rankings with survival time and color
	rankings := make([]protocol.PlayerRanking, len(s.gameState.Players))
	for i, p := range s.gameState.Players {
		// Convert survival ticks to seconds (20 ticks per second)
		survivalSec := p.SurvivalTicks / 20
		rankings[i] = protocol.PlayerRanking{
			PlayerID:     p.ID,
			Name:         p.Name,
			Score:        p.Score,
			SurvivalTime: survivalSec,
			Color:        p.Color,
		}
	}

	// Sort by score (simple bubble sort)
	for i := 0; i < len(rankings)-1; i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[j].Score > rankings[i].Score {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	// Assign ranks
	for i := range rankings {
		rankings[i].Rank = i + 1
	}

	gameOver := protocol.GameOverState{
		Rankings: rankings,
		Reason:   s.gameState.EndReason,
	}

	msg := &protocol.Message{
		Type:    protocol.MsgGameOver,
		Payload: gameOver,
	}

	for _, client := range s.clients {
		client.Send(msg)
	}
}

// removeClient removes a client from the server
func (s *Server) removeClient(clientID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok {
		return
	}

	client.Close()
	delete(s.clients, clientID)

	// If in game, remove player
	if s.gameState != nil && client.PlayerID >= 0 {
		s.gameState.RemovePlayer(client.PlayerID)
	}
}

// NotifyShutdown broadcasts a game over message to all clients before shutdown
func (s *Server) NotifyShutdown() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build rankings from current state if game is in progress
	var rankings []protocol.PlayerRanking
	if s.gameState != nil {
		rankings = make([]protocol.PlayerRanking, len(s.gameState.Players))
		for i, p := range s.gameState.Players {
			rankings[i] = protocol.PlayerRanking{
				PlayerID: p.ID,
				Name:     p.Name,
				Score:    p.Score,
			}
		}

		// Sort by score (descending)
		for i := 0; i < len(rankings)-1; i++ {
			for j := i + 1; j < len(rankings); j++ {
				if rankings[j].Score > rankings[i].Score {
					rankings[i], rankings[j] = rankings[j], rankings[i]
				}
			}
		}

		// Assign ranks
		for i := range rankings {
			rankings[i].Rank = i + 1
		}
	}

	gameOver := protocol.GameOverState{
		Rankings: rankings,
		Reason:   "shutdown",
	}

	msg := &protocol.Message{
		Type:    protocol.MsgGameOver,
		Payload: gameOver,
	}

	for _, client := range s.clients {
		client.Send(msg)
	}
}
