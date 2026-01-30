package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestNewScreen(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	defer sim.Fini()

	screen := NewScreen(sim)
	if screen == nil {
		t.Fatal("NewScreen returned nil")
	}
	if screen.screen != sim {
		t.Error("NewScreen did not wrap the provided screen")
	}
}

func TestScreenSize(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	defer sim.Fini()

	sim.SetSize(80, 24)
	screen := NewScreen(sim)

	w, h := screen.Size()
	if w != 80 || h != 24 {
		t.Errorf("Size() = (%d, %d), want (80, 24)", w, h)
	}
}

func TestDrawText(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	defer sim.Fini()

	sim.SetSize(80, 24)
	screen := NewScreen(sim)

	text := "Hello"
	screen.DrawText(5, 3, text, tcell.StyleDefault)
	sim.Show()

	// Verify each character was drawn
	for i, expected := range text {
		r, _, _, _ := sim.GetContent(5+i, 3)
		if r != expected {
			t.Errorf("GetContent(%d, 3) = %c, want %c", 5+i, r, expected)
		}
	}
}

func TestDrawBox(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	defer sim.Fini()

	sim.SetSize(80, 24)
	screen := NewScreen(sim)

	screen.DrawBox(0, 0, 5, 3, tcell.StyleDefault)
	sim.Show()

	// Check corners
	topLeft, _, _, _ := sim.GetContent(0, 0)
	topRight, _, _, _ := sim.GetContent(4, 0)
	bottomLeft, _, _, _ := sim.GetContent(0, 2)
	bottomRight, _, _, _ := sim.GetContent(4, 2)

	if topLeft != '┌' {
		t.Errorf("Top left corner = %c, want ┌", topLeft)
	}
	if topRight != '┐' {
		t.Errorf("Top right corner = %c, want ┐", topRight)
	}
	if bottomLeft != '└' {
		t.Errorf("Bottom left corner = %c, want └", bottomLeft)
	}
	if bottomRight != '┘' {
		t.Errorf("Bottom right corner = %c, want ┘", bottomRight)
	}

	// Check horizontal edges
	for x := 1; x < 4; x++ {
		top, _, _, _ := sim.GetContent(x, 0)
		bottom, _, _, _ := sim.GetContent(x, 2)
		if top != '─' {
			t.Errorf("Top edge at x=%d = %c, want ─", x, top)
		}
		if bottom != '─' {
			t.Errorf("Bottom edge at x=%d = %c, want ─", x, bottom)
		}
	}

	// Check vertical edges
	left, _, _, _ := sim.GetContent(0, 1)
	right, _, _, _ := sim.GetContent(4, 1)
	if left != '│' {
		t.Errorf("Left edge = %c, want │", left)
	}
	if right != '│' {
		t.Errorf("Right edge = %c, want │", right)
	}
}

func TestFillRect(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	defer sim.Fini()

	sim.SetSize(80, 24)
	screen := NewScreen(sim)

	screen.FillRect(2, 2, 3, 2, tcell.StyleDefault, '#')
	sim.Show()

	// Check all cells in the rectangle
	for y := 2; y < 4; y++ {
		for x := 2; x < 5; x++ {
			r, _, _, _ := sim.GetContent(x, y)
			if r != '#' {
				t.Errorf("GetContent(%d, %d) = %c, want #", x, y, r)
			}
		}
	}
}

func TestSetCell(t *testing.T) {
	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	defer sim.Fini()

	sim.SetSize(80, 24)
	screen := NewScreen(sim)

	screen.SetCell(10, 5, tcell.StyleDefault, 'X')
	sim.Show()

	r, _, _, _ := sim.GetContent(10, 5)
	if r != 'X' {
		t.Errorf("GetContent(10, 5) = %c, want X", r)
	}
}

func TestGetPlayerStyle(t *testing.T) {
	tests := []struct {
		index    int
		expected tcell.Color
	}{
		{0, tcell.ColorRed},
		{1, tcell.ColorBlue},
		{7, tcell.ColorFuchsia},
		{-1, tcell.ColorDefault},
		{100, tcell.ColorDefault},
	}

	for _, tt := range tests {
		style := GetPlayerStyle(tt.index)
		fg, _, _ := style.Decompose()
		if fg != tt.expected {
			t.Errorf("GetPlayerStyle(%d) foreground = %v, want %v", tt.index, fg, tt.expected)
		}
	}
}

func TestGetPlayerBgStyle(t *testing.T) {
	tests := []struct {
		index    int
		expected tcell.Color
	}{
		{0, tcell.ColorRed},
		{1, tcell.ColorBlue},
		{7, tcell.ColorFuchsia},
		{-1, tcell.ColorDefault},
		{100, tcell.ColorDefault},
	}

	for _, tt := range tests {
		style := GetPlayerBgStyle(tt.index)
		_, bg, _ := style.Decompose()
		if bg != tt.expected {
			t.Errorf("GetPlayerBgStyle(%d) background = %v, want %v", tt.index, bg, tt.expected)
		}
	}
}

func TestPlayerColors(t *testing.T) {
	expected := []tcell.Color{
		tcell.ColorRed,
		tcell.ColorBlue,
		tcell.ColorGreen,
		tcell.ColorYellow,
		tcell.ColorPurple,
		tcell.ColorOrange,
		tcell.ColorTeal,
		tcell.ColorFuchsia,
	}

	if len(PlayerColors) != len(expected) {
		t.Errorf("PlayerColors length = %d, want %d", len(PlayerColors), len(expected))
	}

	for i, c := range expected {
		if PlayerColors[i] != c {
			t.Errorf("PlayerColors[%d] = %v, want %v", i, PlayerColors[i], c)
		}
	}
}
