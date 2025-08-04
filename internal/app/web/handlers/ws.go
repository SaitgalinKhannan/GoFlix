package handlers

import (
	"GoFlix/internal/app/torrent"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Для пет-проекта, в продакшене ограничьте origin
}

func HandleWebSocket(torrentClient *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("WS background task panic: %v\n%s", r, debug.Stack())
			}
		}()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ws] Upgrade failed: %v", err) // Логируем
			return
		}
		defer func() {
			if err := conn.Close(); err != nil {
				log.Printf("[ws] Close error: %v", err) // Обрабатываем Close
			}
		}()

		if _, err := torrentClient.GetTorrents(); err != nil {
			if err := conn.WriteJSON(map[string]string{"error": "invalid torrent"}); err != nil {
				log.Printf("[ws] WriteJSON failed: %v", err)
			}
			return
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				torrents, err := torrentClient.GetTorrents()
				if err != nil {
					return
				}

				// Ключевая проверка: если ошибка записи — клиент отключился
				if err := conn.WriteJSON(torrents); err != nil {
					log.Printf("[ws] Client disconnected: %v", err)
					return // Немедленно выходим
				}

			case <-r.Context().Done():
				return
			}
		}
	}
}
