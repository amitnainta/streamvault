package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/amitnainta/streamvault/internal/auth"
)

type ctxKey string

const ClaimsKey ctxKey = "claims"

// Auth validates the Bearer JWT on every protected route.
// Also accepts token via ?token= query param for WebSocket + stream URLs.
func Auth(db *sql.DB) func(http.Handler) http.Handler {
	jwtSvc := auth.NewJWTServiceFromDB(db)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := jwtSvc.Validate(tokenStr)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin returns 403 if the caller is not an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromCtx(r.Context())
		if claims == nil || claims.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ClaimsFromCtx(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(ClaimsKey).(*auth.Claims)
	return c
}

func extractToken(r *http.Request) string {
	// 1. Authorization: Bearer <token>
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// 2. ?token= (for WebSocket + HLS stream URLs)
	return r.URL.Query().Get("token")
}
