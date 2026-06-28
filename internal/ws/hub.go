package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	// Allow all origins — protected by JWT token param
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]bool
	log     *zap.Logger
}

func NewHub(log *zap.Logger) *Hub {
	return &Hub{clients: make(map[*client]bool), log: log}
}

// Broadcast sends an event to all connected WebSocket clients.
func (h *Hub) Broadcast(ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// slow client — skip, will be cleaned up
		}
	}
}

// ServeWS upgrades the connection and registers the client.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("ws upgrade failed", zap.Error(err))
		return
	}

	c := &client{conn: conn, send: make(chan []byte, 256)}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	go h.writePump(c)
	h.readPump(c) // blocks until disconnect
}

func (h *Hub) writePump(c *client) {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
	c.conn.Close()
}

func (h *Hub) readPump(c *client) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		close(c.send)
		c.conn.Close()
	}()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}
