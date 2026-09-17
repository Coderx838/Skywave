package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/skywave-chat/skywave/pkg/protocol"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536 // 64KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin terminal/web clients
	},
}

// Client represents a single connected terminal session.
type Client struct {
	ID        string
	Nickname  string
	Hub       *Hub
	Conn      *websocket.Conn
	SendChan  chan protocol.Packet
	Limiter   *RateLimiter
	CurrentRoom string
	mu        sync.Mutex
	isClosed  bool
}

// NewClient creates a new client session.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:       uuid.New().String()[:8],
		Hub:      hub,
		Conn:     conn,
		SendChan: make(chan protocol.Packet, 128),
		Limiter:  NewRateLimiter(4.0, 10.0), // 4 messages/sec, burst 10
	}
}

// Send enqueues a packet to the client write channel safely.
func (c *Client) Send(p protocol.Packet) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isClosed {
		return
	}
	select {
	case c.SendChan <- p:
	default:
		log.Printf("[Warn] Client %s buffer full, dropping packet", c.ID)
	}
}

// Close gracefully closes the client connection.
func (c *Client) Close() {
	c.mu.Lock()
	if !c.isClosed {
		c.isClosed = true
		close(c.SendChan)
		c.Conn.Close()
	}
	c.mu.Unlock()
}

// ReadPump handles incoming WebSocket frames from the client.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Info] Client %s closed connection: %v", c.ID, err)
			}
			break
		}

		// Rate limiting check
		if !c.Limiter.Allow() {
			c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
				Code:    "RATE_LIMITED",
				Message: "You are sending messages too quickly. Slow down!",
			}))
			continue
		}

		var packet protocol.Packet
		if err := json.Unmarshal(message, &packet); err != nil {
			c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
				Code:    "INVALID_PACKET",
				Message: "Malformed JSON payload",
			}))
			continue
		}

		c.Hub.HandlePacket(c, packet)
	}
}

// WritePump writes queued packets to the client WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case packet, ok := <-c.SendChan:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			if err := json.NewEncoder(w).Encode(packet); err != nil {
				return
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
