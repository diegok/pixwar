package config

import (
	"errors"
	"flag"
	"fmt"
)

// Default values for configuration
const (
	DefaultPort         = 7777
	DefaultGameDuration = 5
	DefaultThreshold    = 95
)

type Config struct {
	IsServer        bool
	ServerAddr      string
	Port            int
	GameDuration    int // minutes
	Threshold       int // percentage
	PowerupsEnabled bool
	PlayerName      string
}

func ParseArgs(args []string) (*Config, error) {
	cfg := &Config{
		Port:         DefaultPort,
		GameDuration: DefaultGameDuration,
		Threshold:    DefaultThreshold,
	}

	fs := flag.NewFlagSet("pixwar", flag.ContinueOnError)
	fs.BoolVar(&cfg.IsServer, "server", false, "Run as server")
	fs.StringVar(&cfg.ServerAddr, "join", "", "Server address to join")
	fs.IntVar(&cfg.Port, "port", DefaultPort, "Server port")
	fs.IntVar(&cfg.GameDuration, "time", DefaultGameDuration, "Game duration in minutes")
	fs.IntVar(&cfg.Threshold, "threshold", DefaultThreshold, "Territory percentage to end game")
	fs.BoolVar(&cfg.PowerupsEnabled, "powerups", false, "Enable power-ups")
	fs.StringVar(&cfg.PlayerName, "name", "", "Player name")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Validate that --server and --join are not both specified
	if cfg.IsServer && cfg.ServerAddr != "" {
		return nil, errors.New("cannot specify both --server and --join")
	}

	if !cfg.IsServer && cfg.ServerAddr == "" {
		return nil, errors.New("must specify --server or --join <address>")
	}

	// Validate port range
	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535, got %d", cfg.Port)
	}

	// Validate game duration
	if cfg.GameDuration <= 0 {
		return nil, fmt.Errorf("game duration must be greater than 0, got %d", cfg.GameDuration)
	}

	// Validate threshold
	if cfg.Threshold < 0 || cfg.Threshold > 100 {
		return nil, fmt.Errorf("threshold must be between 0 and 100, got %d", cfg.Threshold)
	}

	return cfg, nil
}
