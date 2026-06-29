-- Add duration_ms, completed, updated_at to playback_progress
-- (original schema only had position_ms, played_pct, is_watched, last_played_at)
ALTER TABLE playback_progress ADD COLUMN duration_ms  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE playback_progress ADD COLUMN completed    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE playback_progress ADD COLUMN updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
