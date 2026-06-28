package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/transcode"
)

type StreamHandler struct {
	db     *sql.DB
	engine *transcode.Engine
	log    *zap.Logger
}

func NewStreamHandler(db *sql.DB, engine *transcode.Engine, log *zap.Logger) *StreamHandler {
	return &StreamHandler{db: db, engine: engine, log: log}
}

// StartSession picks between direct play and transcode based on client capability.
func (h *StreamHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")

	var filePath, container string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT file_path, container FROM media_items WHERE id=?`, itemID,
	).Scan(&filePath, &container)
	if err == sql.ErrNoRows {
		writeError(w, 404, "item not found")
		return
	}

	// Simple negotiation: if client sends ?direct=true and file is natively playable, redirect to /direct
	if r.URL.Query().Get("direct") == "true" && canDirectPlay(container) {
		writeJSON(w, 200, map[string]string{
			"type":    "direct",
			"url":     "/direct/" + itemID,
		})
		return
	}

	// Otherwise start HLS transcode session
	sessionID, err := h.engine.StartSession(itemID, filePath)
	if err != nil {
		h.log.Error("failed to start transcode session", zap.Error(err))
		writeError(w, 500, "transcode error")
		return
	}

	writeJSON(w, 200, map[string]string{
		"type":       "hls",
		"session_id": sessionID,
		"url":        "/stream/hls/" + sessionID + "/index.m3u8",
	})
}

// HLSManifest serves the HLS playlist for an active transcode session.
func (h *StreamHandler) HLSManifest(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	h.engine.Ping(sessionID)

	manifestPath, err := h.engine.ManifestPath(sessionID)
	if err != nil {
		writeError(w, 404, "session not found")
		return
	}

	// Wait up to 5s for the first segment to appear
	if !waitForFile(manifestPath, 5) {
		writeError(w, 503, "transcode not ready yet")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, manifestPath)
}

// HLSSegment serves an individual .ts segment.
func (h *StreamHandler) HLSSegment(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	segment := chi.URLParam(r, "segment")
	h.engine.Ping(sessionID)

	segPath, err := h.engine.SegmentPath(sessionID, segment)
	if err != nil {
		writeError(w, 404, "session not found")
		return
	}

	if !waitForFile(segPath, 10) {
		writeError(w, 503, "segment not ready")
		return
	}

	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "max-age=3600")
	http.ServeFile(w, r, segPath)
}

// DirectPlay serves the raw file for clients that can play it natively.
func (h *StreamHandler) DirectPlay(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")

	var filePath string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT file_path FROM media_items WHERE id=?`, itemID,
	).Scan(&filePath)
	if err == sql.ErrNoRows {
		writeError(w, 404, "item not found")
		return
	}

	http.ServeFile(w, r, filePath)
}

// ArtworkServe serves cached artwork images from disk.
func (h *StreamHandler) ArtworkServe(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	artType := chi.URLParam(r, "type") // poster | backdrop | thumb

	var filePath string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT file_path FROM artwork WHERE item_id=? AND art_type=?`, itemID, artType,
	).Scan(&filePath)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "max-age=86400")
	http.ServeFile(w, r, filePath)
}

// StopSession terminates an active transcode session and removes temp files.
func (h *StreamHandler) StopSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	h.engine.StopSession(sessionID)
	w.WriteHeader(204)
}

func canDirectPlay(container string) bool {
	switch container {
	case "mp4", "webm", "ogg", "mp3", "flac", "wav", "aac":
		return true
	}
	return false
}

func waitForFile(path string, timeoutSec int) bool {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

