package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	mux := buildMux(newHub())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestWatchServeHTML(t *testing.T) {
	mux := buildMux(newHub())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watch?token=abc", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
}

func TestAgentEndpointRejectsNoToken(t *testing.T) {
	mux := buildMux(newHub())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agent", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestViewerEndpointRejectsNoToken(t *testing.T) {
	mux := buildMux(newHub())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestViewerEndpointRejectsOfflineAgent(t *testing.T) {
	mux := buildMux(newHub())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws?token=notregistered", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
