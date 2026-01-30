package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/diegok/pixwar/internal/protocol"
)

func createTestScreen(t *testing.T) (*Screen, tcell.SimulationScreen) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	sim.SetSize(80, 24)
	return NewScreen(sim), sim
}

func TestNewRenderer(t *testing.T) {
	screen, sim := createTestScreen(t)
	defer sim.Fini()

	renderer := NewRenderer(screen)
	if renderer == nil {
		t.Fatal("NewRenderer returned nil")
	}
	if renderer.screen != screen {
		t.Error("NewRenderer did not set screen correctly")
	}
}

func TestRenderLobby(t *testing.T) {
	screen, sim := createTestScreen(t)
	defer sim.Fini()

	renderer := NewRenderer(screen)
	state := protocol.LobbyState{
		Players: []protocol.LobbyPlayer{
			{ID: 1, Name: "Alice", Color: 0, Ready: true},
			{ID: 2, Name: "Bob", Color: 1, Ready: false},
		},
		IsHost:   true,
		CanStart: true,
	}

	renderer.RenderLobby(state)

	// Verify title is drawn (check for "PIXWAR LOBBY" somewhere on screen)
	found := false
	for y := 0; y < 5; y++ {
		text := ""
		for x := 0; x < 80; x++ {
			r, _, _, _ := sim.GetContent(x, y)
			text += string(r)
		}
		if len(text) > 0 && containsSubstring(text, "PIXWAR LOBBY") {
			found = true
			break
		}
	}
	if !found {
		t.Error("RenderLobby did not render title")
	}
}

func TestRenderConnecting(t *testing.T) {
	screen, sim := createTestScreen(t)
	defer sim.Fini()

	renderer := NewRenderer(screen)
	renderer.RenderConnecting("localhost:8080")

	// Check that connecting message appears
	found := false
	for y := 0; y < 24; y++ {
		text := ""
		for x := 0; x < 80; x++ {
			r, _, _, _ := sim.GetContent(x, y)
			text += string(r)
		}
		if containsSubstring(text, "Connecting") && containsSubstring(text, "localhost:8080") {
			found = true
			break
		}
	}
	if !found {
		t.Error("RenderConnecting did not render connection message")
	}
}

func TestRenderError(t *testing.T) {
	screen, sim := createTestScreen(t)
	defer sim.Fini()

	renderer := NewRenderer(screen)
	renderer.RenderError("Connection failed")

	// Check that error message appears
	foundError := false
	foundMsg := false
	for y := 0; y < 24; y++ {
		text := ""
		for x := 0; x < 80; x++ {
			r, _, _, _ := sim.GetContent(x, y)
			text += string(r)
		}
		if containsSubstring(text, "ERROR") {
			foundError = true
		}
		if containsSubstring(text, "Connection failed") {
			foundMsg = true
		}
	}
	if !foundError {
		t.Error("RenderError did not render ERROR title")
	}
	if !foundMsg {
		t.Error("RenderError did not render error message")
	}
}

func TestRenderGame(t *testing.T) {
	screen, sim := createTestScreen(t)
	defer sim.Fini()

	renderer := NewRenderer(screen)
	state := protocol.GameState{
		Tick:        100,
		BoardWidth:  40,
		BoardHeight: 20,
		TimeLeft:    120,
		Players: []protocol.PlayerState{
			{
				ID:        1,
				Name:      "Alice",
				Color:     0,
				Position:  protocol.Position{X: 10, Y: 5},
				Direction: protocol.DirRight,
				Trail:     []protocol.Position{{X: 9, Y: 5}, {X: 8, Y: 5}},
				Territory: []protocol.Position{{X: 0, Y: 0}, {X: 1, Y: 0}},
				Score:     50,
				Alive:     true,
			},
		},
	}

	renderer.RenderGame(state, 1)

	// Check that player head is drawn at position (10, 5)
	r, _, _, _ := sim.GetContent(10, 5)
	if r != '█' {
		t.Errorf("Player head at (10,5) = %c, want █", r)
	}

	// Check trail dots
	for _, pos := range state.Players[0].Trail {
		r, _, _, _ := sim.GetContent(pos.X, pos.Y)
		if r != '·' {
			t.Errorf("Trail at (%d,%d) = %c, want ·", pos.X, pos.Y, r)
		}
	}
}

func TestRenderGameOver(t *testing.T) {
	screen, sim := createTestScreen(t)
	defer sim.Fini()

	renderer := NewRenderer(screen)
	state := protocol.GameOverState{
		Rankings: []protocol.PlayerRanking{
			{Rank: 1, PlayerID: 1, Name: "Alice", Score: 100, SurvivalTime: 60},
			{Rank: 2, PlayerID: 2, Name: "Bob", Score: 50, SurvivalTime: 30},
		},
		Reason: "laststanding",
	}

	renderer.RenderGameOver(state)

	// Check that GAME OVER appears
	foundGameOver := false
	foundAlice := false
	for y := 0; y < 24; y++ {
		text := ""
		for x := 0; x < 80; x++ {
			r, _, _, _ := sim.GetContent(x, y)
			text += string(r)
		}
		if containsSubstring(text, "GAME OVER") {
			foundGameOver = true
		}
		if containsSubstring(text, "Alice") {
			foundAlice = true
		}
	}
	if !foundGameOver {
		t.Error("RenderGameOver did not render GAME OVER title")
	}
	if !foundAlice {
		t.Error("RenderGameOver did not render player rankings")
	}
}

func TestRenderSpectator(t *testing.T) {
	screen, sim := createTestScreen(t)
	defer sim.Fini()

	renderer := NewRenderer(screen)
	state := protocol.GameState{
		Tick:        100,
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
			},
		},
	}

	renderer.RenderSpectator(state, 1, 3, 25)

	// Check spectator status bar contains ELIMINATED
	text := ""
	for x := 0; x < 80; x++ {
		r, _, _, _ := sim.GetContent(x, 23) // Bottom line
		text += string(r)
	}
	if !containsSubstring(text, "ELIMINATED") {
		t.Error("RenderSpectator did not show ELIMINATED in status bar")
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
