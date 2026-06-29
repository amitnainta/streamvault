package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/api/middleware"
	"github.com/amitnainta/streamvault/internal/library"
	"github.com/amitnainta/streamvault/internal/model"
)

type LibraryHandler struct {
	db  *sql.DB
	log *zap.Logger
}

func NewLibraryHandler(db *sql.DB, log *zap.Logger) *LibraryHandler {
	return &LibraryHandler{db: db, log: log}
}

func (h *LibraryHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())

	var rows *sql.Rows
	var err error
	if claims.Role == "admin" {
		rows, err = h.db.QueryContext(r.Context(), `SELECT id, name, type, paths, created_at, last_scan FROM libraries ORDER BY name`)
	} else {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT l.id, l.name, l.type, l.paths, l.created_at, l.last_scan
			 FROM libraries l
			 JOIN user_library_access a ON a.library_id=l.id
			 WHERE a.user_id=? ORDER BY l.name`, claims.UserID)
	}
	if err != nil {
		writeError(w, 500, "db error")
		return
	}
	defer rows.Close()

	var libs []model.Library
	for rows.Next() {
		var lib model.Library
		var pathsJSON string
		rows.Scan(&lib.ID, &lib.Name, &lib.Type, &pathsJSON, &lib.CreatedAt, &lib.LastScan)
		json.Unmarshal([]byte(pathsJSON), &lib.Paths)
		libs = append(libs, lib)
	}
	if libs == nil {
		libs = []model.Library{}
	}
	writeJSON(w, 200, libs)
}

func (h *LibraryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var lib model.Library
	var pathsJSON string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, type, paths, created_at, last_scan FROM libraries WHERE id=?`, id,
	).Scan(&lib.ID, &lib.Name, &lib.Type, &pathsJSON, &lib.CreatedAt, &lib.LastScan)
	if err == sql.ErrNoRows {
		writeError(w, 404, "library not found")
		return
	}
	json.Unmarshal([]byte(pathsJSON), &lib.Paths)
	writeJSON(w, 200, lib)
}

func (h *LibraryHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	if claims.Role != "admin" {
		writeError(w, 403, "admin only")
		return
	}

	var req struct {
		Name  string           `json:"name"`
		Type  model.LibraryType `json:"type"`
		Paths []string         `json:"paths"`
	}
	if err := readJSON(r, &req); err != nil || req.Name == "" || len(req.Paths) == 0 {
		writeError(w, 400, "name and at least one path required")
		return
	}

	pathsJSON, _ := json.Marshal(req.Paths)
	id := uuid.New().String()
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO libraries(id, name, type, paths, created_at, updated_at) VALUES(?,?,?,?,?,?)`,
		id, req.Name, req.Type, string(pathsJSON), time.Now(), time.Now(),
	)
	if err != nil {
		writeError(w, 500, "failed to create library")
		return
	}

	lib := model.Library{ID: id, Name: req.Name, Type: req.Type, Paths: req.Paths}
	writeJSON(w, 201, lib)
}

func (h *LibraryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	if claims.Role != "admin" {
		writeError(w, 403, "admin only")
		return
	}
	id := chi.URLParam(r, "id")
	h.db.ExecContext(r.Context(), `DELETE FROM libraries WHERE id=?`, id)
	w.WriteHeader(204)
}

func (h *LibraryHandler) Scan(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	if claims.Role != "admin" {
		writeError(w, 403, "admin only")
		return
	}

	id := chi.URLParam(r, "id")
	var lib model.Library
	var pathsJSON string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, type, paths FROM libraries WHERE id=?`, id,
	).Scan(&lib.ID, &lib.Name, &lib.Type, &pathsJSON)
	if err == sql.ErrNoRows {
		writeError(w, 404, "library not found")
		return
	}
	json.Unmarshal([]byte(pathsJSON), &lib.Paths)

	scanner := library.NewScanner(h.db, h.log)
	go scanner.ScanLibrary(context.Background(), lib)

	writeJSON(w, 202, map[string]string{"status": "scan started", "library_id": id})
}
