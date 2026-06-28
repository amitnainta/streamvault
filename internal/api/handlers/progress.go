package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/api/middleware"
)

type ProgressHandler struct {
	db  *sql.DB
	log *zap.Logger
}

func NewProgressHandler(db *sql.DB, log *zap.Logger) *ProgressHandler {
	return &ProgressHandler{db: db, log: log}
}

func (h *ProgressHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	itemID := chi.URLParam(r, "id")

	var pos, dur float64
	var completed bool
	err := h.db.QueryRowContext(r.Context(),
		`SELECT position_ms, duration_ms, completed FROM playback_progress WHERE user_id=? AND item_id=?`,
		claims.UserID, itemID,
	).Scan(&pos, &dur, &completed)
	if err == sql.ErrNoRows {
		writeJSON(w, 200, map[string]any{"position_ms": 0, "duration_ms": 0, "completed": false})
		return
	}
	writeJSON(w, 200, map[string]any{"position_ms": pos, "duration_ms": dur, "completed": completed})
}

func (h *ProgressHandler) ReportProgress(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	itemID := chi.URLParam(r, "id")

	var req struct {
		PositionMs float64 `json:"position_ms"`
		DurationMs float64 `json:"duration_ms"`
		Completed  bool    `json:"completed"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid body")
		return
	}

	id := uuid.New().String()
	h.db.ExecContext(r.Context(),
		`INSERT INTO playback_progress(id, user_id, item_id, position_ms, duration_ms, completed, updated_at)
		 VALUES(?,?,?,?,?,?,?)
		 ON CONFLICT(user_id, item_id) DO UPDATE SET
		   position_ms=excluded.position_ms,
		   duration_ms=excluded.duration_ms,
		   completed=excluded.completed,
		   updated_at=excluded.updated_at`,
		id, claims.UserID, itemID, req.PositionMs, req.DurationMs, req.Completed, time.Now(),
	)
	w.WriteHeader(204)
}

// ListContinueWatching returns in-progress items for the home screen.
func (h *ProgressHandler) ListContinueWatching(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT p.item_id, p.position_ms, p.duration_ms, m.file_path, mt.title
		 FROM playback_progress p
		 JOIN media_items m ON m.id=p.item_id
		 LEFT JOIN metadata mt ON mt.id=p.item_id
		 WHERE p.user_id=? AND p.completed=0 AND p.position_ms > 0
		 ORDER BY p.updated_at DESC LIMIT 20`,
		claims.UserID,
	)
	if err != nil {
		writeError(w, 500, "db error")
		return
	}
	defer rows.Close()

	type item struct {
		ItemID     string  `json:"item_id"`
		PositionMs float64 `json:"position_ms"`
		DurationMs float64 `json:"duration_ms"`
		FilePath   string  `json:"file_path"`
		Title      string  `json:"title"`
	}
	var items []item
	for rows.Next() {
		var i item
		var title sql.NullString
		rows.Scan(&i.ItemID, &i.PositionMs, &i.DurationMs, &i.FilePath, &title)
		if title.Valid {
			i.Title = title.String
		}
		items = append(items, i)
	}
	if items == nil {
		items = []item{}
	}
	writeJSON(w, 200, items)
}
