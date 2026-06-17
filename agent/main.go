package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"peek/agent/capture"
	"peek/agent/setup"
	"peek/agent/streamer"
)

// relayURL is embedded at build time: go build -ldflags "-X main.relayURL=wss://peek.maos.dev"
// Override at runtime with the PEEK_RELAY_URL environment variable.
var relayURL = "wss://peek.maos.dev"

func main() {
	if env := os.Getenv("PEEK_RELAY_URL"); env != "" {
		relayURL = env
	}

	if !isElevated() {
		reportElevationRequired()
		os.Exit(1)
	}

	token, err := generateToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to generate token: %v\n", err)
		os.Exit(1)
	}

	viewerURL := toHTTPS(relayURL) + "/watch?token=" + token
	setup.Show(viewerURL)

	c := capture.New()
	s := streamer.New(relayURL, token, c)
	s.Run()
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func toHTTPS(wsURL string) string {
	s := strings.Replace(wsURL, "wss://", "https://", 1)
	return strings.Replace(s, "ws://", "http://", 1)
}
