package web

import (
	"GoFlix/configs"
	"GoFlix/internal/app/auth"
	"GoFlix/internal/app/torrent"
	"GoFlix/internal/app/web/handlers"
	"GoFlix/internal/pkg/httphelpers"
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server is the HTTP server exposing the torrent-service API.
type Server struct {
	httpServer     *http.Server
	torrentService *torrent.Service
	authService    *auth.Service
	config         *configs.Config
}

// NewServer creates a new web server.
func NewServer(cfg *configs.Config, torrentService *torrent.Service, authService *auth.Service) *Server {
	return &Server{
		torrentService: torrentService,
		authService:    authService,
		config:         cfg,
	}
}

// SetupRouter builds the HTTP router.
func (s *Server) SetupRouter() http.Handler {
	router := chi.NewRouter()

	// Global middleware
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Public auth endpoints — no token required.
	router.Route("/api/v1/auth", func(a chi.Router) {
		a.Use(httphelpers.ErrorHandler)
		a.Post("/login", handlers.LoginHandler(s.authService))
		a.Post("/refresh", handlers.RefreshHandler(s.authService))
	})

	// API v1 — authenticated (JWT access token)
	router.Route("/api/v1", func(v1 chi.Router) {
		v1.Use(httphelpers.ErrorHandler)
		v1.Use(JWTAuth(s.authService))

		v1.Route("/torrents", func(r chi.Router) {
			r.Get("/", handlers.GetTorrentsHandler(s.torrentService))
			r.Post("/", handlers.AddTorrentHandler(s.torrentService))
			r.Get("/{id}", handlers.GetTorrentHandler(s.torrentService))
			r.Delete("/{id}", handlers.DeleteTorrentHandler(s.torrentService))
			r.Post("/{id}/start", handlers.StartSeedingHandler(s.torrentService))
			r.Post("/{id}/stop", handlers.StopSeedingHandler(s.torrentService))
			r.Post("/{id}/recheck", handlers.RecheckHandler(s.torrentService))
		})

		v1.Get("/events", handlers.SSEHandler(s.torrentService))
	})

	// Misc endpoints (no auth) — file browsing + health
	router.Route("/api", func(api chi.Router) {
		api.Use(middleware.Timeout(30*time.Second), httphelpers.ErrorHandler)
		api.Get("/files/tree", handlers.GetFilesTreeHandler(s.config))
		api.Get("/files", handlers.GetFilesHandler(s.config))
		api.Get("/health", handlers.HealthCheck())
	})

	return router
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	router := s.SetupRouter()

	s.httpServer = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: router,
	}

	log.Printf("Server starting at :%s", s.config.Port)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
