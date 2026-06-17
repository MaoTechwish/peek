package hub

import "sync"

// Sender is satisfied by *websocket.Conn in production and mockSender in tests.
type Sender interface {
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// agent holds an agent's connection plus a mutex that serializes writes to it,
// since gorilla forbids concurrent writers on one connection.
type agent struct {
	sender Sender
	mu     sync.Mutex
}

type Hub struct {
	mu      sync.RWMutex
	agents  map[string]*agent
	viewers map[string][]Sender
}

func New() *Hub {
	return &Hub{
		agents:  make(map[string]*agent),
		viewers: make(map[string][]Sender),
	}
}

func (h *Hub) RegisterAgent(token string, s Sender) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agents[token] = &agent{sender: s}
}

func (h *Hub) UnregisterAgent(token string) {
	h.mu.Lock()
	viewers := h.viewers[token]
	delete(h.agents, token)
	delete(h.viewers, token)
	h.mu.Unlock()

	for _, v := range viewers {
		v.Close()
	}
}

func (h *Hub) AgentOnline(token string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.agents[token] != nil
}

// ShutdownAgent sends a shutdown command to the agent registered for token, if any.
// Writes are serialized per agent because the relay may also write pongs on the
// same connection.
func (h *Hub) ShutdownAgent(token string) {
	h.mu.RLock()
	a := h.agents[token]
	h.mu.RUnlock()
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_ = a.sender.WriteMessage(1, []byte(`{"cmd":"shutdown"}`)) // 1 = websocket.TextMessage
}

func (h *Hub) AddViewer(token string, s Sender) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.viewers[token] = append(h.viewers[token], s)
}

func (h *Hub) RemoveViewer(token string, s Sender) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.viewers[token]
	fresh := make([]Sender, 0, len(conns))
	for _, c := range conns {
		if c != s {
			fresh = append(fresh, c)
		}
	}
	h.viewers[token] = fresh
}

func (h *Hub) Broadcast(token string, frame []byte) {
	h.mu.RLock()
	conns := make([]Sender, len(h.viewers[token]))
	copy(conns, h.viewers[token])
	h.mu.RUnlock()

	for _, c := range conns {
		if err := c.WriteMessage(2, frame); err != nil { // 2 = websocket.BinaryMessage
			c.Close()
			h.RemoveViewer(token, c)
		}
	}
}
