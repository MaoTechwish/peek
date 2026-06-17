package streamer

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Capturer is satisfied by *capture.Capture in production.
type Capturer interface {
	Frame() ([]byte, error)
}

type Streamer struct {
	relayURL string
	token    string
	capture  Capturer
	backoff  time.Duration
}

func New(relayURL, token string, c Capturer) *Streamer {
	return &Streamer{relayURL: relayURL, token: token, capture: c}
}

// Run loops forever, connecting to the relay and streaming frames.
// It blocks; call from a goroutine or as the last statement in main.
func (s *Streamer) Run() {
	s.resetBackoff()
	for {
		if s.connect() {
			s.resetBackoff()
		} else {
			d := s.nextBackoff()
			log.Printf("relay connect failed, retrying in %v", d)
			time.Sleep(d)
		}
	}
}

func (s *Streamer) connect() (connected bool) {
	url := s.relayURL + "/agent?token=" + s.token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return false
	}
	defer conn.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-ticker.C:
			frame, err := s.capture.Frame()
			if err != nil || frame == nil {
				continue
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				return true
			}
		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return true
			}
		}
	}
}

func (s *Streamer) resetBackoff() {
	s.backoff = time.Second
}

func (s *Streamer) nextBackoff() time.Duration {
	d := s.backoff
	s.backoff *= 2
	if s.backoff > 30*time.Second {
		s.backoff = 30 * time.Second
	}
	return d
}
