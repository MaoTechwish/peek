package streamer

import (
	"sync/atomic"
	"testing"
	"time"
)

type mockCapturer struct {
	calls  atomic.Int32
	frames [][]byte // frames to return in sequence, then nil
}

func (m *mockCapturer) Frame() ([]byte, error) {
	i := int(m.calls.Add(1)) - 1
	if i < len(m.frames) {
		return m.frames[i], nil
	}
	return nil, nil
}

func TestIsShutdownCommand(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`{"cmd":"shutdown"}`, true},
		{`{"cmd":"other"}`, false},
		{`garbage`, false},
		{``, false},
	}
	for _, c := range cases {
		if got := isShutdownCommand([]byte(c.in)); got != c.want {
			t.Errorf("isShutdownCommand(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestBackoff_resetsAfterSuccessfulConnect(t *testing.T) {
	s := &Streamer{}
	s.resetBackoff()
	if s.backoff != time.Second {
		t.Errorf("initial backoff should be 1s, got %v", s.backoff)
	}
}

func TestBackoff_doublesOnFailure(t *testing.T) {
	s := &Streamer{}
	s.resetBackoff()
	d1 := s.nextBackoff()
	d2 := s.nextBackoff()
	d3 := s.nextBackoff()
	if d1 != time.Second {
		t.Errorf("first backoff should be 1s, got %v", d1)
	}
	if d2 != 2*time.Second {
		t.Errorf("second backoff should be 2s, got %v", d2)
	}
	if d3 != 4*time.Second {
		t.Errorf("third backoff should be 4s, got %v", d3)
	}
}

func TestBackoff_capsAt30s(t *testing.T) {
	s := &Streamer{}
	s.resetBackoff()
	var d time.Duration
	for i := 0; i < 20; i++ {
		d = s.nextBackoff()
	}
	if d > 30*time.Second {
		t.Errorf("backoff should cap at 30s, got %v", d)
	}
}
