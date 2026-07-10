package handlers

import (
	"GoFlix/configs"
	"GoFlix/internal/app/filesystem"
	"log"
	"net/http"
)

// GetFilesTreeHandler returns the full file tree under the torrents directory.
func GetFilesTreeHandler(cfg *configs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := filesystem.GetFilesTree(cfg)
		if err != nil {
			http.Error(w, "error while receiving files", http.StatusBadRequest)
			log.Printf("[api] Error while receiving files: %v", err)
			return
		}
		writeJSON(w, http.StatusOK, files)
	}
}

// GetFilesHandler returns the files in a specific directory.
func GetFilesHandler(cfg *configs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Query().Get("path")

		rootPath, err := filesystem.BuildSafePath(cfg.TorrentsDir, filePath)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			log.Printf("[api] Invalid path attempt: %s", filePath)
			return
		}

		files, err := filesystem.GetFiles(rootPath)
		if err != nil {
			http.Error(w, "error while receiving files", http.StatusBadRequest)
			log.Printf("[api] Error while receiving files: %v", err)
			return
		}
		writeJSON(w, http.StatusOK, files)
	}
}
