package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/amitnainta/streamvault/internal/api/middleware"
	"github.com/amitnainta/streamvault/internal/auth"
	"github.com/amitnainta/streamvault/internal/model"
)

type AuthHandler struct {
	db  *sql.DB
	log *zap.Logger
}

func NewAuthHandler(db *sql.DB, log *zap.Logger) *AuthHandler {
	return &AuthHandler{db: db, log: log}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}

	var user model.User
	var hash string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, username, password_hash, role, is_enabled FROM users WHERE username=?`,
		req.Username,
	).Scan(&user.ID, &user.Username, &hash, &user.Role, &user.IsEnabled)

	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		writeError(w, 401, "invalid credentials")
		return
	}
	if err != nil {
		writeError(w, 500, "internal error")
		return
	}
	if !user.IsEnabled {
		writeError(w, 403, "account disabled")
		return
	}

	// Load library access
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT library_id FROM user_library_access WHERE user_id=?`, user.ID)
	defer rows.Close()
	for rows.Next() {
		var lid string
		rows.Scan(&lid)
		user.LibraryIDs = append(user.LibraryIDs, lid)
	}

	jwtSvc := auth.NewJWTServiceFromDB(h.db)
	pair, err := jwtSvc.Issue(user.ID, string(user.Role), user.LibraryIDs)
	if err != nil {
		writeError(w, 500, "token error")
		return
	}

	// Refresh token in HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "sv_refresh",
		Value:    pair.RefreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 3600,
	})

	// Update last_login
	h.db.ExecContext(r.Context(), `UPDATE users SET last_login_at=? WHERE id=?`, time.Now(), user.ID)

	writeJSON(w, 200, map[string]any{
		"access_token": pair.AccessToken,
		"user":         user,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("sv_refresh")
	if err != nil {
		writeError(w, 401, "no refresh token")
		return
	}

	jwtSvc := auth.NewJWTServiceFromDB(h.db)
	claims, err := jwtSvc.Validate(cookie.Value)
	if err != nil {
		writeError(w, 401, "invalid refresh token")
		return
	}

	// Re-fetch library access (may have changed)
	rows, _ := h.db.QueryContext(r.Context(),
		`SELECT library_id FROM user_library_access WHERE user_id=?`, claims.UserID)
	defer rows.Close()
	var libs []string
	for rows.Next() {
		var lid string
		rows.Scan(&lid)
		libs = append(libs, lid)
	}

	pair, err := jwtSvc.Issue(claims.UserID, claims.Role, libs)
	if err != nil {
		writeError(w, 500, "token error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "sv_refresh",
		Value:    pair.RefreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 3600,
	})

	writeJSON(w, 200, map[string]string{"access_token": pair.AccessToken})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sv_refresh",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.WriteHeader(204)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromCtx(r.Context())
	var user model.User
	h.db.QueryRowContext(r.Context(),
		`SELECT id, username, role, is_enabled, created_at FROM users WHERE id=?`,
		claims.UserID,
	).Scan(&user.ID, &user.Username, &user.Role, &user.IsEnabled, &user.CreatedAt)
	user.LibraryIDs = claims.LibraryIDs
	writeJSON(w, 200, user)
}

// EnsureAdminExists creates the default admin account on first startup if no users exist.
func EnsureAdminExists(db *sql.DB, log *zap.Logger) {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count > 0 {
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), 12)
	id := uuid.New().String()
	_, err := db.Exec(
		`INSERT INTO users(id, username, password_hash, role, is_enabled) VALUES(?,?,?,?,?)`,
		id, "admin", string(hash), "admin", 1,
	)
	if err != nil {
		log.Error("failed to create default admin", zap.Error(err))
		return
	}
	log.Info("created default admin account", zap.String("username", "admin"), zap.String("password", "admin"))
	log.Warn("CHANGE THE DEFAULT PASSWORD immediately after first login")
}
