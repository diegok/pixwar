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

// CaptureTerritory fills territory based on a closed trail
// Returns the number of cells captured (including trail cells)
func (b *Board) CaptureTerritory(playerID int, trail []Point) int {
	if len(trail) == 0 {
		return 0
	}

	// 1. Create a set of trail positions for fast lookup
	trailSet := make(map[Point]bool)
	for _, pt := range trail {
		trailSet[pt] = true
	}

	// 2. Mark all trail cells as territory first
	for _, pt := range trail {
		b.SetTerritory(pt.X, pt.Y, playerID)
	}

	// 3. Find all distinct regions separated by the trail using flood fill
	// The boundary consists of: edges + trail + existing territory
	visited := make([][]bool, b.Height)
	for y := range visited {
		visited[y] = make([]bool, b.Width)
	}

	// Mark ALL edge cells as boundary (visited) - this is critical for Qix-style capture
	for x := 0; x < b.Width; x++ {
		visited[0][x] = true           // Top edge
		visited[b.Height-1][x] = true  // Bottom edge
	}
	for y := 0; y < b.Height; y++ {
		visited[y][0] = true           // Left edge
		visited[y][b.Width-1] = true   // Right edge
	}

	// Mark trail cells as visited so we don't flood fill through them
	for _, pt := range trail {
		if b.IsInBounds(pt.X, pt.Y) {
			visited[pt.Y][pt.X] = true
		}
	}

	// Mark existing player territory as boundary so trails that return
	// to territory form proper closed regions for capture
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			cell := b.GetCell(x, y)
			if cell.Owner == playerID && !trailSet[Point{x, y}] {
				visited[y][x] = true
			}
		}
	}

	// Find all regions in the interior (non-edge cells)
	var regions [][]Point
	for y := 1; y < b.Height-1; y++ {
		for x := 1; x < b.Width-1; x++ {
			if !visited[y][x] {
				region := b.floodFill(x, y, visited)
				if len(region) > 0 {
					regions = append(regions, region)
				}
			}
		}
	}

	// 4. Capture based on Qix rules
	captured := len(trail)

	// If only 0 or 1 region, the trail didn't divide anything - just keep the trail as territory
	if len(regions) <= 1 {
		return captured
	}

	// Multiple regions - capture the smallest one (Qix rule)
	smallestRegion := regions[0]
	for _, region := range regions[1:] {
		if len(region) < len(smallestRegion) {
			smallestRegion = region
		}
	}
	for _, pt := range smallestRegion {
		b.SetTerritory(pt.X, pt.Y, playerID)
	}
	captured += len(smallestRegion)

	return captured
}

// floodFill performs a flood fill from the starting point and returns all connected cells
func (b *Board) floodFill(startX, startY int, visited [][]bool) []Point {
	var region []Point
	stack := []Point{{startX, startY}}

	for len(stack) > 0 {
		// Pop from stack
		pt := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !b.IsInBounds(pt.X, pt.Y) || visited[pt.Y][pt.X] {
			continue
		}

		visited[pt.Y][pt.X] = true
		region = append(region, pt)

		// Add neighbors (4-directional)
		neighbors := []Point{
			{pt.X - 1, pt.Y},
			{pt.X + 1, pt.Y},
			{pt.X, pt.Y - 1},
			{pt.X, pt.Y + 1},
		}
		for _, n := range neighbors {
			if b.IsInBounds(n.X, n.Y) && !visited[n.Y][n.X] {
				stack = append(stack, n)
			}
		}
	}

	return region
}

// GetPlayerTerritory returns all territory cells owned by the specified player
// Does not include trail cells
func (b *Board) GetPlayerTerritory(playerID int) []Point {
	var territory []Point
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			cell := b.GetCell(x, y)
			if cell.Owner == playerID && !cell.IsTrail {
				territory = append(territory, Point{x, y})
			}
		}
	}
	return territory
}

// ClearPlayerTrails clears all trail cells for the specified player
// Does not affect territory cells
func (b *Board) ClearPlayerTrails(playerID int) {
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			cell := b.GetCell(x, y)
			if cell.Owner == playerID && cell.IsTrail {
				b.ClearTrail(x, y)
			}
		}
	}
}

// ClearPlayerTerritory clears all cells (both territory and trails) owned by the specified player
func (b *Board) ClearPlayerTerritory(playerID int) {
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			cell := b.GetCell(x, y)
			if cell.Owner == playerID {
				b.cells[y][x].Owner = -1
				b.cells[y][x].IsTrail = false
			}
		}
	}
}
