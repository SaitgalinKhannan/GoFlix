package handlers

import (
	"GoFlix/internal/app/filesystem"
	"GoFlix/internal/pkg/filehelpers"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
		filePath := r.URL.Query().Get("path")

		// Безопасное построение пути
		rootPath, err := filesystem.BuildSafePath("torrents", filePath)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			log.Printf("[api] Invalid path attempt: %s", filePath)
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

func VideoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Query().Get("path")
		if filePath == "" {
			http.Error(w, "no path specified", http.StatusBadRequest)
			return
		}

		safePath, err := filesystem.BuildSafePath("torrents", filePath)
		if err != nil {
			http.Error(w, "invalid path", http.StatusForbidden)
			return
		}

		file, err := os.Open(safePath)
		if err != nil {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}

		defer filehelpers.CloseFile(file)

		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "cannot get file info", http.StatusInternalServerError)
			return
		}

		ext := filepath.Ext(filePath)
		var contentType string
		switch ext {
		case ".m3u8":
			contentType = "application/vnd.apple.mpegurl"
		case ".ts":
			contentType = "video/mp2t"
		case ".mp4":
			contentType = "video/mp4"
		default:
			contentType = "application/octet-stream"
		}

		// Устанавливаем заголовки
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		// Для .m3u8 файлов отключаем кеширование
		if ext == ".m3u8" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		// Копируем содержимое файла в ответ
		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("Error serving file %s: %v", safePath, err)
		}
	}
}

/*func VideoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Query().Get("path")
		if filePath == "" {
			http.Error(w, "no path specified", http.StatusBadRequest)
			return
		}

		safePath, err := filesystem.BuildSafePath("torrents", filePath)
		if err != nil {
			http.Error(w, "invalid path", http.StatusForbidden)
			return
		}

		file, err := os.Open(safePath)
		if err != nil {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		defer filehelpers.CloseFile(file)

		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "cannot get file info", http.StatusInternalServerError)
			return
		}
		fileSize := fileInfo.Size()

		ext := filepath.Ext(filePath)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// Специальная обработка для HLS плейлистов
		if ext == ".m3u8" {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			//w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
			w.Header().Set("Cache-Control", "no-cache")

			// Отдаем файл целиком
			http.ServeFile(w, r, safePath)
			return
		}

		// Для остальных файлов (включая .ts сегменты) поддерживаем Range
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Type", contentType)

		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			// Без Range - отдаем весь файл
			w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
			http.ServeFile(w, r, safePath)
			return
		}

		// С Range - отдаем часть файла
		start, end, err := httphelpers.ParseRange(rangeHeader, fileSize)
		if err != nil {
			log.Printf("[api] Error parsing range: %v", err)
			http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
			return
		}

		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
		w.WriteHeader(http.StatusPartialContent)

		reader, _, err := filesystem.GetPieceOfFileReader(safePath, start, end)
		if err != nil {
			http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
			return
		}

		defer func(reader *filesystem.FileReadCloser) {
			err := reader.Close()
			if err != nil {
				log.Printf("[api] Error closing reader: %v", err)
			}
		}(reader)

		_, _ = io.Copy(w, reader)
	}
}*/
