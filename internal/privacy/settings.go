package privacy

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"
)

// SettingsReader is consumed by OutboundClient.
type SettingsReader interface {
	InternetEnabled() bool
	FeatureEnabled(f Feature) bool
	RecordActivity(e ActivityEntry)
}

// ActivityLogger records outbound network activity.
type ActivityLogger interface {
	RecordActivity(e ActivityEntry)
}

// Settings loads and caches privacy toggles from the DB.
// Admin API calls Reload() after any toggle change.
type Settings struct {
	mu      sync.RWMutex
	db      *sql.DB
	toggles map[string]bool
}

func NewSettings(db *sql.DB) *Settings {
	s := &Settings{db: db, toggles: make(map[string]bool)}
	s.load()
	return s
}

func (s *Settings) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT key, value FROM settings WHERE key LIKE '%_enabled'`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var key, val string
		rows.Scan(&key, &val)
		var b bool
		json.Unmarshal([]byte(val), &b)
		s.toggles[key] = b
	}
}

func (s *Settings) Reload() { s.load() }

func (s *Settings) InternetEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.toggles["internet_enabled"]
}

func (s *Settings) FeatureEnabled(f Feature) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.toggles[string(f)+"_enabled"]
}

func (s *Settings) SetToggle(key string, enabled bool) error {
	val, _ := json.Marshal(enabled)
	_, err := s.db.Exec(
		`INSERT INTO settings(key, value, updated_at) VALUES(?,?,?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, string(val), time.Now(),
	)
	if err == nil {
		s.load()
	}
	return err
}

func (s *Settings) RecordActivity(e ActivityEntry) {
	s.db.Exec(
		`INSERT INTO network_activity_log(timestamp,feature,url,blocked,block_reason,status_code,duration_ms)
		 VALUES(?,?,?,?,?,?,?)`,
		e.Timestamp, string(e.Feature), e.URL, e.Blocked, e.BlockReason, e.StatusCode, e.DurationMs,
	)
}

// GetActivityLog returns the most recent N entries.
func (s *Settings) GetActivityLog(limit int) ([]ActivityEntry, error) {
	rows, err := s.db.Query(
		`SELECT timestamp,feature,url,blocked,block_reason,status_code,duration_ms
		 FROM network_activity_log ORDER BY timestamp DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []ActivityEntry
	for rows.Next() {
		var e ActivityEntry
		var feature string
		rows.Scan(&e.Timestamp, &feature, &e.URL, &e.Blocked, &e.BlockReason, &e.StatusCode, &e.DurationMs)
		e.Feature = Feature(feature)
		entries = append(entries, e)
	}
	return entries, nil
}
