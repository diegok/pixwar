package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/diegok/pixwar/internal/app"
	"github.com/diegok/pixwar/internal/config"
)

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}

	if cfg.IsServer {
		showServerInfo(cfg.Port)
	}

	application := app.NewApp(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  pixwar --server [options]       Start a game server")
	fmt.Fprintln(os.Stderr, "  pixwar --join <address>         Join a game server")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  --port <port>       Server port (default: 7777)")
	fmt.Fprintln(os.Stderr, "  --name <name>       Player name")
	fmt.Fprintln(os.Stderr, "  --time <minutes>    Game duration (default: 5)")
	fmt.Fprintln(os.Stderr, "  --threshold <pct>   Territory % to win (default: 95)")
	fmt.Fprintln(os.Stderr, "  --powerups          Enable power-ups")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  pixwar --server --name Host")
	fmt.Fprintln(os.Stderr, "  pixwar --join 192.168.1.100 --name Player2")
	fmt.Fprintln(os.Stderr, "  pixwar --join localhost:7777 --name TestPlayer")
}

func showServerInfo(port int) {
	fmt.Printf("Starting PixWar server on port %d\n", port)
	fmt.Println("Players can connect using:")
	fmt.Println("")

	// Get local IP addresses
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Printf("  pixwar --join localhost:%d\n", port)
		return
	}

	for _, addr := range addrs {
		// Check if it's an IP network address
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		// Skip loopback and IPv6 link-local
		ip := ipNet.IP
		if ip.IsLoopback() {
			continue
		}
		if ip.To4() == nil {
			// Skip IPv6 for simplicity
			continue
		}

		fmt.Printf("  pixwar --join %s:%d\n", ip.String(), port)
	}

	fmt.Printf("  pixwar --join localhost:%d  (same machine)\n", port)
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println("")
}
