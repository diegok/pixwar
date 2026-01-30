package game

// Cell represents a single cell on the board
type Cell struct {
	Owner   int  // -1 = unclaimed, 0+ = player ID
	IsTrail bool // true if this cell is part of a trail (not yet converted to territory)
}

// Board represents the game board
type Board struct {
	Width  int
	Height int
	cells  [][]Cell
}

// NewBoard creates a new board with the given dimensions
// All cells are initialized as unclaimed (Owner = -1)
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

// GetCell returns the cell at the given coordinates
// Returns nil if coordinates are out of bounds
func (b *Board) GetCell(x, y int) *Cell {
	if !b.IsInBounds(x, y) {
		return nil
	}
	return &b.cells[y][x]
}

// SetTerritory sets the cell at (x, y) as territory owned by playerID
// The cell is marked as territory (not trail)
// Does nothing if coordinates are out of bounds
func (b *Board) SetTerritory(x, y, playerID int) {
	if !b.IsInBounds(x, y) {
		return
	}
	b.cells[y][x].Owner = playerID
	b.cells[y][x].IsTrail = false
}

// SetTrail sets the cell at (x, y) as trail owned by playerID
// Does nothing if coordinates are out of bounds
func (b *Board) SetTrail(x, y, playerID int) {
	if !b.IsInBounds(x, y) {
		return
	}
	b.cells[y][x].Owner = playerID
	b.cells[y][x].IsTrail = true
}

// ClearTrail clears the trail at (x, y), resetting to unclaimed
// Does nothing if coordinates are out of bounds
func (b *Board) ClearTrail(x, y int) {
	if !b.IsInBounds(x, y) {
		return
	}
	b.cells[y][x].Owner = -1
	b.cells[y][x].IsTrail = false
}

// ConvertTrailToTerritory converts a trail cell to territory
// Keeps the same owner but clears the trail flag
// Does nothing if coordinates are out of bounds
func (b *Board) ConvertTrailToTerritory(x, y int) {
	if !b.IsInBounds(x, y) {
		return
	}
	b.cells[y][x].IsTrail = false
}

// IsOnEdge returns true if the given coordinates are on the board edge
func (b *Board) IsOnEdge(x, y int) bool {
	return x == 0 || y == 0 || x == b.Width-1 || y == b.Height-1
}

// IsInBounds returns true if the given coordinates are within the board
func (b *Board) IsInBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < b.Width && y < b.Height
}

// CountTerritory returns the number of cells owned by the given player
// Only counts territory cells, not trails
func (b *Board) CountTerritory(playerID int) int {
	count := 0
	for y := range b.cells {
		for x := range b.cells[y] {
			cell := &b.cells[y][x]
			if cell.Owner == playerID && !cell.IsTrail {
				count++
			}
		}
	}
	return count
}

// TotalCells returns the total number of cells on the board
func (b *Board) TotalCells() int {
	return b.Width * b.Height
}
