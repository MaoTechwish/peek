package hub

import (
	"sync"
	"testing"
)

type mockSender struct {
	mu       sync.Mutex
	messages [][]byte
	closed   bool
}

func (m *mockSender) WriteMessage(_ int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.messages = append(m.messages, cp)
	return nil
}

func (m *mockSender) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockSender) received() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messages
}

func TestAgentOnline(t *testing.T) {
	h := New()
	if h.AgentOnline("tok1") {
		t.Fatal("agent should not be online before registration")
	}
	h.RegisterAgent("tok1")
	if !h.AgentOnline("tok1") {
		t.Fatal("agent should be online after registration")
	}
	h.UnregisterAgent("tok1")
	if h.AgentOnline("tok1") {
		t.Fatal("agent should not be online after unregistration")
	}
}

func TestBroadcast_deliversToAllViewers(t *testing.T) {
	h := New()
	h.RegisterAgent("tok1")
	v1, v2 := &mockSender{}, &mockSender{}
	h.AddViewer("tok1", v1)
	h.AddViewer("tok1", v2)

	frame := []byte{0xff, 0xd8, 0x01}
	h.Broadcast("tok1", frame)

	if len(v1.received()) != 1 || string(v1.received()[0]) != string(frame) {
		t.Error("v1 did not receive frame")
	}
	if len(v2.received()) != 1 || string(v2.received()[0]) != string(frame) {
		t.Error("v2 did not receive frame")
	}
}

func TestBroadcast_unknownTokenNoPanic(t *testing.T) {
	h := New()
	h.Broadcast("unknown", []byte{1, 2, 3}) // must not panic
}

func TestRemoveViewer(t *testing.T) {
	h := New()
	h.RegisterAgent("tok1")
	v1, v2 := &mockSender{}, &mockSender{}
	h.AddViewer("tok1", v1)
	h.AddViewer("tok1", v2)
	h.RemoveViewer("tok1", v1)

	h.Broadcast("tok1", []byte{1})

	if len(v1.received()) != 0 {
		t.Error("removed viewer should not receive broadcast")
	}
	if len(v2.received()) != 1 {
		t.Error("remaining viewer should receive broadcast")
	}
}

func TestUnregisterAgent_closesViewers(t *testing.T) {
	h := New()
	h.RegisterAgent("tok1")
	v1 := &mockSender{}
	h.AddViewer("tok1", v1)

	h.UnregisterAgent("tok1")

	if !v1.closed {
		t.Error("viewer connection should be closed when agent unregisters")
	}
}
