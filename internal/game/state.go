package game

import "github.com/diegok/pixwar/internal/protocol"

const (
	MaxPlayers       = 8
	ProtectionTicks  = 40 // 2 seconds at 20 ticks/sec
	MoveTickInterval = 4  // Move every 4 ticks = 5 moves/sec at 20 ticks/sec
)

// Powerup represents an active powerup on the board
type Powerup struct {
	Type protocol.PowerupType
	X, Y int
	TTL  int
}

// GameState represents the complete state of a game
type GameState struct {
	Board        *Board
	Players      []*Player
	Tick_        int
	TimeLeft     int // seconds
	TicksPerSec  int
	tickCounter  int
	PowerupsOn   bool
	Powerups     []*Powerup
	nextPlayerID int
	GameOver     bool
	WinnerID     int
	EndReason    string
}

// Player colors (0-based indices matching UI PlayerColors array)
var playerColors = []int{0, 1, 2, 3, 4, 5, 6, 7} // red, blue, green, yellow, purple, orange, teal, fuchsia

// NewGameState creates a new game state with the specified board size and duration
func NewGameState(width, height, durationSec int) *GameState {
	return &GameState{
		Board:        NewBoard(width, height),
		Players:      make([]*Player, 0),
		Tick_:        0,
		TimeLeft:     durationSec,
		TicksPerSec:  20,
		tickCounter:  0,
		PowerupsOn:   false,
		Powerups:     make([]*Powerup, 0),
		nextPlayerID: 0,
		GameOver:     false,
		WinnerID:     -1,
		EndReason:    "",
	}
}

// AddPlayer adds a new player to the game
// Returns nil if max players has been reached
func (gs *GameState) AddPlayer(name string) *Player {
	if len(gs.Players) >= MaxPlayers {
		return nil
	}

	colorIndex := gs.nextPlayerID % len(playerColors)
	player := NewPlayer(gs.nextPlayerID, name, playerColors[colorIndex])
	gs.Players = append(gs.Players, player)
	gs.nextPlayerID++

	return player
}

// GetPlayer returns the player with the specified ID, or nil if not found
func (gs *GameState) GetPlayer(id int) *Player {
	for _, p := range gs.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// RemovePlayer removes a player from the game and clears their territory
func (gs *GameState) RemovePlayer(id int) {
	// Clear player's territory from the board
	gs.Board.ClearPlayerTerritory(id)

	// Remove player from the slice
	for i, p := range gs.Players {
		if p.ID == id {
			gs.Players = append(gs.Players[:i], gs.Players[i+1:]...)
			break
		}
	}
}

// SpawnPlayers positions all players on the edges of the board evenly distributed
func (gs *GameState) SpawnPlayers() {
	if len(gs.Players) == 0 {
		return
	}

	// Calculate spawn positions on edges
	// Distribute players evenly around the perimeter
	perimeter := 2*(gs.Board.Width+gs.Board.Height) - 4
	spacing := perimeter / len(gs.Players)

	for i, p := range gs.Players {
		pos := (i * spacing) % perimeter
		x, y := gs.perimeterToCoords(pos)
		p.SetPosition(x, y)
		p.GrantProtection(ProtectionTicks)
	}
}

// perimeterToCoords converts a perimeter position to x, y coordinates
// Perimeter goes clockwise: top edge (0 to width-1), right edge, bottom edge, left edge
func (gs *GameState) perimeterToCoords(pos int) (int, int) {
	width := gs.Board.Width
	height := gs.Board.Height

	// Top edge
	if pos < width {
		return pos, 0
	}
	pos -= width

	// Right edge (excluding top corner)
	if pos < height-1 {
		return width - 1, pos + 1
	}
	pos -= (height - 1)

	// Bottom edge (excluding right corner)
	if pos < width-1 {
		return width - 2 - pos, height - 1
	}
	pos -= (width - 1)

	// Left edge (excluding bottom corner)
	return 0, height - 2 - pos
}

// ProcessInput handles a direction input from a player
func (gs *GameState) ProcessInput(playerID int, dir protocol.Direction) {
	p := gs.GetPlayer(playerID)
	if p == nil || !p.Alive {
		return
	}
	p.SetDirection(dir)
}

// Tick advances the game state by one tick
func (gs *GameState) Tick() {
	if gs.GameOver {
		return
	}

	gs.Tick_++
	gs.tickCounter++

	// Decrement time every TicksPerSec ticks
	if gs.tickCounter >= gs.TicksPerSec {
		gs.tickCounter = 0
		if gs.TimeLeft > 0 {
			gs.TimeLeft--
		}
	}

	// Process each player
	for _, p := range gs.Players {
		if !p.Alive {
			continue
		}

		// Decrement protection
		if p.Protected {
			p.DecrementProtection()
		}

		// Only move players every MoveTickInterval ticks (to slow down the game)
		if gs.Tick_%MoveTickInterval != 0 {
			continue
		}

		// Move player
		if p.Direction != protocol.DirNone {
			// Store previous position
			prevX, prevY := p.X, p.Y

			// Move
			p.Move()

			// Check if player is outside the board - undo move if so
			if !gs.Board.IsInBounds(p.X, p.Y) {
				p.X, p.Y = prevX, prevY
				continue
			}

			// Handle territory/trail logic
			gs.handlePlayerMovement(p, prevX, prevY)
		}
	}

	// Check win conditions
	gs.checkWinConditions()
}

// handlePlayerMovement handles trail creation and territory capture after a move
func (gs *GameState) handlePlayerMovement(p *Player, prevX, prevY int) {
	cell := gs.Board.GetCell(p.X, p.Y)
	if cell == nil {
		return
	}

	// Check if player hit their own trail
	if p.IsOnOwnTrail(p.X, p.Y) && !p.Protected {
		p.Eliminate()
		gs.Board.ClearPlayerTrails(p.ID)
		return
	}

	// Check if player hit another player's trail (and they're not protected)
	if cell.IsTrail && cell.Owner != p.ID && !p.Protected {
		p.Eliminate()
		gs.Board.ClearPlayerTrails(p.ID)
		return
	}

	// Check if player is on their own territory or on the board edge
	isOnOwnTerritory := cell.Owner == p.ID && !cell.IsTrail
	isOnEdge := gs.Board.IsOnEdge(p.X, p.Y)
	wasOnEdge := gs.Board.IsOnEdge(prevX, prevY)

	// Safe zone = own territory or board edge
	isInSafeZone := isOnOwnTerritory || isOnEdge

	// Capture territory when returning to safe zone with a trail
	if isInSafeZone && len(p.Trail) > 0 {
		captured := gs.Board.CaptureTerritory(p.ID, p.Trail)
		p.Score += captured
		p.ClearTrail()
	} else if !isInSafeZone {
		// Moving in unsafe territory - create trail
		// Only add trail point if we weren't on edge (avoid edge points in trail)
		if !wasOnEdge {
			p.AddTrailPoint(prevX, prevY)
			gs.Board.SetTrail(prevX, prevY, p.ID)
		} else {
			// Starting from edge - just add current trail without the edge point
			// The first point off the edge starts the trail
		}
	}
	// If in safe zone without trail, nothing to do (just moving safely)
}

// checkWinConditions checks if the game should end
func (gs *GameState) checkWinConditions() {
	// Time's up
	if gs.TimeLeft <= 0 {
		gs.endGame("time")
		return
	}

	// Last player standing (if more than 1 player)
	if len(gs.Players) > 1 && gs.AliveCount() <= 1 {
		gs.endGame("laststanding")
		return
	}
}

// endGame sets the game over state
func (gs *GameState) endGame(reason string) {
	gs.GameOver = true
	gs.EndReason = reason

	// Find winner (highest score among alive players, or highest overall if none alive)
	var winner *Player
	highestScore := -1

	for _, p := range gs.Players {
		if p.Alive && p.Score > highestScore {
			highestScore = p.Score
			winner = p
		}
	}

	// If no alive players, find highest score overall
	if winner == nil {
		for _, p := range gs.Players {
			if p.Score > highestScore {
				highestScore = p.Score
				winner = p
			}
		}
	}

	if winner != nil {
		gs.WinnerID = winner.ID
	}
}

// AliveCount returns the number of alive players
func (gs *GameState) AliveCount() int {
	count := 0
	for _, p := range gs.Players {
		if p.Alive {
			count++
		}
	}
	return count
}

// ToProtocolState converts the game state to a protocol.GameState for network transmission
func (gs *GameState) ToProtocolState() protocol.GameState {
	players := make([]protocol.PlayerState, len(gs.Players))

	for i, p := range gs.Players {
		// Convert trail to protocol positions
		trail := make([]protocol.Position, len(p.Trail))
		for j, pt := range p.Trail {
			trail[j] = protocol.Position{X: pt.X, Y: pt.Y}
		}

		// Get territory from board
		territoryPoints := gs.Board.GetPlayerTerritory(p.ID)
		territory := make([]protocol.Position, len(territoryPoints))
		for j, pt := range territoryPoints {
			territory[j] = protocol.Position{X: pt.X, Y: pt.Y}
		}

		players[i] = protocol.PlayerState{
			ID:        p.ID,
			Name:      p.Name,
			Color:     p.Color,
			Position:  protocol.Position{X: p.X, Y: p.Y},
			Direction: p.Direction,
			Trail:     trail,
			Territory: territory,
			Score:     p.Score,
			Alive:     p.Alive,
			Protected: p.Protected,
		}
	}

	// Convert powerups
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
