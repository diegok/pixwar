package ui

import (
	"github.com/gdamore/tcell/v2"
)

// Screen wraps tcell.Screen providing convenient drawing methods
type Screen struct {
	screen tcell.Screen
}

// PlayerColors defines the colors for up to 8 players
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

// NewScreen wraps an existing tcell.Screen
func NewScreen(s tcell.Screen) *Screen {
	return &Screen{screen: s}
}

// InitScreen creates and initializes a real terminal screen
func InitScreen() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	s.EnableMouse()
	s.EnablePaste()
	return NewScreen(s), nil
}

// Size returns the terminal dimensions
func (s *Screen) Size() (int, int) {
	return s.screen.Size()
}

// Clear clears the screen
func (s *Screen) Clear() {
	s.screen.Clear()
}

// Show updates the terminal with buffered content
func (s *Screen) Show() {
	s.screen.Show()
}

// Fini releases the screen resources
func (s *Screen) Fini() {
	s.screen.Fini()
}

// SetCell sets a single cell on the screen
func (s *Screen) SetCell(x, y int, style tcell.Style, r rune) {
	s.screen.SetContent(x, y, r, nil, style)
}

// DrawText renders a string starting at position x, y
func (s *Screen) DrawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		s.screen.SetContent(x+i, y, r, nil, style)
	}
}

// DrawBox draws a Unicode box with the given dimensions
func (s *Screen) DrawBox(x, y, w, h int, style tcell.Style) {
	// Box drawing characters
	const (
		topLeft     = '┌'
		topRight    = '┐'
		bottomLeft  = '└'
		bottomRight = '┘'
		horizontal  = '─'
		vertical    = '│'
	)

	// Corners
	s.screen.SetContent(x, y, topLeft, nil, style)
	s.screen.SetContent(x+w-1, y, topRight, nil, style)
	s.screen.SetContent(x, y+h-1, bottomLeft, nil, style)
	s.screen.SetContent(x+w-1, y+h-1, bottomRight, nil, style)

	// Top and bottom edges
	for i := x + 1; i < x+w-1; i++ {
		s.screen.SetContent(i, y, horizontal, nil, style)
		s.screen.SetContent(i, y+h-1, horizontal, nil, style)
	}

	// Left and right edges
	for j := y + 1; j < y+h-1; j++ {
		s.screen.SetContent(x, j, vertical, nil, style)
		s.screen.SetContent(x+w-1, j, vertical, nil, style)
	}
}

// FillRect fills a rectangle with a rune
func (s *Screen) FillRect(x, y, w, h int, style tcell.Style, r rune) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			s.screen.SetContent(x+dx, y+dy, r, nil, style)
		}
	}
}

// PollEvent waits for and returns the next event
func (s *Screen) PollEvent() tcell.Event {
	return s.screen.PollEvent()
}

// PostEvent posts an event to the screen event queue
func (s *Screen) PostEvent(ev tcell.Event) {
	s.screen.PostEvent(ev)
}

// GetPlayerStyle returns a foreground style for a player color
func GetPlayerStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Foreground(PlayerColors[colorIndex])
}

// GetPlayerBgStyle returns a background style for a player color
func GetPlayerBgStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Background(PlayerColors[colorIndex])
}
