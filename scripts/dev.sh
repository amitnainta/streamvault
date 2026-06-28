#!/usr/bin/env bash
# Start backend + frontend in dev mode with hot reload.
# Requires: Go 1.22+, Node 20+, FFmpeg in PATH

set -e

# Start Go backend (with air for hot reload if available)
if command -v air &>/dev/null; then
  echo "Starting backend with air (hot reload)..."
  air &
else
  echo "Starting backend (no hot reload — install 'air' for hot reload)..."
  go run ./cmd/streamvault &
fi

BACKEND_PID=$!

# Start Vite dev server
echo "Starting frontend dev server..."
cd web && npm run dev &
FRONTEND_PID=$!

# Cleanup on exit
trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null" EXIT

wait
