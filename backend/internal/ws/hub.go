package ws

import (
	"encoding/json"
	"log"
	"sync"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c.UserID] = c
	h.mu.Unlock()
	log.Printf("ws connected: %s", c.UserID)
}

func (h *Hub) Unregister(userID string) {
	h.mu.Lock()
	delete(h.clients, userID)
	h.mu.Unlock()
	log.Printf("ws disconnected: %s", userID)
}

func (h *Hub) Send(to string, msg SignalMessage) bool {
	h.mu.RLock()
	client, ok := h.clients[to]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return false
	}
	select {
	case client.Send <- data:
		return true
	default:
		return false
	}
}
