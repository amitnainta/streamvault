package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/privacy"
)

type PrivacyHandler struct {
	db       *sql.DB
	settings *privacy.Settings
	log      *zap.Logger
}

func NewPrivacyHandler(db *sql.DB, settings *privacy.Settings, log *zap.Logger) *PrivacyHandler {
	return &PrivacyHandler{db: db, settings: settings, log: log}
}

func (h *PrivacyHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `SELECT key, value FROM settings WHERE key LIKE 'internet.%'`)
	if err != nil {
		writeError(w, 500, "db error")
		return
	}
	defer rows.Close()

	result := map[string]bool{}
	for rows.Next() {
		var key, val string
		rows.Scan(&key, &val)
		result[key] = val == "true"
	}
	writeJSON(w, 200, result)
}

func (h *PrivacyHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var patch map[string]bool
	if err := readJSON(r, &patch); err != nil {
		writeError(w, 400, "invalid body")
		return
	}

	for key, val := range patch {
		v := "false"
		if val {
			v = "true"
		}
		h.db.ExecContext(r.Context(),
			`INSERT INTO settings(key, value, updated_at) VALUES(?,?,?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
			key, v, time.Now(),
		)
	}

	// Reload in-memory toggles
	h.settings.Reload()
	w.WriteHeader(204)
}

func (h *PrivacyHandler) GetActivityLog(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, feature, url, direction, bytes, status_code, blocked, occurred_at
		 FROM network_activity_log ORDER BY occurred_at DESC LIMIT 200`)
	if err != nil {
		writeError(w, 500, "db error")
		return
	}
	defer rows.Close()

	type entry struct {
		ID         string    `json:"id"`
		Feature    string    `json:"feature"`
		URL        string    `json:"url"`
		Direction  string    `json:"direction"`
		Bytes      int64     `json:"bytes"`
		StatusCode int       `json:"status_code"`
		Blocked    bool      `json:"blocked"`
		OccurredAt time.Time `json:"occurred_at"`
	}
	var log []entry
	for rows.Next() {
		var e entry
		rows.Scan(&e.ID, &e.Feature, &e.URL, &e.Direction, &e.Bytes, &e.StatusCode, &e.Blocked, &e.OccurredAt)
		log = append(log, e)
	}
	if log == nil {
		log = []entry{}
	}
	writeJSON(w, 200, log)
}
