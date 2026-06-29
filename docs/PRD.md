# StreamVault — Product Requirements Document

**Version:** 0.5
**Date:** 2026-06-28
**Status:** Under Review
**Changelog:**
- v0.5 — F4.4 Search implemented and verified: FTS5 full-text index, debounced sidebar search bar, /search results page with type filter chips. Fixed Device Guard block via go run. Fixed FTS5 alias bug in MATCH query.
- v0.4 — Playback verified end-to-end on both dev (:5180) and production (:8096). F3.1 direct play and F4.9 video player marked tested. .MOV (Canon camera) confirmed playing natively.
- v0.3 — Added implementation tracking columns (Impl / Test / Git) to all feature tables. Status reflects MVP build session through 2026-06-28.
- v0.2 — Added F0 (Offline-First & Privacy Controls) as P0 feature set. Promoted privacy to a first-class design pillar throughout. Every internet-connected feature now documented with: what data leaves the device, what external service receives it, and how to disable it. Metadata section updated to reflect offline-first fallback behavior.

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
   - [F0 — Offline-First & Privacy Controls (NEW — P0)](#f0--offline-first--privacy-controls-new--p0)
   - [F1 — Media Library Management](#f1--media-library-management)
   - [F2 — Metadata](#f2--metadata)
   - [F3 — Playback & Streaming](#f3--playback--streaming)
   - [F4 — Web UI](#f4--web-ui-client)
   - [F5 — Authentication & Multi-User](#f5--authentication--multi-user)
   - [F6 — Remote Access](#f6--remote-access)
   - [Phase 2 Features](#phase-2)
   - [Phase 3 Features](#phase-3)
9. [Internet Data Flow Registry](#9-internet-data-flow-registry)
10. [Technical Requirements](#10-technical-requirements)
11. [Architecture Overview](#11-architecture-overview)
12. [API Requirements](#12-api-requirements)
13. [Security Requirements](#13-security-requirements)
14. [Performance Requirements](#14-performance-requirements)
15. [Deployment Requirements](#15-deployment-requirements)
16. [Competitive Differentiation](#16-competitive-differentiation)
17. [Success Metrics](#17-success-metrics)
18. [Open Questions](#18-open-questions)
19. [Appendix A — Competitive Research](#appendix-a--research-summary-competitive-analysis)

---

## 1. Executive Summary

StreamVault is a self-hosted, open-source home media server that lets users stream their personal movie, TV show, and music collections to any device — TV, phone, tablet, or browser — from anywhere in the world.

It is designed to be the successor to what home media servers should be in 2026: **free**, **private**, **polished**, and **performant at scale**.

**The defining promise of StreamVault is privacy and offline-first operation.** The application works completely and fully on a local network with zero internet access. Every feature that uses the internet is opt-in, explicitly labeled, and fully documented as to what data is sent and where. No data of any kind leaves the device without the user's explicit, informed consent.

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
| Silent data sharing with external services | Plex (reports playback, device info, library stats); Emby (usage analytics) |
| Internet required for metadata even when cached | Jellyfin (can fail silently on no-internet scan) |
| No granular control over what data leaves the device | All platforms — no per-feature internet toggle |
| UI/UX not polished enough for families | Jellyfin (community-acknowledged weakness) |
| Performance degrades past 50k items | All platforms (SQLite lock contention) |
| Not a real server (local player only) | Kodi |
| Legacy technology (DLNA only, Java) | Universal Media Server |
| No built-in watch-party without plugins | Plex, Emby |

### The Opportunity

Plex's 2025–2026 price escalation (lifetime pass: $119 → $249 → $749) triggered a mass migration wave. Jellyfin captured most migrants but users consistently request better UI polish, better performance at scale, and — increasingly — **stronger privacy guarantees**. No existing platform gives users a clear, transparent view of what data leaves their home. **StreamVault makes privacy a first-class feature, not an afterthought.**

---

## 3. Market Analysis

### Competitive Landscape

| Product | Core Model | License | Stack | DB | Key Weakness |
|---------|-----------|---------|-------|----|-------------|
| **Jellyfin** | Free, open-source | GPL-2 | C#/.NET 9 | SQLite/EFCore | UI polish, large-library perf, implicit internet calls |
| **Plex** | Freemium → Paywall | Proprietary | Unknown | SQLite | Aggressive monetization, data collection |
| **Kodi** | Free, open-source | GPL-2 | C++ | SQLite/MySQL | Not a server, local only |
| **Emby** | Freemium | Hybrid | C#/.NET 8 | SQLite | Middle ground, declining |
| **UMS** | Free, open-source | GPL-2 | Java | Unknown | Legacy DLNA only |
| **StreamVault** | Free, open-source | MIT | Go/React | PostgreSQL/SQLite | *To be built* |

### Market Trends (2026)

- Self-hosted media servers growing as streaming subscriptions fragment and increase in price
- Growing privacy consciousness — users want to know what software "phones home"
- Docker is de facto deployment standard for tech-forward users
- Samsung/LG Smart TVs now run web-standard clients (no dedicated app required)
- Hardware transcoding (NVENC, QSV, AMF) expected as free feature
- "Watch party" / SyncPlay features emerging as differentiator

---

## 4. Target Users

### Primary: The Power User (Early Adopters)
- **Profile:** Tech-savvy individual, comfortable with Docker, self-hosting, home server
- **Collection size:** 1,000–50,000+ media items
- **Pain point:** Plex pricing, Jellyfin UI rough edges, unclear data sharing in all platforms
- **Motivation:** Control, privacy, cost savings vs. streaming subscriptions
- **Devices:** Samsung/LG Smart TV, Android phone, iPad, PC browser

### Secondary: The Family Admin
- **Profile:** One technical person managing media for a household (2–6 people)
- **Collection size:** 200–5,000 media items
- **Pain point:** Wants Plex-quality polish without the $70/year fee and without worrying what the software is sending to the cloud
- **Motivation:** Easy to set up, family members can use it without help
- **Devices:** Smart TV (primary), phones (secondary)

### Tertiary: The Collector / Archivist
- **Profile:** Power user with massive library (50,000–500,000+ items), often in air-gapped or limited-internet environments
- **Collection size:** 50k–500k items (films, music, documentaries)
- **Pain point:** Database lock contention, slow scanning, inability to run fully offline
- **Motivation:** Performance at scale, complete offline operation, batch metadata editing

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

**G6 — True Offline Operation:** The application is 100% functional on a local network with zero internet connectivity. A user on an air-gapped home network loses nothing except optional enrichment features, which gracefully degrade.

**G7 — Radical Transparency:** Every byte that leaves the user's device is documented, labeled in the UI, and requires explicit opt-in. Users are never surprised by network activity.

---

## 6. Non-Goals

The following are explicitly out of scope (at least for MVP):

- **Live TV / DVR:** OTA antenna integration (Phase 3+ if at all)
- **Cloud storage backend:** S3/Dropbox as media source (future consideration)
- **Transcoding for 100+ concurrent users:** Not a streaming service, home-scale
- **Content purchasing/rental:** Not a storefront
- **DRM-protected content:** We do not support playback of DRM media
- **Social features (follows, public profiles):** Private/household-scoped only
- **Default-on telemetry of any kind:** Even anonymized usage stats are opt-in only

---

## 7. Core Principles

These guide every design and implementation decision:

1. **Free means free.** No paywalling features that were free yesterday. If a feature ships, it stays free.

2. **Offline-first.** Every core feature works without internet. Internet enriches; it never gatekeeps. If the internet is unavailable, the app behaves identically except optional enrichment features are silently skipped (not errored).

3. **Nothing leaves without consent.** No byte of data — metadata queries, artwork downloads, version checks, analytics, crash reports — is sent to any external server without the user explicitly enabling it. The default state of all internet-connected features is **OFF**.

4. **Transparency by design.** Every setting that involves external communication tells the user: what data is sent, to which service, and why. This information is displayed inline in the settings UI, not buried in a privacy policy.

5. **API-first.** Every feature the UI can do, the API can do. Third-party clients are first-class citizens.

6. **Local network first, remote as opt-in.** Works perfectly without internet. Remote access via user's own infrastructure (reverse proxy, Tailscale, VPN). No proprietary cloud relay.

7. **Fail gracefully.** If metadata lookup fails, media still plays. If transcoding fails, direct play is offered. If internet is down, nothing breaks. Nothing silently disappears.

8. **Boring technology choices.** Pick proven tools with large communities. No exotic frameworks that make contribution harder.

---

## 8. Feature Requirements

### Priority Notation
- **P0** — Must ship in MVP (blocker)
- **P1** — Should ship in MVP
- **P2** — Phase 2
- **P3** — Phase 3 / Future

### Tracking Status

| Symbol | Meaning |
|--------|---------|
| ✅ | Done |
| 🔄 | In progress / partial |
| ⬜ | Not started |

Three columns appear at the right of every feature table:
- **Impl** — backend + frontend code written
- **Test** — manually verified end-to-end
- **Git** — committed to main branch

---

### F0 — Offline-First & Privacy Controls *(NEW — P0)*

This is the foundational feature set. All other internet-dependent features (F2 Metadata, update checks, etc.) are subordinate to the controls defined here. **This section must be implemented before any feature that touches the network.**

#### F0.1 — Master Internet Toggle

| ID | Feature | Priority | Detail | Impl | Test | Git |
|----|---------|----------|--------|------|------|-----|
| F0.1.1 | Master "Allow Internet Access" toggle in Settings | P0 | Single switch. When OFF, StreamVault makes **zero outbound network requests** to any external host. All sub-settings below are automatically disabled and greyed out. | ⬜ | ⬜ | ⬜ |
| F0.1.2 | Master toggle defaults to OFF on first install | P0 | User must consciously enable internet access. Opt-in, not opt-out. | ⬜ | ⬜ | ⬜ |
| F0.1.3 | Master toggle state persisted across restarts | P0 | Stored in server config, not per-user preference. Admin-only setting. | ⬜ | ⬜ | ⬜ |
| F0.1.4 | Visual indicator in UI when internet is disabled | P0 | Subtle badge or status icon in the admin panel header showing "Offline Mode" | ⬜ | ⬜ | ⬜ |
| F0.1.5 | Network activity log (admin only) | P1 | Timestamped log of every outbound request StreamVault has made, including: destination host, purpose, data sent. Viewable in admin panel. | ⬜ | ⬜ | ⬜ |

#### F0.2 — Per-Feature Internet Sub-Settings

Each sub-setting is only visible and configurable when the master toggle (F0.1.1) is ON. Each sub-setting defaults to OFF when master toggle is first enabled.

| ID | Feature | Priority | External Service | Data Sent | Data Received | Impl | Test | Git |
|----|---------|----------|-----------------|-----------|---------------|------|------|-----|
| F0.2.1 | Movie/TV metadata lookup — TMDB | P0 | api.themoviedb.org | Movie/show title, release year | Title, description, genres, cast, rating, poster URL | ⬜ | ⬜ | ⬜ |
| F0.2.2 | Movie/TV artwork download — TMDB | P0 | image.tmdb.org | Artwork file paths (URLs from F0.2.1 results) | JPEG/PNG poster, backdrop, logo images | ⬜ | ⬜ | ⬜ |
| F0.2.3 | Music metadata lookup — MusicBrainz | P1 | musicbrainz.org | Artist name, album title, track title | Track metadata, album art URL | ⬜ | ⬜ | ⬜ |
| F0.2.4 | Music artwork download — Cover Art Archive | P1 | coverartarchive.org | MusicBrainz release ID | Album cover JPEG/PNG | ⬜ | ⬜ | ⬜ |
| F0.2.5 | Software update check | P1 | api.github.com (or self-hosted) | StreamVault version number | Latest release version number | ⬜ | ⬜ | ⬜ |
| F0.2.6 | Subtitle download (if subtitle provider plugin enabled) | P2 | Provider-specific (e.g., OpenSubtitles) | Movie/show title, language, hash | Subtitle file (.srt/.vtt) | ⬜ | ⬜ | ⬜ |
| F0.2.7 | Fan-art / extra artwork (TheTVDB, Fanart.tv) | P2 | thetvdb.com, fanart.tv | Show title or TVDB ID | Posters, banners, clearart | ⬜ | ⬜ | ⬜ |
| F0.2.8 | Crash/error reporting (opt-in) | P2 | Self-hosted or Sentry-compatible endpoint | Stack trace, OS info, StreamVault version — **never** media filenames, user data, or library info | Confirmation receipt | ⬜ | ⬜ | ⬜ |
| F0.2.9 | Anonymous usage statistics (opt-in) | P3 | Self-hosted stats endpoint | Feature usage counts, performance metrics — **no** filenames, titles, or user-identifying data | None | ⬜ | ⬜ | ⬜ |

#### F0.3 — Settings UI Requirements

| ID | Feature | Priority | Detail | Impl | Test | Git |
|----|---------|----------|--------|------|------|-----|
| F0.3.1 | Settings page: "Privacy & Internet" section is the first section in admin settings | P0 | Visual prominence signals that privacy matters | 🔄 | ⬜ | ✅ |
| F0.3.2 | Each sub-setting card shows: toggle, service name, what data is sent, what is received | P0 | No separate privacy policy document — information is inline | ⬜ | ⬜ | ⬜ |
| F0.3.3 | Each sub-setting card shows the external hostname(s) contacted | P0 | e.g., "Connects to: api.themoviedb.org, image.tmdb.org" | ⬜ | ⬜ | ⬜ |
| F0.3.4 | "What happens if I disable this?" explanation on each card | P0 | e.g., "Metadata will not be enriched. Movie posters will show a placeholder. You can add metadata manually." | ⬜ | ⬜ | ⬜ |
| F0.3.5 | Warning when enabling any sub-setting: "This will send data to [Service]. Continue?" | P1 | One-time confirmation per feature; remembered after first accept | ⬜ | ⬜ | ⬜ |
| F0.3.6 | "Disable All Internet Features" quick-action button | P0 | Equivalent to flipping master toggle OFF, but from sub-settings view | ⬜ | ⬜ | ⬜ |
| F0.3.7 | Privacy summary panel: shows count of enabled internet features and last internet activity timestamp | P1 | At-a-glance audit view for the admin | ⬜ | ⬜ | ⬜ |

#### F0.4 — Offline Operation Guarantees

The following must work with zero internet, no exceptions:

| Guarantee | Detail |
|-----------|--------|
| All media playback | Direct play and transcoded streams work fully offline |
| Library scanning | File system scan, fingerprinting, and organization works offline |
| All authentication | Login, session management, user management — local only |
| All search and filtering | Search works against locally stored metadata only |
| All playback state | Resume, history, watchlist — stored locally |
| Watch party / SyncPlay | Works on local network without internet |
| Admin panel | All server management functions work offline |
| Metadata from NFO files | Local `.nfo` files are read and used with no internet needed |
| Previously cached metadata | Once downloaded, metadata images and data are never re-fetched unless manually triggered |

#### F0.5 — Metadata Caching & Staleness Policy

| ID | Feature | Priority | Detail | Impl | Test | Git |
|----|---------|----------|--------|------|------|-----|
| F0.5.1 | All downloaded metadata (text + images) is persisted locally in the database/disk | P0 | After first fetch, no re-fetch unless user explicitly triggers "Refresh Metadata" | ⬜ | ⬜ | ⬜ |
| F0.5.2 | Internet sub-settings can be turned OFF after initial metadata fetch | P0 | User can do a one-time fetch, then go fully offline permanently | ⬜ | ⬜ | ⬜ |
| F0.5.3 | "Last fetched" timestamp shown per media item in admin detail view | P1 | Audit trail for when metadata was pulled | ⬜ | ⬜ | ⬜ |
| F0.5.4 | Manual "Refresh Metadata" action per item or per library (requires internet to be enabled) | P1 | User-initiated only; never automatic background re-fetch | ⬜ | ⬜ | ⬜ |
| F0.5.5 | Metadata images stored in local `data/artwork/` directory, not re-downloaded on each request | P0 | Served from local disk; external image URLs are never proxied live | ⬜ | ⬜ | ⬜ |

---

### F1 — Media Library Management

| ID | Feature | Priority | Notes | Impl | Test | Git |
|----|---------|----------|-------|------|------|-----|
| F1.1 | Scan local directories for media files | P0 | Recursive, configurable paths | ✅ | ✅ | ✅ |
| F1.2 | Incremental scan (detect new/changed/deleted files only) | P0 | Do not re-scan entire library | ⬜ | ⬜ | ⬜ |
| F1.3 | Real-time file watching (inotify / FSEvents / ReadDirectoryChanges) | P1 | Detect additions without manual trigger | ⬜ | ⬜ | ⬜ |
| F1.4 | Organize by Movies / TV Shows / Music | P0 | Separate library views per media type | ✅ | ✅ | ✅ |
| F1.5 | Multi-library support (e.g., separate "Kids" library) | P1 | Different directories → different libraries | ✅ | ✅ | ✅ |
| F1.6 | Library size display (item count, total storage) | P1 | Dashboard stat | ⬜ | ⬜ | ⬜ |
| F1.7 | Batch library operations (re-scan, re-match metadata) | P2 | Needed for large collections | ⬜ | ⬜ | ⬜ |
| F1.8 | Duplicate detection | P2 | Hash-based + fuzzy title match | ⬜ | ⬜ | ⬜ |

---

### F2 — Metadata

> **Privacy note:** All F2 features that contact external services are gated behind F0 sub-settings. If the relevant F0 toggle is OFF, the feature silently skips the network call and uses local data or a placeholder. No error is shown to end users — only a note in the admin scan log.

| ID | Feature | Priority | Internet Required | F0 Gate | Notes | Impl | Test | Git |
|----|---------|----------|------------------|---------|-------|------|------|-----|
| F2.1 | Auto-match movies/shows to TMDB | P0 | Yes (first fetch) | F0.2.1 | Skipped silently if disabled; item shows without metadata | ⬜ | ⬜ | ⬜ |
| F2.2 | IMDb as fallback metadata source | P1 | Yes (first fetch) | F0.2.1 (same toggle) | Secondary lookup when TMDB returns no match | ⬜ | ⬜ | ⬜ |
| F2.3 | Download poster, backdrop, fanart, logo images | P0 | Yes (first fetch) | F0.2.2 | Placeholder shown if disabled; locally cached once downloaded | ⬜ | ⬜ | ⬜ |
| F2.4 | Manual match override (user selects correct TMDB entry) | P0 | Yes (search) | F0.2.1 | Requires internet enabled at time of manual search | ⬜ | ⬜ | ⬜ |
| F2.5 | Filename hints for disambiguation (`{tmdb-12345}`) | P1 | No | — | Pure local; ID baked into filename, no lookup needed | ⬜ | ⬜ | ⬜ |
| F2.6 | Local NFO file support (Kodi-compatible metadata) | P0 | No | — | Fully offline; reads `.nfo` sidecar files | ⬜ | ⬜ | ⬜ |
| F2.7 | Music metadata via MusicBrainz | P1 | Yes (first fetch) | F0.2.3 | Skipped if disabled | ⬜ | ⬜ | ⬜ |
| F2.8 | Batch metadata re-match | P2 | Yes | F0.2.1 | Requires internet enabled; user-initiated only | ⬜ | ⬜ | ⬜ |
| F2.9 | User-submitted metadata corrections | P3 | Optional | F0.2.9 | Community feature; offline editing always supported | ⬜ | ⬜ | ⬜ |
| F2.10 | AI-assisted metadata fuzzy matching | P3 | Optional | TBD | Local model preferred if feasible | ⬜ | ⬜ | ⬜ |
| F2.11 | Manual metadata entry (title, year, description, genre) | P0 | No | — | Full offline metadata editing for any item | ⬜ | ⬜ | ⬜ |
| F2.12 | Manual artwork upload (user provides poster/backdrop image) | P0 | No | — | Upload local image file; stored in `data/artwork/` | ⬜ | ⬜ | ⬜ |

---

### F3 — Playback & Streaming

> All F3 features are fully offline. No internet connection required for any playback feature.

| ID | Feature | Priority | Notes | Impl | Test | Git |
|----|---------|----------|-------|------|------|-----|
| F3.1 | Direct play (no transcoding) when client supports format | P0 | Highest priority — zero CPU, lowest latency | ✅ | ✅ | ✅ |
| F3.2 | Software transcoding via FFmpeg | P0 | Fallback when direct play not possible | ✅ | ⬜ | ✅ |
| F3.3 | Hardware-accelerated transcoding (NVIDIA NVENC) | P0 | Essential for performance | ✅ | ⬜ | ✅ |
| F3.4 | Hardware transcoding: Intel QSV | P1 | NAS/integrated graphics support | ✅ | ⬜ | ✅ |
| F3.5 | Hardware transcoding: AMD AMF | P1 | AMD GPU support | ✅ | ⬜ | ✅ |
| F3.6 | Hardware transcoding: VA-API (Linux) | P1 | Open-source HW accel | ⬜ | ⬜ | ⬜ |
| F3.7 | HLS adaptive bitrate streaming | P0 | Standard for web/TV clients | ✅ | ⬜ | ✅ |
| F3.8 | Bitrate selection (auto + manual) | P1 | User can override quality | ⬜ | ⬜ | ⬜ |
| F3.9 | External subtitle support (.srt, .ass, .vtt) | P0 | Embedded + sidecar files | ⬜ | ⬜ | ⬜ |
| F3.10 | Subtitle burn-in for incompatible clients | P1 | Hardware-accelerated via FFmpeg | ⬜ | ⬜ | ⬜ |
| F3.11 | Resume playback from last position | P0 | Per-user, cross-device | 🔄 | ⬜ | ✅ |
| F3.12 | Multiple audio track selection | P0 | Per-stream | ⬜ | ⬜ | ⬜ |
| F3.13 | Multiple subtitle track selection | P0 | Per-stream | ⬜ | ⬜ | ⬜ |
| F3.14 | Playback speed control | P1 | 0.5x–2x | ⬜ | ⬜ | ⬜ |
| F3.15 | Chapter navigation (for video files with chapters) | P1 | Seek to chapter | ⬜ | ⬜ | ⬜ |
| F3.16 | Skip intro / skip credits detection | P2 | Local analysis only — no crowd-sourced fingerprint service | ⬜ | ⬜ | ⬜ |
| F3.17 | HDR tone-mapping (HDR → SDR for non-HDR displays) | P2 | FFmpeg tone-map filter | ⬜ | ⬜ | ⬜ |

---

### F4 — Web UI (Client)

> All F4 UI features are fully offline. The web UI is served by the local server — no CDN, no external font, no external analytics script, no tracking pixel.

| ID | Feature | Priority | Notes | Impl | Test | Git |
|----|---------|----------|-------|------|------|-----|
| F4.1 | Dashboard (recently added, continue watching, recommended) | P0 | Home screen | ✅ | ✅ | ✅ |
| F4.2 | Library grid view (posters) | P0 | Standard media browser | ✅ | ✅ | ✅ |
| F4.3 | Library list view | P1 | Alternative view mode | ⬜ | ⬜ | ⬜ |
| F4.4 | Search (title, actor, genre, year) | P0 | Instant search, no page refresh | ✅ | ✅ | ✅ |
| F4.5 | Filter & sort (genre, year, rating, resolution, language) | P1 | Faceted filtering | ⬜ | ⬜ | ⬜ |
| F4.6 | Movie detail page (metadata, cast, streams, related) | P0 | Rich detail view | ✅ | ✅ | ✅ |
| F4.7 | TV show detail page (seasons, episodes, progress) | P0 | Episode grid with watch status | ⬜ | ⬜ | ⬜ |
| F4.8 | Music artist / album / track views | P0 | Standard music library UI | ⬜ | ⬜ | ⬜ |
| F4.9 | Built-in video player | P0 | HLS.js or native video element | ✅ | ✅ | ✅ |
| F4.10 | Built-in audio player | P0 | With queue management | ⬜ | ⬜ | ⬜ |
| F4.11 | Mobile-responsive design | P0 | Works on phone browser | 🔄 | ⬜ | ✅ |
| F4.12 | Dark theme (default) | P0 | Standard for media apps | ✅ | ✅ | ✅ |
| F4.13 | Light theme | P2 | Optional toggle | ⬜ | ⬜ | ⬜ |
| F4.14 | Watch history per user | P1 | "Watched" badge, progress rings | ⬜ | ⬜ | ⬜ |
| F4.15 | Favorites / watchlist | P1 | User-curated lists | ⬜ | ⬜ | ⬜ |
| F4.16 | Admin panel (users, libraries, tasks, logs) | P0 | Server management UI | ✅ | ✅ | ✅ |
| F4.17 | Server dashboard (CPU, memory, transcode sessions, storage) | P1 | Real-time stats | 🔄 | ⬜ | ✅ |
| F4.18 | Keyboard shortcuts for player | P1 | Space=play/pause, arrow keys=seek | ⬜ | ⬜ | ⬜ |
| F4.19 | Chromecast / AirPlay support | P2 | Browser cast integration — local network only, no Google/Apple cloud relay | ⬜ | ⬜ | ⬜ |
| F4.20 | No external resources loaded by web UI | P0 | All fonts, icons, and scripts embedded in the single binary. No calls to Google Fonts, CDNs, or analytics from the browser. | ✅ | ✅ | ✅ |
| F4.21 | Privacy & Internet settings page (admin) | P0 | Implements F0.3 UI requirements | 🔄 | ⬜ | ✅ |
| F4.22 | Network activity log viewer (admin) | P1 | Implements F0.1.5 — readable log of all outbound requests | ⬜ | ⬜ | ⬜ |

---

### F5 — Authentication & Multi-User

> All authentication is local. No external identity provider is contacted unless explicitly configured by the admin.

| ID | Feature | Priority | Notes | Impl | Test | Git |
|----|---------|----------|-------|------|------|-----|
| F5.1 | Local username/password accounts | P0 | Built-in auth — no external service | ✅ | ✅ | ✅ |
| F5.2 | Admin role vs. standard user role | P0 | Access control | ✅ | ✅ | ✅ |
| F5.3 | Per-library access controls | P0 | Admin assigns which libraries a user sees | ⬜ | ⬜ | ⬜ |
| F5.4 | PIN-based parental controls | P1 | Restrict content rating per user | ⬜ | ⬜ | ⬜ |
| F5.5 | Invite link system (admin creates invite, user registers) | P1 | No self-registration by default | ⬜ | ⬜ | ⬜ |
| F5.6 | OAuth 2.0 / OIDC integration (optional) | P2 | When enabled, contacts Google/GitHub. Clearly labeled in F0 settings with data disclosure. | ⬜ | ⬜ | ⬜ |
| F5.7 | LDAP/Active Directory integration (optional) | P2 | Contacts LDAP server (local or remote). Labeled in F0 settings. | ⬜ | ⬜ | ⬜ |
| F5.8 | Session management (list active sessions, revoke) | P1 | Admin visibility | ⬜ | ⬜ | ⬜ |
| F5.9 | API token support | P1 | For third-party clients | ⬜ | ⬜ | ⬜ |
| F5.10 | Two-factor authentication (TOTP) | P2 | Local TOTP — no SMS, no cloud auth service | ⬜ | ⬜ | ⬜ |

---

### F6 — Remote Access

| ID | Feature | Priority | Notes | Impl | Test | Git |
|----|---------|----------|-------|------|------|-----|
| F6.1 | Works on local network without any cloud dependency | P0 | Core requirement | ✅ | ✅ | ✅ |
| F6.2 | HTTPS support (user-provided cert or self-signed) | P0 | Let's Encrypt option available but requires internet — optional, labeled in F0 | ⬜ | ⬜ | ⬜ |
| F6.3 | Reverse proxy documentation (nginx, Caddy, Traefik) | P0 | Guide for remote access setup | ⬜ | ⬜ | ⬜ |
| F6.4 | Base URL path prefix support (`/streamvault`) | P1 | For multi-app reverse proxy | ⬜ | ⬜ | ⬜ |
| F6.5 | Bandwidth throttle per user (remote streams) | P1 | Prevent crushing home upload bandwidth | ⬜ | ⬜ | ⬜ |
| F6.6 | Tailscale integration guidance | P1 | Zero-config VPN option — Tailscale itself contacts the internet but StreamVault does not relay any data | ⬜ | ⬜ | ⬜ |
| F6.7 | Built-in optional relay (self-hosted relay server) | P3 | For users who can't port-forward | ⬜ | ⬜ | ⬜ |
| F6.8 | Let's Encrypt automatic cert issuance | P1 | Internet required; gated behind F0 sub-setting "Let's Encrypt Certificate Renewal" — data sent: domain name to letsencrypt.org | ⬜ | ⬜ | ⬜ |

---

### Phase 2

#### F7 — Watch Together (SyncPlay)

| ID | Feature | Priority | Internet Required | Notes | Impl | Test | Git |
|----|---------|----------|------------------|-------|------|------|-----|
| F7.1 | Create watch party session (host + guests) | P2 | No — LAN only | WebSocket-based sync, local network | ⬜ | ⬜ | ⬜ |
| F7.2 | Sync play/pause/seek across all participants | P2 | No | Sub-second sync | ⬜ | ⬜ | ⬜ |
| F7.3 | Text chat during watch party | P2 | No | Sidebar chat, local only | ⬜ | ⬜ | ⬜ |
| F7.4 | Guest access (no account required for watch parties) | P3 | No | Link-based temp access | ⬜ | ⬜ | ⬜ |

#### F8 — Smart Recommendations

| ID | Feature | Priority | Internet Required | Notes | Impl | Test | Git |
|----|---------|----------|------------------|-------|------|------|-----|
| F8.1 | "More like this" (genre/tag-based similarity) | P2 | No | Computed from local metadata | ⬜ | ⬜ | ⬜ |
| F8.2 | Watch history-based recommendations | P2 | No | Local collaborative filtering | ⬜ | ⬜ | ⬜ |
| F8.3 | "New episodes" / "new additions" notifications | P2 | No | In-app, local event | ⬜ | ⬜ | ⬜ |
| F8.4 | AI-powered discovery (natural language: "show me 80s sci-fi") | P3 | Optional | Local model preferred; if cloud LLM used, gated behind F0 with full data disclosure | ⬜ | ⬜ | ⬜ |

#### F9 — Plugin System

| ID | Feature | Priority | Internet Required | Notes | Impl | Test | Git |
|----|---------|----------|------------------|-------|------|------|-----|
| F9.1 | Plugin SDK (Go interface-based) | P2 | No | Typed contracts for plugins | ⬜ | ⬜ | ⬜ |
| F9.2 | Metadata provider plugins | P2 | Plugin-dependent | Each plugin declares its internet requirements; displayed in F0 sub-settings when installed | ⬜ | ⬜ | ⬜ |
| F9.3 | Authentication provider plugins | P2 | Plugin-dependent | Same disclosure requirement | ⬜ | ⬜ | ⬜ |
| F9.4 | Scheduled task plugins | P2 | No | Custom automation | ⬜ | ⬜ | ⬜ |
| F9.5 | Plugin repository (curated community list) | P3 | Yes (browse) | Browsing the plugin catalog requires internet; installing and running a plugin does not unless the plugin itself requires it | ⬜ | ⬜ | ⬜ |
| F9.6 | Plugin internet declaration manifest | P2 | — | Every plugin must declare: `internet_required: true/false`, `external_hosts: [...]`, `data_sent: "..."` in its manifest. Displayed in the F0 settings when plugin is installed. | ⬜ | ⬜ | ⬜ |

---

### Phase 3

#### F10 — TV & Mobile Clients

| ID | Feature | Priority | Notes | Impl | Test | Git |
|----|---------|----------|-------|------|------|-----|
| F10.1 | AndroidTV / FireTV client app | P3 | Native or React Native | ⬜ | ⬜ | ⬜ |
| F10.2 | Apple TV (tvOS) client app | P3 | Native Swift | ⬜ | ⬜ | ⬜ |
| F10.3 | Roku client app | P3 | BrightScript | ⬜ | ⬜ | ⬜ |
| F10.4 | Samsung Tizen app | P3 | Or use web client | ⬜ | ⬜ | ⬜ |
| F10.5 | iOS native app | P3 | Swift / React Native | ⬜ | ⬜ | ⬜ |
| F10.6 | Android native app | P3 | Kotlin / React Native | ⬜ | ⬜ | ⬜ |
| F10.7 | Open client SDK (TypeScript) | P3 | For third-party developers | ⬜ | ⬜ | ⬜ |

#### F11 — Advanced Library Tools

| ID | Feature | Priority | Notes | Impl | Test | Git |
|----|---------|----------|-------|------|------|-----|
| F11.1 | Duplicate file detection & merge UI | P3 | Across all media types | ⬜ | ⬜ | ⬜ |
| F11.2 | Media health check (corrupted files, missing metadata) | P3 | Library audit tool | ⬜ | ⬜ | ⬜ |
| F11.3 | Storage analytics (largest files, codec breakdown) | P3 | Dashboard view | ⬜ | ⬜ | ⬜ |
| F11.4 | Automated file organization (rename/move by convention) | P3 | Optional, with preview | ⬜ | ⬜ | ⬜ |

---

## 9. Internet Data Flow Registry

This section is the authoritative reference for every external network call StreamVault can make. It is also the source of truth for the Privacy & Internet settings UI (F0.3).

> **Rule:** If a network destination is not in this table, the code must not make the call. Adding a new external call requires updating this table and the corresponding F0 sub-setting first (docs-before-code policy for privacy).

| ID | Feature | External Host(s) | Data Sent | Data Received | Default | F0 Sub-Setting |
|----|---------|-----------------|-----------|---------------|---------|----------------|
| NET-01 | TMDB Movie/TV Metadata | `api.themoviedb.org` | Title, year, language | Title, synopsis, genres, cast, rating, poster URL | OFF | F0.2.1 |
| NET-02 | TMDB Artwork Download | `image.tmdb.org` | Image URL path | JPEG/PNG file | OFF | F0.2.2 |
| NET-03 | MusicBrainz Lookup | `musicbrainz.org` | Artist, album, track title | Metadata, MBID | OFF | F0.2.3 |
| NET-04 | Cover Art Archive | `coverartarchive.org` | MusicBrainz release ID | Album cover image | OFF | F0.2.4 |
| NET-05 | Software Update Check | `api.github.com/repos/streamvault/...` | StreamVault version string | Latest release version | OFF | F0.2.5 |
| NET-06 | Let's Encrypt Cert | `acme-v02.api.letsencrypt.org` | Domain name, ACME challenge | TLS certificate | OFF | F6.8 |
| NET-07 | Subtitle Download | Provider-specific (plugin) | Title, language, file hash | Subtitle file | OFF | F0.2.6 |
| NET-08 | TheTVDB / Fanart.tv | `api4.thetvdb.com`, `webservice.fanart.tv` | Show name or TVDB ID | Artwork files | OFF | F0.2.7 |
| NET-09 | Crash Reporting | User-configurable endpoint | Stack trace, OS, version — **never** filenames or user data | Confirmation | OFF | F0.2.8 |
| NET-10 | Usage Statistics | Self-hosted endpoint | Aggregated feature usage counts — **never** filenames, titles, or user data | None | OFF | F0.2.9 |
| NET-11 | OAuth / OIDC | Provider-specific (e.g., `accounts.google.com`) | Auth code, client ID | ID token, user profile | OFF | F5.6 |

**What is never sent under any circumstance:**
- Media file names, paths, or hashes
- User names, emails, or passwords
- Watch history or playback data
- Library structure or item counts
- Device identifiers or IP addresses
- Any data to StreamVault's own servers (StreamVault has no cloud servers)

---

## 10. Technical Requirements

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
| HTTP client | All outbound HTTP calls go through a **single gated client** that checks the F0 master toggle and per-feature toggle before executing. If either is OFF, the call is dropped and logged. |

### HTTP Client Privacy Gate (Required Architecture)

All outbound requests must flow through a single internal `OutboundClient` struct that:

1. Checks `settings.InternetEnabled` (master toggle) — if false, returns error immediately
2. Checks the feature-specific toggle (e.g., `settings.TMDBEnabled`) — if false, returns error immediately
3. Logs the request to the network activity log (F0.1.5) before executing
4. Executes the request
5. Logs the response status

No code in the application is permitted to use `http.DefaultClient` or create its own `http.Client` directly. Linting rule enforces this at build time.

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
| External resource policy | **Zero external resources.** No Google Fonts, no CDN scripts, no analytics. All assets embedded via `embed.FS`. CSP header enforces `default-src 'self'`. |

### Packaging

- Frontend built as static assets, served by Go backend (embedded via `embed.FS`)
- Single binary deployment — no Node.js required at runtime
- Docker image: `streamvault/server:latest` (multi-arch: amd64, arm64)
- Docker Compose template included in repo

---

## 11. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      CLIENT LAYER                           │
│  Web Browser  │  Mobile Browser  │  TV App  │  3rd Party   │
│  (all assets served locally — no external scripts/fonts)    │
└───────┬───────┴────────┬─────────┴────┬─────┴──────┬───────┘
        │                │              │             │
        └────────────────┴──────────────┴─────────────┘
                                │
                         HTTPS / WSS  (local network)
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
│  │(TMDB,MBrainz)│  │ (scan, thumbs)│  │ (FFmpeg wrapper)  │ │
│  └──────┬───────┘  └───────────────┘  └───────────────────┘ │
│         │                                                     │
│  ┌──────┴──────────────────────────────────────────────────┐ │
│  │           PRIVACY GATE (OutboundClient)                 │ │
│  │  Master toggle check → Feature toggle check → Log → Go  │ │
│  └──────┬──────────────────────────────────────────────────┘ │
└─────────┼───────────────────────────────────────────────────┘
          │  (only if both toggles are ON)
          ▼
   ┌──────────────────────────────────────────┐
   │         INTERNET (optional)              │
   │  TMDB  │  MusicBrainz  │  GitHub  │ ...  │
   └──────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                      DATA LAYER                             │
│  PostgreSQL / SQLite          │  Local File System          │
│  (media, users, meta,         │  (media files, artwork,     │
│   settings, activity log)     │   transcode cache, NFOs)    │
└─────────────────────────────────────────────────────────────┘
```

### Key Architectural Decisions

**ADR-1: Monolithic server, modular code**
- Single deployable binary
- Internal package boundaries (auth, library, stream, metadata, tasks, privacy)
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

**ADR-6: Privacy Gate as a single choke point for all outbound traffic**
- All external HTTP calls go through one `OutboundClient`
- Enforced by Go linter rule banning direct `http.Client` usage
- Makes it architecturally impossible to accidentally add a "phone home" call
- Provides a single place to log, audit, and block all outbound traffic

**ADR-7: Offline-first data model**
- All metadata stored locally after first fetch; never fetched live on each request
- Artwork served from `data/artwork/` on local disk; no external image proxy
- All application assets (fonts, icons, JS) embedded — no runtime CDN dependency

---

## 12. API Requirements

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
  PATCH  /api/v1/items/:id            (manual metadata override — offline)
  POST   /api/v1/items/:id/artwork    (upload local artwork — offline)
  GET    /api/v1/items/:id/stream     (negotiate play URL)
  GET    /api/v1/items/:id/playback   (direct play or transcode URL)

TV Shows
  GET    /api/v1/shows/:id/seasons
  GET    /api/v1/shows/:id/seasons/:season/episodes

Users
  GET    /api/v1/users                (admin only)
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
  GET    /api/v1/tasks                (scheduled task list)
  POST   /api/v1/tasks/:id/run        (trigger manually)
  GET    /api/v1/tasks/:id/status

Server
  GET    /api/v1/server/info
  GET    /api/v1/server/stats         (CPU, memory, active sessions)

Privacy & Settings (admin only)
  GET    /api/v1/settings/privacy           (get all internet toggle states)
  PATCH  /api/v1/settings/privacy           (update toggles)
  GET    /api/v1/settings/privacy/activity  (network activity log)
  DELETE /api/v1/settings/privacy/activity  (clear activity log)

Streaming
  GET    /stream/:sessionId/index.m3u8    (HLS manifest)
  GET    /stream/:sessionId/:segment.ts   (HLS segment)
  GET    /direct/:itemId                  (direct play)
```

---

## 13. Security Requirements

| Requirement | Detail |
|-------------|--------|
| Authentication | JWT with short-lived access tokens (15 min) + refresh tokens (30 days) |
| Password storage | bcrypt (min cost 12) |
| HTTPS | Required for remote access; HTTP allowed on local network |
| CORS | Configurable allowed origins |
| Rate limiting | Login endpoint: 5 attempts per minute per IP |
| SQL injection | Parameterized queries only (sqlc enforces this) |
| XSS | React's default escaping + Content-Security-Policy headers |
| CSP | `default-src 'self'` — blocks all external resource loading from the browser |
| Clickjacking | `X-Frame-Options: DENY` |
| File traversal | All file paths validated against allowed library roots |
| API tokens | Scoped (read-only vs. full access), revocable per-token |
| Admin API | All admin endpoints require admin role claim in JWT |
| Telemetry | None by default. No opt-in until F0.2.8 is explicitly enabled. |
| Outbound HTTP | All outbound calls go through `OutboundClient` privacy gate — no exceptions |
| Embedded assets | All frontend assets embedded in binary; no runtime CDN fetches |

---

## 14. Performance Requirements

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
| Offline startup time (no internet, cold start) | < 5 seconds |
| Playback start when internet is OFF | Identical to when internet is ON — zero degradation |

---

## 15. Deployment Requirements

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
      # Internet is OFF by default — no env var needed to stay offline
    restart: unless-stopped
```

### Resource Requirements (Minimum)

| Scenario | CPU | RAM | Storage |
|----------|-----|-----|---------|
| Direct play only | 1 core | 512 MB | 1 GB (app) |
| Software transcode 1x | 2 cores | 1 GB | 5 GB (temp) |
| Hardware transcode 4x | Any (GPU req.) | 2 GB | 5 GB (temp) |

---

## 16. Competitive Differentiation

### Why StreamVault Over Alternatives

| Feature | StreamVault | Jellyfin | Plex | Emby |
|---------|------------|---------|------|------|
| 100% Free (all features) | ✅ | ✅ | ❌ | ❌ |
| Open source | ✅ (MIT) | ✅ (GPL) | ❌ | Partial |
| Hardware transcoding free | ✅ | ✅ | ❌ ($6.99/mo) | ❌ ($4.99/mo) |
| Remote streaming free | ✅ | ✅ | ❌ ($1.99/mo) | ✅ |
| Works 100% offline | ✅ | ⚠️ (partial) | ❌ | ⚠️ (partial) |
| All internet features default OFF | ✅ | ❌ | ❌ | ❌ |
| Inline disclosure of what data is sent | ✅ | ❌ | ❌ | ❌ |
| Network activity log | ✅ | ❌ | ❌ | ❌ |
| No external assets in web UI | ✅ | ⚠️ | ❌ | ⚠️ |
| PostgreSQL support | ✅ (native) | 🔜 (planned) | ❌ | ❌ |
| Modern Go backend | ✅ | ❌ (C#) | ❌ | ❌ |
| Sub-200ms search (50k lib) | ✅ (goal) | ⚠️ (degrades) | ⚠️ | ⚠️ |
| Single binary deployment | ✅ | ❌ | ❌ | ❌ |
| Watch party (SyncPlay) | ✅ (P2) | ✅ | ❌ (Plex Pass) | ❌ |
| Plugin internet manifest (transparency) | ✅ | ❌ | ❌ | ❌ |
| No telemetry (default) | ✅ | ✅ | ❌ | ❌ |

### Our Unique Bets

1. **Privacy Gate architecture** — the only media server where it is architecturally impossible to "phone home" without the user knowing
2. **Offline-first, internet-second** — internet enriches, never gatekeeps; full feature parity offline
3. **Inline data disclosure** — every setting shows what data is sent and why; no separate privacy policy document
4. **Go backend** — faster startup, lower memory than C#/.NET, single binary
5. **PostgreSQL native** — solves the SQLite scale wall all competitors hit
6. **MIT license** — more permissive than GPL; enables commercial client ecosystem

---

## 17. Success Metrics

### Phase 1 (MVP) — 6 months post-launch

| Metric | Target |
|--------|--------|
| GitHub Stars | 2,000+ |
| Docker Hub Pulls | 10,000+ |
| Active installations (telemetry opt-in) | 1,000+ |
| GitHub Issues (bugs) | < 50 open |
| Test coverage | > 70% |
| Time to first stream (new install) | < 15 minutes |
| % of users who keep internet OFF by choice | Tracked (privacy stat — not sent anywhere) |

### Phase 2 — 12 months

| Metric | Target |
|--------|--------|
| GitHub Stars | 10,000+ |
| Community Discord members | 2,000+ |
| Plugin count (community) | 10+ |
| Third-party client apps | 3+ |
| P99 search latency (100k lib) | < 200ms |

---

## 18. Open Questions

| # | Question | Owner | Target Date |
|---|----------|-------|-------------|
| Q1 | Product name — "StreamVault" or alternative? | Product | Before Phase 1 kickoff |
| Q2 | Go vs. Rust for backend? (Rust: faster, steeper curve) | Engineering | Architecture phase |
| Q3 | PostgreSQL migration path from SQLite — how to guide users? | Engineering | Phase 1 |
| Q4 | Plugin system: Go plugins (.so) vs. WebAssembly vs. HTTP sidecar? | Engineering | Phase 2 design |
| Q5 | TV client priority: AndroidTV vs. Apple TV vs. web-first? | Product | Phase 2 planning |
| Q6 | Opt-in telemetry endpoint: self-hosted? Which aggregation? | Engineering | Phase 1 |
| Q7 | Self-hosted relay for remote access: worth the infrastructure cost? | Engineering | Phase 2 |
| Q8 | LLM-powered natural language search: local model (llama.cpp) vs. opt-in cloud? | Engineering | Phase 3 |
| Q9 | Community metadata corrections: voting system or PR-based? | Product | Phase 3 |
| Q10 | Commercial support tier ever? Or fully community-funded (sponsorships)? | Business | Year 1 |
| Q11 | Should the privacy gate apply to LAN requests too (e.g., a plugin calling a local IP)? | Engineering/Privacy | Architecture phase |
| Q12 | How to handle TMDB API key — user provides own key, or we use a shared key (privacy concern)? | Engineering | Phase 1 |

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
- Silent telemetry and "phone home" behavior erodes trust with privacy-conscious users

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
8. **Privacy gate as architecture** — single choke point for all outbound calls, enforced at build time
