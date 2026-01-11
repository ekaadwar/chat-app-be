package ws

import (
	"context"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Client struct {
	UserID uint
	Conn   *websocket.Conn
	Send   chan Event

	ctx    context.Context
	cancel context.CancelFunc
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uint]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: map[uint]map[*Client]struct{}{},
	}
}

func (h *Hub) AddClient(userID uint, conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan Event, 64),
		ctx:    ctx,
		cancel: cancel,
	}

	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*Client]struct{}{}
	}
	h.clients[userID][c] = struct{}{}
	h.mu.Unlock()

	go c.writeLoop()
	go c.keepAliveLoop() // optional ping

	return c
}

func (h *Hub) RemoveClient(c *Client) {
	c.cancel()

	h.mu.Lock()
	defer h.mu.Unlock()

	if set, ok := h.clients[c.UserID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}

	_ = c.Conn.Close(websocket.StatusNormalClosure, "bye")
}

func (h *Hub) BroadcastToUsers(userIDs []uint, ev Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, uid := range userIDs {
		for c := range h.clients[uid] {
			select {
			case c.Send <- ev:
			default:
				// channel penuh → drop (atau bisa kamu ubah jadi strategy lain)
			}
		}
	}
}

func (c *Client) writeLoop() {
	defer close(c.Send)

	for {
		select {
		case <-c.ctx.Done():
			return
		case ev := <-c.Send:
			// Hindari timeout terlalu agresif karena beberapa implementasi deadline bisa menutup koneksi
			// saat ada goroutine read/write aktif.
			writeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = wsjson.Write(writeCtx, c.Conn, ev)
			cancel()
		}
	}
}

func (c *Client) keepAliveLoop() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = c.Conn.Ping(pingCtx)
			cancel()
		}
	}
}
