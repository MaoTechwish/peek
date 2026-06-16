package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"peek/relay/hub"
)

//go:embed viewer.html
var viewerHTML []byte

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 256,
}

func newHub() *hub.Hub { return hub.New() }

func buildMux(h *hub.Hub) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/watch", handleWatch)
	mux.HandleFunc("/agent", handleAgent(h))
	mux.HandleFunc("/ws", handleViewer(h))
	return mux
}

func main() {
	h := newHub()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("relay listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, buildMux(h)))
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleWatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(viewerHTML)
}

func handleAgent(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			return nil
		})

		h.RegisterAgent(token)
		defer h.UnregisterAgent(token)

		for {
			msgType, frame, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if msgType == websocket.BinaryMessage {
				h.Broadcast(token, frame)
			}
		}
	}
}

func handleViewer(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		if !h.AgentOnline(token) {
			http.Error(w, "host offline", http.StatusServiceUnavailable)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		h.AddViewer(token, conn)
		defer h.RemoveViewer(token, conn)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}
}
