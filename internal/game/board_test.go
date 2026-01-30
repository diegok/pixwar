package game

import (
	"testing"
)

func TestNewBoard(t *testing.T) {
	t.Run("creates board with correct dimensions", func(t *testing.T) {
		b := NewBoard(10, 8)

		if b.Width != 10 {
			t.Errorf("expected Width 10, got %d", b.Width)
		}
		if b.Height != 8 {
			t.Errorf("expected Height 8, got %d", b.Height)
		}
	})

	t.Run("cells slice has correct size", func(t *testing.T) {
		b := NewBoard(10, 8)

		if len(b.cells) != 8 {
			t.Errorf("expected 8 rows, got %d", len(b.cells))
		}
		for y, row := range b.cells {
			if len(row) != 10 {
				t.Errorf("expected 10 columns in row %d, got %d", y, len(row))
			}
		}
	})
}

func TestGetCell(t *testing.T) {
	t.Run("returns unclaimed cell for empty board", func(t *testing.T) {
		b := NewBoard(10, 8)

		cell := b.GetCell(5, 4)

		if cell.Owner != -1 {
			t.Errorf("expected Owner -1 (unclaimed), got %d", cell.Owner)
		}
		if cell.IsTrail {
			t.Error("expected IsTrail false, got true")
		}
	})

	t.Run("returns nil for out of bounds coordinates", func(t *testing.T) {
		b := NewBoard(10, 8)

		testCases := []struct {
			name string
			x, y int
		}{
			{"negative x", -1, 4},
			{"negative y", 5, -1},
			{"x >= width", 10, 4},
			{"y >= height", 5, 8},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cell := b.GetCell(tc.x, tc.y)
				if cell != nil {
					t.Errorf("expected nil for out of bounds (%d, %d), got %+v", tc.x, tc.y, cell)
				}
			})
		}
	})
}

func TestSetTerritory(t *testing.T) {
	t.Run("sets cell owner", func(t *testing.T) {
		b := NewBoard(10, 8)

		b.SetTerritory(5, 4, 2)
		cell := b.GetCell(5, 4)

		if cell.Owner != 2 {
			t.Errorf("expected Owner 2, got %d", cell.Owner)
		}
		if cell.IsTrail {
			t.Error("SetTerritory should not set IsTrail")
		}
	})

	t.Run("round-trip with GetCell", func(t *testing.T) {
		b := NewBoard(10, 8)

		b.SetTerritory(3, 2, 1)
		b.SetTerritory(7, 6, 3)

		cell1 := b.GetCell(3, 2)
		cell2 := b.GetCell(7, 6)

		if cell1.Owner != 1 {
			t.Errorf("expected Owner 1, got %d", cell1.Owner)
		}
		if cell2.Owner != 3 {
			t.Errorf("expected Owner 3, got %d", cell2.Owner)
		}
	})

	t.Run("ignores out of bounds", func(t *testing.T) {
		b := NewBoard(10, 8)

		// Should not panic
		b.SetTerritory(-1, 4, 1)
		b.SetTerritory(10, 4, 1)
		b.SetTerritory(5, -1, 1)
		b.SetTerritory(5, 8, 1)
	})
}

func TestSetTrail(t *testing.T) {
	t.Run("marks cell as trail with owner", func(t *testing.T) {
		b := NewBoard(10, 8)

		b.SetTrail(5, 4, 2)
		cell := b.GetCell(5, 4)

		if cell.Owner != 2 {
			t.Errorf("expected Owner 2, got %d", cell.Owner)
		}
		if !cell.IsTrail {
			t.Error("expected IsTrail true, got false")
		}
	})

	t.Run("ignores out of bounds", func(t *testing.T) {
		b := NewBoard(10, 8)

		// Should not panic
		b.SetTrail(-1, 4, 1)
		b.SetTrail(10, 4, 1)
	})
}

func TestClearTrail(t *testing.T) {
	t.Run("clears trail and owner", func(t *testing.T) {
		b := NewBoard(10, 8)

		b.SetTrail(5, 4, 2)
		b.ClearTrail(5, 4)
		cell := b.GetCell(5, 4)

		if cell.Owner != -1 {
			t.Errorf("expected Owner -1 after clear, got %d", cell.Owner)
		}
		if cell.IsTrail {
			t.Error("expected IsTrail false after clear, got true")
		}
	})

	t.Run("ignores out of bounds", func(t *testing.T) {
		b := NewBoard(10, 8)

		// Should not panic
		b.ClearTrail(-1, 4)
		b.ClearTrail(10, 4)
	})
}

func TestConvertTrailToTerritory(t *testing.T) {
	t.Run("converts trail to territory keeping owner", func(t *testing.T) {
		b := NewBoard(10, 8)

		b.SetTrail(5, 4, 2)
		b.ConvertTrailToTerritory(5, 4)
		cell := b.GetCell(5, 4)

		if cell.Owner != 2 {
			t.Errorf("expected Owner 2 after conversion, got %d", cell.Owner)
		}
		if cell.IsTrail {
			t.Error("expected IsTrail false after conversion, got true")
		}
	})

	t.Run("ignores out of bounds", func(t *testing.T) {
		b := NewBoard(10, 8)

		// Should not panic
		b.ConvertTrailToTerritory(-1, 4)
		b.ConvertTrailToTerritory(10, 4)
	})
}

func TestIsOnEdge(t *testing.T) {
	b := NewBoard(10, 8)

	testCases := []struct {
		name     string
		x, y     int
		expected bool
	}{
		// Corners
		{"top-left corner", 0, 0, true},
		{"top-right corner", 9, 0, true},
		{"bottom-left corner", 0, 7, true},
		{"bottom-right corner", 9, 7, true},
		// Edges
		{"top edge", 5, 0, true},
		{"bottom edge", 5, 7, true},
		{"left edge", 0, 4, true},
		{"right edge", 9, 4, true},
		// Center
		{"center", 5, 4, false},
		{"near edge but center", 1, 1, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := b.IsOnEdge(tc.x, tc.y)
			if result != tc.expected {
				t.Errorf("IsOnEdge(%d, %d) = %v, expected %v", tc.x, tc.y, result, tc.expected)
			}
		})
	}
}

func TestIsInBounds(t *testing.T) {
	b := NewBoard(10, 8)

	testCases := []struct {
		name     string
		x, y     int
		expected bool
	}{
		// Valid coordinates
		{"origin", 0, 0, true},
		{"center", 5, 4, true},
		{"max valid", 9, 7, true},
		// Invalid coordinates
		{"negative x", -1, 4, false},
		{"negative y", 5, -1, false},
		{"x == width", 10, 4, false},
		{"y == height", 5, 8, false},
		{"both out of bounds", -1, -1, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := b.IsInBounds(tc.x, tc.y)
			if result != tc.expected {
				t.Errorf("IsInBounds(%d, %d) = %v, expected %v", tc.x, tc.y, result, tc.expected)
			}
		})
	}
}

func TestCountTerritory(t *testing.T) {
	t.Run("returns 0 for empty board", func(t *testing.T) {
		b := NewBoard(10, 8)

		count := b.CountTerritory(1)

		if count != 0 {
			t.Errorf("expected 0 for empty board, got %d", count)
		}
	})

	t.Run("counts only territory not trails", func(t *testing.T) {
		b := NewBoard(10, 8)

		b.SetTerritory(0, 0, 1)
		b.SetTerritory(1, 0, 1)
		b.SetTerritory(2, 0, 1)
		b.SetTrail(3, 0, 1) // Trail should not count

		count := b.CountTerritory(1)

		if count != 3 {
			t.Errorf("expected 3, got %d", count)
		}
	})

	t.Run("counts only specified player", func(t *testing.T) {
		b := NewBoard(10, 8)

		b.SetTerritory(0, 0, 1)
		b.SetTerritory(1, 0, 1)
		b.SetTerritory(2, 0, 2)
		b.SetTerritory(3, 0, 2)
		b.SetTerritory(4, 0, 2)

		count1 := b.CountTerritory(1)
		count2 := b.CountTerritory(2)

		if count1 != 2 {
			t.Errorf("expected 2 for player 1, got %d", count1)
		}
		if count2 != 3 {
			t.Errorf("expected 3 for player 2, got %d", count2)
		}
	})
}

func TestTotalCells(t *testing.T) {
	t.Run("returns width times height", func(t *testing.T) {
		b := NewBoard(10, 8)

		total := b.TotalCells()

		if total != 80 {
			t.Errorf("expected 80, got %d", total)
		}
	})

	t.Run("works for different sizes", func(t *testing.T) {
		b := NewBoard(50, 30)

		total := b.TotalCells()

		if total != 1500 {
			t.Errorf("expected 1500, got %d", total)
		}
	})
}
