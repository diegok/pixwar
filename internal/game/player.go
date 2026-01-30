package game

import "github.com/diegok/pixwar/internal/protocol"

// Point represents a 2D coordinate
type Point struct {
	X, Y int
}

// Player represents a player in the game
type Player struct {
	ID              int
	Name            string
	Color           int
	X               int
	Y               int
	Direction       protocol.Direction
	Trail           []Point
	Score           int
	Alive           bool
	Protected       bool
	ProtectionTicks int
	SurvivalTicks   int    // How many ticks the player survived
	DeathReason     string // Why the player died (empty if alive)
}

// NewPlayer creates a new player with the given ID, name, and color
func NewPlayer(id int, name string, color int) *Player {
	return &Player{
		ID:              id,
		Name:            name,
		Color:           color,
		Direction:       protocol.DirNone,
		Trail:           make([]Point, 0),
		Alive:           true,
		Protected:       false,
		ProtectionTicks: 0,
	}
}

// SetPosition updates the player's position
func (p *Player) SetPosition(x, y int) {
	p.X = x
	p.Y = y
}

// Move moves the player one step in their current direction
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
	case protocol.DirNone:
		// No movement
	}
}

// AddTrailPoint adds a point to the player's trail
func (p *Player) AddTrailPoint(x, y int) {
	p.Trail = append(p.Trail, Point{X: x, Y: y})
}

// ClearTrail removes all points from the player's trail
func (p *Player) ClearTrail() {
	p.Trail = make([]Point, 0)
}

// IsOnOwnTrail checks if the given coordinates match any point in the player's trail
func (p *Player) IsOnOwnTrail(x, y int) bool {
	for _, pt := range p.Trail {
		if pt.X == x && pt.Y == y {
			return true
		}
	}
	return false
}

// Eliminate marks the player as eliminated with a reason
func (p *Player) Eliminate(reason string) {
	p.Alive = false
	p.DeathReason = reason
}

// SetDirection sets the player's direction, preventing 180-degree reversals
func (p *Player) SetDirection(dir protocol.Direction) {
	// Allow setting to None or from None
	if dir == protocol.DirNone || p.Direction == protocol.DirNone {
		p.Direction = dir
		return
	}

	// Check for 180-degree reversal
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

// GrantProtection gives the player spawn protection for the specified number of ticks
func (p *Player) GrantProtection(ticks int) {
	p.Protected = true
	p.ProtectionTicks = ticks
}

// DecrementProtection decreases protection ticks by one, clearing protection when it reaches zero
func (p *Player) DecrementProtection() {
	if p.ProtectionTicks > 0 {
		p.ProtectionTicks--
		if p.ProtectionTicks == 0 {
			p.Protected = false
		}
	}
}
