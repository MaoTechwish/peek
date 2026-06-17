# Peek Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add conditional downscaling (host efficiency), remote shutdown from the viewer, and a togglable recent-frame panel with click-to-copy to the working Peek system.

**Architecture:** Feature 1 inserts a resize step in the agent capture pipeline (agent-local). Feature 2 adds a control channel: viewer sends a JSON command over its existing `/ws` socket → relay routes it to that token's agent socket → the agent's new reader goroutine exits the process. Feature 3 is entirely in `relay/viewer.html` (client-side).

**Tech Stack:** Go 1.26, `github.com/gorilla/websocket`, `golang.org/x/image/draw` (new, agent), `encoding/json` (stdlib), plain HTML/JS canvas + Clipboard API (viewer).

**Spec:** [docs/superpowers/specs/2026-06-17-peek-enhancements-design.md](../specs/2026-06-17-peek-enhancements-design.md)

**Go on this machine:** `go` is not on PATH; prefix commands with `$env:Path = "C:\Program Files\Go\bin;$env:Path"` in PowerShell.

---

## File Map

```
agent/
├── go.mod                       # MODIFY: add golang.org/x/image
├── capture/
│   ├── capture.go               # MODIFY: downscale before JPEG encode
│   └── capture_test.go          # MODIFY: add downscale tests
└── streamer/
    ├── streamer.go              # MODIFY: reader goroutine + shutdown handling
    └── streamer_test.go         # MODIFY: add isShutdownCommand test
relay/
├── hub/
│   ├── hub.go                   # MODIFY: store agent sender, ShutdownAgent()
│   └── hub_test.go              # MODIFY: RegisterAgent signature, shutdown tests
├── main.go                      # MODIFY: register agent conn, parse viewer cmds
├── main_test.go                 # MODIFY: isShutdownCommand test
└── viewer.html                  # REWRITE: panel, copy, toast, shutdown modal
```

---

## Task 1: Feature 1 — Conditional downscaling (agent capture)

**Files:**
- Modify: `agent/go.mod` (add dependency)
- Modify: `agent/capture/capture.go`
- Modify: `agent/capture/capture_test.go`

- [ ] **Step 1: Add the dependency**

Run (PowerShell):
```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\agent; go get golang.org/x/image/draw
```
Expected: `go.mod` gains a `golang.org/x/image` require line.

- [ ] **Step 2: Write the failing tests**

Append to `agent/capture/capture_test.go`:
```go
func TestDownscale_largeImageResizedToFit(t *testing.T) {
	src := solidImage(3840, 2160, color.RGBA{10, 20, 30, 255})
	out := downscale(src)
	b := out.Bounds()
	if b.Dx() > maxWidth || b.Dy() > maxHeight {
		t.Errorf("downscaled image %dx%d exceeds %dx%d box", b.Dx(), b.Dy(), maxWidth, maxHeight)
	}
	// 16:9 source fills the 1920x1080 box exactly.
	if b.Dx() != 1920 || b.Dy() != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestDownscale_smallImageUntouched(t *testing.T) {
	src := solidImage(1280, 720, color.RGBA{1, 2, 3, 255})
	out := downscale(src)
	if out.Bounds() != src.Bounds() {
		t.Errorf("in-bounds image should be untouched, got %v", out.Bounds())
	}
}

func TestDownscale_aspectRatioPreserved(t *testing.T) {
	// Ultrawide 3440x1440 -> limited by width: scale 1920/3440 -> 1920x804.
	src := solidImage(3440, 1440, color.RGBA{9, 9, 9, 255})
	out := downscale(src)
	b := out.Bounds()
	if b.Dx() != 1920 {
		t.Errorf("expected width 1920, got %d", b.Dx())
	}
	if b.Dy() != 804 {
		t.Errorf("expected height 804, got %d", b.Dy())
	}
}
```

- [ ] **Step 3: Run tests to confirm they fail**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\agent; go test ./capture/...
```
Expected: FAIL — `undefined: downscale`, `undefined: maxWidth`, `undefined: maxHeight`.

- [ ] **Step 4: Implement downscaling**

In `agent/capture/capture.go`, update the imports to add `math` and `golang.org/x/image/draw`:
```go
import (
	"bytes"
	"hash/fnv"
	"image"
	"image/jpeg"
	"math"

	"github.com/kbinani/screenshot"
	"golang.org/x/image/draw"
)
```

Add the constants after the imports:
```go
// Frames larger than this bounding box are downscaled before encoding. On
// high-resolution displays this reduces JPEG encode cost and bandwidth; frames
// within the box are encoded unchanged (no added work).
const (
	maxWidth  = 1920
	maxHeight = 1080
)
```

In `processFrame`, downscale before encoding:
```go
func (c *Capture) processFrame(img image.Image, h uint64) []byte {
	if h == c.prevHash {
		return nil
	}
	c.prevHash = h

	img = downscale(img)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return nil
	}
	return buf.Bytes()
}
```

Add the `downscale` function at the end of the file:
```go
// downscale returns img unchanged when it fits within maxWidth x maxHeight;
// otherwise it returns a resized copy that fits the box, preserving aspect ratio.
func downscale(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxWidth && h <= maxHeight {
		return img
	}
	scale := math.Min(float64(maxWidth)/float64(w), float64(maxHeight)/float64(h))
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}
```

- [ ] **Step 5: Run tests to confirm they pass**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\agent; go test ./capture/... -v
```
Expected: all capture tests PASS (the three new ones plus the existing five).

- [ ] **Step 6: Tidy modules and commit**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\agent; go mod tidy
```
Then from repo root:
```bash
git add agent/go.mod agent/go.sum agent/capture/capture.go agent/capture/capture_test.go
git commit -m "feat: downscale agent frames larger than 1080p before encoding"
```

---

## Task 2: Feature 2 — Relay hub shutdown routing

**Files:**
- Modify: `relay/hub/hub.go`
- Modify: `relay/hub/hub_test.go`

The hub must keep the agent's `Sender` (not just a bool) so the relay can write a shutdown command to it, with writes serialized per agent.

- [ ] **Step 1: Update existing tests for the new RegisterAgent signature and add shutdown tests**

In `relay/hub/hub_test.go`, change every `h.RegisterAgent("tok1")` call to `h.RegisterAgent("tok1", &mockSender{})`. There are four such calls (in `TestAgentOnline`, `TestBroadcast_deliversToAllViewers`, `TestRemoveViewer`, `TestUnregisterAgent_closesViewers`).

Then append these tests:
```go
func TestShutdownAgent_sendsCommandToAgent(t *testing.T) {
	h := New()
	a := &mockSender{}
	h.RegisterAgent("tok1", a)

	h.ShutdownAgent("tok1")

	msgs := a.received()
	if len(msgs) != 1 || string(msgs[0]) != `{"cmd":"shutdown"}` {
		t.Errorf("agent should receive one shutdown command, got %v", msgs)
	}
}

func TestShutdownAgent_unknownTokenNoPanic(t *testing.T) {
	h := New()
	h.ShutdownAgent("nobody") // must not panic
}

func TestShutdownAgent_concurrent(t *testing.T) {
	h := New()
	a := &mockSender{}
	h.RegisterAgent("tok1", a)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.ShutdownAgent("tok1")
		}()
	}
	wg.Wait()

	if len(a.received()) != 10 {
		t.Errorf("expected 10 shutdown sends, got %d", len(a.received()))
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\relay; go test ./hub/...
```
Expected: FAIL — `RegisterAgent` now wrong arity / `undefined: (*Hub).ShutdownAgent`.

- [ ] **Step 3: Rewrite hub.go**

Replace the entire contents of `relay/hub/hub.go` with:
```go
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
	defer h.mu.Unlock()
	delete(h.agents, token)
	for _, v := range h.viewers[token] {
		v.Close()
	}
	delete(h.viewers, token)
}

func (h *Hub) AgentOnline(token string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.agents[token] != nil
}

// ShutdownAgent sends a shutdown command to the agent registered for token, if any.
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
	for i, c := range conns {
		if c == s {
			h.viewers[token] = append(conns[:i], conns[i+1:]...)
			return
		}
	}
}

func (h *Hub) Broadcast(token string, frame []byte) {
	h.mu.RLock()
	conns := make([]Sender, len(h.viewers[token]))
	copy(conns, h.viewers[token])
	h.mu.RUnlock()

	for _, c := range conns {
		_ = c.WriteMessage(2, frame) // 2 = websocket.BinaryMessage
	}
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\relay; go test ./hub/... -v
```
Expected: all hub tests PASS.

- [ ] **Step 5: Commit**

```bash
git add relay/hub/hub.go relay/hub/hub_test.go
git commit -m "feat: hub stores agent sender and routes shutdown commands"
```

---

## Task 3: Feature 2 — Relay viewer command handling

**Files:**
- Modify: `relay/main.go`
- Modify: `relay/main_test.go`

- [ ] **Step 1: Write the failing test**

Append to `relay/main_test.go`:
```go
func TestIsShutdownCommand(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`{"cmd":"shutdown"}`, true},
		{`{"cmd":"other"}`, false},
		{`{"foo":"bar"}`, false},
		{`not json`, false},
		{``, false},
	}
	for _, c := range cases {
		if got := isShutdownCommand([]byte(c.in)); got != c.want {
			t.Errorf("isShutdownCommand(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\relay; go test . -run TestIsShutdownCommand
```
Expected: FAIL — `undefined: isShutdownCommand`.

- [ ] **Step 3: Implement command parsing and wiring**

In `relay/main.go`, add `"encoding/json"` to the imports (keep the existing imports).

Register the agent connection so the hub can reach it — change the line in `handleAgent`:
```go
	h.RegisterAgent(token, conn)
```
(was `h.RegisterAgent(token)`).

Replace the viewer read loop in `handleViewer` (the `for { _, _, err := conn.ReadMessage() ... }` block) with:
```go
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if msgType == websocket.TextMessage && isShutdownCommand(data) {
				h.ShutdownAgent(token)
			}
		}
```

Add this helper at the end of `relay/main.go`:
```go
func isShutdownCommand(data []byte) bool {
	var msg struct {
		Cmd string `json:"cmd"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return false
	}
	return msg.Cmd == "shutdown"
}
```

- [ ] **Step 4: Run the full relay test suite**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\relay; go test ./... ; go vet ./...
```
Expected: all PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add relay/main.go relay/main_test.go
git commit -m "feat: relay registers agent conn and forwards viewer shutdown command"
```

---

## Task 4: Feature 2 — Agent shutdown handling

**Files:**
- Modify: `agent/streamer/streamer.go`
- Modify: `agent/streamer/streamer_test.go`

- [ ] **Step 1: Write the failing test**

Append to `agent/streamer/streamer_test.go`:
```go
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
```

- [ ] **Step 2: Run test to confirm it fails**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\agent; go test ./streamer/... -run TestIsShutdownCommand
```
Expected: FAIL — `undefined: isShutdownCommand`.

- [ ] **Step 3: Implement reader goroutine and shutdown handling**

In `agent/streamer/streamer.go`, update imports to add `encoding/json` and `os`:
```go
import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)
```

Add an `onShutdown` field to the struct and set a default in `New`:
```go
type Streamer struct {
	relayURL   string
	token      string
	capture    Capturer
	backoff    time.Duration
	onShutdown func()
}

func New(relayURL, token string, c Capturer) *Streamer {
	return &Streamer{
		relayURL:   relayURL,
		token:      token,
		capture:    c,
		onShutdown: func() { os.Exit(0) },
	}
}
```

In `connect`, after `defer conn.Close()` and before the tickers, add the reader goroutine:
```go
	// Reader goroutine: the relay may send a shutdown command. A read error
	// (normal disconnect) closes the conn so the writer loop returns and
	// reconnects; a shutdown command exits the process.
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				return
			}
			if isShutdownCommand(data) {
				s.onShutdown()
				return
			}
		}
	}()
```

Add the helper at the end of the file:
```go
func isShutdownCommand(data []byte) bool {
	var msg struct {
		Cmd string `json:"cmd"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return false
	}
	return msg.Cmd == "shutdown"
}
```

- [ ] **Step 4: Run the full agent test suite**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\agent; go test ./... ; go vet ./...
```
Expected: all PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add agent/streamer/streamer.go agent/streamer/streamer_test.go
git commit -m "feat: agent reader goroutine exits on relay shutdown command"
```

---

## Task 5: Feature 3 — Viewer panel, click-to-copy, shutdown modal

**Files:**
- Rewrite: `relay/viewer.html`

No automated tests (consistent with the existing viewer). Manual verification is in Task 6.

- [ ] **Step 1: Replace viewer.html**

Replace the entire contents of `relay/viewer.html` with:
```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Peek</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  html, body { height: 100%; }
  body { background: #111; color: #aaa; font-family: monospace; font-size: 14px; display: flex; }
  #main { flex: 1; display: flex; align-items: center; justify-content: center; position: relative; overflow: hidden; }
  #screen { max-width: 100%; max-height: 100vh; object-fit: contain; display: none; cursor: copy; }
  #status { position: absolute; }
  #panel { width: 180px; background: #181818; border-left: 1px solid #2a2a2a; overflow-y: auto; padding: 8px; display: flex; flex-direction: column; gap: 8px; }
  #panel.hidden { display: none; }
  #panel img { width: 100%; display: block; border: 1px solid #2a2a2a; border-radius: 3px; cursor: copy; }
  #panel img:hover { border-color: #4a90d9; }
  #toggle { position: fixed; top: 10px; right: 10px; z-index: 5; background: #222; color: #ccc; border: 1px solid #3a3a3a; border-radius: 5px; padding: 6px 10px; cursor: pointer; }
  #shutdown { position: fixed; top: 10px; right: 64px; z-index: 5; background: #2a1414; color: #e08; border: 1px solid #5a2a2a; border-radius: 5px; padding: 6px 10px; cursor: pointer; }
  #toggle:hover { background: #2c2c2c; }
  #shutdown:hover { background: #3a1a1a; }
  #toast { position: fixed; bottom: 18px; left: 50%; transform: translateX(-50%); background: #000; color: #fff; padding: 8px 16px; border-radius: 6px; opacity: 0; transition: opacity .2s; pointer-events: none; z-index: 10; }
  #toast.show { opacity: .92; }
  #modal { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: none; align-items: center; justify-content: center; z-index: 20; }
  #modal.show { display: flex; }
  #modal .box { background: #1c1c1c; border: 1px solid #3a3a3a; border-radius: 8px; padding: 22px; max-width: 320px; text-align: center; }
  #modal p { color: #ddd; margin-bottom: 18px; line-height: 1.4; }
  #modal .btns { display: flex; gap: 10px; justify-content: center; }
  #modal button { padding: 8px 18px; border-radius: 6px; border: none; cursor: pointer; font-size: 14px; }
  #cancel { background: #333; color: #ddd; }
  #confirm { background: #c0244a; color: #fff; }
</style>
</head>
<body>
<div id="main">
  <img id="screen" alt="remote screen" title="Click to copy this frame">
  <div id="status">Connecting…</div>
</div>
<aside id="panel" class="hidden"></aside>
<button id="toggle" title="Toggle recent frames">▦</button>
<button id="shutdown" title="Shut down host">⏻</button>
<div id="toast"></div>
<div id="modal">
  <div class="box">
    <p>Shut down the host machine? This stops Peek.</p>
    <div class="btns">
      <button id="cancel">Cancel</button>
      <button id="confirm">Shut down</button>
    </div>
  </div>
</div>
<script>
const token = new URLSearchParams(location.search).get('token');
const screen = document.getElementById('screen');
const status = document.getElementById('status');
const panel = document.getElementById('panel');
const toggle = document.getElementById('toggle');
const toastEl = document.getElementById('toast');
const modal = document.getElementById('modal');

const MAX_FRAMES = 15;
let frames = []; // newest first: { blob, url }
let ws = null;

// ---- Panel toggle (persisted) ----
if (localStorage.getItem('peek_panel') === 'open') panel.classList.remove('hidden');
toggle.onclick = () => {
  panel.classList.toggle('hidden');
  localStorage.setItem('peek_panel', panel.classList.contains('hidden') ? 'closed' : 'open');
};

// ---- Toast ----
let toastTimer = null;
function showToast(msg) {
  toastEl.textContent = msg;
  toastEl.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toastEl.classList.remove('show'), 1500);
}

// ---- Copy a JPEG blob to the clipboard as PNG ----
function jpegToPng(blob) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const u = URL.createObjectURL(blob);
    img.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = img.naturalWidth;
      canvas.height = img.naturalHeight;
      canvas.getContext('2d').drawImage(img, 0, 0);
      URL.revokeObjectURL(u);
      canvas.toBlob(b => b ? resolve(b) : reject(new Error('encode failed')), 'image/png');
    };
    img.onerror = () => { URL.revokeObjectURL(u); reject(new Error('decode failed')); };
    img.src = u;
  });
}
async function copyFrame(blob) {
  if (!navigator.clipboard || !window.ClipboardItem) { showToast('Copy not supported'); return; }
  try {
    const png = await jpegToPng(blob);
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': png })]);
    showToast('Copied to clipboard');
  } catch (e) {
    showToast('Copy failed');
  }
}

// ---- Thumbnail panel ----
function renderThumbs() {
  panel.innerHTML = '';
  frames.forEach(f => {
    const img = new Image();
    img.src = f.url;
    img.title = 'Click to copy this frame';
    img.onclick = () => copyFrame(f.blob);
    panel.appendChild(img);
  });
}

// Main view: click copies the current (newest) frame.
screen.onclick = () => { if (frames.length) copyFrame(frames[0].blob); };

// ---- Shutdown modal ----
document.getElementById('shutdown').onclick = () => modal.classList.add('show');
document.getElementById('cancel').onclick = () => modal.classList.remove('show');
document.getElementById('confirm').onclick = () => {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ cmd: 'shutdown' }));
  modal.classList.remove('show');
};

// ---- WebSocket feed ----
if (!token) {
  status.textContent = 'Invalid link — no token.';
} else {
  let retryDelay = 1000;

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${proto}//${location.host}/ws?token=${token}`);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => { retryDelay = 1000; };

    ws.onmessage = (e) => {
      const blob = new Blob([e.data], { type: 'image/jpeg' });
      const url = URL.createObjectURL(blob);
      screen.src = url;
      screen.style.display = 'block';
      status.style.display = 'none';

      frames.unshift({ blob, url });
      while (frames.length > MAX_FRAMES) {
        URL.revokeObjectURL(frames.pop().url);
      }
      renderThumbs();
    };

    ws.onclose = () => {
      screen.style.display = 'none';
      status.style.display = 'block';
      status.textContent = `Host offline — reconnecting in ${retryDelay / 1000}s…`;
      setTimeout(connect, retryDelay);
      retryDelay = Math.min(retryDelay * 2, 30000);
    };
  }

  connect();
}
</script>
</body>
</html>
```

- [ ] **Step 2: Verify the relay still builds (embeds the new HTML)**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek\relay; go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add relay/viewer.html
git commit -m "feat: viewer recent-frame panel, click-to-copy, and shutdown modal"
```

---

## Task 6: Build, deploy, and end-to-end verification

**Files:** none (build/deploy/verify only)

- [ ] **Step 1: Build the agent**

```
$env:Path = "C:\Program Files\Go\bin;$env:Path"; Set-Location C:\mao\repo\peek; .\build.ps1
```
Expected: `dist\main.exe` rebuilt with no errors.

- [ ] **Step 2: Redeploy the relay**

The relay Go code (shutdown routing) and `viewer.html` changed, so the deployed relay must be rebuilt. On the Oracle VM:
```bash
cd peek && git pull && docker compose up -d --build
```
Then confirm health:
```bash
curl https://peek.maos.dev/healthz   # 200
```

- [ ] **Step 2 (alt, local): smoke-test the relay in Docker locally**

If verifying before deploy, from repo root build the relay image and hit `/watch`. Otherwise proceed to Step 3 against the deployed relay.

- [ ] **Step 3: End-to-end manual verification**

1. Run `dist\main.exe` as administrator. Copy the viewer URL from the setup page, dismiss.
2. Open the viewer URL. Confirm the live feed renders.
3. **Downscale:** on a >1080p display, confirm the feed still looks legible (text readable). On a ≤1080p display, confirm unchanged behavior.
4. **Panel:** click ▦ — panel shows recent frames, fills to 15 and rolls. Reload — panel open/closed state persists.
5. **Copy (thumbnail):** click a thumbnail — toast "Copied to clipboard"; paste into an image editor to confirm.
6. **Copy (main):** click the main image — toast appears; paste confirms the current frame.
7. **Shutdown:** click ⏻ — modal appears. Cancel does nothing. Click again → Shut down → the agent process exits (check Task Manager: no `main` process) and the viewer shows "Host offline".

- [ ] **Step 4: Final commit (if any deploy notes/docs changed)**

No code changes expected here; if deploy docs were updated, commit them.

---

## Self-Review

- [x] **Feature 1 (downscale)** — Task 1: `downscale` + constants + tests; no-op within box, resize above it.
- [x] **Feature 2 (remote shutdown)** — Task 2 (hub routing + concurrency), Task 3 (relay viewer cmd parse + agent registration), Task 4 (agent reader goroutine + exit).
- [x] **Feature 3 (panel + copy + main-view copy + toggle + toast)** — Task 5: full viewer.html.
- [x] **Shutdown confirmation modal** — Task 5: `#modal`, cancel/confirm wiring.
- [x] **Clipboard-unavailable fallback** — Task 5: `copyFrame` shows "Copy not supported".
- [x] **Memory management (revoke rolled-off URLs)** — Task 5: `URL.revokeObjectURL(frames.pop().url)`.
- [x] **Type/name consistency** — `RegisterAgent(token, Sender)` used in hub, hub_test, and relay/main; `isShutdownCommand` defined in both relay/main and agent/streamer (separate modules, intentional duplication); `ShutdownAgent`, `onShutdown`, `downscale`, `maxWidth`/`maxHeight` consistent across tasks.
- [x] **No placeholders** — every code step shows complete code.
```
