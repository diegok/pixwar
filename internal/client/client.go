package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/diegok/pixwar/internal/protocol"
)

// Client represents a connection to the game server
type Client struct {
	Name         string
	Width        int
	Height       int
	PlayerID     int
	conn         net.Conn
	codec        *protocol.Codec
	mu           sync.Mutex
	connected    bool
	GameState    chan protocol.GameState
	LobbyState   chan protocol.LobbyState
	GameOver     chan protocol.GameOverState
	RematchState chan protocol.RematchState
	Countdown    chan protocol.Countdown
	GameStart    chan struct{}
	Error        chan error
	done         chan struct{}
}

const (
	channelBufferSize = 16
	connectTimeout    = 5 * time.Second
)

// NewClient creates a new client with the given name and terminal dimensions
func NewClient(name string, width, height int) *Client {
	return &Client{
		Name:         name,
		Width:        width,
		Height:       height,
		PlayerID:     -1,
		GameState:    make(chan protocol.GameState, channelBufferSize),
		LobbyState:   make(chan protocol.LobbyState, channelBufferSize),
		GameOver:     make(chan protocol.GameOverState, channelBufferSize),
		RematchState: make(chan protocol.RematchState, channelBufferSize),
		Countdown:    make(chan protocol.Countdown, channelBufferSize),
		GameStart:    make(chan struct{}, 1),
		Error:        make(chan error, channelBufferSize),
		done:         make(chan struct{}),
	}
}

// Connect establishes a connection to the server and performs the join handshake
func (c *Client) Connect(addr string) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.mu.Unlock()

	// Connect with timeout
	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.codec = protocol.NewCodec(conn)
	c.mu.Unlock()

	// Send join request
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     c.Name,
			TerminalWidth:  c.Width,
			TerminalHeight: c.Height,
		},
	}

	if err := c.codec.Encode(joinReq); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send join request: %w", err)
	}

	// Wait for join response with timeout
	conn.SetReadDeadline(time.Now().Add(connectTimeout))
	msg, err := c.codec.Decode()
	conn.SetReadDeadline(time.Time{}) // Clear deadline

	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to receive join response: %w", err)
	}

	if msg.Type != protocol.MsgJoinResponse {
		conn.Close()
		return fmt.Errorf("unexpected response type: %v", msg.Type)
	}

	response, ok := msg.Payload.(protocol.JoinResponse)
	if !ok {
		conn.Close()
		return fmt.Errorf("invalid join response payload")
	}

	if !response.Accepted {
		conn.Close()
		return fmt.Errorf("join rejected: %s", response.Reason)
	}

	c.mu.Lock()
	c.PlayerID = response.PlayerID
	c.connected = true
	c.mu.Unlock()

	// Start receive loop
	go c.receiveLoop()

	return nil
}

// SendInput sends a direction input to the server
func (c *Client) SendInput(dir protocol.Direction) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	msg := &protocol.Message{
		Type: protocol.MsgPlayerInput,
		Payload: protocol.PlayerInput{
			Direction: dir,
		},
	}

	return c.codec.Encode(msg)
}

// SendRematchReady signals the server that this client is ready for rematch
func (c *Client) SendRematchReady() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	msg := &protocol.Message{
		Type:    protocol.MsgRematchReady,
		Payload: nil,
	}

	return c.codec.Encode(msg)
}

// Close closes the connection to the server
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	select {
	case <-c.done:
		// Already closed
		return
	default:
		close(c.done)
	}

	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected returns whether the client is connected to a server
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// receiveLoop reads messages from the server and dispatches them to channels
func (c *Client) receiveLoop() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		msg, err := c.codec.Decode()
		if err != nil {
			select {
			case <-c.done:
				return
			case c.Error <- fmt.Errorf("receive error: %w", err):
			default:
			}
			return
		}

		c.dispatchMessage(msg)
	}
}

// dispatchMessage sends a received message to the appropriate channel
func (c *Client) dispatchMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgGameState:
		if state, ok := msg.Payload.(protocol.GameState); ok {
			select {
			case c.GameState <- state:
			default:
				// Channel full, drop oldest and add new
				select {
				case <-c.GameState:
				default:
				}
				select {
				case c.GameState <- state:
				default:
				}
			}
		}

	case protocol.MsgLobbyState:
		if state, ok := msg.Payload.(protocol.LobbyState); ok {
			select {
			case c.LobbyState <- state:
			default:
				// Channel full, drop oldest and add new
				select {
				case <-c.LobbyState:
				default:
				}
				select {
				case c.LobbyState <- state:
				default:
				}
			}
		}

	case protocol.MsgGameOver:
		if state, ok := msg.Payload.(protocol.GameOverState); ok {
			select {
			case c.GameOver <- state:
			default:
			}
		}

	case protocol.MsgStartGame:
		select {
		case c.GameStart <- struct{}{}:
		default:
		}

	case protocol.MsgRematchState:
		if state, ok := msg.Payload.(protocol.RematchState); ok {
			select {
			case c.RematchState <- state:
			default:
				// Channel full, drop oldest and add new
				select {
				case <-c.RematchState:
				default:
				}
				select {
				case c.RematchState <- state:
				default:
				}
			}
		}

	case protocol.MsgCountdown:
		if countdown, ok := msg.Payload.(protocol.Countdown); ok {
			select {
			case c.Countdown <- countdown:
			default:
				// Channel full, drop oldest and add new
				select {
				case <-c.Countdown:
				default:
				}
				select {
				case c.Countdown <- countdown:
				default:
				}
			}
		}
	}
}
