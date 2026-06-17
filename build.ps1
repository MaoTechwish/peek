param(
  [string]$RelayURL = "wss://peek.maos.dev"
)

# -H=windowsgui builds a GUI-subsystem binary: no console window is allocated and
# the process is not tied to a terminal, so Peek runs fully in the background.
$ldflags = "-H=windowsgui -X main.relayURL=$RelayURL -s -w"
New-Item -ItemType Directory -Force -Path dist | Out-Null

Write-Host "Building agent for Windows (relay: $RelayURL)..."
Set-Location agent
# kbinani/screenshot uses pure-Go Win32 syscalls, so CGO is not required on Windows.
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
go build -ldflags $ldflags -o ..\dist\main.exe .
Set-Location ..

Write-Host "Done. Output: dist\main.exe"
