package config

import (
	"strings"
	"testing"
)

func TestParseArgs_ServerMode(t *testing.T) {
	cfg, err := ParseArgs([]string{"--server"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsServer {
		t.Error("expected IsServer to be true")
	}
	if cfg.Port != DefaultPort {
		t.Errorf("expected default port %d, got %d", DefaultPort, cfg.Port)
	}
}

func TestParseArgs_ClientMode(t *testing.T) {
	cfg, err := ParseArgs([]string{"--join", "192.168.1.100"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IsServer {
		t.Error("expected IsServer to be false")
	}
	if cfg.ServerAddr != "192.168.1.100" {
		t.Errorf("expected server addr 192.168.1.100, got %s", cfg.ServerAddr)
	}
}

func TestParseArgs_ServerWithOptions(t *testing.T) {
	cfg, err := ParseArgs([]string{"--server", "--port", "8888", "--time", "10", "--threshold", "80", "--powerups"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Port)
	}
	if cfg.GameDuration != 10 {
		t.Errorf("expected duration 10, got %d", cfg.GameDuration)
	}
	if cfg.Threshold != 80 {
		t.Errorf("expected threshold 80, got %d", cfg.Threshold)
	}
	if !cfg.PowerupsEnabled {
		t.Error("expected powerups to be enabled")
	}
}

func TestParseArgs_NoMode_ReturnsError(t *testing.T) {
	_, err := ParseArgs([]string{})
	if err == nil {
		t.Error("expected error for missing mode")
	}
}

func TestParseArgs_NameFlag(t *testing.T) {
	cfg, err := ParseArgs([]string{"--join", "localhost", "--name", "Player1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PlayerName != "Player1" {
		t.Errorf("expected player name Player1, got %s", cfg.PlayerName)
	}
}

func TestParseArgs_ServerAndJoin_ReturnsError(t *testing.T) {
	_, err := ParseArgs([]string{"--server", "--join", "localhost"})
	if err == nil {
		t.Error("expected error when both --server and --join are specified")
	}
	if err != nil && !strings.Contains(err.Error(), "cannot specify both") {
		t.Errorf("expected error about both flags, got: %v", err)
	}
}

func TestParseArgs_InvalidPort_ReturnsError(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{"port too low", "0"},
		{"port negative", "-1"},
		{"port too high", "65536"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseArgs([]string{"--server", "--port", tc.port})
			if err == nil {
				t.Errorf("expected error for port %s", tc.port)
			}
			if err != nil && !strings.Contains(err.Error(), "port must be between") {
				t.Errorf("expected port validation error, got: %v", err)
			}
		})
	}
}

func TestParseArgs_InvalidGameDuration_ReturnsError(t *testing.T) {
	tests := []struct {
		name     string
		duration string
	}{
		{"duration zero", "0"},
		{"duration negative", "-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseArgs([]string{"--server", "--time", tc.duration})
			if err == nil {
				t.Errorf("expected error for duration %s", tc.duration)
			}
			if err != nil && !strings.Contains(err.Error(), "game duration must be greater than 0") {
				t.Errorf("expected duration validation error, got: %v", err)
			}
		})
	}
}

func TestParseArgs_InvalidThreshold_ReturnsError(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
	}{
		{"threshold negative", "-1"},
		{"threshold too high", "101"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseArgs([]string{"--server", "--threshold", tc.threshold})
			if err == nil {
				t.Errorf("expected error for threshold %s", tc.threshold)
			}
			if err != nil && !strings.Contains(err.Error(), "threshold must be between") {
				t.Errorf("expected threshold validation error, got: %v", err)
			}
		})
	}
}

func TestParseArgs_ValidBoundaryValues(t *testing.T) {
	// Test valid boundary values
	cfg, err := ParseArgs([]string{"--server", "--port", "1", "--threshold", "0"})
	if err != nil {
		t.Fatalf("unexpected error for min boundary values: %v", err)
	}
	if cfg.Port != 1 {
		t.Errorf("expected port 1, got %d", cfg.Port)
	}
	if cfg.Threshold != 0 {
		t.Errorf("expected threshold 0, got %d", cfg.Threshold)
	}

	cfg, err = ParseArgs([]string{"--server", "--port", "65535", "--threshold", "100"})
	if err != nil {
		t.Fatalf("unexpected error for max boundary values: %v", err)
	}
	if cfg.Port != 65535 {
		t.Errorf("expected port 65535, got %d", cfg.Port)
	}
	if cfg.Threshold != 100 {
		t.Errorf("expected threshold 100, got %d", cfg.Threshold)
	}
}
