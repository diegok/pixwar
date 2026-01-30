package ui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"github.com/diegok/pixwar/internal/protocol"
)

// Renderer handles drawing game states to the screen
type Renderer struct {
	screen *Screen
}

// NewRenderer creates a new renderer with the given screen
func NewRenderer(screen *Screen) *Renderer {
	return &Renderer{screen: screen}
}

// RenderLobby draws the lobby waiting room
func (r *Renderer) RenderLobby(state protocol.LobbyState) {
	r.screen.Clear()
	w, h := r.screen.Size()

	titleStyle := tcell.StyleDefault.Bold(true)
	defaultStyle := tcell.StyleDefault
	dimStyle := tcell.StyleDefault.Dim(true)

	// Title
	title := "=== PIXWAR LOBBY ==="
	r.screen.DrawText((w-len(title))/2, 1, title, titleStyle)

	// Draw box around player list
	boxW := 40
	boxH := len(state.Players) + 4
	boxX := (w - boxW) / 2
	boxY := 3
	r.screen.DrawBox(boxX, boxY, boxW, boxH, defaultStyle)

	// Header
	header := "Players:"
	r.screen.DrawText(boxX+2, boxY+1, header, titleStyle)

	// Player list
	for i, p := range state.Players {
		playerStyle := GetPlayerStyle(p.Color)
		readyStr := ""
		if p.Ready {
			readyStr = " [READY]"
		}
		line := fmt.Sprintf("%d. %s%s", i+1, p.Name, readyStr)
		r.screen.DrawText(boxX+2, boxY+2+i, line, playerStyle)
	}

	// Show server addresses for host
	if state.IsHost && len(state.ServerAddrs) > 0 {
		addrY := boxY + boxH + 1
		r.screen.DrawText(boxX, addrY, "Others can join with:", dimStyle)
		addrY++
		maxAddrs := 5 // Limit displayed addresses
		for i, addr := range state.ServerAddrs {
			if i >= maxAddrs {
				r.screen.DrawText(boxX+2, addrY, fmt.Sprintf("... and %d more", len(state.ServerAddrs)-maxAddrs), dimStyle)
				break
			}
			r.screen.DrawText(boxX+2, addrY, fmt.Sprintf("pixwar --join %s", addr), dimStyle)
			addrY++
		}
	}

	// Instructions
	var instructions string
	if state.IsHost && state.CanStart {
		instructions = "Press ENTER to start | Q to quit"
	} else if state.IsHost {
		instructions = "Waiting for players... (min 2) | Q to quit"
	} else {
		instructions = "Waiting for host to start... | Q to quit"
	}
	r.screen.DrawText((w-len(instructions))/2, h-2, instructions, defaultStyle)

	r.screen.Show()
}

// RenderGame draws the main game state
func (r *Renderer) RenderGame(state protocol.GameState, myID int) {
	r.screen.Clear()
	_, screenH := r.screen.Size()

	// Render the board
	r.renderBoard(state)

	// Render status bar at the bottom
	r.renderStatusBar(state, myID, screenH)

	r.screen.Show()
}

// renderBoard draws the game board with all players
func (r *Renderer) renderBoard(state protocol.GameState) {
	emptyStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray)

	// Draw empty board
	for y := 0; y < state.BoardHeight; y++ {
		for x := 0; x < state.BoardWidth; x++ {
			r.screen.SetCell(x, y, emptyStyle, ' ')
		}
	}

	// Draw territory for each player
	for _, p := range state.Players {
		if len(p.Territory) > 0 {
			style := GetPlayerBgStyle(p.Color)
			for _, pos := range p.Territory {
				if pos.X >= 0 && pos.X < state.BoardWidth && pos.Y >= 0 && pos.Y < state.BoardHeight {
					r.screen.SetCell(pos.X, pos.Y, style, ' ')
				}
			}
		}
	}

	// Draw trails for each player
	for _, p := range state.Players {
		if len(p.Trail) > 0 {
			style := GetPlayerStyle(p.Color)
			for _, pos := range p.Trail {
				if pos.X >= 0 && pos.X < state.BoardWidth && pos.Y >= 0 && pos.Y < state.BoardHeight {
					r.screen.SetCell(pos.X, pos.Y, style, '·')
				}
			}
		}
	}

	// Draw powerups
	powerupStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	for _, pu := range state.Powerups {
		var char rune
		switch pu.Type {
		case protocol.PowerupSpeed:
			char = '⚡'
		case protocol.PowerupShield:
			char = '🛡'
		case protocol.PowerupFreeze:
			char = '❄'
		default:
			char = '?'
		}
		if pu.Position.X >= 0 && pu.Position.X < state.BoardWidth &&
			pu.Position.Y >= 0 && pu.Position.Y < state.BoardHeight {
			r.screen.SetCell(pu.Position.X, pu.Position.Y, powerupStyle, char)
		}
	}

	// Draw player heads - use player color as background with white foreground for visibility
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		// Use colored background with white foreground so player is always visible
		style := GetPlayerBgStyle(p.Color).Foreground(tcell.ColorWhite).Bold(true)
		if p.Protected {
			style = style.Blink(true)
		}
		char := '▓' // Use a slightly different block character to stand out
		if p.Position.X >= 0 && p.Position.X < state.BoardWidth &&
			p.Position.Y >= 0 && p.Position.Y < state.BoardHeight {
			r.screen.SetCell(p.Position.X, p.Position.Y, style, char)
		}
	}
}

// renderStatusBar draws the status bar at the bottom of the screen
func (r *Renderer) renderStatusBar(state protocol.GameState, myID int, screenHeight int) {
	statusY := screenHeight - 1
	defaultStyle := tcell.StyleDefault.Reverse(true)
	screenW, _ := r.screen.Size()

	// Fill status bar background
	r.screen.FillRect(0, statusY, screenW, 1, defaultStyle, ' ')

	// Find current player
	var me *protocol.PlayerState
	for i := range state.Players {
		if state.Players[i].ID == myID {
			me = &state.Players[i]
			break
		}
	}

	// Build status line
	var statusLine string
	if me != nil {
		statusLine = fmt.Sprintf(" Score: %d | Time: %ds | Tick: %d | WASD/Arrows to move | Q to quit",
			me.Score, state.TimeLeft, state.Tick)
	} else {
		statusLine = fmt.Sprintf(" Time: %ds | Tick: %d | Spectating | Q to quit",
			state.TimeLeft, state.Tick)
	}

	r.screen.DrawText(0, statusY, statusLine, defaultStyle)
}

// RenderGameOver draws the game over screen with rankings
func (r *Renderer) RenderGameOver(state protocol.GameOverState) {
	r.screen.Clear()
	w, h := r.screen.Size()

	titleStyle := tcell.StyleDefault.Bold(true)
	defaultStyle := tcell.StyleDefault

	// Title
	title := "=== GAME OVER ==="
	r.screen.DrawText((w-len(title))/2, 1, title, titleStyle)

	// Reason
	reason := fmt.Sprintf("Ended by: %s", state.Reason)
	r.screen.DrawText((w-len(reason))/2, 3, reason, defaultStyle)

	// Rankings box
	boxW := 50
	boxH := len(state.Rankings) + 4
	boxX := (w - boxW) / 2
	boxY := 5
	r.screen.DrawBox(boxX, boxY, boxW, boxH, defaultStyle)

	// Header
	header := "Final Rankings:"
	r.screen.DrawText(boxX+2, boxY+1, header, titleStyle)

	// Sort rankings by rank
	rankings := make([]protocol.PlayerRanking, len(state.Rankings))
	copy(rankings, state.Rankings)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Rank < rankings[j].Rank
	})

	// Display rankings
	for i, rank := range rankings {
		var medal string
		switch rank.Rank {
		case 1:
			medal = "[1st]"
		case 2:
			medal = "[2nd]"
		case 3:
			medal = "[3rd]"
		default:
			medal = fmt.Sprintf("[%dth]", rank.Rank)
		}
		line := fmt.Sprintf("%s %s - Score: %d, Survived: %ds",
			medal, rank.Name, rank.Score, rank.SurvivalTime)
		r.screen.DrawText(boxX+2, boxY+2+i, line, defaultStyle)
	}

	// Instructions
	instructions := "Press Q to quit | ENTER to play again"
	r.screen.DrawText((w-len(instructions))/2, h-2, instructions, defaultStyle)

	r.screen.Show()
}

// RenderSpectator draws the game in spectator mode
func (r *Renderer) RenderSpectator(state protocol.GameState, watchingID, myRank, myScore int, deathReason string) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Render the board
	r.renderBoard(state)

	// Show death message at top of screen
	if deathReason != "" {
		deathStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
		deathMsg := fmt.Sprintf(" YOU DIED: %s ", deathReason)
		r.screen.DrawText((screenW-len(deathMsg))/2, 0, deathMsg, deathStyle)
	}

	// Render spectator status bar
	r.renderSpectatorStatusBar(state, watchingID, myRank, myScore, screenH)

	r.screen.Show()
}

// renderSpectatorStatusBar draws the spectator status bar
func (r *Renderer) renderSpectatorStatusBar(state protocol.GameState, watchingID, myRank, myScore, screenHeight int) {
	statusY := screenHeight - 1
	defaultStyle := tcell.StyleDefault.Reverse(true)
	screenW, _ := r.screen.Size()

	// Fill status bar background
	r.screen.FillRect(0, statusY, screenW, 1, defaultStyle, ' ')

	// Find watched player
	var watchedName string
	for _, p := range state.Players {
		if p.ID == watchingID {
			watchedName = p.Name
			break
		}
	}

	// Build status line
	statusLine := fmt.Sprintf(" SPECTATING | Your Score: %d | Rank: #%d | Watching: %s | TAB: next | Q: quit",
		myScore, myRank, watchedName)

	r.screen.DrawText(0, statusY, statusLine, defaultStyle)
}

// RenderConnecting draws the connecting screen
func (r *Renderer) RenderConnecting(addr string) {
	r.screen.Clear()
	w, h := r.screen.Size()

	defaultStyle := tcell.StyleDefault

	msg := fmt.Sprintf("Connecting to %s...", addr)
	r.screen.DrawText((w-len(msg))/2, h/2, msg, defaultStyle)

	r.screen.Show()
}

// RenderError draws an error message
func (r *Renderer) RenderError(err string) {
	r.screen.Clear()
	w, h := r.screen.Size()

	errorStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	defaultStyle := tcell.StyleDefault

	title := "ERROR"
	r.screen.DrawText((w-len(title))/2, h/2-1, title, errorStyle)
	r.screen.DrawText((w-len(err))/2, h/2+1, err, defaultStyle)

	instructions := "Press any key to exit"
	r.screen.DrawText((w-len(instructions))/2, h-2, instructions, defaultStyle)

	r.screen.Show()
}
