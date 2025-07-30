package main

import (
	"GoFlix/internal/app/torrent"
	"GoFlix/internal/app/web/handlers"
	"GoFlix/internal/pkg/httplog"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Main panic recovered: %v\n%s", r, debug.Stack())
		}
	}()

	// Хранилище торрентов
	clientBaseDir := "torrents"
	// Хранилище метаданных торрентов
	pieceCompletionDir := "torrent_data"
	// Инициализируем торрент-клиент
	torrentClient, err := torrent.NewClient(clientBaseDir, pieceCompletionDir)
	if err != nil {
		log.Fatal("Failed to init torrent client:", err)
	}

	router := chi.NewRouter()
	// global middleware:
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	// Все API-роуты под префиксом /api
	router.Route("/api", func(api chi.Router) {
		api.Use(middleware.Timeout(30*time.Second), httplog.ErrorHandler)
		api.Post("/torrents/add", handlers.AddTorrentHandler(torrentClient))
		api.Get("/torrents/all", handlers.GetTorrentsHandler(torrentClient))
		api.Get("/files/tree", handlers.GetFilesTreeHandler())
		api.Get("/files", handlers.GetFilesHandler())
		api.Get("/health", handlers.HealthCheck(torrentClient))
	})
	// WebSocket остаётся на корне (без префикса)
	router.Get("/ws", handlers.HandleWebSocket(torrentClient))

	// Создаем HTTP-сервер
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Ожидаем сигнал для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Shutdown goroutine panic: %v\n%s", r, debug.Stack())
			}
		}()

		<-sigChan
		log.Println("Shutting down...")

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, _ := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Завершаем работу HTTP-сервера
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}

		// Закрываем торрент-клиент с обработкой ошибки
		if err := torrentClient.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}

		serverStopCtx()
	}()

	log.Println("Server started at :8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ListenAndServe(): %v", err)
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}
