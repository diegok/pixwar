package game

import (
	"testing"

	"github.com/diegok/pixwar/internal/protocol"
)

func TestNewGameState(t *testing.T) {
	t.Run("creates correct board size", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		if gs.Board == nil {
			t.Fatal("expected Board to be created")
		}
		if gs.Board.Width != 100 {
			t.Errorf("expected Board.Width 100, got %d", gs.Board.Width)
		}
		if gs.Board.Height != 50 {
			t.Errorf("expected Board.Height 50, got %d", gs.Board.Height)
		}
	})

	t.Run("initializes time and tick fields", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		if gs.TimeLeft != 120 {
			t.Errorf("expected TimeLeft 120, got %d", gs.TimeLeft)
		}
		if gs.Tick_ != 0 {
			t.Errorf("expected Tick_ 0, got %d", gs.Tick_)
		}
		if gs.TicksPerSec != 20 {
			t.Errorf("expected TicksPerSec 20, got %d", gs.TicksPerSec)
		}
	})

	t.Run("initializes empty players slice", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		if gs.Players == nil {
			t.Fatal("expected Players slice to be initialized")
		}
		if len(gs.Players) != 0 {
			t.Errorf("expected 0 players, got %d", len(gs.Players))
		}
	})

	t.Run("initializes game state flags", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		if gs.GameOver {
			t.Error("expected GameOver false")
		}
		if gs.WinnerID != -1 {
			t.Errorf("expected WinnerID -1, got %d", gs.WinnerID)
		}
	})
}

func TestAddPlayer(t *testing.T) {
	t.Run("adds player with incremented ID", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p1 := gs.AddPlayer("Alice")
		p2 := gs.AddPlayer("Bob")

		if p1 == nil {
			t.Fatal("expected player 1 to be created")
		}
		if p2 == nil {
			t.Fatal("expected player 2 to be created")
		}
		if p1.ID != 0 {
			t.Errorf("expected first player ID 0, got %d", p1.ID)
		}
		if p2.ID != 1 {
			t.Errorf("expected second player ID 1, got %d", p2.ID)
		}
	})

	t.Run("returns nil after MaxPlayers", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		// Add max players
		for i := 0; i < MaxPlayers; i++ {
			p := gs.AddPlayer("Player")
			if p == nil {
				t.Fatalf("expected player %d to be added", i)
			}
		}

		// Try to add one more
		p := gs.AddPlayer("Extra")
		if p != nil {
			t.Error("expected nil when max players reached")
		}
	})

	t.Run("assigns unique colors", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p1 := gs.AddPlayer("Alice")
		p2 := gs.AddPlayer("Bob")

		if p1.Color == p2.Color {
			t.Error("expected players to have different colors")
		}
	})
}

func TestGetPlayer(t *testing.T) {
	t.Run("finds correct player by ID", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p1 := gs.AddPlayer("Alice")
		gs.AddPlayer("Bob")
		p3 := gs.AddPlayer("Charlie")

		found := gs.GetPlayer(p1.ID)
		if found != p1 {
			t.Error("expected to find player 1")
		}

		found = gs.GetPlayer(p3.ID)
		if found != p3 {
			t.Error("expected to find player 3")
		}
	})

	t.Run("returns nil for non-existent ID", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		gs.AddPlayer("Alice")

		found := gs.GetPlayer(999)
		if found != nil {
			t.Error("expected nil for non-existent player")
		}
	})
}

func TestRemovePlayer(t *testing.T) {
	t.Run("removes player from list", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p1 := gs.AddPlayer("Alice")
		gs.AddPlayer("Bob")

		gs.RemovePlayer(p1.ID)

		if len(gs.Players) != 1 {
			t.Errorf("expected 1 player remaining, got %d", len(gs.Players))
		}
		if gs.GetPlayer(p1.ID) != nil {
			t.Error("expected player 1 to be removed")
		}
	})

	t.Run("clears player territory from board", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		gs.Board.SetTerritory(5, 5, p.ID)
		gs.Board.SetTrail(6, 5, p.ID)

		gs.RemovePlayer(p.ID)

		cell1 := gs.Board.GetCell(5, 5)
		cell2 := gs.Board.GetCell(6, 5)
		if cell1.Owner == p.ID {
			t.Error("expected territory to be cleared")
		}
		if cell2.Owner == p.ID {
			t.Error("expected trail to be cleared")
		}
	})
}

func TestSpawnPlayers(t *testing.T) {
	t.Run("positions players on edges", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		gs.AddPlayer("Alice")
		gs.AddPlayer("Bob")
		gs.AddPlayer("Charlie")
		gs.AddPlayer("Dave")

		gs.SpawnPlayers()

		for _, p := range gs.Players {
			if !gs.Board.IsOnEdge(p.X, p.Y) {
				t.Errorf("player %s at (%d, %d) is not on edge", p.Name, p.X, p.Y)
			}
		}
	})

	t.Run("grants protection to spawned players", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		gs.AddPlayer("Alice")
		gs.SpawnPlayers()

		p := gs.Players[0]
		if !p.Protected {
			t.Error("expected player to be protected after spawn")
		}
		if p.ProtectionTicks != ProtectionTicks {
			t.Errorf("expected %d protection ticks, got %d", ProtectionTicks, p.ProtectionTicks)
		}
	})

	t.Run("distributes players evenly around edges", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		for i := 0; i < 4; i++ {
			gs.AddPlayer("Player")
		}
		gs.SpawnPlayers()

		// Check that players are spaced apart
		for i := 0; i < len(gs.Players); i++ {
			for j := i + 1; j < len(gs.Players); j++ {
				p1 := gs.Players[i]
				p2 := gs.Players[j]
				// Players should not be at the same position
				if p1.X == p2.X && p1.Y == p2.Y {
					t.Errorf("players %d and %d spawned at same position (%d, %d)", i, j, p1.X, p1.Y)
				}
			}
		}
	})
}

func TestProcessInput(t *testing.T) {
	t.Run("changes player direction", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		p.SetPosition(50, 25)

		gs.ProcessInput(p.ID, protocol.DirUp)

		if p.Direction != protocol.DirUp {
			t.Errorf("expected direction Up, got %v", p.Direction)
		}
	})

	t.Run("ignores invalid player ID", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		// Should not panic
		gs.ProcessInput(999, protocol.DirUp)
	})

	t.Run("does not change direction for dead player", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		p.Direction = protocol.DirRight
		p.Eliminate()

		gs.ProcessInput(p.ID, protocol.DirUp)

		if p.Direction != protocol.DirRight {
			t.Error("expected direction to remain unchanged for dead player")
		}
	})
}

func TestTick(t *testing.T) {
	t.Run("moves players in their direction", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		p.SetPosition(50, 25)
		p.Direction = protocol.DirRight
		p.Protected = false

		gs.Tick()

		if p.X != 51 {
			t.Errorf("expected X 51, got %d", p.X)
		}
	})

	t.Run("increments tick counter", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		gs.Tick()
		gs.Tick()

		if gs.Tick_ != 2 {
			t.Errorf("expected Tick_ 2, got %d", gs.Tick_)
		}
	})

	t.Run("decrements time after TicksPerSec ticks", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		for i := 0; i < gs.TicksPerSec; i++ {
			gs.Tick()
		}

		if gs.TimeLeft != 119 {
			t.Errorf("expected TimeLeft 119, got %d", gs.TimeLeft)
		}
	})

	t.Run("decrements protection ticks", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		p.GrantProtection(10)
		p.Direction = protocol.DirRight
		p.SetPosition(50, 25)

		gs.Tick()

		if p.ProtectionTicks != 9 {
			t.Errorf("expected 9 protection ticks, got %d", p.ProtectionTicks)
		}
	})

	t.Run("does not move dead players", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		p.SetPosition(50, 25)
		p.Direction = protocol.DirRight
		p.Eliminate()

		gs.Tick()

		if p.X != 50 {
			t.Error("expected dead player not to move")
		}
	})
}

func TestAliveCount(t *testing.T) {
	t.Run("counts alive players", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		gs.AddPlayer("Alice")
		gs.AddPlayer("Bob")
		gs.AddPlayer("Charlie")

		if gs.AliveCount() != 3 {
			t.Errorf("expected 3 alive, got %d", gs.AliveCount())
		}

		gs.Players[1].Eliminate()

		if gs.AliveCount() != 2 {
			t.Errorf("expected 2 alive, got %d", gs.AliveCount())
		}
	})

	t.Run("returns 0 for no players", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		if gs.AliveCount() != 0 {
			t.Errorf("expected 0 alive, got %d", gs.AliveCount())
		}
	})
}

func TestToProtocolState(t *testing.T) {
	t.Run("converts to protocol GameState", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		p.SetPosition(25, 10)
		p.Score = 500

		gs.Tick_ = 42
		gs.TimeLeft = 90

		state := gs.ToProtocolState()

		if state.Tick != 42 {
			t.Errorf("expected Tick 42, got %d", state.Tick)
		}
		if state.BoardWidth != 100 {
			t.Errorf("expected BoardWidth 100, got %d", state.BoardWidth)
		}
		if state.BoardHeight != 50 {
			t.Errorf("expected BoardHeight 50, got %d", state.BoardHeight)
		}
		if state.TimeLeft != 90 {
			t.Errorf("expected TimeLeft 90, got %d", state.TimeLeft)
		}
		if len(state.Players) != 1 {
			t.Fatalf("expected 1 player, got %d", len(state.Players))
		}

		ps := state.Players[0]
		if ps.Name != "Alice" {
			t.Errorf("expected Name 'Alice', got '%s'", ps.Name)
		}
		if ps.Position.X != 25 || ps.Position.Y != 10 {
			t.Errorf("expected Position (25, 10), got (%d, %d)", ps.Position.X, ps.Position.Y)
		}
		if ps.Score != 500 {
			t.Errorf("expected Score 500, got %d", ps.Score)
		}
	})

	t.Run("includes player territory", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		gs.Board.SetTerritory(5, 5, p.ID)
		gs.Board.SetTerritory(6, 5, p.ID)

		state := gs.ToProtocolState()

		if len(state.Players[0].Territory) != 2 {
			t.Errorf("expected 2 territory cells, got %d", len(state.Players[0].Territory))
		}
	})

	t.Run("includes player trail", func(t *testing.T) {
		gs := NewGameState(100, 50, 120)

		p := gs.AddPlayer("Alice")
		p.AddTrailPoint(10, 10)
		p.AddTrailPoint(11, 10)

		state := gs.ToProtocolState()

		if len(state.Players[0].Trail) != 2 {
			t.Errorf("expected 2 trail points, got %d", len(state.Players[0].Trail))
		}
	})
}
