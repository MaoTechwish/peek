# Peek

A lightweight screen-sharing agent. The **agent** runs on a host machine, captures
the primary display, and streams frames to a **relay**; anyone with the generated
viewer link can watch the host's screen live in a browser.

The agent is a single self-contained binary named `main`, runs in the background,
and requires administrator privileges to start.

- `agent/` — the host-side capture/stream binary (macOS + Windows)
- `relay/` — the WebSocket relay + browser viewer
- `docs/` — design specs and deployment notes

---

## Run on macOS (Intel / x86_64)

These steps cover the Intel build (`main-amd64`). For Apple Silicon use `main-arm64`
— the steps are otherwise identical.

### 1. Get the binary

Download `main-amd64` from the **Build macOS Agent** GitHub Actions run
(artifact `peek-agent-darwin-amd64`), or build it yourself (see *Building* below).
The relay URL `wss://peek.maos.dev` is baked into the CI build; override it at
runtime with `PEEK_RELAY_URL` if you run your own relay.

### 2. Clear the quarantine flag and make it executable

A binary downloaded through a browser is quarantined by Gatekeeper, and the GitHub
artifact zip does **not** preserve the executable bit — so both of these are needed:

```bash
cd ~/Downloads          # wherever you saved it
xattr -d com.apple.quarantine main-amd64   # ignore "No such xattr" if it prints
chmod +x main-amd64
```

If you skip the `xattr` step, macOS blocks the binary with *"cannot be opened because
the developer cannot be verified."* (The binary is unsigned; you can also allow it
via System Settings → Privacy & Security → *Open Anyway*.)

### 3. Run it with sudo

The agent refuses to start unless it is running as root (`uid 0`) — it exits with
`peek requires administrator privileges (run with sudo)` otherwise:

```bash
sudo ./main-amd64
```

To point it at a different relay, pass the env var through sudo:

```bash
sudo PEEK_RELAY_URL=wss://your-relay.example.com ./main-amd64
```

### 4. Grant Screen Recording permission

On the first capture attempt macOS requests **Screen Recording** access. Until it is
granted, the live feed stays blank. Grant it under:

**System Settings → Privacy & Security → Screen Recording**

Add (and enable) **Terminal** — the app you launched the agent from — then quit and
re-run `sudo ./main-amd64`. macOS requires the parent app to be restarted after the
permission is granted.

### 5. Share the viewer link

On start, the agent opens a local browser tab titled **"Peek is ready"** showing the
viewer URL. Click **Copy link** to share it, then **Start & Dismiss** to begin
streaming. If a browser can't be opened, the URL is printed to the terminal instead.

Anyone who opens that link sees the host's screen, refreshed roughly twice a second.

### Stopping the agent

- `Ctrl-C` in the terminal, or
- find the `main` process in Activity Monitor / `sudo pkill -f main-amd64`, or
- restart the machine.

---

## Building

The agent's screen capture uses CoreGraphics through cgo on macOS, so a darwin build
needs `CGO_ENABLED=1` and a C toolchain (Xcode command-line tools). It must be built
on a Mac, or cross-compiled on a macOS runner.

**On a Mac** (builds both architectures):

```bash
./build.sh                 # → dist/main-arm64 and dist/main-amd64
```

**Via GitHub Actions** (no Mac required): run the **Build macOS Agent** workflow
(`.github/workflows/build-agent-macos.yml`). It cross-compiles `darwin/amd64` on a
free Apple Silicon runner and uploads `main-amd64` as an artifact. `macos-13` (Intel)
hosted runners are retired, and the remaining Intel labels are paid large runners, so
the workflow targets a standard `macos-15` runner and relies on macOS's universal
clang/SDK to emit an x86_64 binary.

**Windows** (no cgo required):

```powershell
.\build.ps1                # → dist\main.exe
```
