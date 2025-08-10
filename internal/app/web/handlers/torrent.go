package handlers

import (
	"GoFlix/internal/app/torrent"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type addRequest struct {
	Source string `json:"source"`
}

func AddTorrentHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req addRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println(err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		infoHash, err := client.Add(req.Source)

		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Обрабатываем ошибку Encode
		if err := json.NewEncoder(w).Encode(map[string]string{"infoHash": infoHash}); err != nil {
			log.Printf("[api] Client disconnected before response: %v", err)
		}
	}
}

func GetTorrentHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			http.Error(w, "Missing or invalid hash parameter", http.StatusBadRequest)
			return
		}

		t, err := client.GetTorrent(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Обрабатываем ошибку Encode
		if err := json.NewEncoder(w).Encode(t); err != nil {
			log.Printf("[api] Client disconnected before response: %v", err)
		}
	}
}

func GetTorrentsHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		torrents := client.GetTorrents()

		w.Header().Set("Content-Type", "application/json")

		// Обрабатываем ошибку Encode
		if err := json.NewEncoder(w).Encode(torrents); err != nil {
			log.Printf("[api] Client disconnected before response: %v", err)
		}
	}
}

// PauseTorrentHandler обрабатывает PATCH /{hash}/pause
func PauseTorrentHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			http.Error(w, "Missing or invalid hash parameter", http.StatusBadRequest)
			return
		}

		err := client.PauseTorrent(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError) // Или 400, если ошибка от клиента
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// ResumeTorrentHandler обрабатывает PATCH /{hash}/resume
func ResumeTorrentHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			http.Error(w, "Missing or invalid hash parameter", http.StatusBadRequest)
			return
		}

		err := client.ResumeTorrent(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// DeleteTorrentHandler обрабатывает DELETE /{hash}
func DeleteTorrentHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			http.Error(w, "Missing or invalid hash parameter", http.StatusBadRequest)
			return
		}

		err := client.DeleteTorrent(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// ConvertTorrentHandler обрабатывает POST /{hash}/convert
func ConvertTorrentHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			log.Println("Missing or invalid hash parameter")
			http.Error(w, "Missing or invalid hash parameter", http.StatusBadRequest)
			return
		}

		err := client.ConvertTorrent(hash) // Или client.Convert(hash), в зависимости от вашего API
		if err != nil {
			log.Printf("[api] Converting error: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
