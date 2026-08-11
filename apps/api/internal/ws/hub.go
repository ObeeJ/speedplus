package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/speedplus/api/internal/middleware"
)

// OrderChecker verifies whether a user is a participant (customer or driver) of an order.
// Injected into Hub to enforce ownership on order channel subscriptions.
type OrderChecker interface {
	IsOrderParticipant(ctx context.Context, orderID uuid.UUID, userID string) bool
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
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
	ctx      context.Context // request context for ownership checks
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{} // channel → clients
	rdb     *redis.Client
	orders  OrderChecker // nil = order channel subscriptions denied
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients: make(map[string]map[*client]struct{}),
		rdb:     rdb,
	}
}

// InjectOrderChecker wires the ownership checker after construction to avoid import cycles.
func (h *Hub) InjectOrderChecker(c OrderChecker) { h.orders = c }

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
			// ctx is valid for the connection lifetime: readPump blocks the handler
			// goroutine until the WS closes, so the request context is not cancelled
			// prematurely. If readPump is ever made async, switch to context.WithoutCancel.
			ctx: c.Request.Context(),
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
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(msg)
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *client) readPump(h *Hub, userID string) {
	defer func() {
		h.unsubscribe(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("ws read error", "user_id", userID, "error", err)
			}
			break
		}
		var req struct {
			Action  string `json:"action"`
			Channel string `json:"channel"`
		}
		if json.Unmarshal(msg, &req) == nil && req.Action == "subscribe" {
			if h.isAllowedChannel(c.ctx, req.Channel, userID) {
				h.subscribe(c, req.Channel)
			}
		}
	}
}

func (h *Hub) isAllowedChannel(ctx context.Context, channel, userID string) bool {
	if strings.HasPrefix(channel, "user:") {
		return strings.TrimPrefix(channel, "user:") == userID
	}
	if strings.HasPrefix(channel, "order:") {
		orderID, err := uuid.Parse(strings.TrimPrefix(channel, "order:"))
		if err != nil {
			return false
		}
		if h.orders == nil {
			return false
		}
		return h.orders.IsOrderParticipant(ctx, orderID, userID)
	}
	return false
}
