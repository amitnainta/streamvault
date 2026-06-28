package handlers

import (
	"database/sql"
	"net/http"
	"runtime"
	"time"

	"go.uber.org/zap"
)

type ServerHandler struct {
	db      *sql.DB
	log     *zap.Logger
	startAt time.Time
}

func NewServerHandler(db *sql.DB, log *zap.Logger) *ServerHandler {
	return &ServerHandler{db: db, log: log, startAt: time.Now()}
}

func (h *ServerHandler) Info(w http.ResponseWriter, r *http.Request) {
	var mediaCount, libraryCount int
	h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM media_items WHERE is_missing=0`).Scan(&mediaCount)
	h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM libraries`).Scan(&libraryCount)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	writeJSON(w, 200, map[string]any{
		"version":       "0.1.0",
		"uptime_seconds": int(time.Since(h.startAt).Seconds()),
		"media_count":   mediaCount,
		"library_count": libraryCount,
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"memory_mb":     mem.Alloc / 1024 / 1024,
	})
}
