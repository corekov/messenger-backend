package ws

import (
	"sync"
)

type Hub struct {
	clients    map[string]map[*Client]bool
	mu         sync.RWMutex
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMsg
}

type BroadcastMsg struct {
	UserIDs []string
	Payload []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan *BroadcastMsg, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			if h.clients[c.UserID] == nil {
				h.clients[c.UserID] = make(map[*Client]bool)
			}
			h.clients[c.UserID][c] = true
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[c.UserID]; ok {
				delete(conns, c)
				if len(conns) == 0 {
					delete(h.clients, c.UserID)
				}
			}
			h.mu.Unlock()
			// безопасно закрыть канал
			select {
			case <-c.Send:
			default:
			}
			close(c.Send)

		case msg := <-h.broadcast:
			h.mu.RLock()
			for _, uid := range msg.UserIDs {
				for c := range h.clients[uid] {
					select {
					case c.Send <- msg.Payload:
					default:
						close(c.Send)
						delete(h.clients[uid], c)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) SendToUsers(userIDs []string, payload []byte) {
	h.broadcast <- &BroadcastMsg{UserIDs: userIDs, Payload: payload}
}

func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
