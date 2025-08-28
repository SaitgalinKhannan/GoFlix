package main

import (
	"GoFlix/configs"
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

	log.Printf("Config loaded:\n")
	log.Printf("  Port: %s\n", cfg.Port)
	log.Printf("  TorrentsStatesFile: %s\n", cfg.TorrentsStatesFile)
	log.Printf("  TorrentsDir: %s\n", cfg.TorrentsDir)
	log.Printf("  PieceCompletionDir: %s\n", cfg.PieceCompletionDir)

	// Ожидаем сигнал для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Инициализация файла состояний торрентов
	if err := torrent.InitTorrentStatesFile(cfg.TorrentsStatesFile); err != nil {
		log.Fatalf("Failed to init torrent states file: %v", err)
	}

	// Инициализируем торрент-клиент
	torrentClient, err := torrent.NewClient(cfg.TorrentsDir, cfg.PieceCompletionDir)
	if err != nil {
		log.Fatal("Failed to init torrent client:", err)
	}

	sm := torrent.NewTorrentStateManager(cfg.TorrentsStatesFile)
	torrentService := torrent.NewService(torrentClient, sm)
	eventHandler := torrent.NewEventHandler(torrentService)
	eventHandler.Start()

	// Запускаем фоновые процессы
	torrentService.StartBackgroundWorkers(sigChan)

	// Создаем и запускаем веб-сервер
	server := web.NewServer(cfg, torrentService)

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Goroutine для graceful shutdown
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Shutdown goroutine panic: %v\n%s", r, debug.Stack())
			}
		}()

		<-sigChan
		log.Println("Shutting down...")

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Завершаем работу HTTP-сервера
		if serverShutdownErr := server.Shutdown(shutdownCtx); serverShutdownErr != nil {
			log.Printf("HTTP server Shutdown: %v", serverShutdownErr)
		}

		// Закрываем торрент-клиент с обработкой ошибки
		if torrentClientCloseErr := torrentClient.Close(); torrentClientCloseErr != nil {
			log.Printf("Error closing torrent client: %v", torrentClientCloseErr)
		}

		// Останавливаем компоненты
		eventHandler.Stop()
		sm.Stop()

		serverStopCtx()
	}()

	// Запускаем сервер
	if startErr := server.Start(); startErr != nil {
		log.Fatalf("Server failed to start: %v", startErr)
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}
