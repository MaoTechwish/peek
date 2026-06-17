#!/bin/bash
set -e

RELAY_URL="${RELAY_URL:-wss://peek.maos.dev}"
LDFLAGS="-X main.relayURL=${RELAY_URL} -s -w"

mkdir -p dist

# kbinani/screenshot uses CoreGraphics via cgo on macOS, so CGO must be enabled.
# Each arch must be built on a Mac of that arch (or with a cross C toolchain).
echo "Building agent for macOS ARM64 (relay: ${RELAY_URL})..."
cd agent
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -ldflags "${LDFLAGS}" -o ../dist/main-arm64 .

echo "Building agent for macOS AMD64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "${LDFLAGS}" -o ../dist/main-amd64 .
cd ..

echo "Done. Outputs: dist/main-arm64, dist/main-amd64"
