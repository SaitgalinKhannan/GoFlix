package web

import (
	"GoFlix/configs"
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

type Server struct {
	httpServer     *http.Server
	torrentService *torrent.Service
	config         *configs.Config
}

func NewServer(cfg *configs.Config, torrentService *torrent.Service) *Server {
	return &Server{
		torrentService: torrentService,
		config:         cfg,
	}
}

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
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// API routes
	router.Route("/api", func(api chi.Router) {
		api.Use(middleware.Timeout(30*time.Second), httphelpers.ErrorHandler)
		api.Route("/torrents", func(r chi.Router) {
			r.Get("/", handlers.GetTorrentsHandler(s.torrentService))
			r.Post("/", handlers.AddTorrentHandler(s.torrentService))
			r.Get("/{hash}/pause", handlers.PauseTorrentHandler(s.torrentService))
			r.Get("/{hash}/resume", handlers.ResumeTorrentHandler(s.torrentService))
			r.Get("/{hash}", handlers.GetTorrentHandler(s.torrentService))
			r.Delete("/{hash}", handlers.DeleteTorrentHandler(s.torrentService))
			r.Post("/{hash}/convert", handlers.ConvertTorrentHandler(s.torrentService))
		})
		api.Get("/files/tree", handlers.GetFilesTreeHandler(s.config))
		api.Get("/files", handlers.GetFilesHandler(s.config))
		api.Get("/video", handlers.VideoHandler(s.config))
		api.Get("/health", handlers.HealthCheck())
	})

	// WebSocket and static files
	router.Get("/ws", handlers.HandleWebSocket(s.torrentService))
	router.Get("/starfield/*", handlers.StarfieldHandler("./web"))

	return router
}

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

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}
