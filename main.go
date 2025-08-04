package main

import (
	"GoFlix/internal/app/media"
	"GoFlix/internal/pkg/filehelpers"
	"log"
	"os"
	"path/filepath"
)

func ConvertTorrentToHls(baseDir string, torrentName string) error {
	torrentPath := filepath.Join(baseDir, torrentName)
	abs, err := filepath.Abs(torrentPath)
	if err != nil {
		return err
	}

	stat, err := os.Stat(abs)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		// Обрабатываем ошибку от filepath.Walk
		return filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Пропускаем директории, обрабатываем только файлы
			if info.IsDir() {
				return nil
			}

			// Проверяем, является ли файл видео
			if !filehelpers.IsVideoFile(path) {
				return nil
			}

			// Передаем полный путь, а не только имя файла
			err = media.CopyToHls(path)
			if err != nil {
				return err
			}

			return nil
		})
	} else {
		err = media.CopyToHls(abs)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	clientBaseDir := "torrents"
	if err := ConvertTorrentToHls(clientBaseDir, "Takopii no Genzai - AniLiberty [WEBRip 1080p HEVC]"); err != nil {
		log.Printf("Failed to convert torrent %s: %v", "Takopii no Genzai - AniLiberty [WEBRip 1080p HEVC]", err)
	}
}
