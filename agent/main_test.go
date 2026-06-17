package main

import (
	"encoding/hex"
	"testing"
)

func TestGenerateToken_length(t *testing.T) {
	tok, err := generateToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := hex.DecodeString(tok)
	if err != nil {
		t.Fatalf("token is not valid hex: %v", err)
	}
	if len(b) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(b))
	}
}

func TestGenerateToken_uniqueness(t *testing.T) {
	tok1, _ := generateToken()
	tok2, _ := generateToken()
	if tok1 == tok2 {
		t.Error("two tokens should not be equal")
	}
}

func TestToHTTPS(t *testing.T) {
	if got := toHTTPS("wss://peek.maos.dev"); got != "https://peek.maos.dev" {
		t.Errorf("wss should map to https, got %s", got)
	}
	if got := toHTTPS("ws://localhost:8080"); got != "http://localhost:8080" {
		t.Errorf("ws should map to http, got %s", got)
	}
}
