-- ─────────────────────────────────────────────
-- LIBRARIES
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS libraries (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK(type IN ('movie','show','music')),
    paths      TEXT NOT NULL,  -- JSON array of directory paths
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_scan  TIMESTAMP
);

-- ─────────────────────────────────────────────
-- MEDIA ITEMS
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS media_items (
    id             TEXT PRIMARY KEY,
    library_id     TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    type           TEXT NOT NULL CHECK(type IN ('movie','episode','track')),
    file_path      TEXT NOT NULL UNIQUE,
    file_size      INTEGER NOT NULL,
    file_hash      TEXT,
    duration_ms    INTEGER,
    added_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    video_codec    TEXT,
    video_width    INTEGER,
    video_height   INTEGER,
    video_bitrate  INTEGER,
    hdr_format     TEXT,
    audio_codec    TEXT,
    audio_channels INTEGER,
    audio_language TEXT,
    container      TEXT
);

CREATE INDEX IF NOT EXISTS idx_media_items_library ON media_items(library_id);
CREATE INDEX IF NOT EXISTS idx_media_items_type    ON media_items(type);

-- ─────────────────────────────────────────────
-- METADATA
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS metadata (
    id                  TEXT PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    title               TEXT NOT NULL,
    sort_title          TEXT,
    original_title      TEXT,
    year                INTEGER,
    description         TEXT,
    tagline             TEXT,
    genres              TEXT,  -- JSON array
    studios             TEXT,  -- JSON array
    rating              REAL,
    content_rating      TEXT,
    language            TEXT,
    country             TEXT,
    tmdb_id             INTEGER,
    imdb_id             TEXT,
    tvdb_id             INTEGER,
    mbid                TEXT,
    metadata_source     TEXT,
    metadata_fetched_at TIMESTAMP,
    is_manually_edited  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_metadata_title      ON metadata(title);
CREATE INDEX IF NOT EXISTS idx_metadata_year       ON metadata(year);
CREATE INDEX IF NOT EXISTS idx_metadata_sort_title ON metadata(sort_title);
CREATE INDEX IF NOT EXISTS idx_metadata_tmdb_id    ON metadata(tmdb_id);

-- ─────────────────────────────────────────────
-- TV SHOWS / SEASONS / EPISODES
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS shows (
    id             TEXT PRIMARY KEY,
    library_id     TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    tmdb_id        INTEGER,
    title          TEXT NOT NULL,
    sort_title     TEXT,
    year           INTEGER,
    description    TEXT,
    genres         TEXT,
    rating         REAL,
    content_rating TEXT,
    status         TEXT,
    added_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS seasons (
    id          TEXT PRIMARY KEY,
    show_id     TEXT NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    season_num  INTEGER NOT NULL,
    title       TEXT,
    description TEXT,
    year        INTEGER,
    UNIQUE(show_id, season_num)
);

CREATE TABLE IF NOT EXISTS episodes (
    id          TEXT PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    show_id     TEXT NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    season_id   TEXT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    season_num  INTEGER NOT NULL,
    episode_num INTEGER NOT NULL
);

-- ─────────────────────────────────────────────
-- MUSIC
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS artists (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    mbid        TEXT,
    description TEXT,
    added_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS albums (
    id          TEXT PRIMARY KEY,
    artist_id   TEXT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    year        INTEGER,
    mbid        TEXT,
    description TEXT
);

CREATE TABLE IF NOT EXISTS tracks (
    id        TEXT PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    album_id  TEXT NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    track_num INTEGER,
    disc_num  INTEGER NOT NULL DEFAULT 1,
    title     TEXT NOT NULL
);

-- ─────────────────────────────────────────────
-- ARTWORK
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS artwork (
    id         TEXT PRIMARY KEY,
    item_id    TEXT NOT NULL,
    item_type  TEXT NOT NULL,
    art_type   TEXT NOT NULL CHECK(art_type IN ('poster','backdrop','logo','thumb','cover')),
    file_path  TEXT NOT NULL,
    source     TEXT,
    width      INTEGER,
    height     INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_artwork_item ON artwork(item_id, item_type, art_type);

-- ─────────────────────────────────────────────
-- MEDIA STREAMS (audio / subtitle tracks)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS media_streams (
    id          TEXT PRIMARY KEY,
    item_id     TEXT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    stream_type TEXT NOT NULL CHECK(stream_type IN ('audio','subtitle')),
    index_num   INTEGER NOT NULL,
    codec       TEXT,
    language    TEXT,
    title       TEXT,
    is_default  INTEGER NOT NULL DEFAULT 0,
    is_forced   INTEGER NOT NULL DEFAULT 0,
    is_external INTEGER NOT NULL DEFAULT 0,
    file_path   TEXT
);

CREATE INDEX IF NOT EXISTS idx_streams_item ON media_streams(item_id);

-- ─────────────────────────────────────────────
-- USERS
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'viewer' CHECK(role IN ('admin','viewer')),
    is_enabled    INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_library_access (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, library_id)
);

-- ─────────────────────────────────────────────
-- PLAYBACK STATE
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS playback_progress (
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id        TEXT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    position_ms    INTEGER NOT NULL DEFAULT 0,
    played_pct     REAL NOT NULL DEFAULT 0,
    is_watched     INTEGER NOT NULL DEFAULT 0,
    last_played_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_progress_user ON playback_progress(user_id, last_played_at DESC);

CREATE TABLE IF NOT EXISTS watchlist (
    user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id  TEXT NOT NULL,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, item_id)
);

-- ─────────────────────────────────────────────
-- API TOKENS
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS api_tokens (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    scope        TEXT NOT NULL DEFAULT 'read' CHECK(scope IN ('read','full')),
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    expires_at   TIMESTAMP
);

-- ─────────────────────────────────────────────
-- PRIVACY & SETTINGS
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- All internet features default to OFF
INSERT OR IGNORE INTO settings(key, value) VALUES
    ('internet_enabled',         'false'),
    ('tmdb_metadata_enabled',    'false'),
    ('tmdb_artwork_enabled',     'false'),
    ('musicbrainz_enabled',      'false'),
    ('cover_art_enabled',        'false'),
    ('update_check_enabled',     'false'),
    ('lets_encrypt_enabled',     'false'),
    ('crash_reporting_enabled',  'false'),
    ('usage_stats_enabled',      'false'),
    ('jwt_secret',               ''),
    ('setup_complete',           'false');

-- ─────────────────────────────────────────────
-- NETWORK ACTIVITY LOG
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS network_activity_log (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    feature      TEXT NOT NULL,
    url          TEXT NOT NULL,
    blocked      INTEGER NOT NULL,
    block_reason TEXT,
    status_code  INTEGER,
    duration_ms  INTEGER
);

CREATE INDEX IF NOT EXISTS idx_activity_log_ts ON network_activity_log(timestamp DESC);

-- ─────────────────────────────────────────────
-- FULL-TEXT SEARCH (SQLite FTS5)
-- ─────────────────────────────────────────────
CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
    item_id  UNINDEXED,
    title,
    original_title,
    description,
    genres
);
