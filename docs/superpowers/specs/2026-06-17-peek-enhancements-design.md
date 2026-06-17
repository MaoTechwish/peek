# Peek Enhancements — Design

**Date:** 2026-06-17
**Status:** Approved
**Builds on:** [2026-06-16-peek-design.md](2026-06-16-peek-design.md)

---

## Overview

Three enhancements to the working Peek system (agent → relay → browser viewer):

1. **Conditional downscaling** — reduce host CPU/bandwidth on high-resolution displays, with zero added cost on ordinary displays.
2. **Remote shutdown** — let a viewer stop the host agent, guarded by a confirmation popup.
3. **Viewer frame panel + click-to-copy** — a togglable side panel of recent frames, plus click-to-copy on both the thumbnails and the main view.

**Guiding constraint:** nothing may add meaningful recurring compute on the host. Feature 2 adds only a one-shot control message; feature 3 is entirely client-side (runs in the viewer's browser); feature 1 only ever *reduces* host work.

---

## Feature 1 — Conditional downscaling (agent)

**Where:** `agent/capture/capture.go`, in `processFrame`, before JPEG encoding.

**Behavior:**
- After capture, if the frame width or height exceeds a 1080p bounding box (default **1920×1080**), resize down to fit inside the box, preserving aspect ratio.
- If the frame already fits within the box, encode it unchanged (no resize step).
- Resize uses `golang.org/x/image/draw` with the `ApproxBiLinear` kernel — fast, and since the destination is ~1080p, text remains legible.
- The bounding box is a package constant (`maxWidth = 1920`, `maxHeight = 1080`). No runtime config in this version.
- Change detection (the 32×32 grid FNV hash) continues to run on the **original** captured frame, so detection sensitivity is unchanged.

**Compute rationale:**
- ≤1080p displays: strict no-op — no resize, identical to today.
- 1440p/4K/5K displays: JPEG encoding dominates host cost and scales with pixel count. Cutting pixels 2–4× saves far more encode time than the cheap bilinear resize adds, so total host CPU *drops*. Bandwidth drops correspondingly.

**Dependency:** add `golang.org/x/image/draw` to the agent module.

---

## Feature 2 — Remote shutdown (viewer → relay → agent)

Introduces a control channel. Today the agent never reads from the relay, and the relay discards everything viewers send; both change minimally.

### Viewer
- A small, unobtrusive "Shut down host" control.
- Clicking opens a minimalistic confirmation modal: *"Shut down the host machine? This stops Peek."* with Cancel / Shut down buttons.
- On confirm, the viewer sends a JSON **text** frame over its existing `/ws` connection: `{"cmd":"shutdown"}`.

### Relay
- `handleViewer` reads text frames (instead of discarding). On a `shutdown` command, it signals that token's agent.
- The relay sends `{"cmd":"shutdown"}` as a text frame to the agent's connection.
- **Concurrency:** gorilla forbids concurrent writes to one connection. The relay already writes pong frames on the agent socket (via the default ping handler). All writes to the agent socket must be serialized — the hub owns the agent `Sender` and guards writes with a per-agent mutex (or a single writer goroutine). The shutdown send goes through that guard.
- Routing is token-scoped: a shutdown from a viewer only reaches the agent registered under the same token; other tokens' agents are unaffected.

### Agent
- The streamer gains a **reader goroutine** on the relay connection.
- A normal read error / disconnect → existing behavior (close connection, reconnect with exponential backoff).
- A `{"cmd":"shutdown"}` control message → the agent exits via `os.Exit(0)`. Silent: no host-visible message, consistent with the privacy requirements.
- The command-handling logic is factored so the "is this a shutdown command?" decision is unit-testable without terminating the test process.

### Security
- Anyone holding the token can trigger shutdown. This matches the existing trust model — the token already grants full view access, and the host deliberately shares it.
- The confirmation modal exists to prevent accidental clicks (the stated goal), not to defend against a malicious token holder.
- Rejected alternative: closing the agent socket with a custom close code. A close is indistinguishable from a network drop, so the agent would reconnect instead of exiting. An explicit command message is required.

---

## Feature 3 — Viewer frame panel + click-to-copy (client-side only)

All in `relay/viewer.html`. No host or relay cost.

### Rolling frame panel
- A togglable panel along one side (default collapsed; toggle state remembered in `localStorage`). A small toggle button is always visible.
- Holds the **last N = 15 frames received** in a ring buffer (simple "last N", not time-spaced).
- Each arriving frame's blob is pushed; when the buffer overflows, the oldest entry is dropped and its object URL is `revokeObjectURL`'d. Memory stays flat (a few MB).
- Thumbnails render newest-first (most recent at the top of the panel).

### Click-to-copy (shared helper)
- **Left-click a thumbnail** → copy that frame to the clipboard.
- **Left-click the main viewing area** → copy the currently displayed frame to the clipboard.
- Both call one `copyFrame(blob)` helper: draw the JPEG onto an offscreen `<canvas>`, export via `canvas.toBlob('image/png')`, and write it with `navigator.clipboard.write([new ClipboardItem({'image/png': pngBlob})])`. PNG is used because clipboard image support is reliable for PNG, not JPEG. The conversion happens once per click, on the viewer's machine.

### Toast
- A small toast at the bottom shows **"Copied to clipboard"**, auto-dismissing after ~1.5s.
- If `navigator.clipboard` / `ClipboardItem` is unavailable (e.g., non-secure context or older browser), the toast shows **"Copy not supported"** instead of failing silently.

---

## Data Flow

```
Agent:  capture ─► [downscale if > 1080p] ─► hash (original) ─► JPEG ─► WS ─► Relay ─► Viewers
Viewer (render): each frame ─► <img> + push into 15-frame ring buffer (panel thumbnails)
Viewer (copy):  click main view OR thumbnail ─► canvas ─► PNG ─► clipboard ─► toast
Viewer (shutdown): confirm modal ─► {"cmd":"shutdown"} ─► Relay ─► Agent reader ─► os.Exit(0)
```

---

## Error Handling

| Scenario | Behavior |
|---|---|
| Frame larger than 1080p box | Downscaled to fit, aspect preserved |
| Frame within 1080p box | Encoded unchanged (no resize) |
| Viewer sends shutdown, agent online | Agent exits silently; viewers then see existing "Host offline" |
| Viewer sends shutdown, no agent online | No-op on the relay (nothing to signal) |
| Concurrent writes to agent socket | Serialized via per-agent write guard in the hub |
| Clipboard API unavailable | Toast: "Copy not supported"; no crash |
| Frame buffer overflow | Oldest blob dropped and object URL revoked |

---

## Testing

- **Downscale (`agent/capture`):** oversized image → resized to fit within box with aspect preserved; in-bounds image → returned unchanged; output remains a valid JPEG (FF D8 magic). Hash still computed on the original.
- **Shutdown routing (`relay/hub`):** a shutdown request for a token signals that token's agent and not others; safe when no agent is registered; concurrent writes to the agent sender are serialized (race-detector test).
- **Shutdown decision (`agent`):** the control-message parser identifies a `shutdown` command vs. ordinary/garbage messages, tested without exiting the process.
- **Viewer (`relay/viewer.html`):** manual verification (consistent with the existing viewer, which has no JS test harness):
  1. Toggle panel on/off; state persists across reload.
  2. Thumbnails roll as frames arrive; buffer stays at 15.
  3. Click a thumbnail → image on clipboard, toast appears and auto-dismisses.
  4. Click the main view → current frame on clipboard, toast appears.
  5. Shutdown control → confirm modal → confirm → host process exits → viewer shows "Host offline".

---

## Out of Scope (unchanged from V1)

- Audio, input forwarding, multi-monitor, auto-start on boot.
- Platform-native capture (DXGI Desktop Duplication / ScreenCaptureKit) — explicitly considered for Feature 1 and deferred; conditional downscaling delivers the wanted efficiency without platform-specific code.
- Time-spaced or de-duplicated frame history — the panel keeps the literal last N frames.
- Runtime configuration of the downscale cap or panel size (compile-time constants for now).
