package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/config"
	apimw "github.com/amitnainta/streamvault/internal/api/middleware"
	"github.com/amitnainta/streamvault/internal/api/handlers"
	"github.com/amitnainta/streamvault/internal/privacy"
	"github.com/amitnainta/streamvault/internal/scheduler"
	"github.com/amitnainta/streamvault/internal/transcode"
)

// Deps are all dependencies wired in main.go and injected here.
type Deps struct {
	Config     *config.Config
	DB         *sql.DB
	Privacy    *privacy.Settings
	Outbound   *privacy.OutboundClient
	Transcoder *transcode.Engine
	Scheduler  *scheduler.Scheduler
	Logger     *zap.Logger
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware ──────────────────────────────────────────────────
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(apimw.Logger(d.Logger))
	r.Use(apimw.Recover(d.Logger))
	r.Use(apimw.SecurityHeaders())
	r.Use(apimw.CORS(d.Config.Server.BaseURL))

	// ── Health check (no auth) ─────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// ── API v1 ────────────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public: auth
		authH := handlers.NewAuthHandler(d.DB, d.Logger)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(apimw.Auth(d.DB))

			r.Post("/auth/logout", authH.Logout)
			r.Get("/auth/me", authH.Me)

			// Libraries
			libraryH := handlers.NewLibraryHandler(d.DB, d.Logger)
			r.Get("/libraries", libraryH.List)
			r.Post("/libraries", libraryH.Create)
			r.Get("/libraries/{id}", libraryH.Get)
			r.Delete("/libraries/{id}", libraryH.Delete)
			r.Post("/libraries/{id}/scan", libraryH.Scan)

			// Media items
			itemH := handlers.NewItemHandler(d.DB, d.Logger)
			r.Get("/items", itemH.List)
			r.Get("/items/{id}", itemH.Get)
			r.Patch("/items/{id}", itemH.Update)
			r.Post("/items/{id}/artwork", itemH.UploadArtwork)

			// Playback
			streamH := handlers.NewStreamHandler(d.DB, d.Transcoder, d.Logger)
			r.Get("/items/{id}/playback", streamH.Negotiate)

			// TV Shows
			showH := handlers.NewShowHandler(d.DB, d.Logger)
			r.Get("/shows/{id}/seasons", showH.ListSeasons)
			r.Get("/shows/{id}/seasons/{season}/episodes", showH.ListEpisodes)

			// Users (admin) + self-service progress
			userH := handlers.NewUserHandler(d.DB, d.Logger)
			r.Get("/users", userH.List)           // admin only
			r.Post("/users", userH.Create)         // admin only
			r.Get("/users/{id}", userH.Get)
			r.Patch("/users/{id}", userH.Update)
			r.Delete("/users/{id}", userH.Delete)  // admin only
			r.Get("/users/{id}/progress", userH.GetProgress)
			r.Put("/users/{id}/progress/{itemId}", userH.UpdateProgress)
			r.Get("/users/{id}/history", userH.GetHistory)
			r.Get("/users/{id}/watchlist", userH.GetWatchlist)
			r.Post("/users/{id}/watchlist/{itemId}", userH.AddToWatchlist)
			r.Delete("/users/{id}/watchlist/{itemId}", userH.RemoveFromWatchlist)

			// Tasks (admin)
			taskH := handlers.NewTaskHandler(d.Scheduler, d.Logger)
			r.Get("/tasks", taskH.List)
			r.Post("/tasks/{id}/run", taskH.Run)
			r.Get("/tasks/{id}/status", taskH.Status)

			// Server stats (admin)
			serverH := handlers.NewServerHandler(d.Logger)
			r.Get("/server/info", serverH.Info)
			r.Get("/server/stats", serverH.Stats)

			// Privacy settings (admin)
			privacyH := handlers.NewPrivacyHandler(d.Privacy, d.Logger)
			r.Get("/settings/privacy", privacyH.Get)
			r.Patch("/settings/privacy", privacyH.Update)
			r.Get("/settings/privacy/activity", privacyH.ActivityLog)
			r.Delete("/settings/privacy/activity", privacyH.ClearActivityLog)
		})
	})

	// ── Streaming (authenticated via token query param) ───────────────────
	streamH := handlers.NewStreamHandler(d.DB, d.Transcoder, d.Logger)
	r.Get("/stream/{sessionId}/index.m3u8", streamH.HLSManifest)
	r.Get("/stream/{sessionId}/{segment}", streamH.HLSSegment)
	r.Get("/direct/{itemId}", streamH.DirectPlay)

	// ── Artwork (local, no external) ──────────────────────────────────────
	r.Get("/artwork/{itemId}/{artType}", handlers.ServeArtwork(d.Config.Storage.DataDir))

	// ── WebSocket ─────────────────────────────────────────────────────────
	r.Get("/ws", handlers.WebSocket(d.DB, d.Logger))

	// ── Frontend SPA (embedded static assets) ────────────────────────────
	r.Handle("/*", handlers.StaticFiles())

	return r
}
