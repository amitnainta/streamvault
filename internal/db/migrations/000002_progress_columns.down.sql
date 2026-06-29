-- SQLite does not support DROP COLUMN before 3.35.0; recreate table
CREATE TABLE playback_progress_old AS SELECT user_id, item_id, position_ms, played_pct, is_watched, last_played_at FROM playback_progress;
DROP TABLE playback_progress;
ALTER TABLE playback_progress_old RENAME TO playback_progress;
