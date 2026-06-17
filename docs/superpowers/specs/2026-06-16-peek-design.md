# Peek — Remote Screen Viewer Design

**Date:** 2026-06-16  
**Status:** Approved

---

## Overview

Peek is a lightweight remote screen-viewing tool. A host agent runs silently on a monitored machine and streams screen captures to a relay server. Authorized viewers watch the live feed from any browser over the internet.

---

## Components

### 1. `peek-agent` (Go binary, host machine)

- Cross-platform: macOS Apple Silicon, macOS Intel, Windows
- Requires elevated permissions at launch (UAC on Windows, sudo-equivalent on macOS); exits with a clear error if not elevated
- Process is named `main` via Go build flags and runtime process title override
- Runs entirely in the background with no visible window or tray icon after setup is dismissed
- Single outbound WebSocket connection to the relay (no inbound ports required)
- Relay URL embedded at build time via Go `ldflags`; overridable at runtime via `PEEK_RELAY_URL` environment variable

### 2. `peek-relay` (Go HTTP server, Fly.io free tier)

- Accepts one authenticated WebSocket connection per token from an agent
- Accepts N authenticated WebSocket connections per token from viewers
- Forwards binary JPEG frames from agent → all connected viewers (fan-out)
- Serves the static viewer HTML page
- Validates tokens on connection; rejects with 401 if token is unknown or no agent is connected
- Stateless — all state in-memory, no database

### 3. Viewer (browser)

- Single-page HTML/JS app (~100 lines, no framework) served by the relay
- URL format: `https://<relay-host>/watch?token=XYZ`
- Connects via WebSocket: `wss://<relay-host>/ws?token=XYZ`
- Renders incoming binary JPEG frames by updating an `<img>` src via Blob URL (previous Blob URL revoked each frame to prevent memory leaks)
- Displays "host offline" message when no agent is connected

---

## Startup Flow

1. User launches `peek-agent` with elevated permissions
2. Agent generates a cryptographically random 32-byte token (in-memory only, never written to disk)
3. Agent starts a temporary local HTTP server on a random OS-assigned ephemeral port and opens `http://localhost:<PORT>/setup` in the default browser
4. Setup page displays the full viewer URL (`https://<relay>/watch?token=<TOKEN>`) and a "Start & Dismiss" button
5. User copies the URL, clicks Dismiss — local HTTP server shuts down, browser tab closes
6. Agent connects outbound to relay via WebSocket, authenticates with token
7. Agent runs silently from this point; no UI, no tray icon, no output

---

## Frame Pipeline

Runs on a 100ms tick:

1. Capture full screenshot of primary monitor
2. Downscale captured frame to thumbnail
3. Hash the thumbnail (fast, e.g. FNV-64)
4. Compare hash to previous frame's hash — if identical, discard and wait for next tick
5. If changed: encode full screenshot as JPEG at ~70% quality
6. Send encoded JPEG as a binary WebSocket frame to the relay
7. Relay fans out the frame to all viewers holding a valid connection for that token

**Bandwidth characteristics:**
- Active screen (frequent changes): up to ~10 frames/sec possible, realistic ~3–5fps
- Idle screen (no changes): 0 frames sent
- Typical JPEG at 70% quality for 1080p: ~100–200KB per frame

---

## Error Handling

| Scenario | Behavior |
|---|---|
| Agent loses relay connection | Reconnects with exponential backoff: 1s, 2s, 4s … up to 30s cap, indefinitely |
| Viewer loses relay connection | JS reconnects with same backoff strategy |
| Viewer connects but no agent online | Relay accepts connection, viewer shows "host offline" |
| Viewer provides invalid token | Relay rejects with HTTP 401 before WebSocket upgrade |
| Agent launched without elevation | Exits immediately with a human-readable error message |
| Multiple monitors | Captures primary monitor only |

---

## Shutdown

- Agent stops via process kill, task manager, or shell (`kill <pid>`)
- No graceful shutdown required — relay detects dropped WebSocket and marks token offline
- System restart stops the agent (no auto-start on boot unless user configures it separately)
- Remote stop: not implemented (kept simple per requirements)

---

## Security

- Token is 32 bytes of cryptographic randomness (~256-bit entropy); not stored anywhere
- Token is never logged or written to disk by either agent or relay
- All traffic between agent ↔ relay ↔ viewer is over WSS (TLS provided by Fly.io)
- Relay does not store frames — each frame is forwarded and discarded immediately
- If the token is compromised, the only mitigation is restarting the agent (which generates a new token)

---

## Repository Structure

```
peek/
├── agent/           # peek-agent Go source
│   ├── main.go
│   ├── capture/     # screenshot + change detection
│   ├── relay/       # WebSocket client to relay
│   └── setup/       # temporary local HTTP setup page
├── relay/           # peek-relay Go source
│   ├── main.go
│   ├── hub/         # token registry + fan-out logic
│   └── static/      # viewer HTML/JS
├── fly.toml         # Fly.io deployment config
└── docs/
    └── superpowers/
        └── specs/
            └── 2026-06-16-peek-design.md
```

---

## Deployment

- Relay deployed to Fly.io free tier (shared-CPU VM, 256MB RAM — sufficient for light usage)
- Agent distributed as a single compiled binary per platform:
  - `peek-agent-darwin-arm64`
  - `peek-agent-darwin-amd64`
  - `peek-agent-windows-amd64.exe`
- No installer required — run directly with elevated permissions

---

## Out of Scope

- Audio streaming
- Input forwarding (keyboard/mouse control)
- Multiple simultaneous host agents under one relay account
- Auto-start on boot
- Remote agent shutdown from viewer
