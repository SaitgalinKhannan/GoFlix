package main

import (
	"GoFlix/configs"
	"GoFlix/internal/app/auth"
	"GoFlix/internal/app/torrent"
	"GoFlix/internal/app/web"
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Main panic recovered: %v\n%s", r, debug.Stack())
		}
	}()

	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded:")
	log.Printf("  Port: %s", cfg.Port)
	log.Printf("  TorrentsStatesFile: %s", cfg.TorrentsStatesFile)
	log.Printf("  TorrentsDir: %s", cfg.TorrentsDir)
	log.Printf("  PieceCompletionDir: %s", cfg.PieceCompletionDir)
	log.Printf("  SpeedLimitsFile: %s", cfg.SpeedLimitsFile)
	log.Printf("  Initial rate limits: download=%d B/s upload=%d B/s", cfg.DownloadLimit, cfg.UploadLimit)
	log.Printf("  MaxConcurrentDownloads: %d", cfg.MaxConcurrentDownloads)
	log.Printf("  Auth user: %s", cfg.AuthUsername)
	if cfg.JellyfinURL == "" {
		log.Println("  Jellyfin hook disabled (JELLYFIN_URL not set)")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Persisted torrent state
	if err := torrent.InitStateFile(cfg.TorrentsStatesFile); err != nil {
		log.Fatalf("Failed to init torrent states file: %v", err)
	}

	// Torrent client
	torrentClient, err := torrent.NewClient(
		cfg.TorrentsDir,
		cfg.PieceCompletionDir,
		cfg.SpeedLimitsFile,
		torrent.SpeedLimits{Download: cfg.DownloadLimit, Upload: cfg.UploadLimit},
	)
	if err != nil {
		log.Fatal("Failed to init torrent client:", err)
	}

	stateManager := torrent.NewStateManager(cfg.TorrentsStatesFile)
	statusHub := torrent.NewStatusHub()
	torrentService := torrent.NewService(torrentClient, stateManager, statusHub, cfg.MaxConcurrentDownloads)

	// Jellyfin hook on completion
	eventHandler := torrent.NewEventHandler(stateManager, cfg.JellyfinURL, cfg.JellyfinToken)
	eventHandler.Start()

	// Background workers (broadcaster + scheduler)
	torrentService.StartBackgroundWorkers(sigChan)

	// Auth: hash the configured password once at startup.
	passwordHash, err := auth.HashPassword(cfg.AuthPassword)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	authService := auth.NewService(cfg.AuthUsername, passwordHash, []byte(cfg.JWTSecret), []byte("goflix"))

	server := web.NewServer(cfg, torrentService, authService)

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Shutdown goroutine panic: %v\n%s", r, debug.Stack())
			}
		}()

		<-sigChan
		log.Println("Shutting down...")

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		if err := torrentClient.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}

		stateManager.Stop()
		serverStopCtx()
	}()

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

	<-serverCtx.Done()
}
