package handlers

import (
	"GoFlix/internal/app/torrent"
	"encoding/json"
	"log"
	"net/http"
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

func GetTorrentsHandler(client *torrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		torrents, err := client.GetTorrents()

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Обрабатываем ошибку Encode
		if err := json.NewEncoder(w).Encode(torrents); err != nil {
			log.Printf("[api] Client disconnected before response: %v", err)
		}
	}
}
