package config

import (
	"errors"
	"flag"
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
		Port:         7777,
		GameDuration: 5,
		Threshold:    95,
	}

	fs := flag.NewFlagSet("pixwar", flag.ContinueOnError)
	fs.BoolVar(&cfg.IsServer, "server", false, "Run as server")
	fs.StringVar(&cfg.ServerAddr, "join", "", "Server address to join")
	fs.IntVar(&cfg.Port, "port", 7777, "Server port")
	fs.IntVar(&cfg.GameDuration, "time", 5, "Game duration in minutes")
	fs.IntVar(&cfg.Threshold, "threshold", 95, "Territory percentage to end game")
	fs.BoolVar(&cfg.PowerupsEnabled, "powerups", false, "Enable power-ups")
	fs.StringVar(&cfg.PlayerName, "name", "", "Player name")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if !cfg.IsServer && cfg.ServerAddr == "" {
		return nil, errors.New("must specify --server or --join <address>")
	}

	return cfg, nil
}
