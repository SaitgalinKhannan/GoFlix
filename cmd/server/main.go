package main

import (
	"GoFlix/configs"
	"GoFlix/internal/app/torrent"
	"GoFlix/internal/app/web/handlers"
	"GoFlix/internal/pkg/filehelpers"
	"GoFlix/internal/pkg/httphelpers"
	"context"
	"encoding/json"
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

	// Хранилище состояний торрентов
	torrentStates := cfg.TorrentsStatesFile
	// Проверяем, существует ли файл
	if _, err := os.Stat(torrentStates); os.IsNotExist(err) {
		// Файл не существует — создаём его
		file, createErr := os.Create(torrentStates)
		if createErr != nil {
			panic("Не удалось создать файл: " + createErr.Error())
		}

		defer filehelpers.CloseFile(file)

		// Записываем пустой JSON-объект: {}
		encoder := json.NewEncoder(file)
		encodeErr := encoder.Encode(map[string]interface{}{})
		if encodeErr != nil {
			panic("Не удалось записать JSON в файл: " + encodeErr.Error())
		}

		log.Println("Файл", torrentStates, "создан.")
	} else {
		log.Println("Файл", torrentStates, "уже существует.")
	}

	// Хранилище торрентов
	clientBaseDir := cfg.TorrentsDir
	// Хранилище метаданных торрентов
	pieceCompletionDir := cfg.PieceCompletionDir
	// Инициализируем торрент-клиент
	torrentClient, err := torrent.NewClient(clientBaseDir, pieceCompletionDir)
	if err != nil {
		log.Fatal("Failed to init torrent client:", err)
	}

	sm := torrent.NewTorrentStateManager(torrentStates)
	torrentService := torrent.NewService(torrentClient, sm)
	eventHandler := torrent.NewEventHandler(torrentService)
	eventHandler.Start()

	// Обрабатываем очередь конвертации
	go func() {
		for {
			select {
			case t, ok := <-sm.GetConversionQueue():
				if !ok {
					log.Println("Conversion queue closed, stopping conversion worker...")
					return
				}

				log.Printf("Starting conversion for torrent: %s", t.InfoHash)

				// Помечаем как конвертируемый
				if err := sm.MarkAsConverting(t.InfoHash); err != nil {
					log.Printf("Failed to mark torrent as converting: %v", err)
					continue
				}

				// Выполняем конвертацию
				if err := torrent.ConvertTorrentToHls(cfg, t); err != nil {
					log.Printf("Failed to convert torrent %s: %v", t.InfoHash, err)
					// Помечаем как ошибку
					if markErr := sm.MarkAsError(t.InfoHash); markErr != nil {
						log.Printf("Failed to mark torrent as error: %v", markErr)
					}
				} else {
					// После успешной конвертации
					if err := sm.MarkAsConverted(t.InfoHash); err != nil {
						log.Printf("Failed to mark torrent as converted: %v", err)
					} else {
						log.Printf("Successfully converted torrent: %s", t.Name)
					}
				}

			case <-sigChan:
				log.Println("Received shutdown signal, stopping conversion worker...")
				return
			}
		}
	}()

	router := chi.NewRouter()
	// global middleware:
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	router.Route("/api", func(api chi.Router) {
		api.Use(middleware.Timeout(30*time.Second), httphelpers.ErrorHandler)
		api.Route("/torrents", func(r chi.Router) {
			r.Get("/", handlers.GetTorrentsHandler(torrentService))
			r.Post("/", handlers.AddTorrentHandler(torrentService))
			r.Get("/{hash}/pause", handlers.PauseTorrentHandler(torrentService))
			r.Get("/{hash}/resume", handlers.ResumeTorrentHandler(torrentService))
			r.Get("/{hash}", handlers.GetTorrentHandler(torrentService))
			r.Delete("/{hash}", handlers.DeleteTorrentHandler(torrentService))
			r.Post("/{hash}/convert", handlers.ConvertTorrentHandler(torrentService))
		})
		api.Get("/files/tree", handlers.GetFilesTreeHandler(cfg))
		api.Get("/files", handlers.GetFilesHandler(cfg))
		api.Get("/video", handlers.VideoHandler(cfg))
		api.Get("/health", handlers.HealthCheck(torrentClient))
	})
	router.Get("/ws", handlers.HandleWebSocket(torrentService))
	router.Get("/starfield/*", handlers.StarfieldHandler("./web"))

	// Создаем HTTP-сервер
	server := &http.Server{
		Addr:    ":8081", // Changed port to 8082
		Handler: router,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

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

		// Останавливаем компоненты
		eventHandler.Stop()
		sm.Stop()

		serverStopCtx()
	}()

	log.Println("Server started at :8081") // Updated log message
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ListenAndServe(): %v", err)
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}
