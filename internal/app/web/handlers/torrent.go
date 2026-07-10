package handlers

import (
	"GoFlix/internal/app/torrent"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

const maxUploadBytes = 5 << 20 // 5 MB cap for .torrent files

// AddTorrentHandler adds a torrent via multipart/form-data.
// Accepts either a "magnet" field (string) or a "file" field (.torrent file).
func AddTorrentHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow multipart or JSON body for flexibility.
		ct := r.Header.Get("Content-Type")

		if isMultipart(ct) {
			r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
			if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
				http.Error(w, "failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Magnet takes precedence if present.
			if magnet := r.FormValue("magnet"); magnet != "" {
				status, err := service.AddMagnet(magnet)
				if err != nil {
					log.Println(err)
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				writeJSON(w, http.StatusCreated, status)
				return
			}

			// Otherwise expect a .torrent file.
			file, header, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "provide a 'magnet' field or a 'file' .torrent upload", http.StatusBadRequest)
				return
			}
			defer file.Close()

			tmpPath, magnet, err := saveTorrentFile(file, header.Filename)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer os.Remove(tmpPath)

			status, err := service.AddFile(tmpPath, magnet)
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusCreated, status)
			return
		}

		// Fallback: JSON body { "source": "<magnet|path>" } (kept for compatibility).
		var req struct {
			Source string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		status, err := service.AddMagnet(req.Source)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, status)
	}
}

// GetTorrentsHandler returns all torrents.
func GetTorrentsHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.List())
	}
}

// GetTorrentHandler returns a single torrent by id (infoHash).
func GetTorrentHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing or invalid id parameter", http.StatusBadRequest)
			return
		}
		status, err := service.Get(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

// DeleteTorrentHandler deletes a torrent. With ?deleteFiles=true the downloaded
// files are removed from disk.
func DeleteTorrentHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing or invalid id parameter", http.StatusBadRequest)
			return
		}
		deleteFiles := r.URL.Query().Has("deleteFiles")
		if err := service.Delete(id, deleteFiles); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// StartSeedingHandler starts/resumes a torrent.
func StartSeedingHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing or invalid id parameter", http.StatusBadRequest)
			return
		}
		if err := service.StartSeeding(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// StopSeedingHandler stops/pauses a torrent (files preserved).
func StopSeedingHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing or invalid id parameter", http.StatusBadRequest)
			return
		}
		if err := service.StopSeeding(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// RecheckHandler forces a full re-hash of a torrent's existing data.
func RecheckHandler(service *torrent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing or invalid id parameter", http.StatusBadRequest)
			return
		}
		if err := service.Recheck(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

// saveTorrentFile writes an uploaded .torrent to a temp file and returns its path + magnet.
func saveTorrentFile(src io.Reader, filename string) (path, magnet string, err error) {
	tmp, err := os.CreateTemp("", "goflix-*.torrent")
	if err != nil {
		return "", "", err
	}
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", "", err
	}
	tmp.Close()

	magnet, magErr := torrent.MagnetFromFile(tmp.Name())
	if magErr != nil {
		// not fatal — we still have the file; magnet will be empty.
		log.Printf("[api] could not derive magnet from %s: %v", filepath.Base(filename), magErr)
	}
	return tmp.Name(), magnet, nil
}

func isMultipart(ct string) bool {
	return len(ct) >= 10 && ct[:10] == "multipart/"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[api] Client disconnected before response: %v", err)
	}
}
