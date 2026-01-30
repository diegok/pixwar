package config

import "testing"

func TestParseArgs_ServerMode(t *testing.T) {
	cfg, err := ParseArgs([]string{"--server"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsServer {
		t.Error("expected IsServer to be true")
	}
	if cfg.Port != 7777 {
		t.Errorf("expected default port 7777, got %d", cfg.Port)
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
