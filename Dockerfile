# ── Stage 1: Build frontend ──────────────────────────────────────────────────
FROM node:20-alpine AS frontend
WORKDIR /build/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── Stage 2: Build backend (embeds frontend) ─────────────────────────────────
FROM golang:1.22-alpine AS backend
WORKDIR /build

# CGO needed for sqlite3
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY --from=frontend /build/web/dist ./web/dist
COPY . .

RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w" \
    -o streamvault \
    ./cmd/streamvault

# ── Stage 3: Runtime image ────────────────────────────────────────────────────
FROM linuxserver/ffmpeg:latest

COPY --from=backend /build/streamvault /usr/local/bin/streamvault

ENV SV_SERVER_PORT=8096 \
    SV_DATABASE_TYPE=sqlite \
    SV_DATABASE_URL=/config/streamvault.db \
    SV_STORAGE_DATA_DIR=/config

EXPOSE 8096

VOLUME ["/config", "/media"]

ENTRYPOINT ["streamvault"]
