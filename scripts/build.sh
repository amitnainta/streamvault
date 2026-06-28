#!/usr/bin/env bash
# Full production build: frontend → backend (embeds frontend) → single binary
set -e

echo "→ Building frontend..."
cd web && npm ci && npm run build
cd ..

echo "→ Building backend (embedding frontend)..."
CGO_ENABLED=1 go build \
  -ldflags="-s -w" \
  -o streamvault \
  ./cmd/streamvault

echo "✓ Build complete: ./streamvault"
