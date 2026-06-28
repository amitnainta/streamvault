package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/amitnainta/streamvault/internal/api/middleware"
	"github.com/amitnainta/streamvault/internal/model"
)

type UserHandler struct {
	db  *sql.DB
	log *zap.Logger
}

func NewUserHandler(db *sql.DB, log *zap.Logger) *UserHandler {
	return &UserHandler{db: db, log: log}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, username, role, is_enabled, created_at, last_login_at FROM users ORDER BY username`)
	if err != nil {
		writeError(w, 500, "db error")
		return
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		rows.Scan(&u.ID, &u.Username, &u.Role, &u.IsEnabled, &u.CreatedAt, &u.LastLoginAt)
		users = append(users, u)
	}
	if users == nil {
		users = []model.User{}
	}
	writeJSON(w, 200, users)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := readJSON(r, &req); err != nil || req.Username == "" || req.Password == "" {
		writeError(w, 400, "username and password required")
		return
	}
	if req.Role != "admin" && req.Role != "viewer" {
		req.Role = "viewer"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		writeError(w, 500, "password error")
		return
	}

	id := uuid.New().String()
	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO users(id, username, password_hash, role, is_enabled, created_at, updated_at) VALUES(?,?,?,?,1,?,?)`,
		id, req.Username, string(hash), req.Role, time.Now(), time.Now())
	if err != nil {
		writeError(w, 409, "username already exists")
		return
	}

	writeJSON(w, 201, model.User{ID: id, Username: req.Username, Role: model.Role(req.Role), IsEnabled: true})
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	// Users can update their own password; admins can change anything
	if claims.Role != "admin" && claims.UserID != id {
		writeError(w, 403, "forbidden")
		return
	}

	var req struct {
		Password  *string `json:"password"`
		Role      *string `json:"role"`
		IsEnabled *bool   `json:"is_enabled"`
	}
	readJSON(r, &req)

	if req.Password != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte(*req.Password), 12)
		h.db.ExecContext(r.Context(), `UPDATE users SET password_hash=?,updated_at=? WHERE id=?`, string(hash), time.Now(), id)
	}
	if req.Role != nil && claims.Role == "admin" {
		h.db.ExecContext(r.Context(), `UPDATE users SET role=?,updated_at=? WHERE id=?`, *req.Role, time.Now(), id)
	}
	if req.IsEnabled != nil && claims.Role == "admin" {
		h.db.ExecContext(r.Context(), `UPDATE users SET is_enabled=?,updated_at=? WHERE id=?`, *req.IsEnabled, time.Now(), id)
	}

	w.WriteHeader(204)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	if claims.UserID == id {
		writeError(w, 400, "cannot delete yourself")
		return
	}
	h.db.ExecContext(r.Context(), `DELETE FROM users WHERE id=?`, id)
	w.WriteHeader(204)
}
