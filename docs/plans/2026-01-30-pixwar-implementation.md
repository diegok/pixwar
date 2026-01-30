# Pixwar Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a multiplayer Qix-inspired territory capture game as a single Go binary with TUI interface.

**Architecture:** TCP client-server with authoritative server. Single binary runs in server mode (`--server`) or client mode (`--join`). Server handles all game logic and broadcasts state at 20 ticks/sec. Clients send only directional input.

**Tech Stack:** Go 1.21+, tcell (TUI), encoding/gob (serialization), net (TCP)

---

## Phase 1: Project Foundation

### Task 1.1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `cmd/pixwar/main.go`

**Step 1: Create go.mod**

```bash
cd /home/diegok/devel/sandbox/fosdem-ia
go mod init github.com/diegok/pixwar
```

**Step 2: Create minimal main.go**

```go
// cmd/pixwar/main.go
package main

import "fmt"

func main() {
	fmt.Println("pixwar")
}
```

**Step 3: Verify it compiles**

Run: `go build -o pixwar ./cmd/pixwar`
Expected: Binary created, no errors

**Step 4: Verify it runs**

Run: `./pixwar`
Expected: Output "pixwar"

**Step 5: Commit**

```bash
git add go.mod cmd/pixwar/main.go
git commit -m "feat: initialize pixwar project structure"
```

---

### Task 1.2: Add CLI Flag Parsing

**Files:**
- Modify: `cmd/pixwar/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test for config parsing**

```go
// internal/config/config_test.go
package config

import "testing"

func TestParseArgs_ServerMode(t *testing.T) {
	cfg, err := ParseArgs([]string{"--server"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsServer {
		t.Error("expected IsServer to be true")
	}
	if cfg.Port != 7777 {
		t.Errorf("expected default port 7777, got %d", cfg.Port)
	}
}

func TestParseArgs_ClientMode(t *testing.T) {
	cfg, err := ParseArgs([]string{"--join", "192.168.1.100"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IsServer {
		t.Error("expected IsServer to be false")
	}
	if cfg.ServerAddr != "192.168.1.100" {
		t.Errorf("expected server addr 192.168.1.100, got %s", cfg.ServerAddr)
	}
}

func TestParseArgs_ServerWithOptions(t *testing.T) {
	cfg, err := ParseArgs([]string{"--server", "--port", "8888", "--time", "10", "--threshold", "80", "--powerups"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Port)
	}
	if cfg.GameDuration != 10 {
		t.Errorf("expected duration 10, got %d", cfg.GameDuration)
	}
	if cfg.Threshold != 80 {
		t.Errorf("expected threshold 80, got %d", cfg.Threshold)
	}
	if !cfg.PowerupsEnabled {
		t.Error("expected powerups to be enabled")
	}
}

func TestParseArgs_NoMode_ReturnsError(t *testing.T) {
	_, err := ParseArgs([]string{})
	if err == nil {
		t.Error("expected error for missing mode")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL - package not found

**Step 3: Write minimal implementation**

```go
// internal/config/config.go
package config

import (
	"errors"
	"flag"
)

type Config struct {
	IsServer        bool
	ServerAddr      string
	Port            int
	GameDuration    int // minutes
	Threshold       int // percentage
	PowerupsEnabled bool
	PlayerName      string
}

func ParseArgs(args []string) (*Config, error) {
	cfg := &Config{
		Port:         7777,
		GameDuration: 5,
		Threshold:    95,
	}

	fs := flag.NewFlagSet("pixwar", flag.ContinueOnError)
	fs.BoolVar(&cfg.IsServer, "server", false, "Run as server")
	fs.StringVar(&cfg.ServerAddr, "join", "", "Server address to join")
	fs.IntVar(&cfg.Port, "port", 7777, "Server port")
	fs.IntVar(&cfg.GameDuration, "time", 5, "Game duration in minutes")
	fs.IntVar(&cfg.Threshold, "threshold", 95, "Territory percentage to end game")
	fs.BoolVar(&cfg.PowerupsEnabled, "powerups", false, "Enable power-ups")
	fs.StringVar(&cfg.PlayerName, "name", "", "Player name")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if !cfg.IsServer && cfg.ServerAddr == "" {
		return nil, errors.New("must specify --server or --join <address>")
	}

	return cfg, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -v`
Expected: PASS

**Step 5: Update main.go to use config**

```go
// cmd/pixwar/main.go
package main

import (
	"fmt"
	"os"

	"github.com/diegok/pixwar/internal/config"
)

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Usage: pixwar --server [options] | pixwar --join <address>\n")
		os.Exit(1)
	}

	if cfg.IsServer {
		fmt.Printf("Starting server on port %d\n", cfg.Port)
	} else {
		fmt.Printf("Connecting to %s:%d\n", cfg.ServerAddr, cfg.Port)
	}
}
```

**Step 6: Verify build and basic usage**

Run: `go build -o pixwar ./cmd/pixwar && ./pixwar --server`
Expected: "Starting server on port 7777"

Run: `./pixwar --join 192.168.1.1`
Expected: "Connecting to 192.168.1.1:7777"

**Step 7: Commit**

```bash
git add internal/config/ cmd/pixwar/main.go
git commit -m "feat: add CLI flag parsing for server/client modes"
```

---

## Phase 2: Protocol Layer

### Task 2.1: Define Core Message Types

**Files:**
- Create: `internal/protocol/types.go`
- Create: `internal/protocol/types_test.go`

**Step 1: Write the failing test for message types**

```go
// internal/protocol/types_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/protocol/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/protocol/types.go
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
	Players  []LobbyPlayer
	IsHost   bool
	CanStart bool
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/protocol/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: define protocol message types for network communication"
```

---

### Task 2.2: Add Message Encoding/Decoding Helpers

**Files:**
- Create: `internal/protocol/codec.go`
- Create: `internal/protocol/codec_test.go`

**Step 1: Write the failing test**

```go
// internal/protocol/codec_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/protocol/... -v`
Expected: FAIL - NewCodec undefined

**Step 3: Write implementation**

```go
// internal/protocol/codec.go
package protocol

import (
	"encoding/gob"
	"io"
)

// Codec handles message encoding/decoding
type Codec struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

// NewCodec creates a codec for the given read/writer
func NewCodec(rw io.ReadWriter) *Codec {
	return &Codec{
		enc: gob.NewEncoder(rw),
		dec: gob.NewDecoder(rw),
	}
}

// NewEncoder creates an encoder-only codec
func NewEncoder(w io.Writer) *Codec {
	return &Codec{
		enc: gob.NewEncoder(w),
	}
}

// NewDecoder creates a decoder-only codec
func NewDecoder(r io.Reader) *Codec {
	return &Codec{
		dec: gob.NewDecoder(r),
	}
}

// Encode writes a message
func (c *Codec) Encode(msg *Message) error {
	return c.enc.Encode(msg)
}

// Decode reads a message
func (c *Codec) Decode() (*Message, error) {
	var msg Message
	if err := c.dec.Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/protocol/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/protocol/codec.go internal/protocol/codec_test.go
git commit -m "feat: add protocol codec for message encoding/decoding"
```

---

## Phase 3: Game Core Logic

### Task 3.1: Implement Board and Basic Structures

**Files:**
- Create: `internal/game/board.go`
- Create: `internal/game/board_test.go`

**Step 1: Write the failing test**

```go
// internal/game/board_test.go
package game

import "testing"

func TestNewBoard(t *testing.T) {
	b := NewBoard(80, 40)
	if b.Width != 80 {
		t.Errorf("expected width 80, got %d", b.Width)
	}
	if b.Height != 40 {
		t.Errorf("expected height 40, got %d", b.Height)
	}
}

func TestBoard_GetCell_Empty(t *testing.T) {
	b := NewBoard(10, 10)
	cell := b.GetCell(5, 5)
	if cell.Owner != -1 {
		t.Errorf("expected owner -1 (unclaimed), got %d", cell.Owner)
	}
	if cell.IsTrail {
		t.Error("expected cell not to be trail")
	}
}

func TestBoard_SetTerritory(t *testing.T) {
	b := NewBoard(10, 10)
	b.SetTerritory(5, 5, 1) // player 1 claims cell
	cell := b.GetCell(5, 5)
	if cell.Owner != 1 {
		t.Errorf("expected owner 1, got %d", cell.Owner)
	}
}

func TestBoard_SetTrail(t *testing.T) {
	b := NewBoard(10, 10)
	b.SetTrail(3, 3, 2) // player 2 leaves trail
	cell := b.GetCell(3, 3)
	if cell.Owner != 2 {
		t.Errorf("expected owner 2, got %d", cell.Owner)
	}
	if !cell.IsTrail {
		t.Error("expected cell to be trail")
	}
}

func TestBoard_ClearTrail(t *testing.T) {
	b := NewBoard(10, 10)
	b.SetTrail(3, 3, 2)
	b.ClearTrail(3, 3)
	cell := b.GetCell(3, 3)
	if cell.IsTrail {
		t.Error("expected trail to be cleared")
	}
	if cell.Owner != -1 {
		t.Errorf("expected owner -1 after clear, got %d", cell.Owner)
	}
}

func TestBoard_IsOnEdge(t *testing.T) {
	b := NewBoard(10, 10)

	// Corners
	if !b.IsOnEdge(0, 0) {
		t.Error("(0,0) should be on edge")
	}
	if !b.IsOnEdge(9, 9) {
		t.Error("(9,9) should be on edge")
	}

	// Edges
	if !b.IsOnEdge(5, 0) {
		t.Error("(5,0) should be on edge")
	}
	if !b.IsOnEdge(0, 5) {
		t.Error("(0,5) should be on edge")
	}

	// Center
	if b.IsOnEdge(5, 5) {
		t.Error("(5,5) should not be on edge")
	}
}

func TestBoard_IsInBounds(t *testing.T) {
	b := NewBoard(10, 10)

	if !b.IsInBounds(0, 0) {
		t.Error("(0,0) should be in bounds")
	}
	if !b.IsInBounds(9, 9) {
		t.Error("(9,9) should be in bounds")
	}
	if b.IsInBounds(-1, 0) {
		t.Error("(-1,0) should be out of bounds")
	}
	if b.IsInBounds(10, 5) {
		t.Error("(10,5) should be out of bounds")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/game/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/game/board.go
package game

// Cell represents a single cell on the board
type Cell struct {
	Owner   int  // -1 = unclaimed, otherwise player ID
	IsTrail bool // true if this is an active trail (vulnerable)
}

// Board represents the game board
type Board struct {
	Width  int
	Height int
	cells  [][]Cell
}

// NewBoard creates a new board with given dimensions
func NewBoard(width, height int) *Board {
	cells := make([][]Cell, height)
	for y := range cells {
		cells[y] = make([]Cell, width)
		for x := range cells[y] {
			cells[y][x] = Cell{Owner: -1, IsTrail: false}
		}
	}
	return &Board{
		Width:  width,
		Height: height,
		cells:  cells,
	}
}

// GetCell returns the cell at the given position
func (b *Board) GetCell(x, y int) Cell {
	if !b.IsInBounds(x, y) {
		return Cell{Owner: -1, IsTrail: false}
	}
	return b.cells[y][x]
}

// SetTerritory marks a cell as owned territory (not trail)
func (b *Board) SetTerritory(x, y, playerID int) {
	if b.IsInBounds(x, y) {
		b.cells[y][x] = Cell{Owner: playerID, IsTrail: false}
	}
}

// SetTrail marks a cell as trail
func (b *Board) SetTrail(x, y, playerID int) {
	if b.IsInBounds(x, y) {
		b.cells[y][x] = Cell{Owner: playerID, IsTrail: true}
	}
}

// ClearTrail removes trail from a cell, making it unclaimed
func (b *Board) ClearTrail(x, y int) {
	if b.IsInBounds(x, y) {
		b.cells[y][x] = Cell{Owner: -1, IsTrail: false}
	}
}

// ConvertTrailToTerritory converts a trail cell to territory
func (b *Board) ConvertTrailToTerritory(x, y int) {
	if b.IsInBounds(x, y) && b.cells[y][x].IsTrail {
		b.cells[y][x].IsTrail = false
	}
}

// IsOnEdge returns true if position is on the board edge
func (b *Board) IsOnEdge(x, y int) bool {
	return x == 0 || y == 0 || x == b.Width-1 || y == b.Height-1
}

// IsInBounds returns true if position is within board
func (b *Board) IsInBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < b.Width && y < b.Height
}

// CountTerritory returns the number of cells owned by a player
func (b *Board) CountTerritory(playerID int) int {
	count := 0
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.cells[y][x].Owner == playerID && !b.cells[y][x].IsTrail {
				count++
			}
		}
	}
	return count
}

// TotalCells returns total board cells
func (b *Board) TotalCells() int {
	return b.Width * b.Height
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/game/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/game/
git commit -m "feat: implement board with cells, territory, and trail support"
```

---

### Task 3.2: Implement Player Movement

**Files:**
- Create: `internal/game/player.go`
- Create: `internal/game/player_test.go`

**Step 1: Write the failing test**

```go
// internal/game/player_test.go
package game

import (
	"testing"

	"github.com/diegok/pixwar/internal/protocol"
)

func TestNewPlayer(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
	if p.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", p.Name)
	}
	if !p.Alive {
		t.Error("expected player to be alive")
	}
}

func TestPlayer_SetPosition(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.SetPosition(10, 20)
	if p.X != 10 || p.Y != 20 {
		t.Errorf("expected position (10,20), got (%d,%d)", p.X, p.Y)
	}
}

func TestPlayer_Move_Up(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.SetPosition(5, 5)
	p.Direction = protocol.DirUp
	p.Move()
	if p.Y != 4 {
		t.Errorf("expected Y=4 after moving up, got %d", p.Y)
	}
}

func TestPlayer_Move_Down(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.SetPosition(5, 5)
	p.Direction = protocol.DirDown
	p.Move()
	if p.Y != 6 {
		t.Errorf("expected Y=6 after moving down, got %d", p.Y)
	}
}

func TestPlayer_Move_Left(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.SetPosition(5, 5)
	p.Direction = protocol.DirLeft
	p.Move()
	if p.X != 4 {
		t.Errorf("expected X=4 after moving left, got %d", p.X)
	}
}

func TestPlayer_Move_Right(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.SetPosition(5, 5)
	p.Direction = protocol.DirRight
	p.Move()
	if p.X != 6 {
		t.Errorf("expected X=6 after moving right, got %d", p.X)
	}
}

func TestPlayer_Move_None(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.SetPosition(5, 5)
	p.Direction = protocol.DirNone
	p.Move()
	if p.X != 5 || p.Y != 5 {
		t.Errorf("expected no movement, got (%d,%d)", p.X, p.Y)
	}
}

func TestPlayer_AddTrail(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.AddTrailPoint(1, 1)
	p.AddTrailPoint(1, 2)
	if len(p.Trail) != 2 {
		t.Errorf("expected 2 trail points, got %d", len(p.Trail))
	}
}

func TestPlayer_ClearTrail(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.AddTrailPoint(1, 1)
	p.AddTrailPoint(1, 2)
	p.ClearTrail()
	if len(p.Trail) != 0 {
		t.Errorf("expected empty trail, got %d points", len(p.Trail))
	}
}

func TestPlayer_IsOnOwnTrail(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.AddTrailPoint(1, 1)
	p.AddTrailPoint(1, 2)

	// Not on trail (head position is never in trail)
	if p.IsOnOwnTrail(5, 5) {
		t.Error("(5,5) should not be on trail")
	}

	// On trail
	if !p.IsOnOwnTrail(1, 1) {
		t.Error("(1,1) should be on trail")
	}
}

func TestPlayer_Eliminate(t *testing.T) {
	p := NewPlayer(1, "Alice", 0)
	p.Eliminate()
	if p.Alive {
		t.Error("expected player to be dead")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/game/... -v`
Expected: FAIL - NewPlayer undefined

**Step 3: Write implementation**

```go
// internal/game/player.go
package game

import "github.com/diegok/pixwar/internal/protocol"

// Point represents a 2D coordinate
type Point struct {
	X, Y int
}

// Player represents a game player
type Player struct {
	ID              int
	Name            string
	Color           int
	X, Y            int
	Direction       protocol.Direction
	Trail           []Point
	Score           int
	Alive           bool
	Protected       bool // spawn protection
	ProtectionTicks int  // remaining protection ticks
}

// NewPlayer creates a new player
func NewPlayer(id int, name string, color int) *Player {
	return &Player{
		ID:    id,
		Name:  name,
		Color: color,
		Alive: true,
		Trail: make([]Point, 0),
	}
}

// SetPosition sets the player's position
func (p *Player) SetPosition(x, y int) {
	p.X = x
	p.Y = y
}

// Move updates position based on current direction
func (p *Player) Move() {
	switch p.Direction {
	case protocol.DirUp:
		p.Y--
	case protocol.DirDown:
		p.Y++
	case protocol.DirLeft:
		p.X--
	case protocol.DirRight:
		p.X++
	}
}

// AddTrailPoint adds a point to the player's trail
func (p *Player) AddTrailPoint(x, y int) {
	p.Trail = append(p.Trail, Point{X: x, Y: y})
}

// ClearTrail removes all trail points
func (p *Player) ClearTrail() {
	p.Trail = p.Trail[:0]
}

// IsOnOwnTrail checks if a position is on this player's trail
func (p *Player) IsOnOwnTrail(x, y int) bool {
	for _, pt := range p.Trail {
		if pt.X == x && pt.Y == y {
			return true
		}
	}
	return false
}

// Eliminate marks player as dead
func (p *Player) Eliminate() {
	p.Alive = false
}

// SetDirection changes direction (prevents 180 turns)
func (p *Player) SetDirection(dir protocol.Direction) {
	// Prevent reversing direction
	if p.Direction == protocol.DirUp && dir == protocol.DirDown {
		return
	}
	if p.Direction == protocol.DirDown && dir == protocol.DirUp {
		return
	}
	if p.Direction == protocol.DirLeft && dir == protocol.DirRight {
		return
	}
	if p.Direction == protocol.DirRight && dir == protocol.DirLeft {
		return
	}
	p.Direction = dir
}

// DecrementProtection reduces protection ticks
func (p *Player) DecrementProtection() {
	if p.ProtectionTicks > 0 {
		p.ProtectionTicks--
		if p.ProtectionTicks == 0 {
			p.Protected = false
		}
	}
}

// GrantProtection gives spawn protection
func (p *Player) GrantProtection(ticks int) {
	p.Protected = true
	p.ProtectionTicks = ticks
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/game/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/game/player.go internal/game/player_test.go
git commit -m "feat: implement player movement and trail tracking"
```

---

### Task 3.3: Implement Territory Capture (Flood Fill)

**Files:**
- Modify: `internal/game/board.go`
- Modify: `internal/game/board_test.go`

**Step 1: Write the failing test for flood fill**

Add to `internal/game/board_test.go`:

```go
func TestBoard_FloodFill_SimpleCapture(t *testing.T) {
	b := NewBoard(10, 10)

	// Create a trail that forms a closed loop touching an edge
	// Trail goes: (0,2) -> (3,2) -> (3,5) -> (0,5)
	trail := []Point{
		{0, 2}, {1, 2}, {2, 2}, {3, 2},
		{3, 3}, {3, 4}, {3, 5},
		{2, 5}, {1, 5}, {0, 5},
	}

	captured := b.CaptureTerritory(1, trail)

	// Should capture the interior cells plus the trail
	if captured < 10 {
		t.Errorf("expected at least trail cells captured, got %d", captured)
	}

	// Trail cells should now be territory
	cell := b.GetCell(1, 2)
	if cell.Owner != 1 {
		t.Errorf("trail cell (1,2) should be owned by player 1")
	}
	if cell.IsTrail {
		t.Error("trail cell should be converted to territory")
	}
}

func TestBoard_CaptureTerritory_FillsInterior(t *testing.T) {
	b := NewBoard(10, 10)

	// Create a box trail from edge
	// (0,1) -> (4,1) -> (4,4) -> (0,4)
	trail := []Point{
		{0, 1}, {1, 1}, {2, 1}, {3, 1}, {4, 1},
		{4, 2}, {4, 3}, {4, 4},
		{3, 4}, {2, 4}, {1, 4}, {0, 4},
	}

	b.CaptureTerritory(1, trail)

	// Interior cell (2,2) should be captured
	cell := b.GetCell(2, 2)
	if cell.Owner != 1 {
		t.Errorf("interior cell (2,2) should be owned by player 1, got owner %d", cell.Owner)
	}
}

func TestBoard_CaptureTerritory_SmallerSide(t *testing.T) {
	b := NewBoard(20, 10)

	// Vertical line from top to bottom edge at x=3
	// This divides board into left (3 columns) and right (16 columns)
	// Smaller side (left) should be captured
	trail := []Point{}
	for y := 0; y < 10; y++ {
		trail = append(trail, Point{3, y})
	}

	b.CaptureTerritory(1, trail)

	// Left side (x=0,1,2) should be captured (smaller)
	for x := 0; x < 3; x++ {
		cell := b.GetCell(x, 5)
		if cell.Owner != 1 {
			t.Errorf("cell (%d,5) should be captured, got owner %d", x, cell.Owner)
		}
	}

	// Right side should NOT be captured (larger)
	cell := b.GetCell(10, 5)
	if cell.Owner == 1 {
		t.Error("larger side should not be captured")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/game/... -v -run FloodFill`
Expected: FAIL - CaptureTerritory undefined

**Step 3: Write implementation**

Add to `internal/game/board.go`:

```go
// CaptureTerritory fills territory based on a closed trail
// Returns the number of cells captured
func (b *Board) CaptureTerritory(playerID int, trail []Point) int {
	if len(trail) == 0 {
		return 0
	}

	// First, mark all trail cells as territory
	for _, pt := range trail {
		b.SetTerritory(pt.X, pt.Y, playerID)
	}

	// Create a set of trail positions for quick lookup
	trailSet := make(map[Point]bool)
	for _, pt := range trail {
		trailSet[pt] = true
	}

	// Find regions separated by the trail
	// We'll flood fill from each non-trail, non-owned cell
	visited := make([][]bool, b.Height)
	for y := range visited {
		visited[y] = make([]bool, b.Width)
	}

	// Mark trail and existing territory as visited
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.cells[y][x].Owner == playerID {
				visited[y][x] = true
			}
		}
	}

	type region struct {
		cells      []Point
		touchesEdge bool
	}

	var regions []region

	// Find all disconnected regions
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if visited[y][x] {
				continue
			}

			// Flood fill this region
			r := region{cells: []Point{}, touchesEdge: false}
			stack := []Point{{x, y}}

			for len(stack) > 0 {
				pt := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				if pt.X < 0 || pt.Y < 0 || pt.X >= b.Width || pt.Y >= b.Height {
					continue
				}
				if visited[pt.Y][pt.X] {
					continue
				}

				visited[pt.Y][pt.X] = true
				r.cells = append(r.cells, pt)

				if b.IsOnEdge(pt.X, pt.Y) {
					r.touchesEdge = true
				}

				// Add neighbors
				stack = append(stack, Point{pt.X - 1, pt.Y})
				stack = append(stack, Point{pt.X + 1, pt.Y})
				stack = append(stack, Point{pt.X, pt.Y - 1})
				stack = append(stack, Point{pt.X, pt.Y + 1})
			}

			if len(r.cells) > 0 {
				regions = append(regions, r)
			}
		}
	}

	// Find the smallest region that doesn't touch an edge,
	// or if all touch edges, the smallest one overall
	captured := 0

	// If trail forms closed loop (both ends on edge or connecting to territory),
	// capture the smaller enclosed region
	var smallestEnclosed *region
	for i := range regions {
		r := &regions[i]
		// Skip regions that touch the edge if we have enclosed ones
		if !r.touchesEdge {
			if smallestEnclosed == nil || len(r.cells) < len(smallestEnclosed.cells) {
				smallestEnclosed = r
			}
		}
	}

	if smallestEnclosed != nil {
		// Capture the enclosed region
		for _, pt := range smallestEnclosed.cells {
			b.SetTerritory(pt.X, pt.Y, playerID)
			captured++
		}
	} else if len(regions) > 1 {
		// All regions touch edge, capture the smallest one
		var smallest *region
		for i := range regions {
			if smallest == nil || len(regions[i].cells) < len(smallest.cells) {
				smallest = &regions[i]
			}
		}
		if smallest != nil {
			for _, pt := range smallest.cells {
				b.SetTerritory(pt.X, pt.Y, playerID)
				captured++
			}
		}
	}

	// Add trail cells to captured count
	captured += len(trail)

	return captured
}

// GetPlayerTerritory returns all cells owned by a player (not trail)
func (b *Board) GetPlayerTerritory(playerID int) []Point {
	var cells []Point
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.cells[y][x].Owner == playerID && !b.cells[y][x].IsTrail {
				cells = append(cells, Point{X: x, Y: y})
			}
		}
	}
	return cells
}

// ClearPlayerTrails removes all trail cells for a player
func (b *Board) ClearPlayerTrails(playerID int) {
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.cells[y][x].Owner == playerID && b.cells[y][x].IsTrail {
				b.cells[y][x] = Cell{Owner: -1, IsTrail: false}
			}
		}
	}
}

// ClearPlayerTerritory removes all territory for a player (on disconnect/death)
func (b *Board) ClearPlayerTerritory(playerID int) {
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.cells[y][x].Owner == playerID {
				b.cells[y][x] = Cell{Owner: -1, IsTrail: false}
			}
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/game/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/game/board.go internal/game/board_test.go
git commit -m "feat: implement territory capture with flood fill algorithm"
```

---

### Task 3.4: Implement Game State Manager

**Files:**
- Create: `internal/game/state.go`
- Create: `internal/game/state_test.go`

**Step 1: Write the failing test**

```go
// internal/game/state_test.go
package game

import (
	"testing"

	"github.com/diegok/pixwar/internal/protocol"
)

func TestNewGameState(t *testing.T) {
	gs := NewGameState(80, 40, 300) // 5 min = 300 sec
	if gs.Board.Width != 80 {
		t.Errorf("expected board width 80, got %d", gs.Board.Width)
	}
	if gs.TimeLeft != 300 {
		t.Errorf("expected 300 seconds, got %d", gs.TimeLeft)
	}
}

func TestGameState_AddPlayer(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	p := gs.AddPlayer("Alice")
	if p == nil {
		t.Fatal("expected player to be added")
	}
	if p.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", p.Name)
	}
	if len(gs.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(gs.Players))
	}
}

func TestGameState_AddPlayer_MaxPlayers(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	for i := 0; i < 8; i++ {
		gs.AddPlayer("Player")
	}
	p := gs.AddPlayer("TooMany")
	if p != nil {
		t.Error("expected nil when max players reached")
	}
}

func TestGameState_SpawnPlayers(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	gs.AddPlayer("Alice")
	gs.AddPlayer("Bob")
	gs.SpawnPlayers()

	// Players should be at different positions on edges
	p1 := gs.Players[0]
	p2 := gs.Players[1]

	if p1.X == p2.X && p1.Y == p2.Y {
		t.Error("players should spawn at different positions")
	}

	// Players should be on edge
	if !gs.Board.IsOnEdge(p1.X, p1.Y) {
		t.Error("player 1 should spawn on edge")
	}
	if !gs.Board.IsOnEdge(p2.X, p2.Y) {
		t.Error("player 2 should spawn on edge")
	}
}

func TestGameState_ProcessInput(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	p := gs.AddPlayer("Alice")
	gs.SpawnPlayers()

	gs.ProcessInput(p.ID, protocol.DirRight)
	if p.Direction != protocol.DirRight {
		t.Errorf("expected direction Right, got %v", p.Direction)
	}
}

func TestGameState_Tick_PlayerMoves(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	p := gs.AddPlayer("Alice")
	p.SetPosition(40, 20)
	p.Direction = protocol.DirRight

	gs.Tick()

	if p.X != 41 {
		t.Errorf("expected X=41 after tick, got %d", p.X)
	}
}

func TestGameState_GetPlayer(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	p := gs.AddPlayer("Alice")

	found := gs.GetPlayer(p.ID)
	if found == nil {
		t.Fatal("expected to find player")
	}
	if found.Name != "Alice" {
		t.Errorf("expected Alice, got %s", found.Name)
	}
}

func TestGameState_RemovePlayer(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	p := gs.AddPlayer("Alice")
	gs.RemovePlayer(p.ID)

	if len(gs.Players) != 0 {
		t.Errorf("expected 0 players, got %d", len(gs.Players))
	}
}

func TestGameState_AliveCount(t *testing.T) {
	gs := NewGameState(80, 40, 300)
	gs.AddPlayer("Alice")
	gs.AddPlayer("Bob")

	if gs.AliveCount() != 2 {
		t.Errorf("expected 2 alive, got %d", gs.AliveCount())
	}

	gs.Players[0].Eliminate()
	if gs.AliveCount() != 1 {
		t.Errorf("expected 1 alive after elimination, got %d", gs.AliveCount())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/game/... -v -run GameState`
Expected: FAIL - NewGameState undefined

**Step 3: Write implementation**

```go
// internal/game/state.go
package game

import (
	"github.com/diegok/pixwar/internal/protocol"
)

const (
	MaxPlayers       = 8
	ProtectionTicks  = 40 // 2 seconds at 20 ticks/sec
)

// GameState holds all game state
type GameState struct {
	Board         *Board
	Players       []*Player
	Tick_         int
	TimeLeft      int // seconds
	TicksPerSec   int
	tickCounter   int
	PowerupsOn    bool
	Powerups      []*Powerup
	nextPlayerID  int
	GameOver      bool
	WinnerID      int
	EndReason     string
}

// Powerup represents an active powerup on the board
type Powerup struct {
	Type     protocol.PowerupType
	X, Y     int
	TTL      int // ticks until despawn
}

// NewGameState creates a new game state
func NewGameState(width, height, durationSec int) *GameState {
	return &GameState{
		Board:        NewBoard(width, height),
		Players:      make([]*Player, 0, MaxPlayers),
		TimeLeft:     durationSec,
		TicksPerSec:  20,
		nextPlayerID: 1,
	}
}

// AddPlayer adds a new player if room available
func (gs *GameState) AddPlayer(name string) *Player {
	if len(gs.Players) >= MaxPlayers {
		return nil
	}

	color := len(gs.Players) // 0-7 for 8 colors
	p := NewPlayer(gs.nextPlayerID, name, color)
	gs.nextPlayerID++
	gs.Players = append(gs.Players, p)
	return p
}

// GetPlayer finds a player by ID
func (gs *GameState) GetPlayer(id int) *Player {
	for _, p := range gs.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// RemovePlayer removes a player from the game
func (gs *GameState) RemovePlayer(id int) {
	for i, p := range gs.Players {
		if p.ID == id {
			gs.Board.ClearPlayerTerritory(id)
			gs.Players = append(gs.Players[:i], gs.Players[i+1:]...)
			return
		}
	}
}

// SpawnPlayers positions all players on edges
func (gs *GameState) SpawnPlayers() {
	n := len(gs.Players)
	if n == 0 {
		return
	}

	// Distribute players evenly along edges
	// Perimeter = 2 * (width + height - 2)
	perimeter := 2 * (gs.Board.Width + gs.Board.Height - 2)
	spacing := perimeter / n

	for i, p := range gs.Players {
		pos := (spacing * i) + (spacing / 2)
		x, y := gs.perimeterToXY(pos)
		p.SetPosition(x, y)
		p.GrantProtection(ProtectionTicks)

		// Set initial direction (inward)
		p.Direction = gs.initialDirection(x, y)
	}
}

// perimeterToXY converts perimeter position to coordinates
func (gs *GameState) perimeterToXY(pos int) (int, int) {
	w, h := gs.Board.Width, gs.Board.Height

	// Top edge
	if pos < w {
		return pos, 0
	}
	pos -= w

	// Right edge
	if pos < h-1 {
		return w - 1, pos + 1
	}
	pos -= h - 1

	// Bottom edge
	if pos < w-1 {
		return w - 2 - pos, h - 1
	}
	pos -= w - 1

	// Left edge
	return 0, h - 2 - pos
}

// initialDirection returns direction pointing inward from edge
func (gs *GameState) initialDirection(x, y int) protocol.Direction {
	w, h := gs.Board.Width, gs.Board.Height

	if y == 0 {
		return protocol.DirDown
	}
	if y == h-1 {
		return protocol.DirUp
	}
	if x == 0 {
		return protocol.DirRight
	}
	if x == w-1 {
		return protocol.DirLeft
	}
	return protocol.DirDown
}

// ProcessInput handles player input
func (gs *GameState) ProcessInput(playerID int, dir protocol.Direction) {
	p := gs.GetPlayer(playerID)
	if p != nil && p.Alive {
		p.SetDirection(dir)
	}
}

// Tick advances game state by one tick
func (gs *GameState) Tick() {
	if gs.GameOver {
		return
	}

	gs.Tick_++
	gs.tickCounter++

	// Decrement time every second
	if gs.tickCounter >= gs.TicksPerSec {
		gs.tickCounter = 0
		gs.TimeLeft--
	}

	// Move players and update trails
	for _, p := range gs.Players {
		if !p.Alive {
			continue
		}

		p.DecrementProtection()
		oldX, oldY := p.X, p.Y
		p.Move()

		// Check bounds
		if !gs.Board.IsInBounds(p.X, p.Y) {
			p.X, p.Y = oldX, oldY // undo move
			continue
		}

		// Handle trail and territory
		gs.handlePlayerMove(p, oldX, oldY)
	}

	// Check collisions
	gs.checkCollisions()

	// Check win conditions
	gs.checkWinConditions()
}

// handlePlayerMove manages trail creation and territory capture
func (gs *GameState) handlePlayerMove(p *Player, oldX, oldY int) {
	cell := gs.Board.GetCell(p.X, p.Y)
	wasOnTerritory := gs.Board.GetCell(oldX, oldY).Owner == p.ID && !gs.Board.GetCell(oldX, oldY).IsTrail
	isOnTerritory := cell.Owner == p.ID && !cell.IsTrail
	isOnEdge := gs.Board.IsOnEdge(p.X, p.Y)

	if len(p.Trail) > 0 {
		// We have an active trail
		if isOnTerritory || isOnEdge {
			// Close the trail - capture territory
			gs.Board.CaptureTerritory(p.ID, p.Trail)
			p.ClearTrail()
		}
	} else if !wasOnTerritory && !isOnTerritory && !isOnEdge {
		// Start a new trail (left territory or edge)
	}

	// If outside territory and not on edge, add to trail
	if !isOnTerritory && !isOnEdge {
		p.AddTrailPoint(p.X, p.Y)
		gs.Board.SetTrail(p.X, p.Y, p.ID)
	}
}

// checkCollisions checks for player-trail collisions
func (gs *GameState) checkCollisions() {
	for _, p := range gs.Players {
		if !p.Alive || p.Protected {
			continue
		}

		// Check if player hit another player's trail
		cell := gs.Board.GetCell(p.X, p.Y)
		if cell.IsTrail && cell.Owner != p.ID {
			// Hit another player's trail - both die (killer gets points)
			victim := gs.GetPlayer(cell.Owner)
			if victim != nil && victim.Alive {
				p.Score += victim.Score // transfer points
				victim.Eliminate()
				gs.Board.ClearPlayerTrails(victim.ID)
			}
		}

		// Check if another player hit our trail
		for _, other := range gs.Players {
			if other.ID == p.ID || !other.Alive {
				continue
			}
			if p.IsOnOwnTrail(other.X, other.Y) && !other.Protected {
				// Other player hit our trail
				other.Score += p.Score
				p.Eliminate()
				gs.Board.ClearPlayerTrails(p.ID)
				break
			}
		}
	}
}

// checkWinConditions checks if game should end
func (gs *GameState) checkWinConditions() {
	// Time up
	if gs.TimeLeft <= 0 {
		gs.endGame("time")
		return
	}

	// Last player standing
	alive := gs.AliveCount()
	if alive <= 1 && len(gs.Players) > 1 {
		gs.endGame("laststanding")
		return
	}

	// Territory threshold (check highest territory percentage)
	totalCells := gs.Board.TotalCells()
	for _, p := range gs.Players {
		if !p.Alive {
			continue
		}
		territory := gs.Board.CountTerritory(p.ID)
		pct := (territory * 100) / totalCells
		if pct >= 95 { // threshold
			gs.endGame("territory")
			return
		}
	}
}

// endGame marks game as over
func (gs *GameState) endGame(reason string) {
	gs.GameOver = true
	gs.EndReason = reason

	// Find winner (highest score or last alive)
	var winner *Player
	for _, p := range gs.Players {
		if winner == nil || p.Score > winner.Score {
			winner = p
		}
	}
	if winner != nil {
		gs.WinnerID = winner.ID
	}
}

// AliveCount returns number of alive players
func (gs *GameState) AliveCount() int {
	count := 0
	for _, p := range gs.Players {
		if p.Alive {
			count++
		}
	}
	return count
}

// ToProtocolState converts to protocol.GameState for network transmission
func (gs *GameState) ToProtocolState() protocol.GameState {
	players := make([]protocol.PlayerState, len(gs.Players))
	for i, p := range gs.Players {
		trail := make([]protocol.Position, len(p.Trail))
		for j, pt := range p.Trail {
			trail[j] = protocol.Position{X: pt.X, Y: pt.Y}
		}

		territory := gs.Board.GetPlayerTerritory(p.ID)
		protoTerritory := make([]protocol.Position, len(territory))
		for j, pt := range territory {
			protoTerritory[j] = protocol.Position{X: pt.X, Y: pt.Y}
		}

		players[i] = protocol.PlayerState{
			ID:        p.ID,
			Name:      p.Name,
			Color:     p.Color,
			Position:  protocol.Position{X: p.X, Y: p.Y},
			Direction: p.Direction,
			Trail:     trail,
			Territory: protoTerritory,
			Score:     p.Score,
			Alive:     p.Alive,
			Protected: p.Protected,
		}
	}

	powerups := make([]protocol.PowerupState, len(gs.Powerups))
	for i, pu := range gs.Powerups {
		powerups[i] = protocol.PowerupState{
			Type:     pu.Type,
			Position: protocol.Position{X: pu.X, Y: pu.Y},
			TTL:      pu.TTL,
		}
	}

	return protocol.GameState{
		Tick:        gs.Tick_,
		Players:     players,
		BoardWidth:  gs.Board.Width,
		BoardHeight: gs.Board.Height,
		TimeLeft:    gs.TimeLeft,
		Powerups:    powerups,
	}
}
```

**Step 4: Update protocol types to include Territory**

Add to `internal/protocol/types.go` in the `PlayerState` struct:

```go
// Add Territory field to PlayerState
type PlayerState struct {
	ID        int
	Name      string
	Color     int
	Position  Position
	Direction Direction
	Trail     []Position
	Territory []Position // Add this field
	Score     int
	Alive     bool
	Protected bool
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/game/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/game/state.go internal/game/state_test.go internal/protocol/types.go
git commit -m "feat: implement game state manager with tick processing"
```

---

## Phase 4: Server Implementation

### Task 4.1: Implement TCP Server and Client Connections

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`
- Create: `internal/server/client.go`

**Step 1: Write the failing test**

```go
// internal/server/server_test.go
package server

import (
	"net"
	"testing"
	"time"

	"github.com/diegok/pixwar/internal/protocol"
)

func TestServer_StartsAndAcceptsConnections(t *testing.T) {
	srv := NewServer(0) // port 0 = auto-assign

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Get the actual port
	addr := srv.Addr()
	if addr == "" {
		t.Fatal("server should have an address")
	}

	// Connect a client
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	srv.Stop()
}

func TestServer_HandlesJoinRequest(t *testing.T) {
	srv := NewServer(0)

	go srv.Start()
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send join request
	codec := protocol.NewCodec(conn)
	err = codec.Encode(&protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     "TestPlayer",
			TerminalWidth:  80,
			TerminalHeight: 24,
		},
	})
	if err != nil {
		t.Fatalf("failed to send join: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(time.Second))
	msg, err := codec.Decode()
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if msg.Type != protocol.MsgJoinResponse {
		t.Errorf("expected JoinResponse, got %v", msg.Type)
	}

	resp, ok := msg.Payload.(protocol.JoinResponse)
	if !ok {
		t.Fatal("payload is not JoinResponse")
	}
	if !resp.Accepted {
		t.Errorf("expected join to be accepted, reason: %s", resp.Reason)
	}

	srv.Stop()
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/server/... -v`
Expected: FAIL - package not found

**Step 3: Write the client connection handler**

```go
// internal/server/client.go
package server

import (
	"net"
	"sync"

	"github.com/diegok/pixwar/internal/protocol"
)

// Client represents a connected client
type Client struct {
	ID       int
	Name     string
	PlayerID int
	Conn     net.Conn
	Codec    *protocol.Codec
	Width    int
	Height   int

	mu       sync.Mutex
	sendCh   chan *protocol.Message
	done     chan struct{}
}

// NewClient creates a new client from a connection
func NewClient(id int, conn net.Conn) *Client {
	return &Client{
		ID:     id,
		Conn:   conn,
		Codec:  protocol.NewCodec(conn),
		sendCh: make(chan *protocol.Message, 64),
		done:   make(chan struct{}),
	}
}

// Send queues a message to be sent
func (c *Client) Send(msg *protocol.Message) {
	select {
	case c.sendCh <- msg:
	case <-c.done:
	default:
		// Channel full, drop message
	}
}

// SendDirect sends immediately (blocking)
func (c *Client) SendDirect(msg *protocol.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Codec.Encode(msg)
}

// Close shuts down the client
func (c *Client) Close() {
	close(c.done)
	c.Conn.Close()
}

// StartWriter starts the send goroutine
func (c *Client) StartWriter() {
	go func() {
		for {
			select {
			case msg := <-c.sendCh:
				c.mu.Lock()
				c.Codec.Encode(msg)
				c.mu.Unlock()
			case <-c.done:
				return
			}
		}
	}()
}
```

**Step 4: Write the server implementation**

```go
// internal/server/server.go
package server

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/diegok/pixwar/internal/config"
	"github.com/diegok/pixwar/internal/game"
	"github.com/diegok/pixwar/internal/protocol"
)

const (
	TickRate      = 20 // ticks per second
	MinTermWidth  = 40
	MinTermHeight = 20
)

// Server handles the game server
type Server struct {
	cfg      *config.Config
	listener net.Listener

	mu         sync.RWMutex
	clients    map[int]*Client
	nextID     int

	gameState  *game.GameState
	inLobby    bool

	minWidth   int
	minHeight  int

	done       chan struct{}
	started    bool
}

// NewServer creates a new server
func NewServer(port int) *Server {
	return &Server{
		cfg: &config.Config{
			Port:         port,
			GameDuration: 5,
			Threshold:    95,
		},
		clients:   make(map[int]*Client),
		nextID:    1,
		inLobby:   true,
		minWidth:  80,
		minHeight: 24,
		done:      make(chan struct{}),
	}
}

// NewServerWithConfig creates a server with full config
func NewServerWithConfig(cfg *config.Config) *Server {
	return &Server{
		cfg:       cfg,
		clients:   make(map[int]*Client),
		nextID:    1,
		inLobby:   true,
		minWidth:  80,
		minHeight: 24,
		done:      make(chan struct{}),
	}
}

// Start begins listening for connections
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = ln
	s.started = true

	log.Printf("Server listening on %s", ln.Addr().String())

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// Addr returns the server's address
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Stop shuts down the server
func (s *Server) Stop() {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for _, c := range s.clients {
		c.Close()
	}
	s.mu.Unlock()
}

// handleConnection processes a new client connection
func (s *Server) handleConnection(conn net.Conn) {
	s.mu.Lock()
	clientID := s.nextID
	s.nextID++
	client := NewClient(clientID, conn)
	s.clients[clientID] = client
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		if s.gameState != nil && client.PlayerID > 0 {
			s.gameState.RemovePlayer(client.PlayerID)
		}
		s.mu.Unlock()
		client.Close()
	}()

	client.StartWriter()

	// Wait for join request
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	msg, err := client.Codec.Decode()
	if err != nil {
		log.Printf("Client %d: failed to read join: %v", clientID, err)
		return
	}

	if msg.Type != protocol.MsgJoinRequest {
		log.Printf("Client %d: expected join request, got %v", clientID, msg.Type)
		return
	}

	req, ok := msg.Payload.(protocol.JoinRequest)
	if !ok {
		log.Printf("Client %d: invalid join payload", clientID)
		return
	}

	// Validate terminal size
	if req.TerminalWidth < MinTermWidth || req.TerminalHeight < MinTermHeight {
		client.SendDirect(&protocol.Message{
			Type: protocol.MsgJoinResponse,
			Payload: protocol.JoinResponse{
				Accepted: false,
				Reason:   fmt.Sprintf("Terminal too small. Minimum: %dx%d", MinTermWidth, MinTermHeight),
			},
		})
		return
	}

	// Accept the client
	client.Name = req.PlayerName
	client.Width = req.TerminalWidth
	client.Height = req.TerminalHeight

	// Update minimum dimensions
	s.mu.Lock()
	if req.TerminalWidth < s.minWidth {
		s.minWidth = req.TerminalWidth
	}
	if req.TerminalHeight < s.minHeight {
		s.minHeight = req.TerminalHeight
	}
	s.mu.Unlock()

	client.SendDirect(&protocol.Message{
		Type: protocol.MsgJoinResponse,
		Payload: protocol.JoinResponse{
			PlayerID: clientID,
			Accepted: true,
		},
	})

	conn.SetReadDeadline(time.Time{}) // Remove deadline

	// Handle messages
	for {
		select {
		case <-s.done:
			return
		default:
		}

		msg, err := client.Codec.Decode()
		if err != nil {
			log.Printf("Client %d: read error: %v", clientID, err)
			return
		}

		s.handleMessage(client, msg)
	}
}

// handleMessage processes a message from a client
func (s *Server) handleMessage(client *Client, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgPlayerInput:
		if input, ok := msg.Payload.(protocol.PlayerInput); ok && s.gameState != nil {
			s.gameState.ProcessInput(client.PlayerID, input.Direction)
		}
	}
}

// StartGame begins the game
func (s *Server) StartGame() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.inLobby {
		return
	}

	// Calculate board size based on client terminals
	boardWidth := s.minWidth - 2   // Leave margin
	boardHeight := s.minHeight - 4 // Leave space for UI

	s.gameState = game.NewGameState(boardWidth, boardHeight, s.cfg.GameDuration*60)
	s.gameState.PowerupsOn = s.cfg.PowerupsEnabled

	// Add all clients as players
	for _, c := range s.clients {
		p := s.gameState.AddPlayer(c.Name)
		if p != nil {
			c.PlayerID = p.ID
		}
	}

	s.gameState.SpawnPlayers()
	s.inLobby = false

	// Notify clients
	s.broadcast(&protocol.Message{
		Type: protocol.MsgStartGame,
	})

	// Start game loop
	go s.gameLoop()
}

// gameLoop runs the main game tick
func (s *Server) gameLoop() {
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.gameState == nil || s.gameState.GameOver {
				s.mu.Unlock()
				return
			}

			s.gameState.Tick()
			state := s.gameState.ToProtocolState()
			s.mu.Unlock()

			s.broadcast(&protocol.Message{
				Type:    protocol.MsgGameState,
				Payload: state,
			})
		}
	}
}

// broadcast sends a message to all clients
func (s *Server) broadcast(msg *protocol.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.clients {
		c.Send(msg)
	}
}

// BroadcastLobbyState sends lobby state to all clients
func (s *Server) BroadcastLobbyState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	players := make([]protocol.LobbyPlayer, 0, len(s.clients))
	for _, c := range s.clients {
		players = append(players, protocol.LobbyPlayer{
			ID:    c.ID,
			Name:  c.Name,
			Color: c.ID - 1,
			Ready: true,
		})
	}

	for i, c := range s.clients {
		isHost := i == 0 // First client is host
		msg := &protocol.Message{
			Type: protocol.MsgLobbyState,
			Payload: protocol.LobbyState{
				Players:  players,
				IsHost:   isHost,
				CanStart: len(s.clients) >= 1,
			},
		}
		c.Send(msg)
	}
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/server/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/server/
git commit -m "feat: implement TCP server with client connection handling"
```

---

### Task 4.2: Implement Client Connection

**Files:**
- Create: `internal/client/client.go`
- Create: `internal/client/client_test.go`

**Step 1: Write the failing test**

```go
// internal/client/client_test.go
package client

import (
	"net"
	"testing"
	"time"

	"github.com/diegok/pixwar/internal/protocol"
)

func TestClient_ConnectsAndJoins(t *testing.T) {
	// Start a mock server
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		codec := protocol.NewCodec(conn)
		msg, _ := codec.Decode()

		if msg.Type == protocol.MsgJoinRequest {
			codec.Encode(&protocol.Message{
				Type: protocol.MsgJoinResponse,
				Payload: protocol.JoinResponse{
					PlayerID: 1,
					Accepted: true,
				},
			})
		}
		<-serverDone
	}()

	// Create and connect client
	c := NewClient("TestPlayer", 80, 24)
	err = c.Connect(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer c.Close()

	// Wait for join response
	time.Sleep(100 * time.Millisecond)

	if c.PlayerID != 1 {
		t.Errorf("expected player ID 1, got %d", c.PlayerID)
	}

	close(serverDone)
}

func TestClient_SendsInput(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	inputReceived := make(chan protocol.Direction, 1)
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()

		codec := protocol.NewCodec(conn)
		// Read join
		codec.Decode()
		// Send response
		codec.Encode(&protocol.Message{
			Type: protocol.MsgJoinResponse,
			Payload: protocol.JoinResponse{
				PlayerID: 1,
				Accepted: true,
			},
		})

		// Read input
		msg, _ := codec.Decode()
		if input, ok := msg.Payload.(protocol.PlayerInput); ok {
			inputReceived <- input.Direction
		}
	}()

	c := NewClient("Test", 80, 24)
	c.Connect(ln.Addr().String())
	defer c.Close()

	time.Sleep(50 * time.Millisecond)
	c.SendInput(protocol.DirUp)

	select {
	case dir := <-inputReceived:
		if dir != protocol.DirUp {
			t.Errorf("expected DirUp, got %v", dir)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for input")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/client/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/client/client.go
package client

import (
	"fmt"
	"net"
	"sync"

	"github.com/diegok/pixwar/internal/protocol"
)

// Client handles connection to the game server
type Client struct {
	Name       string
	Width      int
	Height     int
	PlayerID   int

	conn       net.Conn
	codec      *protocol.Codec

	mu         sync.Mutex
	connected  bool

	// Channels for received state
	GameState  chan protocol.GameState
	LobbyState chan protocol.LobbyState
	GameOver   chan protocol.GameOverState
	GameStart  chan struct{}
	Error      chan error

	done       chan struct{}
}

// NewClient creates a new game client
func NewClient(name string, width, height int) *Client {
	return &Client{
		Name:       name,
		Width:      width,
		Height:     height,
		GameState:  make(chan protocol.GameState, 10),
		LobbyState: make(chan protocol.LobbyState, 10),
		GameOver:   make(chan protocol.GameOverState, 1),
		GameStart:  make(chan struct{}, 1),
		Error:      make(chan error, 1),
		done:       make(chan struct{}),
	}
}

// Connect establishes connection to the server
func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	c.conn = conn
	c.codec = protocol.NewCodec(conn)
	c.connected = true

	// Send join request
	err = c.codec.Encode(&protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     c.Name,
			TerminalWidth:  c.Width,
			TerminalHeight: c.Height,
		},
	})
	if err != nil {
		conn.Close()
		return fmt.Errorf("send join failed: %w", err)
	}

	// Wait for response
	msg, err := c.codec.Decode()
	if err != nil {
		conn.Close()
		return fmt.Errorf("read join response failed: %w", err)
	}

	if msg.Type != protocol.MsgJoinResponse {
		conn.Close()
		return fmt.Errorf("expected join response, got %v", msg.Type)
	}

	resp, ok := msg.Payload.(protocol.JoinResponse)
	if !ok {
		conn.Close()
		return fmt.Errorf("invalid join response payload")
	}

	if !resp.Accepted {
		conn.Close()
		return fmt.Errorf("join rejected: %s", resp.Reason)
	}

	c.PlayerID = resp.PlayerID

	// Start receiver
	go c.receiveLoop()

	return nil
}

// receiveLoop handles incoming messages
func (c *Client) receiveLoop() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		msg, err := c.codec.Decode()
		if err != nil {
			select {
			case c.Error <- err:
			default:
			}
			return
		}

		c.handleMessage(msg)
	}
}

// handleMessage processes received messages
func (c *Client) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgGameState:
		if state, ok := msg.Payload.(protocol.GameState); ok {
			select {
			case c.GameState <- state:
			default:
				// Drop if channel full
			}
		}
	case protocol.MsgLobbyState:
		if state, ok := msg.Payload.(protocol.LobbyState); ok {
			select {
			case c.LobbyState <- state:
			default:
			}
		}
	case protocol.MsgStartGame:
		select {
		case c.GameStart <- struct{}{}:
		default:
		}
	case protocol.MsgGameOver:
		if state, ok := msg.Payload.(protocol.GameOverState); ok {
			select {
			case c.GameOver <- state:
			default:
			}
		}
	}
}

// SendInput sends a direction input to the server
func (c *Client) SendInput(dir protocol.Direction) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	return c.codec.Encode(&protocol.Message{
		Type: protocol.MsgPlayerInput,
		Payload: protocol.PlayerInput{
			Direction: dir,
		},
	})
}

// Close disconnects from the server
func (c *Client) Close() {
	close(c.done)
	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/client/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/
git commit -m "feat: implement client connection and message handling"
```

---

## Phase 5: TUI Implementation

### Task 5.1: Add tcell Dependency and Basic Screen

**Files:**
- Modify: `go.mod`
- Create: `internal/ui/screen.go`
- Create: `internal/ui/screen_test.go`

**Step 1: Add tcell dependency**

```bash
go get github.com/gdamore/tcell/v2
```

**Step 2: Write the failing test**

```go
// internal/ui/screen_test.go
package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestNewScreen_ReturnsScreen(t *testing.T) {
	// Use simulation screen for testing
	simScreen := tcell.NewSimulationScreen("")
	simScreen.Init()
	defer simScreen.Fini()

	screen := NewScreen(simScreen)
	if screen == nil {
		t.Fatal("expected non-nil screen")
	}
}

func TestScreen_Size(t *testing.T) {
	simScreen := tcell.NewSimulationScreen("")
	simScreen.Init()
	simScreen.SetSize(80, 24)
	defer simScreen.Fini()

	screen := NewScreen(simScreen)
	w, h := screen.Size()

	if w != 80 {
		t.Errorf("expected width 80, got %d", w)
	}
	if h != 24 {
		t.Errorf("expected height 24, got %d", h)
	}
}

func TestScreen_DrawText(t *testing.T) {
	simScreen := tcell.NewSimulationScreen("")
	simScreen.Init()
	simScreen.SetSize(80, 24)
	defer simScreen.Fini()

	screen := NewScreen(simScreen)
	screen.DrawText(0, 0, "Hello", tcell.StyleDefault)
	screen.Show()

	// Check that text was drawn
	cells, _, _ := simScreen.GetContents()
	if len(cells) == 0 {
		t.Error("expected content to be drawn")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/ui/... -v`
Expected: FAIL - package not found

**Step 4: Write implementation**

```go
// internal/ui/screen.go
package ui

import (
	"github.com/gdamore/tcell/v2"
)

// Screen wraps tcell screen functionality
type Screen struct {
	screen tcell.Screen
}

// NewScreen creates a new screen wrapper
func NewScreen(s tcell.Screen) *Screen {
	return &Screen{screen: s}
}

// InitScreen initializes a real terminal screen
func InitScreen() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	s.EnableMouse()
	s.Clear()
	return &Screen{screen: s}, nil
}

// Size returns terminal dimensions
func (s *Screen) Size() (int, int) {
	return s.screen.Size()
}

// Clear clears the screen
func (s *Screen) Clear() {
	s.screen.Clear()
}

// Show syncs screen buffer to terminal
func (s *Screen) Show() {
	s.screen.Show()
}

// Fini cleans up the screen
func (s *Screen) Fini() {
	s.screen.Fini()
}

// SetCell sets a single cell
func (s *Screen) SetCell(x, y int, style tcell.Style, r rune) {
	s.screen.SetContent(x, y, r, nil, style)
}

// DrawText draws a string at position
func (s *Screen) DrawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		s.screen.SetContent(x+i, y, r, nil, style)
	}
}

// DrawBox draws a bordered box
func (s *Screen) DrawBox(x, y, w, h int, style tcell.Style) {
	// Corners
	s.SetCell(x, y, style, '┌')
	s.SetCell(x+w-1, y, style, '┐')
	s.SetCell(x, y+h-1, style, '└')
	s.SetCell(x+w-1, y+h-1, style, '┘')

	// Horizontal lines
	for i := x + 1; i < x+w-1; i++ {
		s.SetCell(i, y, style, '─')
		s.SetCell(i, y+h-1, style, '─')
	}

	// Vertical lines
	for i := y + 1; i < y+h-1; i++ {
		s.SetCell(x, i, style, '│')
		s.SetCell(x+w-1, i, style, '│')
	}
}

// FillRect fills a rectangle with a character
func (s *Screen) FillRect(x, y, w, h int, style tcell.Style, r rune) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			s.SetCell(x+dx, y+dy, style, r)
		}
	}
}

// PollEvent returns the next event
func (s *Screen) PollEvent() tcell.Event {
	return s.screen.PollEvent()
}

// PostEvent posts an event (for testing)
func (s *Screen) PostEvent(ev tcell.Event) {
	s.screen.PostEvent(ev)
}

// Player colors
var PlayerColors = []tcell.Color{
	tcell.ColorRed,
	tcell.ColorBlue,
	tcell.ColorGreen,
	tcell.ColorYellow,
	tcell.ColorPurple,
	tcell.ColorOrange,
	tcell.ColorTeal,
	tcell.ColorFuchsia,
}

// GetPlayerStyle returns style for a player
func GetPlayerStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Foreground(PlayerColors[colorIndex])
}

// GetPlayerBgStyle returns style with background for a player
func GetPlayerBgStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Background(PlayerColors[colorIndex])
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/ui/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/ui/ go.mod go.sum
git commit -m "feat: add tcell-based screen abstraction for TUI"
```

---

### Task 5.2: Implement Game Renderer

**Files:**
- Create: `internal/ui/renderer.go`
- Create: `internal/ui/renderer_test.go`

**Step 1: Write the failing test**

```go
// internal/ui/renderer_test.go
package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixwar/internal/protocol"
)

func TestRenderer_RenderGame(t *testing.T) {
	simScreen := tcell.NewSimulationScreen("")
	simScreen.Init()
	simScreen.SetSize(80, 24)
	defer simScreen.Fini()

	screen := NewScreen(simScreen)
	renderer := NewRenderer(screen)

	state := protocol.GameState{
		BoardWidth:  40,
		BoardHeight: 20,
		TimeLeft:    120,
		Players: []protocol.PlayerState{
			{
				ID:       1,
				Name:     "Alice",
				Color:    0,
				Position: protocol.Position{X: 10, Y: 5},
				Alive:    true,
				Score:    100,
			},
		},
	}

	// Should not panic
	renderer.RenderGame(state, 1)
	screen.Show()
}

func TestRenderer_RenderLobby(t *testing.T) {
	simScreen := tcell.NewSimulationScreen("")
	simScreen.Init()
	simScreen.SetSize(80, 24)
	defer simScreen.Fini()

	screen := NewScreen(simScreen)
	renderer := NewRenderer(screen)

	state := protocol.LobbyState{
		Players: []protocol.LobbyPlayer{
			{ID: 1, Name: "Alice", Color: 0},
			{ID: 2, Name: "Bob", Color: 1},
		},
		IsHost:   true,
		CanStart: true,
	}

	// Should not panic
	renderer.RenderLobby(state)
	screen.Show()
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/... -v -run Renderer`
Expected: FAIL - NewRenderer undefined

**Step 3: Write implementation**

```go
// internal/ui/renderer.go
package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixwar/internal/protocol"
)

// Renderer handles game rendering
type Renderer struct {
	screen *Screen
}

// NewRenderer creates a new renderer
func NewRenderer(screen *Screen) *Renderer {
	return &Renderer{screen: screen}
}

// RenderLobby renders the lobby screen
func (r *Renderer) RenderLobby(state protocol.LobbyState) {
	r.screen.Clear()
	w, h := r.screen.Size()

	// Title
	title := "PIXWAR - Waiting for Players"
	r.screen.DrawText((w-len(title))/2, 2, title, tcell.StyleDefault.Bold(true))

	// Player list
	y := 5
	r.screen.DrawText(10, y, "Players:", tcell.StyleDefault)
	y += 2

	for _, p := range state.Players {
		style := GetPlayerStyle(p.Color)
		text := fmt.Sprintf("  %s", p.Name)
		r.screen.DrawText(10, y, text, style)
		y++
	}

	// Instructions
	y = h - 4
	if state.IsHost {
		if state.CanStart {
			r.screen.DrawText(10, y, "Press ENTER to start the game", tcell.StyleDefault)
		} else {
			r.screen.DrawText(10, y, "Waiting for more players...", tcell.StyleDefault)
		}
	} else {
		r.screen.DrawText(10, y, "Waiting for host to start...", tcell.StyleDefault)
	}

	r.screen.DrawText(10, y+1, "Press Q to quit", tcell.StyleDefault.Dim(true))
}

// RenderGame renders the game screen
func (r *Renderer) RenderGame(state protocol.GameState, myID int) {
	r.screen.Clear()
	w, h := r.screen.Size()

	// Calculate board position (centered)
	boardX := (w - state.BoardWidth) / 2
	boardY := 1
	if boardX < 0 {
		boardX = 0
	}

	// Draw board border
	r.screen.DrawBox(boardX-1, boardY-1, state.BoardWidth+2, state.BoardHeight+2, tcell.StyleDefault)

	// Draw territory for each player
	for _, p := range state.Players {
		style := GetPlayerBgStyle(p.Color)
		for _, pos := range p.Territory {
			r.screen.SetCell(boardX+pos.X, boardY+pos.Y, style, ' ')
		}
	}

	// Draw trails
	for _, p := range state.Players {
		style := GetPlayerStyle(p.Color)
		for _, pos := range p.Trail {
			r.screen.SetCell(boardX+pos.X, boardY+pos.Y, style, '░')
		}
	}

	// Draw player heads
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		style := GetPlayerStyle(p.Color).Bold(true)
		char := '●'
		if p.Protected {
			char = '◆' // Different symbol for protected
		}
		r.screen.SetCell(boardX+p.Position.X, boardY+p.Position.Y, style, char)
	}

	// Draw powerups
	for _, pu := range state.Powerups {
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		var char rune
		switch pu.Type {
		case protocol.PowerupSpeed:
			char = '⚡'
		case protocol.PowerupShield:
			char = '+'
		case protocol.PowerupFreeze:
			char = '*'
		}
		r.screen.SetCell(boardX+pu.Position.X, boardY+pu.Position.Y, style, char)
	}

	// Draw status bar
	r.renderStatusBar(state, myID, h)
}

// renderStatusBar draws the bottom status bar
func (r *Renderer) renderStatusBar(state protocol.GameState, myID int, screenHeight int) {
	y := screenHeight - 2
	w, _ := r.screen.Size()

	// Find my player
	var myScore int
	for _, p := range state.Players {
		if p.ID == myID {
			myScore = p.Score
			break
		}
	}

	// Time remaining
	minutes := state.TimeLeft / 60
	seconds := state.TimeLeft % 60
	timeStr := fmt.Sprintf("Time: %d:%02d", minutes, seconds)
	r.screen.DrawText(2, y, timeStr, tcell.StyleDefault)

	// Score
	scoreStr := fmt.Sprintf("Score: %d", myScore)
	r.screen.DrawText(20, y, scoreStr, tcell.StyleDefault)

	// Players alive
	alive := 0
	total := len(state.Players)
	for _, p := range state.Players {
		if p.Alive {
			alive++
		}
	}
	playersStr := fmt.Sprintf("Players: %d/%d", alive, total)
	r.screen.DrawText(w-len(playersStr)-2, y, playersStr, tcell.StyleDefault)
}

// RenderGameOver renders the game over screen
func (r *Renderer) RenderGameOver(state protocol.GameOverState) {
	r.screen.Clear()
	w, h := r.screen.Size()

	// Title
	title := "GAME OVER"
	r.screen.DrawText((w-len(title))/2, 3, title, tcell.StyleDefault.Bold(true))

	// Reason
	var reason string
	switch state.Reason {
	case "time":
		reason = "Time's up!"
	case "territory":
		reason = "Territory threshold reached!"
	case "laststanding":
		reason = "Last player standing!"
	}
	r.screen.DrawText((w-len(reason))/2, 5, reason, tcell.StyleDefault)

	// Rankings
	y := 8
	r.screen.DrawText(10, y, "Final Rankings:", tcell.StyleDefault.Bold(true))
	y += 2

	for _, rank := range state.Rankings {
		line := fmt.Sprintf("#%d  %-15s  Score: %d", rank.Rank, rank.Name, rank.Score)
		r.screen.DrawText(10, y, line, tcell.StyleDefault)
		y++
	}

	// Instructions
	r.screen.DrawText(10, h-3, "Press Q to quit", tcell.StyleDefault)
}

// RenderSpectator renders spectator mode
func (r *Renderer) RenderSpectator(state protocol.GameState, watchingID int, myRank int, myScore int) {
	// Render game normally
	r.RenderGame(state, watchingID)

	w, _ := r.screen.Size()

	// Overlay spectator banner
	banner := fmt.Sprintf("ELIMINATED - Rank #%d - Score: %d | TAB to cycle players", myRank, myScore)
	r.screen.DrawText((w-len(banner))/2, 0, banner, tcell.StyleDefault.Background(tcell.ColorDarkRed).Foreground(tcell.ColorWhite))
}

// RenderConnecting shows connecting screen
func (r *Renderer) RenderConnecting(addr string) {
	r.screen.Clear()
	w, h := r.screen.Size()

	text := fmt.Sprintf("Connecting to %s...", addr)
	r.screen.DrawText((w-len(text))/2, h/2, text, tcell.StyleDefault)
}

// RenderError shows error screen
func (r *Renderer) RenderError(err string) {
	r.screen.Clear()
	w, h := r.screen.Size()

	title := "Error"
	r.screen.DrawText((w-len(title))/2, h/2-1, title, tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true))
	r.screen.DrawText((w-len(err))/2, h/2+1, err, tcell.StyleDefault)
	r.screen.DrawText((w-15)/2, h/2+3, "Press Q to quit", tcell.StyleDefault)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/renderer.go internal/ui/renderer_test.go
git commit -m "feat: implement game renderer for all screens"
```

---

### Task 5.3: Implement Input Handler

**Files:**
- Create: `internal/ui/input.go`
- Create: `internal/ui/input_test.go`

**Step 1: Write the failing test**

```go
// internal/ui/input_test.go
package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixwar/internal/protocol"
)

func TestKeyToDirection(t *testing.T) {
	tests := []struct {
		key      tcell.Key
		rune     rune
		expected protocol.Direction
	}{
		{tcell.KeyUp, 0, protocol.DirUp},
		{tcell.KeyDown, 0, protocol.DirDown},
		{tcell.KeyLeft, 0, protocol.DirLeft},
		{tcell.KeyRight, 0, protocol.DirRight},
		{tcell.KeyRune, 'w', protocol.DirUp},
		{tcell.KeyRune, 'W', protocol.DirUp},
		{tcell.KeyRune, 's', protocol.DirDown},
		{tcell.KeyRune, 'S', protocol.DirDown},
		{tcell.KeyRune, 'a', protocol.DirLeft},
		{tcell.KeyRune, 'A', protocol.DirLeft},
		{tcell.KeyRune, 'd', protocol.DirRight},
		{tcell.KeyRune, 'D', protocol.DirRight},
		{tcell.KeyRune, 'x', protocol.DirNone},
	}

	for _, tt := range tests {
		dir := KeyToDirection(tt.key, tt.rune)
		if dir != tt.expected {
			t.Errorf("KeyToDirection(%v, %c) = %v, want %v", tt.key, tt.rune, dir, tt.expected)
		}
	}
}

func TestIsQuitKey(t *testing.T) {
	if !IsQuitKey(tcell.KeyRune, 'q') {
		t.Error("'q' should be quit key")
	}
	if !IsQuitKey(tcell.KeyRune, 'Q') {
		t.Error("'Q' should be quit key")
	}
	if !IsQuitKey(tcell.KeyEscape, 0) {
		t.Error("Escape should be quit key")
	}
	if IsQuitKey(tcell.KeyRune, 'x') {
		t.Error("'x' should not be quit key")
	}
}

func TestIsStartKey(t *testing.T) {
	if !IsStartKey(tcell.KeyEnter) {
		t.Error("Enter should be start key")
	}
	if IsStartKey(tcell.KeyRune) {
		t.Error("Rune key should not be start key")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/... -v -run Key`
Expected: FAIL - KeyToDirection undefined

**Step 3: Write implementation**

```go
// internal/ui/input.go
package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixwar/internal/protocol"
)

// KeyToDirection converts a key press to a direction
func KeyToDirection(key tcell.Key, r rune) protocol.Direction {
	switch key {
	case tcell.KeyUp:
		return protocol.DirUp
	case tcell.KeyDown:
		return protocol.DirDown
	case tcell.KeyLeft:
		return protocol.DirLeft
	case tcell.KeyRight:
		return protocol.DirRight
	case tcell.KeyRune:
		switch r {
		case 'w', 'W':
			return protocol.DirUp
		case 's', 'S':
			return protocol.DirDown
		case 'a', 'A':
			return protocol.DirLeft
		case 'd', 'D':
			return protocol.DirRight
		}
	}
	return protocol.DirNone
}

// IsQuitKey checks if the key is a quit key
func IsQuitKey(key tcell.Key, r rune) bool {
	if key == tcell.KeyEscape || key == tcell.KeyCtrlC {
		return true
	}
	if key == tcell.KeyRune && (r == 'q' || r == 'Q') {
		return true
	}
	return false
}

// IsStartKey checks if the key starts the game
func IsStartKey(key tcell.Key) bool {
	return key == tcell.KeyEnter
}

// IsTabKey checks for tab (spectator cycling)
func IsTabKey(key tcell.Key) bool {
	return key == tcell.KeyTab
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/input.go internal/ui/input_test.go
git commit -m "feat: implement input handling for keyboard controls"
```

---

## Phase 6: Main Application Loop

### Task 6.1: Implement Application Controller

**Files:**
- Create: `internal/app/app.go`
- Modify: `cmd/pixwar/main.go`

**Step 1: Write the application controller**

```go
// internal/app/app.go
package app

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixwar/internal/client"
	"github.com/diegok/pixwar/internal/config"
	"github.com/diegok/pixwar/internal/protocol"
	"github.com/diegok/pixwar/internal/server"
	"github.com/diegok/pixwar/internal/ui"
)

// App is the main application
type App struct {
	cfg      *config.Config
	screen   *ui.Screen
	renderer *ui.Renderer
	client   *client.Client
	server   *server.Server

	// State
	inLobby     bool
	inGame      bool
	gameOver    bool
	spectating  bool
	lobbyState  protocol.LobbyState
	gameState   protocol.GameState
	overState   protocol.GameOverState
	myRank      int
	myScore     int
	watchingID  int
}

// NewApp creates a new application
func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:     cfg,
		inLobby: true,
	}
}

// Run starts the application
func (a *App) Run() error {
	// Initialize screen
	screen, err := ui.InitScreen()
	if err != nil {
		return fmt.Errorf("failed to init screen: %w", err)
	}
	a.screen = screen
	a.renderer = ui.NewRenderer(screen)
	defer screen.Fini()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		a.cleanup()
		os.Exit(0)
	}()

	// Get terminal size
	w, h := screen.Size()

	// Start server or connect to server
	if a.cfg.IsServer {
		return a.runServer(w, h)
	}
	return a.runClient(w, h)
}

// runServer runs in server mode
func (a *App) runServer(w, h int) error {
	a.server = server.NewServerWithConfig(a.cfg)

	// Get player name
	name := a.cfg.PlayerName
	if name == "" {
		name = "Host"
	}

	// Start server in background
	go func() {
		if err := a.server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Display server info
	a.renderer.RenderConnecting(fmt.Sprintf("localhost:%d", a.cfg.Port))
	a.screen.Show()

	// Connect as client to own server
	a.client = client.NewClient(name, w, h)
	addr := fmt.Sprintf("localhost:%d", a.cfg.Port)
	if err := a.client.Connect(addr); err != nil {
		return fmt.Errorf("failed to connect to own server: %w", err)
	}

	return a.mainLoop()
}

// runClient runs in client mode
func (a *App) runClient(w, h int) error {
	name := a.cfg.PlayerName
	if name == "" {
		name = fmt.Sprintf("Player%d", os.Getpid()%1000)
	}

	addr := fmt.Sprintf("%s:%d", a.cfg.ServerAddr, a.cfg.Port)
	a.renderer.RenderConnecting(addr)
	a.screen.Show()

	a.client = client.NewClient(name, w, h)
	if err := a.client.Connect(addr); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	return a.mainLoop()
}

// mainLoop is the main event loop
func (a *App) mainLoop() error {
	eventCh := make(chan tcell.Event, 10)

	// Event polling goroutine
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				return
			}
			eventCh <- ev
		}
	}()

	for {
		select {
		case ev := <-eventCh:
			if a.handleEvent(ev) {
				a.cleanup()
				return nil
			}

		case state := <-a.client.LobbyState:
			a.lobbyState = state
			a.inLobby = true
			a.inGame = false

		case <-a.client.GameStart:
			a.inLobby = false
			a.inGame = true

		case state := <-a.client.GameState:
			a.gameState = state
			// Check if we're eliminated
			for _, p := range state.Players {
				if p.ID == a.client.PlayerID && !p.Alive && !a.spectating {
					a.spectating = true
					a.myScore = p.Score
					// Calculate rank
					a.myRank = 1
					for _, other := range state.Players {
						if other.Alive || other.Score > p.Score {
							a.myRank++
						}
					}
					// Pick someone to watch
					for _, other := range state.Players {
						if other.Alive {
							a.watchingID = other.ID
							break
						}
					}
				}
			}

		case state := <-a.client.GameOver:
			a.overState = state
			a.gameOver = true
			a.inGame = false

		case err := <-a.client.Error:
			a.renderer.RenderError(err.Error())
			a.screen.Show()
			// Wait for quit
			for {
				ev := <-eventCh
				if keyEv, ok := ev.(*tcell.EventKey); ok {
					if ui.IsQuitKey(keyEv.Key(), keyEv.Rune()) {
						a.cleanup()
						return nil
					}
				}
			}
		}

		a.render()
	}
}

// handleEvent processes a tcell event, returns true to quit
func (a *App) handleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ui.IsQuitKey(ev.Key(), ev.Rune()) {
			return true
		}

		if a.inLobby {
			// Host can start game
			if a.lobbyState.IsHost && ui.IsStartKey(ev.Key()) && a.server != nil {
				a.server.StartGame()
			}
		} else if a.inGame {
			if a.spectating {
				// Tab to cycle players
				if ui.IsTabKey(ev.Key()) {
					a.cycleWatching()
				}
			} else {
				// Send direction input
				dir := ui.KeyToDirection(ev.Key(), ev.Rune())
				if dir != protocol.DirNone {
					a.client.SendInput(dir)
				}
			}
		}

	case *tcell.EventResize:
		a.screen.Clear()
	}

	return false
}

// cycleWatching switches to next alive player
func (a *App) cycleWatching() {
	var aliveIDs []int
	for _, p := range a.gameState.Players {
		if p.Alive {
			aliveIDs = append(aliveIDs, p.ID)
		}
	}
	if len(aliveIDs) == 0 {
		return
	}

	// Find current index and move to next
	for i, id := range aliveIDs {
		if id == a.watchingID {
			a.watchingID = aliveIDs[(i+1)%len(aliveIDs)]
			return
		}
	}
	a.watchingID = aliveIDs[0]
}

// render draws the current state
func (a *App) render() {
	if a.gameOver {
		a.renderer.RenderGameOver(a.overState)
	} else if a.inLobby {
		a.renderer.RenderLobby(a.lobbyState)
	} else if a.inGame {
		if a.spectating {
			a.renderer.RenderSpectator(a.gameState, a.watchingID, a.myRank, a.myScore)
		} else {
			a.renderer.RenderGame(a.gameState, a.client.PlayerID)
		}
	}
	a.screen.Show()
}

// cleanup shuts down resources
func (a *App) cleanup() {
	if a.client != nil {
		a.client.Close()
	}
	if a.server != nil {
		a.server.Stop()
	}
}
```

**Step 2: Update main.go**

```go
// cmd/pixwar/main.go
package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/diegok/pixwar/internal/app"
	"github.com/diegok/pixwar/internal/config"
)

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Usage: pixwar --server [options] | pixwar --join <address>\n")
		fmt.Fprintf(os.Stderr, "\nServer options:\n")
		fmt.Fprintf(os.Stderr, "  --port <port>      Listen port (default: 7777)\n")
		fmt.Fprintf(os.Stderr, "  --time <minutes>   Game duration (default: 5)\n")
		fmt.Fprintf(os.Stderr, "  --threshold <pct>  Territory %% to end (default: 95)\n")
		fmt.Fprintf(os.Stderr, "  --powerups         Enable power-ups\n")
		fmt.Fprintf(os.Stderr, "  --name <name>      Player name\n")
		os.Exit(1)
	}

	// If server, show IP address
	if cfg.IsServer {
		showServerInfo(cfg.Port)
	}

	application := app.NewApp(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func showServerInfo(port int) {
	fmt.Printf("Starting Pixwar server...\n")
	fmt.Printf("Port: %d\n", port)

	// Get local IP
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					fmt.Printf("Connect with: pixwar --join %s\n", ipnet.IP.String())
				}
			}
		}
	}
	fmt.Println()
}
```

**Step 3: Build and test**

Run: `go build -o pixwar ./cmd/pixwar`
Expected: Binary created successfully

**Step 4: Commit**

```bash
git add internal/app/ cmd/pixwar/main.go
git commit -m "feat: implement main application loop and entry point"
```

---

## Phase 7: Polish and Integration

### Task 7.1: Add Graceful Shutdown and Error Handling

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/app/app.go`

**Step 1: Update server for graceful shutdown**

Add to `internal/server/server.go`:

```go
// NotifyShutdown sends shutdown notice to all clients
func (s *Server) NotifyShutdown() {
	s.broadcast(&protocol.Message{
		Type: protocol.MsgGameOver,
		Payload: protocol.GameOverState{
			Reason: "Server shutting down",
		},
	})
}
```

**Step 2: Update app cleanup**

In `internal/app/app.go`, update cleanup:

```go
func (a *App) cleanup() {
	if a.server != nil {
		a.server.NotifyShutdown()
	}
	if a.client != nil {
		a.client.Close()
	}
	if a.server != nil {
		a.server.Stop()
	}
}
```

**Step 3: Run tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/server/server.go internal/app/app.go
git commit -m "feat: add graceful shutdown with client notification"
```

---

### Task 7.2: Final Integration Test

**Step 1: Build the binary**

Run: `go build -o pixwar ./cmd/pixwar`
Expected: Success

**Step 2: Run the tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 3: Manual test (in two terminals)**

Terminal 1:
```bash
./pixwar --server --name Host
```

Terminal 2:
```bash
./pixwar --join localhost --name Player2
```

**Step 4: Final commit**

```bash
git add .
git commit -m "feat: complete pixwar multiplayer game implementation"
```

---

## Summary

This plan implements Pixwar in 7 phases with 14 main tasks:

1. **Foundation** (Tasks 1.1-1.2): Go module, CLI parsing
2. **Protocol** (Tasks 2.1-2.2): Message types, codec
3. **Game Core** (Tasks 3.1-3.4): Board, player, territory capture, state manager
4. **Server** (Tasks 4.1-4.2): TCP server, client handler, lobby, game loop
5. **TUI** (Tasks 5.1-5.3): Screen, renderer, input handling
6. **Application** (Task 6.1): Main loop, integration
7. **Polish** (Tasks 7.1-7.2): Graceful shutdown, final testing

Each task follows TDD: write failing test, implement, verify pass, commit.

---

**Plan complete and saved to `docs/plans/2026-01-30-pixwar-implementation.md`. Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
