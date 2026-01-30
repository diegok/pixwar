package game

import (
	"testing"

	"github.com/diegok/pixwar/internal/protocol"
)

func TestNewPlayer(t *testing.T) {
	t.Run("creates alive player", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if !p.Alive {
			t.Error("expected Alive true, got false")
		}
	})

	t.Run("sets ID, Name, and Color", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if p.ID != 1 {
			t.Errorf("expected ID 1, got %d", p.ID)
		}
		if p.Name != "Alice" {
			t.Errorf("expected Name 'Alice', got '%s'", p.Name)
		}
		if p.Color != 2 {
			t.Errorf("expected Color 2, got %d", p.Color)
		}
	})

	t.Run("creates player with empty trail", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if len(p.Trail) != 0 {
			t.Errorf("expected empty trail, got %d elements", len(p.Trail))
		}
	})

	t.Run("initial direction is None", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if p.Direction != protocol.DirNone {
			t.Errorf("expected Direction DirNone, got %d", p.Direction)
		}
	})

	t.Run("initial position is zero", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if p.X != 0 || p.Y != 0 {
			t.Errorf("expected position (0,0), got (%d,%d)", p.X, p.Y)
		}
	})

	t.Run("initial score is zero", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if p.Score != 0 {
			t.Errorf("expected Score 0, got %d", p.Score)
		}
	})

	t.Run("initial protection is false", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if p.Protected {
			t.Error("expected Protected false, got true")
		}
		if p.ProtectionTicks != 0 {
			t.Errorf("expected ProtectionTicks 0, got %d", p.ProtectionTicks)
		}
	})
}

func TestSetPosition(t *testing.T) {
	t.Run("updates X and Y", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.SetPosition(10, 20)

		if p.X != 10 {
			t.Errorf("expected X 10, got %d", p.X)
		}
		if p.Y != 20 {
			t.Errorf("expected Y 20, got %d", p.Y)
		}
	})

	t.Run("can set negative positions", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.SetPosition(-5, -10)

		if p.X != -5 {
			t.Errorf("expected X -5, got %d", p.X)
		}
		if p.Y != -10 {
			t.Errorf("expected Y -10, got %d", p.Y)
		}
	})
}

func TestMove(t *testing.T) {
	testCases := []struct {
		name         string
		direction    protocol.Direction
		startX       int
		startY       int
		expectedX    int
		expectedY    int
	}{
		{"up decreases Y", protocol.DirUp, 5, 5, 5, 4},
		{"down increases Y", protocol.DirDown, 5, 5, 5, 6},
		{"left decreases X", protocol.DirLeft, 5, 5, 4, 5},
		{"right increases X", protocol.DirRight, 5, 5, 6, 5},
		{"none does not move", protocol.DirNone, 5, 5, 5, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlayer(1, "Alice", 2)
			p.SetPosition(tc.startX, tc.startY)
			p.Direction = tc.direction

			p.Move()

			if p.X != tc.expectedX {
				t.Errorf("expected X %d, got %d", tc.expectedX, p.X)
			}
			if p.Y != tc.expectedY {
				t.Errorf("expected Y %d, got %d", tc.expectedY, p.Y)
			}
		})
	}
}

func TestAddTrailPoint(t *testing.T) {
	t.Run("adds point to trail", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.AddTrailPoint(5, 10)

		if len(p.Trail) != 1 {
			t.Fatalf("expected 1 trail point, got %d", len(p.Trail))
		}
		if p.Trail[0].X != 5 || p.Trail[0].Y != 10 {
			t.Errorf("expected point (5,10), got (%d,%d)", p.Trail[0].X, p.Trail[0].Y)
		}
	})

	t.Run("adds multiple points in order", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.AddTrailPoint(1, 1)
		p.AddTrailPoint(2, 2)
		p.AddTrailPoint(3, 3)

		if len(p.Trail) != 3 {
			t.Fatalf("expected 3 trail points, got %d", len(p.Trail))
		}
		if p.Trail[0].X != 1 || p.Trail[0].Y != 1 {
			t.Errorf("expected first point (1,1), got (%d,%d)", p.Trail[0].X, p.Trail[0].Y)
		}
		if p.Trail[2].X != 3 || p.Trail[2].Y != 3 {
			t.Errorf("expected last point (3,3), got (%d,%d)", p.Trail[2].X, p.Trail[2].Y)
		}
	})
}

func TestPlayerClearTrail(t *testing.T) {
	t.Run("clears all trail points", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.AddTrailPoint(1, 1)
		p.AddTrailPoint(2, 2)
		p.AddTrailPoint(3, 3)

		p.ClearTrail()

		if len(p.Trail) != 0 {
			t.Errorf("expected empty trail after clear, got %d points", len(p.Trail))
		}
	})

	t.Run("clearing empty trail is safe", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.ClearTrail()

		if len(p.Trail) != 0 {
			t.Errorf("expected empty trail, got %d points", len(p.Trail))
		}
	})
}

func TestIsOnOwnTrail(t *testing.T) {
	t.Run("returns true when on trail", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.AddTrailPoint(5, 10)
		p.AddTrailPoint(6, 10)
		p.AddTrailPoint(7, 10)

		if !p.IsOnOwnTrail(6, 10) {
			t.Error("expected true for point on trail")
		}
	})

	t.Run("returns false when not on trail", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.AddTrailPoint(5, 10)
		p.AddTrailPoint(6, 10)
		p.AddTrailPoint(7, 10)

		if p.IsOnOwnTrail(8, 10) {
			t.Error("expected false for point not on trail")
		}
	})

	t.Run("returns false for empty trail", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		if p.IsOnOwnTrail(5, 5) {
			t.Error("expected false for empty trail")
		}
	})
}

func TestEliminate(t *testing.T) {
	t.Run("sets Alive to false", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.Eliminate()

		if p.Alive {
			t.Error("expected Alive false after elimination, got true")
		}
	})

	t.Run("multiple eliminations are safe", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.Eliminate()
		p.Eliminate()

		if p.Alive {
			t.Error("expected Alive false after multiple eliminations")
		}
	})
}

func TestSetDirection(t *testing.T) {
	t.Run("sets new direction", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.SetDirection(protocol.DirUp)

		if p.Direction != protocol.DirUp {
			t.Errorf("expected Direction DirUp, got %d", p.Direction)
		}
	})

	t.Run("prevents 180 reversal Up to Down", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.Direction = protocol.DirUp

		p.SetDirection(protocol.DirDown)

		if p.Direction != protocol.DirUp {
			t.Error("expected Direction to remain DirUp, 180 reversal should be blocked")
		}
	})

	t.Run("prevents 180 reversal Down to Up", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.Direction = protocol.DirDown

		p.SetDirection(protocol.DirUp)

		if p.Direction != protocol.DirDown {
			t.Error("expected Direction to remain DirDown, 180 reversal should be blocked")
		}
	})

	t.Run("prevents 180 reversal Left to Right", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.Direction = protocol.DirLeft

		p.SetDirection(protocol.DirRight)

		if p.Direction != protocol.DirLeft {
			t.Error("expected Direction to remain DirLeft, 180 reversal should be blocked")
		}
	})

	t.Run("prevents 180 reversal Right to Left", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.Direction = protocol.DirRight

		p.SetDirection(protocol.DirLeft)

		if p.Direction != protocol.DirRight {
			t.Error("expected Direction to remain DirRight, 180 reversal should be blocked")
		}
	})

	t.Run("allows perpendicular changes", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.Direction = protocol.DirUp

		p.SetDirection(protocol.DirLeft)
		if p.Direction != protocol.DirLeft {
			t.Error("expected Direction to change to DirLeft")
		}

		p.SetDirection(protocol.DirDown)
		if p.Direction != protocol.DirDown {
			t.Error("expected Direction to change to DirDown")
		}

		p.SetDirection(protocol.DirRight)
		if p.Direction != protocol.DirRight {
			t.Error("expected Direction to change to DirRight")
		}
	})

	t.Run("allows setting direction from None", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		// Direction starts as DirNone

		p.SetDirection(protocol.DirUp)
		if p.Direction != protocol.DirUp {
			t.Error("expected Direction to change to DirUp from DirNone")
		}
	})

	t.Run("allows setting to None", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.Direction = protocol.DirUp

		p.SetDirection(protocol.DirNone)
		if p.Direction != protocol.DirNone {
			t.Error("expected Direction to change to DirNone")
		}
	})
}

func TestGrantProtection(t *testing.T) {
	t.Run("sets Protected true and ticks", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)

		p.GrantProtection(40)

		if !p.Protected {
			t.Error("expected Protected true after grant")
		}
		if p.ProtectionTicks != 40 {
			t.Errorf("expected ProtectionTicks 40, got %d", p.ProtectionTicks)
		}
	})
}

func TestDecrementProtection(t *testing.T) {
	t.Run("decrements ticks", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.GrantProtection(40)

		p.DecrementProtection()

		if p.ProtectionTicks != 39 {
			t.Errorf("expected ProtectionTicks 39, got %d", p.ProtectionTicks)
		}
		if !p.Protected {
			t.Error("expected Protected to remain true")
		}
	})

	t.Run("clears protection when ticks reach zero", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		p.GrantProtection(1)

		p.DecrementProtection()

		if p.ProtectionTicks != 0 {
			t.Errorf("expected ProtectionTicks 0, got %d", p.ProtectionTicks)
		}
		if p.Protected {
			t.Error("expected Protected to be false when ticks reach 0")
		}
	})

	t.Run("does nothing when already zero", func(t *testing.T) {
		p := NewPlayer(1, "Alice", 2)
		// No protection granted, ticks are 0

		p.DecrementProtection()

		if p.ProtectionTicks != 0 {
			t.Errorf("expected ProtectionTicks to remain 0, got %d", p.ProtectionTicks)
		}
		if p.Protected {
			t.Error("expected Protected to remain false")
		}
	})
}

func TestPoint(t *testing.T) {
	t.Run("stores X and Y", func(t *testing.T) {
		pt := Point{X: 10, Y: 20}

		if pt.X != 10 {
			t.Errorf("expected X 10, got %d", pt.X)
		}
		if pt.Y != 20 {
			t.Errorf("expected Y 20, got %d", pt.Y)
		}
	})
}
