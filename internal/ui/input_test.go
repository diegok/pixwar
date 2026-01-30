package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/diegok/pixwar/internal/protocol"
)

func TestKeyToDirection_ArrowKeys(t *testing.T) {
	tests := []struct {
		key      tcell.Key
		expected protocol.Direction
	}{
		{tcell.KeyUp, protocol.DirUp},
		{tcell.KeyDown, protocol.DirDown},
		{tcell.KeyLeft, protocol.DirLeft},
		{tcell.KeyRight, protocol.DirRight},
	}

	for _, tt := range tests {
		result := KeyToDirection(tt.key, 0)
		if result != tt.expected {
			t.Errorf("KeyToDirection(%v, 0) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestKeyToDirection_WASD(t *testing.T) {
	tests := []struct {
		r        rune
		expected protocol.Direction
	}{
		{'w', protocol.DirUp},
		{'W', protocol.DirUp},
		{'s', protocol.DirDown},
		{'S', protocol.DirDown},
		{'a', protocol.DirLeft},
		{'A', protocol.DirLeft},
		{'d', protocol.DirRight},
		{'D', protocol.DirRight},
	}

	for _, tt := range tests {
		result := KeyToDirection(tcell.KeyRune, tt.r)
		if result != tt.expected {
			t.Errorf("KeyToDirection(KeyRune, '%c') = %v, want %v", tt.r, result, tt.expected)
		}
	}
}

func TestKeyToDirection_Other(t *testing.T) {
	// Other keys should return DirNone
	tests := []struct {
		key tcell.Key
		r   rune
	}{
		{tcell.KeyEnter, 0},
		{tcell.KeyTab, 0},
		{tcell.KeyRune, 'x'},
		{tcell.KeyRune, 'q'},
		{tcell.KeyEscape, 0},
	}

	for _, tt := range tests {
		result := KeyToDirection(tt.key, tt.r)
		if result != protocol.DirNone {
			t.Errorf("KeyToDirection(%v, '%c') = %v, want DirNone", tt.key, tt.r, result)
		}
	}
}

func TestIsQuitKey(t *testing.T) {
	tests := []struct {
		key      tcell.Key
		r        rune
		expected bool
	}{
		{tcell.KeyEscape, 0, true},
		{tcell.KeyCtrlC, 0, true},
		{tcell.KeyRune, 'q', true},
		{tcell.KeyRune, 'Q', true},
		{tcell.KeyEnter, 0, false},
		{tcell.KeyRune, 'x', false},
		{tcell.KeyTab, 0, false},
		{tcell.KeyUp, 0, false},
	}

	for _, tt := range tests {
		result := IsQuitKey(tt.key, tt.r)
		if result != tt.expected {
			t.Errorf("IsQuitKey(%v, '%c') = %v, want %v", tt.key, tt.r, result, tt.expected)
		}
	}
}

func TestIsStartKey(t *testing.T) {
	tests := []struct {
		key      tcell.Key
		expected bool
	}{
		{tcell.KeyEnter, true},
		{tcell.KeyEscape, false},
		{tcell.KeyTab, false},
		{tcell.KeyRune, false},
		{tcell.KeyUp, false},
	}

	for _, tt := range tests {
		result := IsStartKey(tt.key)
		if result != tt.expected {
			t.Errorf("IsStartKey(%v) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestIsTabKey(t *testing.T) {
	tests := []struct {
		key      tcell.Key
		expected bool
	}{
		{tcell.KeyTab, true},
		{tcell.KeyEnter, false},
		{tcell.KeyEscape, false},
		{tcell.KeyRune, false},
		{tcell.KeyUp, false},
	}

	for _, tt := range tests {
		result := IsTabKey(tt.key)
		if result != tt.expected {
			t.Errorf("IsTabKey(%v) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}
