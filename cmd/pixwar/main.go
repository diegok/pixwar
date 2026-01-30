package main

import (
	"fmt"
	"os"

	"github.com/diegok/pixwar/internal/config"
)

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Usage: pixwar --server [options] | pixwar --join <address>\n")
		os.Exit(1)
	}

	if cfg.IsServer {
		fmt.Printf("Starting server on port %d\n", cfg.Port)
	} else {
		fmt.Printf("Connecting to %s:%d\n", cfg.ServerAddr, cfg.Port)
	}
}
