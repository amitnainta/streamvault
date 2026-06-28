package api

import (
	"database/sql"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/amitnainta/streamvault/internal/api/handlers"
	apimw "github.com/amitnainta/streamvault/internal/api/middleware"
	"github.com/amitnainta/streamvault/internal/config"
	"github.com/amitnainta/streamvault/internal/privacy"
	"github.com/amitnainta/streamvault/internal/scheduler"
	"github.com/amitnainta/streamvault/internal/transcode"
	"github.com/amitnainta/streamvault/internal/ws"
)

// Deps are all dependencies wired in main.go and injected here.
type Deps struct {
	Config     *config.Config
	DB         *sql.DB
	Privacy    *privacy.Settings
	Outbound   *privacy.OutboundClient
	Transcoder *transcode.Engine
	Scheduler  *scheduler.Scheduler
	Hub        *ws.Hub
	WebFS      fs.FS // embedded web/dist
	Logger     *zap.Logger
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware ──────────────────────────────────────────────────
	r.Use(chimw.RealIP)
	r.Use(chimw.RequestID)
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
		authH := handlers.NewAuthHandler(d.DB, d.Logger)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(apimw.Auth(d.DB))

			r.Post("/auth/logout", authH.Logout)
			r.Get("/auth/me", authH.Me)

			// Libraries
			libH := handlers.NewLibraryHandler(d.DB, d.Logger)
			r.Get("/libraries", libH.List)
			r.Post("/libraries", libH.Create)
			r.Get("/libraries/{id}", libH.Get)
			r.Delete("/libraries/{id}", libH.Delete)
			r.Post("/libraries/{id}/scan", libH.Scan)

			// Media items
			itemH := handlers.NewItemHandler(d.DB, d.Logger)
			r.Get("/items", itemH.List)
			r.Get("/items/{id}", itemH.Get)
			r.Patch("/items/{id}", itemH.Update)
			r.Post("/items/{id}/artwork", itemH.UploadArtwork)

			// Playback session start
			streamH := handlers.NewStreamHandler(d.DB, d.Transcoder, d.Logger)
			r.Get("/items/{id}/playback", streamH.StartSession)
			r.Delete("/stream/sessions/{sessionId}", streamH.StopSession)

			// Progress
			progH := handlers.NewProgressHandler(d.DB, d.Logger)
			r.Get("/progress/continue-watching", progH.ListContinueWatching)
			r.Get("/progress/{id}", progH.GetProgress)
			r.Put("/progress/{id}", progH.ReportProgress)

			// Users (admin)
			userH := handlers.NewUserHandler(d.DB, d.Logger)
			r.Get("/users", userH.List)
			r.Post("/users", userH.Create)
			r.Patch("/users/{id}", userH.Update)
			r.Delete("/users/{id}", userH.Delete)

			// Tasks (admin)
			taskH := handlers.NewTaskHandler(d.Scheduler, d.Logger)
			r.Get("/tasks", taskH.List)
			r.Get("/tasks/{id}", taskH.Get)
			r.Post("/tasks/{id}/run", taskH.RunNow)

			// Server stats
			srvH := handlers.NewServerHandler(d.DB, d.Logger)
			r.Get("/server/info", srvH.Info)

			// Privacy settings (admin)
			privH := handlers.NewPrivacyHandler(d.DB, d.Privacy, d.Logger)
			r.Get("/settings/privacy", privH.GetSettings)
			r.Patch("/settings/privacy", privH.UpdateSettings)
			r.Get("/settings/privacy/activity", privH.GetActivityLog)
		})
	})

	// ── Streaming (auth via ?token= query param) ──────────────────────────
	streamH := handlers.NewStreamHandler(d.DB, d.Transcoder, d.Logger)
	r.With(apimw.Auth(d.DB)).Get("/stream/hls/{sessionId}/index.m3u8", streamH.HLSManifest)
	r.With(apimw.Auth(d.DB)).Get("/stream/hls/{sessionId}/{segment}", streamH.HLSSegment)
	r.With(apimw.Auth(d.DB)).Get("/direct/{id}", streamH.DirectPlay)

	// ── Artwork ────────────────────────────────────────────────────────────
	artH := handlers.NewStreamHandler(d.DB, d.Transcoder, d.Logger)
	r.Get("/artwork/{id}/{type}", artH.ArtworkServe)

	// ── WebSocket ──────────────────────────────────────────────────────────
	r.With(apimw.Auth(d.DB)).Get("/ws", d.Hub.ServeWS)

	// ── Frontend SPA (embedded static assets) ─────────────────────────────
	if d.WebFS != nil {
		r.Handle("/*", handlers.SPAHandler(d.WebFS))
	}

	return r
}
