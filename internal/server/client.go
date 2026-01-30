package server

import (
	"net"
	"sync"

	"github.com/diegok/pixwar/internal/protocol"
)

const sendBufferSize = 64

// Client represents a connected client
type Client struct {
	ID       int
	Name     string
	PlayerID int
	Conn     net.Conn
	Codec    *protocol.Codec
	Width    int
	Height   int
	mu       sync.Mutex
	sendCh   chan *protocol.Message
	done     chan struct{}
}

// NewClient creates a new client with the given ID and connection
func NewClient(id int, conn net.Conn) *Client {
	return &Client{
		ID:       id,
		PlayerID: -1,
		Conn:     conn,
		Codec:    protocol.NewCodec(conn),
		sendCh:   make(chan *protocol.Message, sendBufferSize),
		done:     make(chan struct{}),
	}
}

// Send queues a message to be sent to the client (non-blocking)
// If the send channel is full, the message is dropped
func (c *Client) Send(msg *protocol.Message) {
	select {
	case c.sendCh <- msg:
	default:
		// Channel full, drop message
	}
}

// SendDirect sends a message directly to the client (blocking)
func (c *Client) SendDirect(msg *protocol.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Codec.Encode(msg)
}

// Close closes the client connection and signals the writer to stop
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		// Already closed
		return
	default:
		close(c.done)
	}
	c.Conn.Close()
}

// StartWriter starts the goroutine that sends messages from the send channel
func (c *Client) StartWriter() {
	go func() {
		for {
			select {
			case <-c.done:
				return
			case msg := <-c.sendCh:
				c.mu.Lock()
				err := c.Codec.Encode(msg)
				c.mu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()
}
