# StreamVault — Architecture Document

**Version:** 0.1
**Date:** 2026-06-28
**Status:** Draft
**Companion:** [PRD v0.2](PRD.md)

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Repository Structure](#2-repository-structure)
3. [Backend Architecture](#3-backend-architecture)
   - [Package Design](#31-package-design)
   - [Request Lifecycle](#32-request-lifecycle)
   - [Privacy Gate](#33-privacy-gate)
   - [Configuration System](#34-configuration-system)
4. [Database Architecture](#4-database-architecture)
   - [Schema Design](#41-schema-design)
   - [Migration Strategy](#42-migration-strategy)
   - [SQLite vs PostgreSQL](#43-sqlite-vs-postgresql)
5. [Media Library & Scanner](#5-media-library--scanner)
   - [Directory Watcher](#51-directory-watcher)
   - [Scanner Pipeline](#52-scanner-pipeline)
   - [File Fingerprinting](#53-file-fingerprinting)
6. [Metadata System](#6-metadata-system)
   - [Metadata Pipeline](#61-metadata-pipeline)
   - [Local NFO Support](#62-local-nfo-support)
   - [Artwork Storage](#63-artwork-storage)
7. [Transcoding Engine](#7-transcoding-engine)
   - [Decision Tree](#71-decision-tree)
   - [FFmpeg Pipeline](#72-ffmpeg-pipeline)
   - [Hardware Acceleration](#73-hardware-acceleration)
   - [Session Management](#74-session-management)
8. [Streaming Architecture](#8-streaming-architecture)
   - [HLS Pipeline](#81-hls-pipeline)
   - [Direct Play](#82-direct-play)
9. [Authentication & Authorization](#9-authentication--authorization)
   - [JWT Design](#91-jwt-design)
   - [RBAC Model](#92-rbac-model)
10. [Frontend Architecture](#10-frontend-architecture)
    - [Component Structure](#101-component-structure)
    - [State Management](#102-state-management)
    - [Player Architecture](#103-player-architecture)
11. [Task Scheduler](#11-task-scheduler)
12. [WebSocket Event System](#12-websocket-event-system)
13. [Deployment Architecture](#13-deployment-architecture)
    - [Single Binary](#131-single-binary)
    - [Docker](#132-docker)
    - [Data Directory Layout](#133-data-directory-layout)
14. [Security Architecture](#14-security-architecture)
15. [Key Interfaces & Contracts](#15-key-interfaces--contracts)

---

## 1. System Overview

StreamVault is a **monolithic, single-binary server** that embeds the React frontend as static assets. It exposes a REST API and WebSocket endpoint consumed by the web UI and any third-party clients.

```
┌─────────────────────────────────────────────────────────────────────┐
│                          USER DEVICES                               │
│                                                                     │
│   Browser / Mobile Browser        Smart TV / 3rd-party client      │
│   (React SPA served locally)      (REST API + HLS stream)          │
└───────────────┬─────────────────────────────┬───────────────────────┘
                │  HTTP/HTTPS + WebSocket      │  HTTP/HTTPS + HLS
                └──────────────┬──────────────┘
                               │
                    ┌──────────▼──────────┐
                    │   StreamVault       │  Single Go binary
                    │   :8096             │  (API + UI + streams)
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
    ┌─────────▼──────┐ ┌───────▼──────┐ ┌──────▼────────┐
    │  PostgreSQL /  │ │  Local File  │ │  FFmpeg       │
    │  SQLite        │ │  System      │ │  (subprocess) │
    └────────────────┘ └──────────────┘ └───────────────┘
                                │
                    (only when internet enabled)
                               │
              ┌────────────────┼──────────────┐
              │                │              │
         ┌────▼───┐    ┌───────▼──┐    ┌─────▼───┐
         │  TMDB  │    │MusicBrainz│    │ GitHub  │  ...
         └────────┘    └──────────┘    └─────────┘
```

### Design Constraints

| Constraint | Rationale |
|-----------|-----------|
| Single binary | Simplest possible self-hosted deployment |
| No external services at startup | App starts and runs fully offline |
| All outbound HTTP through Privacy Gate | Architecturally enforce the privacy model |
| FFmpeg as separate process | Avoids CGO; FFmpeg crashes don't take down the server |
| Embedded frontend | No separate web server; no CDN dependency |

---

## 2. Repository Structure

```
streamvault/
├── cmd/
│   └── streamvault/
│       └── main.go              # Entry point — wires everything together
│
├── internal/                    # Private packages (not importable externally)
│   ├── api/                     # HTTP handlers and routing
│   │   ├── router.go            # Route registration
│   │   ├── middleware/
│   │   │   ├── auth.go          # JWT validation middleware
│   │   │   ├── cors.go
│   │   │   └── ratelimit.go
│   │   └── handlers/
│   │       ├── auth.go
│   │       ├── libraries.go
│   │       ├── items.go
│   │       ├── stream.go
│   │       ├── users.go
│   │       ├── tasks.go
│   │       └── settings.go
│   │
│   ├── auth/                    # Authentication & authorization
│   │   ├── jwt.go               # Token issue / validate
│   │   ├── password.go          # bcrypt helpers
│   │   └── rbac.go              # Role checks
│   │
│   ├── config/                  # Configuration loading & validation
│   │   ├── config.go            # Config struct
│   │   └── loader.go            # YAML + env var loading
│   │
│   ├── db/                      # Database layer
│   │   ├── db.go                # DB connection setup (SQLite / PostgreSQL)
│   │   ├── migrations/          # golang-migrate SQL files
│   │   │   ├── 000001_init.up.sql
│   │   │   ├── 000001_init.down.sql
│   │   │   └── ...
│   │   └── query/               # sqlc-generated type-safe query code
│   │       ├── models.go
│   │       ├── items.sql.go
│   │       ├── users.sql.go
│   │       └── ...
│   │
│   ├── library/                 # Media library management
│   │   ├── scanner.go           # Directory scanner
│   │   ├── watcher.go           # fsnotify file watcher
│   │   ├── fingerprint.go       # File hashing / identification
│   │   ├── matcher.go           # Title/year extraction from filenames
│   │   └── organizer.go         # Movie vs TV vs Music classification
│   │
│   ├── metadata/                # Metadata enrichment
│   │   ├── provider.go          # Provider interface
│   │   ├── tmdb/
│   │   │   ├── client.go        # TMDB API client (uses OutboundClient)
│   │   │   └── mapper.go        # TMDB response → internal model
│   │   ├── musicbrainz/
│   │   │   └── client.go
│   │   ├── nfo/
│   │   │   └── parser.go        # Local .nfo file reader
│   │   └── artwork/
│   │       ├── downloader.go    # Image fetch + local store
│   │       └── server.go        # Serve artwork from local disk
│   │
│   ├── privacy/                 # Privacy Gate — ALL outbound HTTP goes here
│   │   ├── gate.go              # OutboundClient: toggle checks + logging
│   │   ├── settings.go          # Privacy settings model + persistence
│   │   └── activitylog.go       # Network activity log storage
│   │
│   ├── transcode/               # Transcoding engine
│   │   ├── engine.go            # Main transcoding coordinator
│   │   ├── ffmpeg.go            # FFmpeg subprocess wrapper
│   │   ├── hwaccel.go           # Hardware acceleration detection
│   │   ├── profiles.go          # Codec/container profiles per client
│   │   └── session.go           # Active transcode session management
│   │
│   ├── stream/                  # Streaming delivery
│   │   ├── hls.go               # HLS manifest + segment serving
│   │   ├── direct.go            # Direct play handler
│   │   └── negotiate.go         # Direct play vs transcode decision
│   │
│   ├── scheduler/               # Background task runner
│   │   ├── scheduler.go         # Cron-style task scheduler
│   │   └── tasks/
│   │       ├── scan.go          # Library scan task
│   │       ├── thumbnail.go     # Thumbnail generation task
│   │       └── cleanup.go       # Temp file cleanup task
│   │
│   ├── ws/                      # WebSocket event bus
│   │   ├── hub.go               # Connection hub
│   │   └── events.go            # Event type definitions
│   │
│   └── model/                   # Shared domain models (not DB-specific)
│       ├── media.go
│       ├── user.go
│       └── library.go
│
├── web/                         # React frontend source
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── api/                 # API client (typed fetch wrappers)
│   │   ├── components/
│   │   │   ├── ui/              # shadcn/ui base components
│   │   │   ├── layout/
│   │   │   ├── player/
│   │   │   ├── library/
│   │   │   └── admin/
│   │   ├── pages/
│   │   │   ├── Home.tsx
│   │   │   ├── Library.tsx
│   │   │   ├── Item.tsx
│   │   │   ├── Player.tsx
│   │   │   └── admin/
│   │   │       ├── Dashboard.tsx
│   │   │       ├── Privacy.tsx   # F0 Privacy & Internet settings
│   │   │       ├── Libraries.tsx
│   │   │       └── Users.tsx
│   │   ├── store/               # Zustand stores
│   │   └── hooks/               # Custom React hooks
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
│
├── docs/
│   ├── PRD.md
│   ├── ARCHITECTURE.md          # This document
│   ├── API.md                   # OpenAPI spec (generated)
│   └── reverse-proxy/
│       ├── nginx.conf
│       ├── Caddyfile
│       └── traefik.yml
│
├── scripts/
│   ├── build.sh                 # Full build: frontend + backend + embed
│   └── dev.sh                   # Dev mode: hot-reload frontend + backend
│
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── sqlc.yaml                    # sqlc configuration
└── .golangci.yml                # Linter config (includes OutboundClient rule)
```

---

## 3. Backend Architecture

### 3.1 Package Design

The backend follows a **layered architecture** with strict dependency rules:

```
cmd/streamvault/main.go
        │
        ▼
   internal/api          ← HTTP layer: receives requests, calls services
        │
        ▼
   internal/{service}    ← Business logic: library, metadata, transcode, auth
        │
        ▼
   internal/db           ← Data access: sqlc-generated queries
        │
        ▼
   PostgreSQL / SQLite
```

**Dependency rules (enforced by package visibility):**
- `api/handlers` may import any service package
- Service packages (`library`, `metadata`, `transcode`, `stream`, `auth`) may import `db` and `model`
- Service packages must NOT import each other directly — communicate via interfaces or `model` types
- `db` package may NOT import any service package
- All outbound HTTP calls must import `privacy` — never `net/http` directly

### 3.2 Request Lifecycle

```
Client Request (HTTP)
        │
        ▼
┌───────────────────┐
│  Chi Router       │  Route matching
└────────┬──────────┘
         │
┌────────▼──────────┐
│  Middleware Chain │
│  1. CORS          │
│  2. Rate Limiter  │
│  3. Auth (JWT)    │  → 401 if invalid token
│  4. RBAC          │  → 403 if insufficient role
└────────┬──────────┘
         │
┌────────▼──────────┐
│  Handler          │  Decode request, call service, encode response
└────────┬──────────┘
         │
┌────────▼──────────┐
│  Service          │  Business logic, validation
└────────┬──────────┘
         │
┌────────▼──────────┐
│  DB Query (sqlc)  │  Type-safe parameterized SQL
└────────┬──────────┘
         │
┌────────▼──────────┐
│  PostgreSQL/SQLite│
└───────────────────┘
```

**Error handling convention:**

```go
// Services return typed errors
type NotFoundError struct{ ID string }
type ValidationError struct{ Field, Message string }
type ForbiddenError struct{}

// Handlers map error types to HTTP status codes
func handleError(w http.ResponseWriter, err error) {
    switch {
    case errors.As(err, &NotFoundError{}):    writeJSON(w, 404, ...)
    case errors.As(err, &ValidationError{}):  writeJSON(w, 400, ...)
    case errors.As(err, &ForbiddenError{}):   writeJSON(w, 403, ...)
    default:                                   writeJSON(w, 500, ...)
    }
}
```

### 3.3 Privacy Gate

The Privacy Gate is the most critical architectural component. It is the **only** place in the codebase allowed to make outbound HTTP calls.

```go
// internal/privacy/gate.go

// Feature constants — each maps to a settings toggle
type Feature string

const (
    FeatureTMDBMetadata    Feature = "tmdb_metadata"
    FeatureTMDBArtwork     Feature = "tmdb_artwork"
    FeatureMusicBrainz     Feature = "musicbrainz"
    FeatureCoverArt        Feature = "cover_art"
    FeatureUpdateCheck     Feature = "update_check"
    FeatureLetsEncrypt     Feature = "lets_encrypt"
    FeatureCrashReporting  Feature = "crash_reporting"
    FeatureUsageStats      Feature = "usage_stats"
)

// OutboundClient is the only HTTP client allowed in the codebase.
// All other packages must use this instead of net/http directly.
type OutboundClient struct {
    settings   SettingsReader
    activityLog ActivityLogger
    httpClient *http.Client
}

func (c *OutboundClient) Do(ctx context.Context, feature Feature, req *http.Request) (*http.Response, error) {
    // 1. Check master toggle
    if !c.settings.InternetEnabled() {
        c.activityLog.Record(ActivityEntry{
            Feature: feature,
            URL:     req.URL.String(),
            Blocked: true,
            Reason:  "master internet toggle is OFF",
        })
        return nil, ErrInternetDisabled
    }

    // 2. Check feature-specific toggle
    if !c.settings.FeatureEnabled(feature) {
        c.activityLog.Record(ActivityEntry{
            Feature: feature,
            URL:     req.URL.String(),
            Blocked: true,
            Reason:  "feature toggle is OFF",
        })
        return nil, ErrFeatureDisabled{Feature: feature}
    }

    // 3. Log the outbound attempt
    c.activityLog.Record(ActivityEntry{
        Feature:   feature,
        URL:       req.URL.String(),
        Blocked:   false,
        Timestamp: time.Now(),
    })

    // 4. Execute
    return c.httpClient.Do(req.WithContext(ctx))
}
```

**Linter rule** (`.golangci.yml`) to enforce no direct `http.Client` usage:

```yaml
linters-settings:
  forbidigo:
    forbid:
      - pattern: 'http\.DefaultClient'
        msg: "Use privacy.OutboundClient instead of http.DefaultClient"
      - pattern: '&http\.Client{'
        msg: "Use privacy.OutboundClient instead of creating http.Client directly"
      - pattern: 'http\.Get\('
        msg: "Use privacy.OutboundClient instead of http.Get"
      - pattern: 'http\.Post\('
        msg: "Use privacy.OutboundClient instead of http.Post"
```

### 3.4 Configuration System

Config is loaded from (in priority order, highest first):
1. Environment variables (`SV_` prefix)
2. Config file (`/config/streamvault.yaml`)
3. Defaults

```go
// internal/config/config.go

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Storage  StorageConfig
    Privacy  PrivacyConfig
    Log      LogConfig
}

type ServerConfig struct {
    Host        string        `yaml:"host" env:"SV_HOST" default:"0.0.0.0"`
    Port        int           `yaml:"port" env:"SV_PORT" default:"8096"`
    BaseURL     string        `yaml:"base_url" env:"SV_BASE_URL" default:""`
    TLSCertFile string        `yaml:"tls_cert" env:"SV_TLS_CERT"`
    TLSKeyFile  string        `yaml:"tls_key" env:"SV_TLS_KEY"`
    ReadTimeout time.Duration `yaml:"read_timeout" default:"30s"`
}

type DatabaseConfig struct {
    Type string `yaml:"type" env:"SV_DB_TYPE" default:"sqlite"` // "sqlite" | "postgres"
    URL  string `yaml:"url"  env:"SV_DB_URL"`                    // DSN
    // SQLite default: /config/streamvault.db
}

type StorageConfig struct {
    DataDir    string `yaml:"data_dir" env:"SV_DATA_DIR" default:"/config"`
    // Subdirs created automatically:
    // /config/artwork/    — downloaded poster/backdrop images
    // /config/transcode/  — temporary HLS segment files
    // /config/logs/       — server logs
    // /config/db/         — SQLite database (if using SQLite)
}

type PrivacyConfig struct {
    // These are the persisted defaults loaded at startup.
    // Runtime state is stored in DB and takes precedence.
    InternetEnabledDefault bool `yaml:"internet_enabled" default:"false"`
}
```

---

## 4. Database Architecture

### 4.1 Schema Design

#### Core Tables

```sql
-- migrations/000001_init.up.sql

-- ─────────────────────────────────────────────
-- LIBRARIES
-- ─────────────────────────────────────────────
CREATE TABLE libraries (
    id          TEXT PRIMARY KEY,          -- UUID
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,             -- 'movie' | 'show' | 'music'
    paths       TEXT NOT NULL,             -- JSON array of directory paths
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_scan   TIMESTAMP
);

-- ─────────────────────────────────────────────
-- MEDIA ITEMS
-- ─────────────────────────────────────────────
CREATE TABLE media_items (
    id              TEXT PRIMARY KEY,       -- UUID
    library_id      TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    type            TEXT NOT NULL,          -- 'movie' | 'episode' | 'track'
    file_path       TEXT NOT NULL UNIQUE,   -- absolute path on server
    file_size       BIGINT NOT NULL,
    file_hash       TEXT,                   -- SHA-256 of first 64KB (fast fingerprint)
    duration_ms     BIGINT,                 -- milliseconds
    added_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- video properties
    video_codec     TEXT,                   -- 'h264' | 'hevc' | 'av1' | ...
    video_width     INTEGER,
    video_height    INTEGER,
    video_bitrate   BIGINT,
    hdr_format      TEXT,                   -- 'hdr10' | 'dolby_vision' | NULL

    -- audio properties (primary stream)
    audio_codec     TEXT,                   -- 'aac' | 'ac3' | 'dts' | ...
    audio_channels  INTEGER,
    audio_language  TEXT,

    -- container
    container       TEXT                    -- 'mkv' | 'mp4' | 'avi' | ...
);

CREATE INDEX idx_media_items_library ON media_items(library_id);
CREATE INDEX idx_media_items_type    ON media_items(type);

-- ─────────────────────────────────────────────
-- METADATA
-- ─────────────────────────────────────────────
CREATE TABLE metadata (
    id              TEXT PRIMARY KEY,       -- same as media_items.id
    title           TEXT NOT NULL,
    sort_title      TEXT,                   -- for alphabetical sort (e.g., "Dark Knight, The")
    original_title  TEXT,
    year            INTEGER,
    description     TEXT,
    tagline         TEXT,
    genres          TEXT,                   -- JSON array: ["Action","Drama"]
    studios         TEXT,                   -- JSON array
    rating          REAL,                   -- 0.0–10.0
    content_rating  TEXT,                   -- 'PG-13' | 'R' | 'TV-MA' | ...
    language        TEXT,                   -- ISO 639-1
    country         TEXT,

    -- external IDs (nullable — only set if internet was used)
    tmdb_id         INTEGER,
    imdb_id         TEXT,
    tvdb_id         INTEGER,
    mbid            TEXT,                   -- MusicBrainz ID

    -- source tracking
    metadata_source TEXT,                   -- 'tmdb' | 'nfo' | 'manual' | 'musicbrainz'
    metadata_fetched_at TIMESTAMP,          -- NULL if never fetched from internet
    is_manually_edited  BOOLEAN NOT NULL DEFAULT FALSE,

    FOREIGN KEY (id) REFERENCES media_items(id) ON DELETE CASCADE
);

CREATE INDEX idx_metadata_title      ON metadata(title);
CREATE INDEX idx_metadata_year       ON metadata(year);
CREATE INDEX idx_metadata_tmdb_id    ON metadata(tmdb_id);
CREATE INDEX idx_metadata_sort_title ON metadata(sort_title);

-- ─────────────────────────────────────────────
-- TV SHOWS & SEASONS (parent of episodes)
-- ─────────────────────────────────────────────
CREATE TABLE shows (
    id          TEXT PRIMARY KEY,
    library_id  TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    tmdb_id     INTEGER,
    title       TEXT NOT NULL,
    sort_title  TEXT,
    year        INTEGER,
    description TEXT,
    genres      TEXT,
    rating      REAL,
    content_rating TEXT,
    status      TEXT,                       -- 'Continuing' | 'Ended'
    added_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE seasons (
    id          TEXT PRIMARY KEY,
    show_id     TEXT NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    season_num  INTEGER NOT NULL,
    title       TEXT,
    description TEXT,
    year        INTEGER,
    UNIQUE(show_id, season_num)
);

CREATE TABLE episodes (
    id          TEXT PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    show_id     TEXT NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    season_id   TEXT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    season_num  INTEGER NOT NULL,
    episode_num INTEGER NOT NULL
);

-- ─────────────────────────────────────────────
-- MUSIC (artist → album → track hierarchy)
-- ─────────────────────────────────────────────
CREATE TABLE artists (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    mbid        TEXT,
    description TEXT,
    added_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE albums (
    id          TEXT PRIMARY KEY,
    artist_id   TEXT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    year        INTEGER,
    mbid        TEXT,
    description TEXT
);

CREATE TABLE tracks (
    id          TEXT PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    album_id    TEXT NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    track_num   INTEGER,
    disc_num    INTEGER DEFAULT 1,
    title       TEXT NOT NULL
);

-- ─────────────────────────────────────────────
-- ARTWORK
-- ─────────────────────────────────────────────
CREATE TABLE artwork (
    id          TEXT PRIMARY KEY,
    item_id     TEXT NOT NULL,              -- references media_items, shows, albums
    item_type   TEXT NOT NULL,              -- 'movie' | 'show' | 'album' | ...
    art_type    TEXT NOT NULL,              -- 'poster' | 'backdrop' | 'logo' | 'thumb' | 'cover'
    file_path   TEXT NOT NULL,              -- relative to storage.DataDir/artwork/
    source      TEXT,                       -- 'tmdb' | 'upload' | 'embedded'
    width       INTEGER,
    height      INTEGER,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_artwork_item ON artwork(item_id, item_type, art_type);

-- ─────────────────────────────────────────────
-- AUDIO STREAMS / SUBTITLE STREAMS
-- ─────────────────────────────────────────────
CREATE TABLE media_streams (
    id          TEXT PRIMARY KEY,
    item_id     TEXT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    stream_type TEXT NOT NULL,              -- 'audio' | 'subtitle'
    index_num   INTEGER NOT NULL,           -- FFmpeg stream index
    codec       TEXT,
    language    TEXT,                       -- ISO 639-2
    title       TEXT,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    is_forced   BOOLEAN NOT NULL DEFAULT FALSE,
    -- subtitle-specific
    is_external BOOLEAN NOT NULL DEFAULT FALSE,
    file_path   TEXT                        -- for external .srt/.ass files
);

CREATE INDEX idx_streams_item ON media_streams(item_id);

-- ─────────────────────────────────────────────
-- USERS
-- ─────────────────────────────────────────────
CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,          -- bcrypt
    role            TEXT NOT NULL DEFAULT 'viewer', -- 'admin' | 'viewer'
    is_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at   TIMESTAMP
);

CREATE TABLE user_library_access (
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    library_id  TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, library_id)
);

-- ─────────────────────────────────────────────
-- PLAYBACK STATE
-- ─────────────────────────────────────────────
CREATE TABLE playback_progress (
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id         TEXT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    position_ms     BIGINT NOT NULL DEFAULT 0,
    played_pct      REAL NOT NULL DEFAULT 0,    -- 0.0–1.0
    is_watched      BOOLEAN NOT NULL DEFAULT FALSE,
    last_played_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, item_id)
);

CREATE INDEX idx_progress_user ON playback_progress(user_id, last_played_at DESC);

CREATE TABLE watchlist (
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id     TEXT NOT NULL,
    added_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, item_id)
);

-- ─────────────────────────────────────────────
-- API TOKENS
-- ─────────────────────────────────────────────
CREATE TABLE api_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,       -- SHA-256 of the token
    scope       TEXT NOT NULL DEFAULT 'read', -- 'read' | 'full'
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    expires_at  TIMESTAMP                   -- NULL = never expires
);

-- ─────────────────────────────────────────────
-- PRIVACY & SETTINGS
-- ─────────────────────────────────────────────
CREATE TABLE settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL,              -- JSON value
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Default rows inserted by migration:
-- ('internet_enabled', 'false')
-- ('tmdb_metadata_enabled', 'false')
-- ('tmdb_artwork_enabled', 'false')
-- ('musicbrainz_enabled', 'false')
-- ('cover_art_enabled', 'false')
-- ('update_check_enabled', 'false')
-- etc.

CREATE TABLE network_activity_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,  -- or SERIAL for postgres
    timestamp   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    feature     TEXT NOT NULL,
    url         TEXT NOT NULL,
    blocked     BOOLEAN NOT NULL,
    block_reason TEXT,
    status_code INTEGER,                    -- NULL if blocked
    duration_ms INTEGER                     -- NULL if blocked
);

CREATE INDEX idx_activity_log_ts ON network_activity_log(timestamp DESC);
```

#### Full-Text Search

```sql
-- SQLite FTS5
CREATE VIRTUAL TABLE search_index USING fts5(
    item_id UNINDEXED,
    title,
    original_title,
    description,
    genres,
    cast_names,
    content='metadata',
    content_rowid='rowid'
);

-- PostgreSQL equivalent — use tsvector column
ALTER TABLE metadata ADD COLUMN search_vector tsvector;
CREATE INDEX idx_metadata_fts ON metadata USING GIN(search_vector);
```

### 4.2 Migration Strategy

- Tool: `golang-migrate`
- Files: `internal/db/migrations/NNNNNN_description.{up,down}.sql`
- Migrations run automatically at server startup if pending
- No manual intervention required for upgrades

```go
// internal/db/db.go
func runMigrations(db *sql.DB, dbType string) error {
    driver, err := getMigrateDriver(db, dbType)
    m, err := migrate.NewWithDatabaseInstance(
        "embed://migrations", dbType, driver,
    )
    return m.Up() // no-op if already at latest
}
```

### 4.3 SQLite vs PostgreSQL

Both are supported via a **database-agnostic interface**. The sqlc-generated code has two build targets:

```
db/query/sqlite/    — SQLite dialect queries
db/query/postgres/  — PostgreSQL dialect queries
```

Selected at startup based on `database.type` config. The service layer only sees the interface:

```go
type Querier interface {
    GetMediaItem(ctx context.Context, id string) (MediaItem, error)
    ListMediaItems(ctx context.Context, params ListMediaItemsParams) ([]MediaItem, error)
    // ... all queries
}
```

**When to use which:**

| SQLite | PostgreSQL |
|--------|-----------|
| Single user / small library | Multi-user / large library |
| Easiest setup (default) | Best performance at scale |
| < 50k items recommended | 500k+ items supported |
| No separate DB process | Requires PostgreSQL server |

---

## 5. Media Library & Scanner

### 5.1 Directory Watcher

```
Library configured with path(s)
         │
         ▼
┌─────────────────────┐
│   fsnotify Watcher  │  OS-native: inotify (Linux), FSEvents (macOS),
│                     │  ReadDirectoryChanges (Windows)
└─────────┬───────────┘
          │ CREATE / MODIFY / DELETE events
          ▼
┌─────────────────────┐
│   Event Debouncer   │  100ms debounce window — batch rapid changes
│                     │  (e.g., file copy-in-progress)
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   Scanner Queue     │  Buffered channel, single goroutine consumer
└─────────────────────┘
```

### 5.2 Scanner Pipeline

```
File Path(s) to scan
        │
        ▼
┌───────────────────┐
│  File Walker      │  filepath.WalkDir — skips hidden files, non-media extensions
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  Media Probe      │  FFprobe subprocess: extract codec, duration,
│  (ffprobe)        │  resolution, streams, container format
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  Type Classifier  │  Movie / TV Episode / Music Track
│                   │  Rules: folder structure + filename patterns
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  Title Extractor  │  Parse: "Movie Name (2023).mkv"
│                   │  TV: "Show Name S01E03.mkv"
│                   │  Hint: "{tmdb-12345}" in filename
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  DB Upsert        │  Insert new / update changed / mark deleted
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│  Metadata Trigger │  If internet enabled + feature enabled:
│                   │  enqueue metadata lookup job
└───────────────────┘
```

**Supported media extensions:**

```go
var videoExtensions = map[string]bool{
    ".mkv":true, ".mp4":true, ".avi":true, ".mov":true,
    ".wmv":true, ".m4v":true, ".ts":true,  ".m2ts":true,
    ".webm":true, ".flv":true, ".iso":true,
}

var audioExtensions = map[string]bool{
    ".mp3":true, ".flac":true, ".aac":true, ".ogg":true,
    ".opus":true, ".wav":true, ".m4a":true, ".wma":true,
    ".alac":true, ".aiff":true,
}
```

**TV episode filename patterns (in priority order):**

```
S01E02          → Season 1, Episode 2       (most common)
1x02            → Season 1, Episode 2
s01.e02         → Season 1, Episode 2
Season 1/E02    → Season 1, Episode 2       (folder-based)
2023.01.15      → Air-date based episodes
```

### 5.3 File Fingerprinting

Full SHA-256 of large files is too slow. We use a **fast fingerprint**:

```go
// internal/library/fingerprint.go

// FastHash reads the first 64KB + last 64KB + file size.
// Collision probability negligible for media files.
// Runtime: < 5ms even for 50GB files.
func FastHash(path string) (string, error) {
    f, _ := os.Open(path)
    stat, _ := f.Stat()
    size := stat.Size()

    h := sha256.New()
    buf := make([]byte, 65536)

    // First 64KB
    io.ReadFull(f, buf)
    h.Write(buf)

    // File size
    binary.Write(h, binary.LittleEndian, size)

    // Last 64KB (if file > 128KB)
    if size > 131072 {
        f.Seek(-65536, io.SeekEnd)
        io.ReadFull(f, buf)
        h.Write(buf)
    }

    return hex.EncodeToString(h.Sum(nil)), nil
}
```

---

## 6. Metadata System

### 6.1 Metadata Pipeline

```go
// internal/metadata/provider.go

type Provider interface {
    // Search returns candidate matches for a title/year.
    Search(ctx context.Context, query MetadataQuery) ([]SearchResult, error)

    // Fetch returns full metadata for a specific external ID.
    Fetch(ctx context.Context, id ExternalID) (*Metadata, error)

    // Name returns a human-readable provider name for logging.
    Name() string
}

type MetadataQuery struct {
    Title    string
    Year     int     // 0 = unknown
    Type     string  // 'movie' | 'show' | 'music'
    Language string  // ISO 639-1
}
```

**Provider resolution order:**

```
1. NFO file exists?         → Use NFO (offline, highest priority)
2. Filename hint {tmdb-N}?  → Direct TMDB fetch (if enabled)
3. TMDB enabled?            → Search TMDB by title + year
4. TMDB returns no match?   → Try IMDb via TMDB's find API
5. Still no match?          → Leave metadata empty; log to scan report
```

The metadata pipeline is **non-blocking** — a failed metadata lookup never blocks a file from appearing in the library. Items appear immediately after scanning; metadata enriches asynchronously.

### 6.2 Local NFO Support

NFO files use the Kodi/Jellyfin standard XML format:

```xml
<!-- Movie.nfo -->
<movie>
  <title>The Dark Knight</title>
  <year>2008</year>
  <plot>Batman faces the Joker...</plot>
  <genre>Action</genre>
  <genre>Crime</genre>
  <rating>9.0</rating>
  <mpaa>PG-13</mpaa>
  <tmdbid>155</tmdbid>
  <imdbid>tt0468569</imdbid>
</movie>
```

When an NFO file exists alongside a media file, it is always preferred over any internet source. `metadata_source` is set to `'nfo'`.

### 6.3 Artwork Storage

```
/config/artwork/
├── movies/
│   └── {tmdb_id}/
│       ├── poster.jpg       (primary poster, 500px wide)
│       ├── backdrop.jpg     (16:9 backdrop, 1280px wide)
│       └── logo.png         (transparent logo, if available)
├── shows/
│   └── {tmdb_id}/
│       ├── poster.jpg
│       ├── backdrop.jpg
│       └── seasons/
│           └── s01_poster.jpg
├── albums/
│   └── {mbid}/
│       └── cover.jpg
└── uploads/                 # User-uploaded artwork
    └── {item_id}/
        └── poster.jpg
```

**Artwork served from local disk:**

```
GET /api/v1/artwork/{item_id}/{art_type}
                    ↓
            Reads from /config/artwork/...
                    ↓
            Streams file with proper Content-Type
            Cache-Control: max-age=86400
```

The external URL (e.g., `https://image.tmdb.org/t/p/...`) is **never** used as a `<img src>` in the frontend. All images are proxied through the local server after being downloaded once.

---

## 7. Transcoding Engine

### 7.1 Decision Tree

```
Client requests playback of item
        │
        ▼
┌────────────────────────┐
│  Client Capabilities   │  From request headers / explicit params
│  (codecs, containers)  │
└────────────┬───────────┘
             │
    ┌────────▼────────┐
    │ Direct Play     │  Client supports: video codec + audio codec
    │ Possible?       │  + container + resolution + bitrate
    └────────┬────────┘
          YES│             NO
             │              │
             ▼              ▼
     ┌───────────┐   ┌──────────────────────┐
     │  Direct   │   │ What needs changing? │
     │  Play     │   └──────────┬───────────┘
     └───────────┘              │
                    ┌───────────┼────────────┐
                    │           │            │
              Video only  Audio only  Both (remux
              transcode   transcode   or full code)
                    │           │            │
                    └───────────┴────────────┘
                                │
                    ┌───────────▼───────────┐
                    │  Hardware Accel?      │
                    │  (NVENC/QSV/AMF)      │
                    └───────────┬───────────┘
                             YES│         NO
                                │          │
                        HW Transcode  SW Transcode
                        (FFmpeg NVENC) (FFmpeg x264)
```

**Common direct play scenarios (no transcoding):**

| Video | Audio | Container | Client | Result |
|-------|-------|-----------|--------|--------|
| H.264 | AAC | MP4 | Any modern browser | Direct play |
| H.264 | AC3 | MKV | Browser | Remux to MP4 (copy streams) |
| HEVC | EAC3 | MKV | Browser | Transcode video + audio |
| H.264 | AAC | MP4 | Smart TV | Direct play |
| HEVC | AAC | MP4 | Smart TV (HEVC capable) | Direct play |

### 7.2 FFmpeg Pipeline

FFmpeg is run as a **subprocess** (not via CGO). This isolates crashes.

```go
// internal/transcode/ffmpeg.go

type TranscodeJob struct {
    InputPath    string
    OutputDir    string      // temp dir for HLS segments
    VideoCodec   string      // 'copy' | 'h264' | 'hevc' | 'av1'
    AudioCodec   string      // 'copy' | 'aac' | 'ac3'
    Width        int         // 0 = keep original
    Height       int         // 0 = keep original
    VideoBitrate string      // e.g., "4000k"
    AudioBitrate string      // e.g., "192k"
    StartTimeSec float64     // for seeking
    HWAccel      HWAccelType // 'none' | 'nvenc' | 'qsv' | 'amf' | 'vaapi'
    SubtitlePath string      // for burn-in; empty = no burn-in
}

func (f *FFmpegRunner) BuildArgs(job TranscodeJob) []string {
    args := []string{
        "-hide_banner", "-loglevel", "error",
        "-ss", strconv.FormatFloat(job.StartTimeSec, 'f', 3, 64),
        "-i", job.InputPath,
    }

    // Hardware acceleration input
    if job.HWAccel != HWAccelNone {
        args = append(args, hwAccelInputArgs(job.HWAccel)...)
    }

    // Video codec
    args = append(args, "-c:v", videoCodecArg(job))

    // Audio codec
    args = append(args, "-c:a", audioCodecArg(job))

    // HLS output
    args = append(args,
        "-f", "hls",
        "-hls_time", "6",
        "-hls_list_size", "0",
        "-hls_segment_filename", filepath.Join(job.OutputDir, "seg%05d.ts"),
        filepath.Join(job.OutputDir, "index.m3u8"),
    )

    return args
}
```

### 7.3 Hardware Acceleration

Hardware acceleration is detected at startup and cached:

```go
// internal/transcode/hwaccel.go

type HWAccelCapability struct {
    Type      HWAccelType
    Available bool
    Encoder   string  // 'h264_nvenc', 'h264_qsv', 'h264_amf', 'h264_vaapi'
    Decoder   string  // 'h264_cuvid', 'h264_qsv', ...
}

func DetectHWAccel(ffmpegPath string) []HWAccelCapability {
    // Run: ffmpeg -hide_banner -encoders | grep -E 'nvenc|qsv|amf|vaapi'
    // Parse output, probe each encoder with a short test
    // Cache result; re-detect only if admin triggers manually
}
```

**Priority order when multiple HW accelerators present:**

```
1. NVIDIA NVENC    (best quality + performance)
2. Intel QSV       (good for NAS/NUC)
3. AMD AMF         (AMD GPUs)
4. VA-API          (Linux open-source)
5. Software (x264) (fallback — always available)
```

### 7.4 Session Management

```go
// internal/transcode/session.go

type Session struct {
    ID          string
    UserID      string
    ItemID      string
    StartedAt   time.Time
    LastPingAt  time.Time     // updated by HLS segment requests
    OutputDir   string        // /config/transcode/{sessionID}/
    Process     *os.Process   // FFmpeg process
    VideoCodec  string
    AudioCodec  string
    Done        chan struct{}  // closed when FFmpeg exits
}

// Sessions idle for > 5 minutes with no segment requests are killed.
// Cleanup goroutine runs every 60 seconds.
func (sm *SessionManager) cleanup() {
    ticker := time.NewTicker(60 * time.Second)
    for range ticker.C {
        sm.mu.Lock()
        for id, sess := range sm.sessions {
            if time.Since(sess.LastPingAt) > 5*time.Minute {
                sess.Process.Kill()
                os.RemoveAll(sess.OutputDir)
                delete(sm.sessions, id)
            }
        }
        sm.mu.Unlock()
    }
}
```

---

## 8. Streaming Architecture

### 8.1 HLS Pipeline

```
Client: GET /api/v1/items/{id}/playback
                    │
                    ▼
          ┌─────────────────┐
          │  Negotiator     │  Decide: direct play or transcode
          └────────┬────────┘
                   │ → Transcode
                   ▼
          ┌─────────────────┐
          │ Session Manager │  Create session, start FFmpeg
          └────────┬────────┘
                   │
                   ▼
          Returns: { streamUrl: "/stream/{sessionId}/index.m3u8" }

Client: GET /stream/{sessionId}/index.m3u8
                    │
                    ▼
          ┌─────────────────┐
          │  HLS Handler    │  Wait for first segment (up to 3s)
          │                 │  Return M3U8 manifest
          └────────┬────────┘

Client: GET /stream/{sessionId}/seg00001.ts  (repeated for each segment)
                    │
                    ▼
          ┌─────────────────┐
          │  Segment Server │  Stream file from OutputDir
          │                 │  Update Session.LastPingAt
          └─────────────────┘
```

**HLS manifest format:**

```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:6.000000,
seg00000.ts
#EXTINF:6.000000,
seg00001.ts
...
```

### 8.2 Direct Play

```
Client: GET /api/v1/items/{id}/playback
              (with Accept: video/mp4 or client_profile=browser)
                    │
                    ▼
          ┌─────────────────┐
          │  Negotiator     │  Determines direct play is possible
          └────────┬────────┘
                   │
                   ▼
          Returns: { streamUrl: "/direct/{itemId}", directPlay: true }

Client: GET /direct/{itemId}
                    │
                    ▼
          ┌─────────────────┐
          │  Direct Handler │  http.ServeContent(file)
          │                 │  Supports Range requests (seek)
          │                 │  Content-Type from container
          └─────────────────┘
```

---

## 9. Authentication & Authorization

### 9.1 JWT Design

```
Access Token  — expires in 15 minutes
Refresh Token — expires in 30 days, stored in HttpOnly cookie

Access Token Payload:
{
  "sub": "user-uuid",
  "role": "admin",           // or "viewer"
  "libraries": ["lib-1", "lib-2"],   // accessible library IDs
  "scope": "full",           // or "read" for API tokens
  "iat": 1719561600,
  "exp": 1719562500
}
```

**Token flow:**

```
POST /api/v1/auth/login  { username, password }
        │
        ├── Verify password (bcrypt)
        ├── Generate access token (15min)
        ├── Generate refresh token (30d)
        └── Response: { accessToken } + Set-Cookie: refreshToken (HttpOnly, Secure, SameSite=Strict)

POST /api/v1/auth/refresh  (sends refreshToken cookie automatically)
        │
        ├── Validate refresh token signature + expiry
        ├── Generate new access token
        └── Response: { accessToken }
```

### 9.2 RBAC Model

```
Role: admin
  - Full access to all API endpoints
  - Can manage users, libraries, settings
  - Can view network activity log
  - Can change privacy settings

Role: viewer
  - Can browse and play libraries they have access to
  - Can manage their own playback progress, watchlist
  - Cannot access admin endpoints
  - Cannot see other users' data
```

Library access check:

```go
func (m *AuthMiddleware) requireLibraryAccess(libraryID string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        claims := ClaimsFromContext(r.Context())
        if claims.Role == RoleAdmin {
            return // admins access all libraries
        }
        if !slices.Contains(claims.Libraries, libraryID) {
            writeJSON(w, 403, ErrForbidden)
            return
        }
    }
}
```

---

## 10. Frontend Architecture

### 10.1 Component Structure

```
web/src/
├── api/
│   ├── client.ts          # Base fetch wrapper with auth token injection
│   ├── items.ts           # Item API calls
│   ├── libraries.ts
│   ├── stream.ts          # Playback negotiation
│   ├── users.ts
│   └── settings.ts        # Privacy settings API
│
├── components/
│   ├── ui/                # shadcn/ui primitives (Button, Card, Dialog, etc.)
│   ├── layout/
│   │   ├── AppShell.tsx   # Sidebar + header + main content area
│   │   ├── Sidebar.tsx
│   │   └── Header.tsx
│   ├── library/
│   │   ├── LibraryGrid.tsx        # Poster grid view
│   │   ├── LibraryList.tsx        # List view
│   │   ├── MediaCard.tsx          # Single poster card
│   │   ├── SearchBar.tsx
│   │   └── FilterPanel.tsx
│   ├── player/
│   │   ├── VideoPlayer.tsx        # HLS.js video element wrapper
│   │   ├── AudioPlayer.tsx        # Music player
│   │   ├── PlayerControls.tsx     # Play/pause/seek/volume/subtitle picker
│   │   └── PlayerOverlay.tsx      # Title, episode info overlay
│   └── admin/
│       ├── PrivacyCard.tsx        # Single internet feature toggle card
│       └── ActivityLogTable.tsx   # Network activity log
│
├── pages/
│   ├── Home.tsx           # Dashboard: continue watching, recently added
│   ├── Movies.tsx
│   ├── Shows.tsx
│   ├── Music.tsx
│   ├── Item.tsx           # Movie / episode / track detail page
│   ├── Player.tsx         # Full-screen player page
│   └── admin/
│       ├── Dashboard.tsx  # Server stats
│       ├── Privacy.tsx    # F0 privacy settings — first admin page
│       ├── Libraries.tsx
│       ├── Users.tsx
│       └── Tasks.tsx
│
├── store/
│   ├── authStore.ts       # Zustand: current user, access token
│   ├── playerStore.ts     # Zustand: current playback state
│   └── settingsStore.ts   # Zustand: UI preferences (theme, etc.)
│
└── hooks/
    ├── useItems.ts        # React Query: fetch + cache items
    ├── useProgress.ts     # React Query: playback progress sync
    └── useWebSocket.ts    # WebSocket connection + event subscription
```

### 10.2 State Management

```
Server State (React Query)     Local UI State (Zustand)
───────────────────────────    ─────────────────────────
Library items                  Current user / auth token
Search results                 Player state (playing, position)
Playback progress              Theme preference
User list (admin)              Sidebar collapsed state
Privacy settings               Active filters
Network activity log
```

**React Query key conventions:**

```ts
// Consistent cache keys prevent stale data
['items', libraryId, { search, genre, year }]
['item', itemId]
['progress', userId, itemId]
['settings', 'privacy']
['activity-log', { page, limit }]
```

### 10.3 Player Architecture

```
VideoPlayer.tsx
    │
    ├── HLS.js instance
    │   ├── Loads /stream/{sessionId}/index.m3u8
    │   ├── Fetches segments automatically
    │   └── Handles adaptive quality switching
    │
    ├── <video> element (HTML5)
    │   ├── Receives HLS.js source
    │   └── Native controls disabled (custom PlayerControls used)
    │
    ├── Progress Reporter
    │   ├── Sends PUT /api/v1/users/{id}/progress every 10 seconds
    │   └── Sends final position on pause/exit
    │
    └── PlayerControls.tsx
        ├── Play / Pause
        ├── Seek bar (with chapter markers if available)
        ├── Volume + mute
        ├── Subtitle track selector
        ├── Audio track selector
        ├── Quality selector (auto / 1080p / 720p / 480p)
        ├── Playback speed (0.5x – 2x)
        └── Fullscreen
```

**Direct play path:** `<video src="/direct/{itemId}">` — HLS.js not used; browser handles Range requests natively.

---

## 11. Task Scheduler

The scheduler manages background jobs without an external queue (Redis, RabbitMQ, etc.).

```go
// internal/scheduler/scheduler.go

type Task struct {
    ID          string
    Name        string
    Schedule    string          // cron expression or "" for manual-only
    Handler     func(ctx context.Context) error
    Timeout     time.Duration
    LastRun     time.Time
    LastStatus  string          // 'success' | 'error' | 'running'
    LastError   string
}

// Built-in tasks registered at startup:
var defaultTasks = []Task{
    {
        ID:       "library.scan_all",
        Name:     "Scan All Libraries",
        Schedule: "0 3 * * *",   // 3 AM daily
        Timeout:  2 * time.Hour,
    },
    {
        ID:       "transcode.cleanup",
        Name:     "Clean Transcode Cache",
        Schedule: "*/15 * * * *", // every 15 minutes
        Timeout:  5 * time.Minute,
    },
    {
        ID:       "metadata.refresh_missing",
        Name:     "Fetch Missing Metadata",
        Schedule: "0 4 * * *",    // 4 AM daily (only if internet enabled)
        Timeout:  1 * time.Hour,
    },
}
```

Tasks run in a goroutine pool. Maximum 2 concurrent tasks (configurable). Task status persisted to `settings` table and broadcast via WebSocket.

---

## 12. WebSocket Event System

All real-time UI updates (scan progress, transcode status, task logs) flow through a single WebSocket endpoint.

```
ws://host:8096/ws  (authenticated via token query param)
```

```go
// internal/ws/events.go

type EventType string

const (
    EventScanProgress    EventType = "library.scan.progress"
    EventScanComplete    EventType = "library.scan.complete"
    EventTaskStarted     EventType = "task.started"
    EventTaskCompleted   EventType = "task.completed"
    EventTaskFailed      EventType = "task.failed"
    EventTranscodeUpdate EventType = "transcode.session.update"
    EventServerStats     EventType = "server.stats"         // sent every 5s
)

type Event struct {
    Type    EventType       `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

// Example scan progress event:
// {
//   "type": "library.scan.progress",
//   "payload": {
//     "library_id": "...",
//     "scanned": 1243,
//     "total": 5000,
//     "current_file": "The Dark Knight (2008).mkv"
//   }
// }
```

**Hub design:**

```go
// internal/ws/hub.go

type Hub struct {
    clients    map[string]*Client   // userID → client
    broadcast  chan Event
    register   chan *Client
    unregister chan *Client
}

// Events are broadcast to all connected clients.
// Admin-only events (task logs, privacy activity) are filtered by role.
func (h *Hub) run() {
    for {
        select {
        case event := <-h.broadcast:
            for _, client := range h.clients {
                if canReceive(client, event) {
                    client.send <- event
                }
            }
        }
    }
}
```

---

## 13. Deployment Architecture

### 13.1 Single Binary

```
Build process:
  1. npm run build              → web/dist/ (static assets)
  2. go build -o streamvault   → embeds web/dist/ via go:embed
  3. Result: single ~30MB binary

Startup sequence:
  1. Load config (YAML + env vars)
  2. Run DB migrations
  3. Load privacy settings from DB
  4. Detect FFmpeg + hardware acceleration
  5. Start file watchers for all libraries
  6. Start task scheduler
  7. Start HTTP server
  8. Log: "StreamVault ready on :8096"
```

### 13.2 Docker

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build

# Install Node for frontend build
RUN apk add --no-cache nodejs npm

# Build frontend
COPY web/package*.json web/
RUN cd web && npm ci
COPY web/ web/
RUN cd web && npm run build

# Build backend (embeds web/dist)
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o streamvault ./cmd/streamvault

# Final image — FFmpeg included
FROM linuxserver/ffmpeg:latest
COPY --from=builder /build/streamvault /usr/local/bin/streamvault

ENV SV_PORT=8096
ENV SV_DB_TYPE=sqlite

EXPOSE 8096
VOLUME ["/config", "/media"]

ENTRYPOINT ["streamvault"]
```

```yaml
# docker-compose.yml
services:
  streamvault:
    image: streamvault/server:latest
    container_name: streamvault
    ports:
      - "8096:8096"
    volumes:
      - ./config:/config        # DB, artwork, transcode temp, logs
      - /path/to/movies:/media/movies:ro
      - /path/to/music:/media/music:ro
    environment:
      - SV_DB_TYPE=sqlite
      # - SV_DB_TYPE=postgres
      # - SV_DB_URL=postgres://user:pass@db:5432/streamvault
    devices:
      # Uncomment for NVIDIA hardware transcoding:
      # - /dev/nvidia0
      # Uncomment for Intel QSV / VA-API:
      # - /dev/dri:/dev/dri
    restart: unless-stopped

  # Optional: PostgreSQL for large libraries
  # db:
  #   image: postgres:16-alpine
  #   environment:
  #     POSTGRES_DB: streamvault
  #     POSTGRES_USER: streamvault
  #     POSTGRES_PASSWORD: changeme
  #   volumes:
  #     - pgdata:/var/lib/postgresql/data
  #
  # volumes:
  #   pgdata:
```

### 13.3 Data Directory Layout

```
/config/                         (mapped via Docker volume)
├── streamvault.yaml             # Server config
├── streamvault.db               # SQLite database (if using SQLite)
├── artwork/                     # All downloaded + uploaded artwork
│   ├── movies/{tmdb_id}/
│   ├── shows/{tmdb_id}/
│   └── albums/{mbid}/
├── transcode/                   # Temporary HLS segments (auto-cleaned)
│   └── {session_id}/
│       ├── index.m3u8
│       └── seg00001.ts
└── logs/
    └── streamvault.log          # Structured JSON log
```

---

## 14. Security Architecture

### Headers (every response)

```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

### Input Validation

```go
// All file paths validated against library root
func validatePath(base, target string) error {
    abs, err := filepath.Abs(target)
    if err != nil || !strings.HasPrefix(abs, base) {
        return ErrPathTraversal
    }
    return nil
}
```

### Rate Limiting

```
POST /api/v1/auth/login     → 5 requests / minute / IP
POST /api/v1/auth/refresh   → 20 requests / minute / IP
GET  /api/v1/*              → 300 requests / minute / user
POST /api/v1/*              → 60 requests / minute / user
```

### Secrets

- JWT signing key: 32-byte random key, generated on first start, stored in `settings` table
- bcrypt cost: 12 (≈ 250ms on modern hardware — slow enough to resist brute force)
- API tokens: stored as SHA-256 hash only; raw token shown once at creation

---

## 15. Key Interfaces & Contracts

These interfaces are the boundaries between packages. Changing a signature here is a breaking change.

```go
// Provider — metadata enrichment
type MetadataProvider interface {
    Search(ctx context.Context, q MetadataQuery) ([]SearchResult, error)
    Fetch(ctx context.Context, id ExternalID) (*Metadata, error)
    Name() string
}

// PluginMetadataProvider — future plugin system
type PluginProvider interface {
    MetadataProvider
    // Plugins must declare what internet they need
    InternetManifest() PluginInternetManifest
}

type PluginInternetManifest struct {
    RequiresInternet bool
    ExternalHosts    []string
    DataSent         string  // Human-readable description
    DataReceived     string
}

// Scanner event — emitted by library.Scanner
type ScanEvent struct {
    Type      ScanEventType  // 'added' | 'changed' | 'removed'
    FilePath  string
    LibraryID string
}

// OutboundClient — all external HTTP (enforced by linter)
type OutboundRequester interface {
    Do(ctx context.Context, feature Feature, req *http.Request) (*http.Response, error)
}

// TaskHandler — registered tasks
type TaskHandler interface {
    Run(ctx context.Context) error
    ID() string
    Name() string
}
```

---

## Appendix: Technology Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go 1.22+ | Single binary, fast, low memory, great concurrency |
| HTTP router | Chi | Lightweight, idiomatic Go, no magic |
| DB (primary) | PostgreSQL 16 | Concurrent writes, no lock contention at scale |
| DB (default) | SQLite (modernc) | Zero setup, pure Go (no CGO) |
| Query layer | sqlc | Type-safe, no ORM magic, compile-time checked |
| Migrations | golang-migrate | Battle-tested, rollback support |
| Transcoding | FFmpeg subprocess | No CGO, crash isolation, hardware accel native |
| File watching | fsnotify | Cross-platform, OS-native events |
| Frontend | React 19 + TypeScript | Large ecosystem, type safety |
| Build | Vite | Fast HMR, ES module native |
| UI components | shadcn/ui | Accessible, unstyled primitives, Tailwind |
| Video player | HLS.js | Browser HLS support, well maintained |
| Auth | JWT (local) | No external dependency |
| Logging | zap | Structured JSON, zero-allocation hot path |
| Linting | golangci-lint | Includes custom OutboundClient rule |
