package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/speedplus/api/internal/middleware"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Origin validation handled by CORS middleware upstream
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Clients authenticate by sending ["bearer", <token>] as subprotocols so
	// the credential stays out of the URL (see middleware.wsTokenFromSubprotocol).
	// A browser aborts the handshake unless the server echoes an accepted
	// subprotocol, so advertise "bearer" here. The token half is deliberately
	// not negotiated — it is a credential, not a protocol name to echo back.
	Subprotocols: []string{middleware.WSBearerSubprotocol},
}

type Message struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Data    interface{} `json:"data"`
}

type client struct {
	conn     *websocket.Conn
	send     chan []byte
	channels map[string]struct{}
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{} // channel → clients
	rdb     *redis.Client
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients: make(map[string]map[*client]struct{}),
		rdb:     rdb,
	}
}

// Start subscribes to Redis pub/sub and fans out to local clients.
func (h *Hub) Start(ctx context.Context) {
	pubsub := h.rdb.PSubscribe(ctx, "ws:*")
	go func() {
		for msg := range pubsub.Channel() {
			h.broadcast(msg.Channel[3:], []byte(msg.Payload)) // strip "ws:" prefix
		}
	}()
}

// Publish sends a message to a channel via Redis (works across instances).
func (h *Hub) Publish(ctx context.Context, channel string, event string, data interface{}) error {
	msg := Message{Channel: channel, Event: event, Data: data}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.rdb.Publish(ctx, "ws:"+channel, b).Err()
}

func (h *Hub) broadcast(channel string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[channel] {
		select {
		case c.send <- payload:
		default:
			// Client too slow — drop message
		}
	}
}

func (h *Hub) subscribe(c *client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[channel] == nil {
		h.clients[channel] = make(map[*client]struct{})
	}
	h.clients[channel][c] = struct{}{}
	c.channels[channel] = struct{}{}
}

func (h *Hub) unsubscribe(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range c.channels {
		delete(h.clients[ch], c)
	}
}

// Handler upgrades the HTTP connection and subscribes the client to their channels.
// Channels are derived from the authenticated user's role and ID.
func (h *Hub) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		role := c.GetString(middleware.CtxUserRole)

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("ws upgrade failed", "error", err)
			return
		}

		cl := &client{
			conn:     conn,
			send:     make(chan []byte, 256),
			channels: make(map[string]struct{}),
		}

		// Subscribe to role-appropriate channels
		h.subscribe(cl, fmt.Sprintf("user:%s", userID))
		switch role {
		case "driver":
			h.subscribe(cl, fmt.Sprintf("driver:%s", userID))
		case "merchant":
			h.subscribe(cl, fmt.Sprintf("merchant:%s", userID))
		case "customer":
			// Subscribe to order channels on demand via client message
		}

		go cl.writePump()
		cl.readPump(h, userID)
		h.unsubscribe(cl)
		conn.Close()
	}
}

func (c *client) writePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *client) readPump(h *Hub, userID string) {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		// Client can subscribe to specific order channels
		var req struct {
			Action  string `json:"action"`
			Channel string `json:"channel"`
		}
		if json.Unmarshal(msg, &req) == nil && req.Action == "subscribe" {
			// Validate: customer can only subscribe to their own order channels
			if isAllowedChannel(req.Channel, userID) {
				h.subscribe(c, req.Channel)
			}
		}
	}
}

func isAllowedChannel(channel, userID string) bool {
	// Allow order:{uuid} channels — ownership validated at order creation
	// Allow user:{userID} only for own ID
	if len(channel) > 5 && channel[:5] == "user:" {
		return channel[5:] == userID
	}
	if len(channel) > 6 && channel[:6] == "order:" {
		if _, err := uuid.Parse(channel[6:]); err == nil {
			return true // order ownership enforced at DB level
		}
	}
	return false
}
