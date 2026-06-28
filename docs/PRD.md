# StreamVault — Product Requirements Document

**Version:** 0.1 (Draft)
**Date:** 2026-06-28
**Status:** Under Review

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Market Analysis](#3-market-analysis)
4. [Target Users](#4-target-users)
5. [Product Vision & Goals](#5-product-vision--goals)
6. [Non-Goals](#6-non-goals)
7. [Core Principles](#7-core-principles)
8. [Feature Requirements](#8-feature-requirements)
   - [MVP (Phase 1)](#mvp-phase-1)
   - [Phase 2](#phase-2)
   - [Phase 3](#phase-3)
9. [Technical Requirements](#9-technical-requirements)
10. [Architecture Overview](#10-architecture-overview)
11. [API Requirements](#11-api-requirements)
12. [Security Requirements](#12-security-requirements)
13. [Performance Requirements](#13-performance-requirements)
14. [Deployment Requirements](#14-deployment-requirements)
15. [Competitive Differentiation](#15-competitive-differentiation)
16. [Success Metrics](#16-success-metrics)
17. [Open Questions](#17-open-questions)

---

## 1. Executive Summary

StreamVault is a self-hosted, open-source home media server that lets users stream their personal movie, TV show, and music collections to any device — TV, phone, tablet, or browser — from anywhere in the world.

It is designed to be the successor to what home media servers should be in 2026: **free**, **private**, **polished**, and **performant at scale**. It learns from what Plex, Jellyfin, Kodi, Emby, and Universal Media Server did right — and deliberately avoids their mistakes.

---

## 2. Problem Statement

### The Core Pain

People accumulate personal media collections (movies, TV shows, music) on local hard drives. Playing this media on modern smart TVs, phones, and tablets requires either:

- A physical connection (USB) — clunky, no remote access
- Manual file transfers — slow and tedious
- A media server software — the right solution, but current options have critical flaws

### What's Wrong With Current Options

| Problem | Who Has It |
|---------|-----------|
| Aggressive paywalls on basic features | Plex (remote streaming: $1.99/mo; HW transcoding: $6.99/mo; lifetime now $749.99) |
| UI/UX not polished enough for families | Jellyfin (community-acknowledged weakness) |
| Performance degrades past 50k items | All platforms (SQLite lock contention) |
| Not a real server (local player only) | Kodi |
| Legacy technology (DLNA only, Java) | Universal Media Server |
| Uncomfortable middle ground | Emby (paywalled, less open than Jellyfin) |
| No AI-powered discovery | All platforms |
| Metadata corrections require manual work | All platforms |
| No built-in watch-party without plugins | Plex, Emby |

### The Opportunity

Plex's 2025–2026 price escalation (lifetime pass: $119 → $249 → $749) triggered a mass migration wave. Jellyfin captured most migrants but users consistently request better UI polish and better performance at scale. **The market wants a free, polished, modern alternative and none exists yet.**

---

## 3. Market Analysis

### Competitive Landscape

| Product | Core Model | License | Stack | DB | Key Weakness |
|---------|-----------|---------|-------|----|-------------|
| **Jellyfin** | Free, open-source | GPL-2 | C#/.NET 9 | SQLite/EFCore | UI polish, large-library perf |
| **Plex** | Freemium → Paywall | Proprietary | Unknown | SQLite | Aggressive monetization |
| **Kodi** | Free, open-source | GPL-2 | C++ | SQLite/MySQL | Not a server, local only |
| **Emby** | Freemium | Hybrid | C#/.NET 8 | SQLite | Middle ground, declining |
| **UMS** | Free, open-source | GPL-2 | Java | Unknown | Legacy DLNA only |
| **StreamVault** | Free, open-source | MIT | Go/React | PostgreSQL/SQLite | *To be built* |

### Market Trends (2026)

- Self-hosted media servers growing as streaming subscriptions fragment and increase in price
- Docker is de facto deployment standard for tech-forward users
- Samsung/LG Smart TVs now run web-standard clients (no dedicated app required)
- Hardware transcoding (NVENC, QSV, AMF) expected as free feature
- "Watch party" / SyncPlay features emerging as differentiator
- AI-powered recommendations still unexploited in self-hosted space

---

## 4. Target Users

### Primary: The Power User (Early Adopters)
- **Profile:** Tech-savvy individual, comfortable with Docker, self-hosting, home server
- **Collection size:** 1,000–50,000+ media items
- **Pain point:** Plex pricing, Jellyfin UI rough edges
- **Motivation:** Control, privacy, cost savings vs. streaming subscriptions
- **Devices:** Samsung/LG Smart TV, Android phone, iPad, PC browser

### Secondary: The Family Admin
- **Profile:** One technical person managing media for a household (2–6 people)
- **Collection size:** 200–5,000 media items
- **Pain point:** Wants Plex-quality polish without the $70/year fee
- **Motivation:** Easy to set up, family members can use it without help
- **Devices:** Smart TV (primary), phones (secondary)

### Tertiary: The Collector / Archivist
- **Profile:** Power user with massive library (50,000–500,000+ items)
- **Collection size:** 50k–500k items (films, music, documentaries)
- **Pain point:** Database lock contention, slow scanning, poor metadata for obscure content
- **Motivation:** Performance at scale, batch metadata editing, duplicate detection

---

## 5. Product Vision & Goals

### Vision Statement

> StreamVault makes your personal media collection feel as good as Netflix — without giving up your data, your money, or your control.

### Goals

**G1 — Free Forever:** Zero paywalls. All core features free. No "Plex Pass equivalent."

**G2 — Family-Friendly Polish:** A non-technical family member can find and play a movie in under 30 seconds without asking for help.

**G3 — Scale Without Pain:** Library of 200,000+ items with sub-second search, no UI blocking during scans, no database lock issues.

**G4 — Everywhere Access:** Play your media on Samsung TV, iPhone, Android, browser — from your couch or from a hotel 5,000 miles away.

**G5 — Developer Ecosystem:** A public REST API and plugin SDK that make it easy to build third-party clients, integrations, and extensions.

---

## 6. Non-Goals

The following are explicitly out of scope (at least for MVP):

- **Live TV / DVR:** OTA antenna integration (Phase 3+ if at all)
- **Cloud storage backend:** S3/Dropbox as media source (future consideration)
- **Transcoding for 100+ concurrent users:** Not a streaming service, home-scale
- **Content purchasing/rental:** Not a storefront
- **DRM-protected content:** We do not support playback of DRM media
- **Social features (follows, public profiles):** Private/household-scoped only

---

## 7. Core Principles

These guide every design and implementation decision:

1. **Free means free.** No paywalling features that were free yesterday. If a feature ships, it stays free.

2. **Privacy by design.** No telemetry without explicit opt-in. No calling home. All data stays on user hardware.

3. **API-first.** Every feature the UI can do, the API can do. Third-party clients are first-class citizens.

4. **Local network first, remote as opt-in.** Works perfectly without internet. Remote access via user's own infrastructure (reverse proxy, Tailscale, VPN). No proprietary cloud relay.

5. **Fail gracefully.** If metadata lookup fails, media still plays. If transcoding fails, direct play is offered. Nothing silently disappears.

6. **Boring technology choices.** Pick proven tools with large communities. No exotic frameworks that make contribution harder.

---

## 8. Feature Requirements

### Priority Notation
- **P0** — Must ship in MVP (blocker)
- **P1** — Should ship in MVP
- **P2** — Phase 2
- **P3** — Phase 3 / Future

---

### MVP (Phase 1)

#### F1 — Media Library Management

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F1.1 | Scan local directories for media files | P0 | Recursive, configurable paths |
| F1.2 | Incremental scan (detect new/changed/deleted files only) | P0 | Do not re-scan entire library |
| F1.3 | Real-time file watching (inotify / FSEvents / ReadDirectoryChanges) | P1 | Detect additions without manual trigger |
| F1.4 | Organize by Movies / TV Shows / Music | P0 | Separate library views per media type |
| F1.5 | Multi-library support (e.g., separate "Kids" library) | P1 | Different directories → different libraries |
| F1.6 | Library size display (item count, total storage) | P1 | Dashboard stat |
| F1.7 | Batch library operations (re-scan, re-match metadata) | P2 | Needed for large collections |
| F1.8 | Duplicate detection | P2 | Hash-based + fuzzy title match |

#### F2 — Metadata

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F2.1 | Auto-match movies/shows to TMDB | P0 | Primary metadata source |
| F2.2 | IMDb as fallback metadata source | P1 | For content not in TMDB |
| F2.3 | Download poster, backdrop, fanart, logo images | P0 | From TMDB |
| F2.4 | Manual match override (user selects correct TMDB entry) | P0 | Essential for mismatch correction |
| F2.5 | Filename hints for disambiguation (`{tmdb-12345}`) | P1 | Folder/file name based forcing |
| F2.6 | Local NFO file support (Kodi-compatible metadata) | P1 | Respect existing `.nfo` files |
| F2.7 | Music metadata via MusicBrainz | P1 | Standard for audio |
| F2.8 | Batch metadata re-match | P2 | Apply corrections to multiple items |
| F2.9 | User-submitted metadata corrections | P3 | Community feature |
| F2.10 | AI-assisted metadata fuzzy matching | P3 | For unidentified content |

#### F3 — Playback & Streaming

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F3.1 | Direct play (no transcoding) when client supports format | P0 | Highest priority — zero CPU, lowest latency |
| F3.2 | Software transcoding via FFmpeg | P0 | Fallback when direct play not possible |
| F3.3 | Hardware-accelerated transcoding (NVIDIA NVENC) | P0 | Essential for performance |
| F3.4 | Hardware transcoding: Intel QSV | P1 | NAS/integrated graphics support |
| F3.5 | Hardware transcoding: AMD AMF | P1 | AMD GPU support |
| F3.6 | Hardware transcoding: VA-API (Linux) | P1 | Open-source HW accel |
| F3.7 | HLS adaptive bitrate streaming | P0 | Standard for web/TV clients |
| F3.8 | Bitrate selection (auto + manual) | P1 | User can override quality |
| F3.9 | External subtitle support (.srt, .ass, .vtt) | P0 | Embedded + sidecar files |
| F3.10 | Subtitle burn-in for incompatible clients | P1 | Hardware-accelerated via FFmpeg |
| F3.11 | Resume playback from last position | P0 | Per-user, cross-device |
| F3.12 | Multiple audio track selection | P0 | Per-stream |
| F3.13 | Multiple subtitle track selection | P0 | Per-stream |
| F3.14 | Playback speed control | P1 | 0.5x–2x |
| F3.15 | Chapter navigation (for video files with chapters) | P1 | Seek to chapter |
| F3.16 | Skip intro / skip credits detection | P2 | Analyze opening/closing credits |
| F3.17 | HDR tone-mapping (HDR → SDR for non-HDR displays) | P2 | FFmpeg tone-map filter |

#### F4 — Web UI (Client)

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F4.1 | Dashboard (recently added, continue watching, recommended) | P0 | Home screen |
| F4.2 | Library grid view (posters) | P0 | Standard media browser |
| F4.3 | Library list view | P1 | Alternative view mode |
| F4.4 | Search (title, actor, genre, year) | P0 | Instant search, no page refresh |
| F4.5 | Filter & sort (genre, year, rating, resolution, language) | P1 | Faceted filtering |
| F4.6 | Movie detail page (metadata, cast, streams, related) | P0 | Rich detail view |
| F4.7 | TV show detail page (seasons, episodes, progress) | P0 | Episode grid with watch status |
| F4.8 | Music artist / album / track views | P0 | Standard music library UI |
| F4.9 | Built-in video player | P0 | HLS.js or native video element |
| F4.10 | Built-in audio player | P0 | With queue management |
| F4.11 | Mobile-responsive design | P0 | Works on phone browser |
| F4.12 | Dark theme (default) | P0 | Standard for media apps |
| F4.13 | Light theme | P2 | Optional toggle |
| F4.14 | Watch history per user | P1 | "Watched" badge, progress rings |
| F4.15 | Favorites / watchlist | P1 | User-curated lists |
| F4.16 | Admin panel (users, libraries, tasks, logs) | P0 | Server management UI |
| F4.17 | Server dashboard (CPU, memory, transcode sessions, storage) | P1 | Real-time stats |
| F4.18 | Keyboard shortcuts for player | P1 | Space=play/pause, arrow keys=seek |
| F4.19 | Chromecast / AirPlay support | P2 | Browser cast integration |

#### F5 — Authentication & Multi-User

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F5.1 | Local username/password accounts | P0 | Built-in auth |
| F5.2 | Admin role vs. standard user role | P0 | Access control |
| F5.3 | Per-library access controls | P0 | Admin assigns which libraries a user sees |
| F5.4 | PIN-based parental controls | P1 | Restrict content rating per user |
| F5.5 | Invite link system (admin creates invite, user registers) | P1 | No self-registration by default |
| F5.6 | OAuth 2.0 / OIDC integration | P2 | Google, GitHub SSO |
| F5.7 | LDAP/Active Directory integration | P2 | Enterprise/home lab |
| F5.8 | Session management (list active sessions, revoke) | P1 | Admin visibility |
| F5.9 | API token support | P1 | For third-party clients |
| F5.10 | Two-factor authentication (TOTP) | P2 | Optional security layer |

#### F6 — Remote Access

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F6.1 | Works on local network without any cloud dependency | P0 | Core requirement |
| F6.2 | HTTPS support (Let's Encrypt / custom cert) | P0 | Secure remote access |
| F6.3 | Reverse proxy documentation (nginx, Caddy, Traefik) | P0 | Guide for remote access setup |
| F6.4 | Base URL path prefix support (`/streamvault`) | P1 | For multi-app reverse proxy |
| F6.5 | Bandwidth throttle per user (remote streams) | P1 | Prevent crushing home upload bandwidth |
| F6.6 | Tailscale integration guidance | P1 | Zero-config VPN option |
| F6.7 | Built-in optional relay (self-hosted relay server) | P3 | For users who can't port-forward |

---

### Phase 2

#### F7 — Watch Together (SyncPlay)

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F7.1 | Create watch party session (host + guests) | P2 | WebSocket-based sync |
| F7.2 | Sync play/pause/seek across all participants | P2 | Sub-second sync |
| F7.3 | Text chat during watch party | P2 | Sidebar chat |
| F7.4 | Guest access (no account required for watch parties) | P3 | Link-based temp access |

#### F8 — Smart Recommendations

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F8.1 | "More like this" (genre/tag-based similarity) | P2 | Content-based filtering |
| F8.2 | Watch history-based recommendations | P2 | Collaborative filtering |
| F8.3 | "New episodes" / "new additions" notifications | P2 | In-app notifications |
| F8.4 | AI-powered discovery (LLM query: "show me 80s sci-fi thrillers") | P3 | Natural language search |

#### F9 — Plugin System

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F9.1 | Plugin SDK (Go interface-based) | P2 | Typed contracts for plugins |
| F9.2 | Metadata provider plugins | P2 | Alternative metadata sources |
| F9.3 | Authentication provider plugins | P2 | Custom auth backends |
| F9.4 | Scheduled task plugins | P2 | Custom automation |
| F9.5 | Plugin repository (curated community list) | P3 | Discovery and install via UI |
| F9.6 | WebAssembly plugin support | P3 | Language-agnostic, sandboxed |

---

### Phase 3

#### F10 — TV & Mobile Clients

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F10.1 | AndroidTV / FireTV client app | P3 | Native or React Native |
| F10.2 | Apple TV (tvOS) client app | P3 | Native Swift |
| F10.3 | Roku client app | P3 | BrightScript |
| F10.4 | Samsung Tizen app | P3 | Or use web client |
| F10.5 | iOS native app | P3 | Swift / React Native |
| F10.6 | Android native app | P3 | Kotlin / React Native |
| F10.7 | Open client SDK (TypeScript) | P3 | For third-party developers |

#### F11 — Advanced Library Tools

| ID | Feature | Priority | Notes |
|----|---------|----------|-------|
| F11.1 | Duplicate file detection & merge UI | P3 | Across all media types |
| F11.2 | Media health check (corrupted files, missing metadata) | P3 | Library audit tool |
| F11.3 | Storage analytics (largest files, codec breakdown) | P3 | Dashboard view |
| F11.4 | Automated file organization (rename/move by convention) | P3 | Optional, with preview |

---

## 9. Technical Requirements

### Backend

| Requirement | Specification |
|-------------|---------------|
| Language | **Go 1.22+** — compiled, fast, low memory, excellent concurrency |
| HTTP Framework | **Gin** or **Chi** — lightweight, idiomatic Go |
| Database (primary) | **PostgreSQL 16+** — better concurrency, no write-lock issues |
| Database (default/simple) | **SQLite** (via `modernc.org/sqlite`) — zero setup for single-user |
| ORM / Query Builder | **sqlc** (type-safe SQL code gen) or **pgx** directly |
| Database Migrations | **golang-migrate** — versioned, rollback-capable |
| Transcoding | **FFmpeg** (latest stable, dynamically linked) |
| Hardware Accel | NVENC (NVIDIA), QSV (Intel), AMF (AMD), VA-API (Linux) |
| File Watching | OS-native (inotify / kqueue / ReadDirectoryChanges) via `fsnotify` |
| Task Queue | Built-in goroutine pool (no external queue required for MVP) |
| Caching | In-process LRU cache (ristretto) + optional Redis for multi-instance |
| Logging | **zap** (structured JSON logging) |
| Config | YAML/TOML config file + environment variable overrides |
| Metrics | Prometheus `/metrics` endpoint |
| WebSockets | For real-time events (scan progress, transcode status, SyncPlay) |

### Frontend

| Requirement | Specification |
|-------------|---------------|
| Framework | **React 19 + TypeScript** |
| Build Tool | **Vite** |
| UI Component Library | **shadcn/ui** (Radix primitives + Tailwind) |
| State Management | **Zustand** (lightweight) + React Query for server state |
| Video Player | **HLS.js** + HTML5 `<video>` |
| Audio Player | **Howler.js** or Web Audio API |
| Icons | **Lucide React** |
| Routing | **React Router v7** |
| i18n | **i18next** (internationalization from day one) |
| Testing | **Vitest** + **Playwright** (E2E) |

### Packaging

- Frontend built as static assets, served by Go backend (embedded via `embed.FS`)
- Single binary deployment — no Node.js required at runtime
- Docker image: `streamvault/server:latest` (multi-arch: amd64, arm64)
- Docker Compose template included in repo

---

## 10. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      CLIENT LAYER                           │
│  Web Browser  │  Mobile Browser  │  TV App  │  3rd Party   │
└───────┬───────┴────────┬─────────┴────┬─────┴──────┬───────┘
        │                │              │             │
        └────────────────┴──────────────┴─────────────┘
                                │
                         HTTPS / WSS
                                │
┌───────────────────────────────┼─────────────────────────────┐
│                      API GATEWAY                            │
│  REST API v1  │  WebSocket (events)  │  Static File Server  │
└───────┬───────┴──────────┬───────────┴──────────────────────┘
        │                  │
┌───────┴──────────────────┴──────────────────────────────────┐
│                      CORE SERVICES                          │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐ │
│  │  Auth Service│  │Library Service│  │ Streaming Service │ │
│  │  (JWT + RBAC)│  │(scan,metadata)│  │ (HLS + Direct)    │ │
│  └──────────────┘  └──────┬───────┘  └────────┬──────────┘ │
│                            │                    │            │
│  ┌──────────────┐  ┌───────┴───────┐  ┌────────┴─────────┐ │
│  │Metadata Svc  │  │ Task Scheduler│  │ Transcode Engine  │ │
│  │(TMDB, MBrainz│  │ (scan, thumbs)│  │ (FFmpeg wrapper)  │ │
│  └──────────────┘  └───────────────┘  └───────────────────┘ │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │         DATA LAYER          │
        │  PostgreSQL / SQLite  │ File │
        │  (media, users, meta) │ System│
        └─────────────────────────────┘
```

### Key Architectural Decisions

**ADR-1: Monolithic server, modular code**
- Single deployable binary
- Internal package boundaries (auth, library, stream, metadata, tasks)
- Rationale: Simpler for self-hosters, no orchestration overhead. Can extract services later if needed.

**ADR-2: PostgreSQL as first-class DB (with SQLite fallback)**
- PostgreSQL has real concurrent write support, no lock contention
- SQLite offered for zero-config single-user installs
- Jellyfin and Plex suffer from SQLite limits at scale — we won't
- Database-agnostic via sqlc / interface layer

**ADR-3: No proprietary cloud relay**
- Local network always works without internet
- Remote access = user's own reverse proxy / Tailscale / VPN
- We provide documentation, not infrastructure
- Eliminates operational costs and privacy concerns

**ADR-4: FFmpeg for all transcoding**
- Industry standard, battle-tested, hardware acceleration native
- Custom FFmpeg arguments exposed via config for power users
- No proprietary transcoder

**ADR-5: Frontend as embedded static assets**
- Go's `embed.FS` bundles the React app into the server binary
- No separate web server process needed
- Single binary simplifies deployment dramatically

---

## 11. API Requirements

### Design Principles
- REST with versioned paths (`/api/v1/...`)
- JSON request/response bodies
- Bearer token authentication (JWT)
- OpenAPI 3.0 spec generated from code annotations
- WebSocket endpoint for real-time events (`/ws`)
- Pagination via `limit` + `cursor` (not page number)

### Core Endpoints (MVP)

```
Authentication
  POST   /api/v1/auth/login
  POST   /api/v1/auth/logout
  POST   /api/v1/auth/refresh
  GET    /api/v1/auth/me

Libraries
  GET    /api/v1/libraries
  POST   /api/v1/libraries
  GET    /api/v1/libraries/:id
  DELETE /api/v1/libraries/:id
  POST   /api/v1/libraries/:id/scan

Media Items
  GET    /api/v1/items?library=&type=&search=&limit=&cursor=
  GET    /api/v1/items/:id
  PATCH  /api/v1/items/:id           (manual metadata override)
  GET    /api/v1/items/:id/stream    (negotiate play URL)
  GET    /api/v1/items/:id/playback  (direct play or transcode URL)

TV Shows
  GET    /api/v1/shows/:id/seasons
  GET    /api/v1/shows/:id/seasons/:season/episodes

Users
  GET    /api/v1/users               (admin only)
  POST   /api/v1/users
  GET    /api/v1/users/:id
  PATCH  /api/v1/users/:id
  DELETE /api/v1/users/:id

Playback State
  GET    /api/v1/users/:id/progress
  PUT    /api/v1/users/:id/progress/:itemId
  GET    /api/v1/users/:id/history
  GET    /api/v1/users/:id/watchlist
  POST   /api/v1/users/:id/watchlist/:itemId

Tasks
  GET    /api/v1/tasks               (scheduled task list)
  POST   /api/v1/tasks/:id/run       (trigger manually)
  GET    /api/v1/tasks/:id/status

Server
  GET    /api/v1/server/info
  GET    /api/v1/server/stats        (CPU, memory, active sessions)

Streaming
  GET    /stream/:sessionId/index.m3u8    (HLS manifest)
  GET    /stream/:sessionId/:segment.ts   (HLS segment)
  GET    /direct/:itemId                  (direct play)
```

---

## 12. Security Requirements

| Requirement | Detail |
|-------------|--------|
| Authentication | JWT with short-lived access tokens (15 min) + refresh tokens (30 days) |
| Password storage | bcrypt (min cost 12) |
| HTTPS | Required for remote access; HTTP allowed on local network |
| CORS | Configurable allowed origins |
| Rate limiting | Login endpoint: 5 attempts per minute per IP |
| SQL injection | Parameterized queries only (sqlc enforces this) |
| XSS | React's default escaping + Content-Security-Policy headers |
| Clickjacking | `X-Frame-Options: DENY` |
| File traversal | All file paths validated against allowed library roots |
| API tokens | Scoped (read-only vs. full access), revocable per-token |
| Admin API | All admin endpoints require admin role claim in JWT |
| Telemetry | None by default. Optional opt-in crash reports only. |

---

## 13. Performance Requirements

| Metric | Target |
|--------|--------|
| Library scan (initial, 10k items) | < 5 minutes |
| Library scan (incremental, 10k items, 10 new) | < 10 seconds |
| Search response (50k item library) | < 200ms (p99) |
| HLS stream start (first segment) | < 3 seconds (transcode), < 1 second (direct play) |
| Web UI initial load | < 2 seconds on home network |
| API response (list items, 100 items) | < 100ms (p99) |
| Concurrent transcode sessions | ≥ 4 simultaneous (hardware), ≥ 2 (software on 4-core CPU) |
| Library size supported without degradation | 500,000+ items |
| Memory footprint (idle, 10k library) | < 512 MB |
| Memory footprint (active 4 transcode sessions) | < 2 GB |

---

## 14. Deployment Requirements

### Supported Platforms

| Platform | Priority |
|----------|----------|
| Docker (Linux x86_64) | P0 |
| Docker (Linux ARM64) | P0 |
| Linux bare metal (amd64) | P1 |
| Windows bare metal | P1 |
| macOS (Apple Silicon + Intel) | P1 |
| Synology NAS (SPK package) | P2 |
| QNAP NAS | P2 |
| Unraid Community App | P2 |

### Docker Requirements

```yaml
# Minimum docker-compose.yml
services:
  streamvault:
    image: streamvault/server:latest
    ports:
      - "8096:8096"
    volumes:
      - ./config:/config
      - /path/to/media:/media:ro
    environment:
      - SV_DB_TYPE=sqlite          # or postgres
      - SV_DB_URL=postgresql://... # if postgres
    restart: unless-stopped
```

### Resource Requirements (Minimum)

| Scenario | CPU | RAM | Storage |
|----------|-----|-----|---------|
| Direct play only | 1 core | 512 MB | 1 GB (app) |
| Software transcode 1x | 2 cores | 1 GB | 5 GB (temp) |
| Hardware transcode 4x | Any (GPU req.) | 2 GB | 5 GB (temp) |

---

## 15. Competitive Differentiation

### Why StreamVault Over Alternatives

| Feature | StreamVault | Jellyfin | Plex | Emby |
|---------|------------|---------|------|------|
| 100% Free (all features) | ✅ | ✅ | ❌ | ❌ |
| Open source | ✅ (MIT) | ✅ (GPL) | ❌ | Partial |
| Hardware transcoding free | ✅ | ✅ | ❌ ($6.99/mo) | ❌ ($4.99/mo) |
| Remote streaming free | ✅ | ✅ | ❌ ($1.99/mo) | ✅ |
| PostgreSQL support | ✅ (native) | 🔜 (planned) | ❌ | ❌ |
| Modern Go backend | ✅ | ❌ (C#) | ❌ | ❌ |
| Sub-200ms search (50k lib) | ✅ (goal) | ⚠️ (degrades) | ⚠️ | ⚠️ |
| Single binary deployment | ✅ | ❌ | ❌ | ❌ |
| Watch party (SyncPlay) | ✅ (P2) | ✅ | ❌ (Plex Pass) | ❌ |
| Plugin system | ✅ (P2) | ✅ | ❌ | Partial |
| No telemetry (default) | ✅ | ✅ | ❌ | ❌ |

### Our Unique Bets

1. **Go backend** — faster startup, lower memory than C#/.NET, single binary
2. **PostgreSQL native** — solves the SQLite scale wall all competitors hit
3. **Polish-first UI** — investing design budget Jellyfin doesn't have
4. **Single binary** — `./streamvault` and it works; no JVM, no .NET runtime
5. **MIT license** — more permissive than GPL; enables commercial client ecosystem

---

## 16. Success Metrics

### Phase 1 (MVP) — 6 months post-launch

| Metric | Target |
|--------|--------|
| GitHub Stars | 2,000+ |
| Docker Hub Pulls | 10,000+ |
| Active installations (telemetry opt-in) | 1,000+ |
| GitHub Issues (bugs) | < 50 open |
| Test coverage | > 70% |
| Time to first stream (new install) | < 15 minutes |

### Phase 2 — 12 months

| Metric | Target |
|--------|--------|
| GitHub Stars | 10,000+ |
| Community Discord members | 2,000+ |
| Plugin count (community) | 10+ |
| Third-party client apps | 3+ |
| P99 search latency (100k lib) | < 200ms |

---

## 17. Open Questions

| # | Question | Owner | Target Date |
|---|----------|-------|-------------|
| Q1 | Product name — "StreamVault" or alternative? | Product | Before Phase 1 kickoff |
| Q2 | Go vs. Rust for backend? (Rust: faster, steeper curve) | Engineering | Architecture phase |
| Q3 | PostgreSQL migration path from SQLite — how to guide users? | Engineering | Phase 1 |
| Q4 | Plugin system: Go plugins (.so) vs. WebAssembly vs. HTTP sidecar? | Engineering | Phase 2 design |
| Q5 | TV client priority: AndroidTV vs. Apple TV vs. web-first? | Product | Phase 2 planning |
| Q6 | Opt-in telemetry: what to collect, where to send? | Product/Legal | Phase 1 |
| Q7 | Self-hosted relay for remote access: worth the infrastructure cost? | Engineering | Phase 2 |
| Q8 | LLM-powered natural language search: which model, local or cloud? | Engineering | Phase 3 |
| Q9 | Community metadata corrections: voting system or PR-based? | Product | Phase 3 |
| Q10 | Commercial support tier ever? Or fully community-funded (sponsorships)? | Business | Year 1 |

---

## Appendix A — Research Summary (Competitive Analysis)

*Based on deep analysis of Plex, Jellyfin, Kodi, Emby, and Universal Media Server codebases, documentation, and community feedback.*

### Key Lessons Learned

**From Jellyfin (what works):**
- 100% free model wins community loyalty
- Open REST API creates client ecosystem (70+ clients)
- Hardware transcoding must be free
- SyncPlay (watch together) is a highly-valued differentiator
- EFCore migration shows SQLite limits forced their hand — we should start with PostgreSQL

**From Plex (what not to do):**
- Paywalling remote streaming ($1.99/mo) triggered mass exodus
- Closed source limits contribution and trust
- Lifetime price increases ($119 → $249 → $749) destroy brand loyalty
- SQLite at scale creates database lock contention

**From Kodi (what to adopt):**
- Best-in-class video player quality
- Python plugin system is accessible but not sandboxed enough
- MySQL support for multi-device sync is the right pattern

**From UMS (what to avoid):**
- Java JVM overhead is unnecessary for modern systems
- DLNA-only is a dead end — HLS is the standard
- Not investing in UI makes adoption impossible outside of enthusiasts

**From Emby (the cautionary tale):**
- Forking then closing source erodes community trust permanently
- The "middle ground" (half-free, half-paid) satisfies nobody
- A product without a clear identity loses to clearer alternatives

### Architecture Patterns to Adopt

1. **API-first** — every feature has an API endpoint (Jellyfin model)
2. **Plugin via dependency injection** — modular, discoverable (Jellyfin model)
3. **FFmpeg for all transcoding** — no proprietary alternatives
4. **HLS as primary protocol** — forget DLNA for new features
5. **Docker-first deployment** — standard for self-hosters
6. **PostgreSQL over SQLite at scale** — solved problem, not an innovation
7. **File watching over polling** — real-time detection via OS events
