package protocol

import "encoding/gob"

// Direction represents movement direction
type Direction int

const (
	DirNone Direction = iota
	DirUp
	DirDown
	DirLeft
	DirRight
)

// MessageType identifies the kind of message
type MessageType int

const (
	MsgPlayerInput MessageType = iota
	MsgGameState
	MsgLobbyState
	MsgJoinRequest
	MsgJoinResponse
	MsgStartGame
	MsgGameOver
	MsgPlayerEliminated
)

// Message is the envelope for all network communication
type Message struct {
	Type    MessageType
	Payload interface{}
}

// PlayerInput sent from client to server
type PlayerInput struct {
	Direction Direction
}

// JoinRequest sent when client connects
type JoinRequest struct {
	PlayerName     string
	TerminalWidth  int
	TerminalHeight int
}

// JoinResponse sent by server after join
type JoinResponse struct {
	PlayerID int
	Accepted bool
	Reason   string
}

// Position on the board
type Position struct {
	X, Y int
}

// PlayerState represents a single player's state
type PlayerState struct {
	ID        int
	Name      string
	Color     int
	Position  Position
	Direction Direction
	Trail     []Position
	Territory []Position
	Score     int
	Alive     bool
	Protected bool // spawn protection
}

// GameState broadcast by server each tick
type GameState struct {
	Tick        int
	Players     []PlayerState
	BoardWidth  int
	BoardHeight int
	TimeLeft    int // seconds
	Powerups    []PowerupState
}

// PowerupState for active powerups on board
type PowerupState struct {
	Type     PowerupType
	Position Position
	TTL      int // ticks until despawn
}

// PowerupType identifies powerup kinds
type PowerupType int

const (
	PowerupSpeed PowerupType = iota
	PowerupShield
	PowerupFreeze
)

// LobbyState broadcast while waiting
type LobbyState struct {
	Players     []LobbyPlayer
	IsHost      bool
	CanStart    bool
	ServerAddrs []string // IP addresses to connect to (shown to host)
}

// LobbyPlayer in waiting room
type LobbyPlayer struct {
	ID    int
	Name  string
	Color int
	Ready bool
}

// GameOverState sent when game ends
type GameOverState struct {
	Rankings []PlayerRanking
	Reason   string // "time", "territory", "laststanding"
}

// PlayerRanking for final results
type PlayerRanking struct {
	Rank         int
	PlayerID     int
	Name         string
	Score        int
	SurvivalTime int // seconds
}

func init() {
	// Register types for gob encoding
	gob.Register(PlayerInput{})
	gob.Register(JoinRequest{})
	gob.Register(JoinResponse{})
	gob.Register(GameState{})
	gob.Register(LobbyState{})
	gob.Register(GameOverState{})
	gob.Register(Position{})
	gob.Register(PlayerState{})
	gob.Register(PowerupState{})
	gob.Register(LobbyPlayer{})
	gob.Register(PlayerRanking{})
}
