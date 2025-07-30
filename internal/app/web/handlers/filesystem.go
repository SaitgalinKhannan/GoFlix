package handlers

import (
	"GoFlix/internal/app/filesystem"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func GetFilesTreeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, fErr := filesystem.GetFilesTree()

		if fErr != nil {
			http.Error(w, "error while receiving files", http.StatusBadRequest)
			log.Printf("[api] Error while receiving files: %v", fErr)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Обрабатываем ошибку Encode
		if err := json.NewEncoder(w).Encode(files); err != nil {
			log.Printf("[api] Client disconnected before response: %v", err)
		}
	}
}

func GetFilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")

		// Безопасное построение пути
		rootPath, err := buildSafePath("torrents", path)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			log.Printf("[api] Invalid path attempt: %s", path)
			return
		}

		files, fErr := filesystem.GetFiles(rootPath)

		if fErr != nil {
			http.Error(w, "error while receiving files", http.StatusBadRequest)
			log.Printf("[api] Error while receiving files: %v", fErr)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Обрабатываем ошибку Encode
		if err := json.NewEncoder(w).Encode(files); err != nil {
			log.Printf("[api] Client disconnected before response: %v", err)
		}
	}
}

func buildSafePath(baseDir, userPath string) (string, error) {
	// Очищаем путь от множественных слешей и относительных переходов
	cleanPath := filepath.Clean(userPath)

	// Убираем ведущий слеш если есть
	if strings.HasPrefix(cleanPath, "/") {
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}

	// Строим полный путь
	fullPath := cleanPath

	if !strings.HasPrefix(fullPath, baseDir) {
		fullPath = filepath.Join(baseDir, cleanPath)
	}

	// Получаем абсолютные пути для проверки
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	// Проверяем, что результирующий путь находится внутри базовой директории
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path traversal detected")
	}

	return fullPath, nil
}
