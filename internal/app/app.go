package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/diegok/pixwar/internal/client"
	"github.com/diegok/pixwar/internal/config"
	"github.com/diegok/pixwar/internal/protocol"
	"github.com/diegok/pixwar/internal/server"
	"github.com/diegok/pixwar/internal/ui"
)

// App is the main application controller
type App struct {
	cfg      *config.Config
	screen   *ui.Screen
	renderer *ui.Renderer
	client   *client.Client
	server   *server.Server

	// State
	inLobby    bool
	inGame     bool
	gameOver   bool
	spectating bool
	lobbyState protocol.LobbyState
	gameState  protocol.GameState
	overState  protocol.GameOverState
	myRank     int
	myScore    int
	watchingID int

	// Control channels
	quit    chan struct{}
	sigChan chan os.Signal
}

// NewApp creates a new application with the given configuration
func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:  cfg,
		quit: make(chan struct{}),
	}
}

// Run is the main entry point for the application
func (a *App) Run() error {
	// Initialize screen
	screen, err := ui.InitScreen()
	if err != nil {
		return fmt.Errorf("failed to initialize screen: %w", err)
	}
	a.screen = screen
	a.renderer = ui.NewRenderer(screen)

	// Set up signal handling
	a.sigChan = make(chan os.Signal, 1)
	signal.Notify(a.sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-a.sigChan
		close(a.quit)
	}()

	// Get terminal dimensions
	w, h := a.screen.Size()

	var runErr error
	if a.cfg.IsServer {
		runErr = a.runServer(w, h)
	} else {
		runErr = a.runClient(w, h)
	}

	a.cleanup()
	return runErr
}

// runServer starts the server and connects as a client
func (a *App) runServer(w, h int) error {
	// Create and start the server
	a.server = server.NewServerWithConfig(a.cfg)
	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Connect to our own server as a client
	addr := fmt.Sprintf("localhost:%d", a.cfg.Port)
	return a.connectAndRun(addr, w, h)
}

// runClient connects to a remote server
func (a *App) runClient(w, h int) error {
	addr := a.cfg.ServerAddr
	// Add port if not specified in the address
	if _, _, err := parseHostPort(addr); err != nil {
		addr = fmt.Sprintf("%s:%d", addr, a.cfg.Port)
	}
	return a.connectAndRun(addr, w, h)
}

// connectAndRun establishes the connection and runs the main loop
func (a *App) connectAndRun(addr string, w, h int) error {
	// Show connecting screen
	a.renderer.RenderConnecting(addr)

	// Determine player name
	name := a.cfg.PlayerName
	if name == "" {
		name = fmt.Sprintf("Player%d", time.Now().UnixNano()%1000)
	}

	// Create and connect client
	// Reserve some space for UI elements
	boardHeight := h - 2
	a.client = client.NewClient(name, w, boardHeight)

	if err := a.client.Connect(addr); err != nil {
		a.renderer.RenderError(fmt.Sprintf("Connection failed: %v", err))
		// Wait for key press before exiting
		a.screen.PollEvent()
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Start in lobby
	a.inLobby = true

	return a.mainLoop()
}

// mainLoop is the main event loop
func (a *App) mainLoop() error {
	// Create event channel from screen
	eventChan := make(chan tcell.Event, 16)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				return
			}
			select {
			case eventChan <- ev:
			case <-a.quit:
				return
			}
		}
	}()

	// Ticker for periodic rendering
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.quit:
			return nil

		case ev := <-eventChan:
			if a.handleEvent(ev) {
				return nil
			}

		case state := <-a.client.LobbyState:
			a.lobbyState = state
			a.render()

		case <-a.client.GameStart:
			a.inLobby = false
			a.inGame = true
			a.gameOver = false
			a.spectating = false
			a.render()

		case state := <-a.client.GameState:
			a.gameState = state
			// Check if we're eliminated
			a.checkElimination()
			a.render()

		case state := <-a.client.GameOver:
			a.overState = state
			a.inGame = false
			a.gameOver = true
			a.spectating = false
			// Find our rank
			a.findMyRank()
			a.render()

		case err := <-a.client.Error:
			a.renderer.RenderError(fmt.Sprintf("Connection error: %v", err))
			a.screen.PollEvent()
			return err

		case <-ticker.C:
			// Periodic render to handle any missed updates
			a.render()
		}
	}
}

// handleEvent processes a tcell event. Returns true to quit.
func (a *App) handleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		// Check for quit
		if ui.IsQuitKey(e.Key(), e.Rune()) {
			return true
		}

		if a.inLobby {
			// In lobby: handle start key if host
			if ui.IsStartKey(e.Key()) && a.lobbyState.IsHost && a.lobbyState.CanStart {
				if a.server != nil {
					a.server.StartGame()
				}
			}
		} else if a.inGame {
			if a.spectating {
				// Spectator mode: Tab to cycle watched player
				if ui.IsTabKey(e.Key()) {
					a.cycleWatching()
				}
			} else {
				// Playing: send direction input
				dir := ui.KeyToDirection(e.Key(), e.Rune())
				if dir != protocol.DirNone {
					a.client.SendInput(dir)
				}
			}
		} else if a.gameOver {
			// Game over: Enter to play again (only if server host)
			if ui.IsStartKey(e.Key()) {
				// For now, just quit - restart would require more complex logic
				return true
			}
		}

	case *tcell.EventResize:
		a.screen.Clear()
		a.render()
	}

	return false
}

// cycleWatching cycles through alive players in spectator mode
func (a *App) cycleWatching() {
	// Find all alive players
	var alivePlayers []int
	for _, p := range a.gameState.Players {
		if p.Alive {
			alivePlayers = append(alivePlayers, p.ID)
		}
	}

	if len(alivePlayers) == 0 {
		return
	}

	// Find current index
	currentIdx := -1
	for i, id := range alivePlayers {
		if id == a.watchingID {
			currentIdx = i
			break
		}
	}

	// Move to next player
	nextIdx := (currentIdx + 1) % len(alivePlayers)
	a.watchingID = alivePlayers[nextIdx]
}

// checkElimination checks if we've been eliminated
func (a *App) checkElimination() {
	if a.spectating {
		return
	}

	myID := a.client.PlayerID
	for _, p := range a.gameState.Players {
		if p.ID == myID {
			if !p.Alive {
				a.spectating = true
				a.myScore = p.Score

				// Calculate rank based on alive players
				aliveCount := 0
				for _, other := range a.gameState.Players {
					if other.Alive {
						aliveCount++
					}
				}
				a.myRank = aliveCount + 1

				// Start watching the first alive player
				for _, other := range a.gameState.Players {
					if other.Alive {
						a.watchingID = other.ID
						break
					}
				}
			}
			break
		}
	}
}

// findMyRank finds our rank in the game over state
func (a *App) findMyRank() {
	myID := a.client.PlayerID
	for _, r := range a.overState.Rankings {
		if r.PlayerID == myID {
			a.myRank = r.Rank
			a.myScore = r.Score
			break
		}
	}
}

// render draws the current state
func (a *App) render() {
	if a.inLobby {
		a.renderer.RenderLobby(a.lobbyState)
	} else if a.gameOver {
		a.renderer.RenderGameOver(a.overState)
	} else if a.inGame {
		if a.spectating {
			a.renderer.RenderSpectator(a.gameState, a.watchingID, a.myRank, a.myScore)
		} else {
			a.renderer.RenderGame(a.gameState, a.client.PlayerID)
		}
	}
}

// cleanup shuts down resources
func (a *App) cleanup() {
	signal.Stop(a.sigChan)

	if a.client != nil {
		a.client.Close()
	}

	if a.server != nil {
		a.server.Stop()
	}

	if a.screen != nil {
		a.screen.Fini()
	}
}

// parseHostPort is a helper to check if an address contains a port
func parseHostPort(addr string) (host, port string, err error) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
		if addr[i] == ']' {
			// IPv6 address without port
			break
		}
	}
	return "", "", fmt.Errorf("no port found")
}
