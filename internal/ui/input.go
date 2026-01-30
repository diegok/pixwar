package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/diegok/pixwar/internal/protocol"
)

// KeyToDirection converts a key event to a direction
// Arrow keys and WASD are supported
func KeyToDirection(key tcell.Key, r rune) protocol.Direction {
	// Check arrow keys first
	switch key {
	case tcell.KeyUp:
		return protocol.DirUp
	case tcell.KeyDown:
		return protocol.DirDown
	case tcell.KeyLeft:
		return protocol.DirLeft
	case tcell.KeyRight:
		return protocol.DirRight
	}

	// Check WASD (for regular character keys, key will be KeyRune)
	// Also handle case where key might not be KeyRune but rune is set
	switch r {
	case 'w', 'W':
		return protocol.DirUp
	case 's', 'S':
		return protocol.DirDown
	case 'a', 'A':
		return protocol.DirLeft
	case 'd', 'D':
		return protocol.DirRight
	}

	return protocol.DirNone
}

// IsQuitKey returns true if the key is a quit key (q, Q, Escape, Ctrl+C)
func IsQuitKey(key tcell.Key, r rune) bool {
	if key == tcell.KeyEscape || key == tcell.KeyCtrlC {
		return true
	}
	if key == tcell.KeyRune && (r == 'q' || r == 'Q') {
		return true
	}
	return false
}

// IsStartKey returns true if the key is the start key (Enter)
func IsStartKey(key tcell.Key) bool {
	return key == tcell.KeyEnter
}

// IsTabKey returns true if the key is Tab (for spectator cycling)
func IsTabKey(key tcell.Key) bool {
	return key == tcell.KeyTab
}
