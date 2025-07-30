package handlers

import (
	"GoFlix/internal/app/torrent"
	"log"
	"net/http"
	"strings"
)

func HealthCheck(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверяем критичные зависимости
		if client == nil {
			log.Println("[health] CRITICAL: Torrent client not initialized")
			http.Error(w, "Torrent client not initialized", http.StatusServiceUnavailable)
			return
		}

		// 2. Проверяем работоспособность
		if _, err := client.Add("magnet:?xt=urn:btih:DEADBEEF"); err != nil {
			if !strings.Contains(err.Error(), "not found") {
				log.Printf("[health] CRITICAL: Torrent client error: %v", err)
				http.Error(w, "Torrent client error", http.StatusServiceUnavailable)
				return
			}
		}

		// 3. Отправляем ответ с обработкой ошибок
		if _, err := w.Write([]byte("OK")); err != nil {
			// Это НЕ критично для health check
			log.Printf("[health] NON-CRITICAL: Failed to write response: %v", err)
		}
	}
}
