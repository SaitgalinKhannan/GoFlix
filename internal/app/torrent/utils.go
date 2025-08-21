package torrent

import (
	"GoFlix/configs"
	"GoFlix/internal/app/media"
	"GoFlix/internal/pkg/filehelpers"
	"os"
	"path/filepath"
)

func getPercent(n, total int64) float32 {
	if total == 0 {
		return float32(0)
	}
	return float32(int(float64(10000)*(float64(n)/float64(total)))) / 100
}

func ConvertTorrentToHls(cfg *configs.Config, torrent *Torrent) error {
	torrentPath := filepath.Join(cfg.TorrentsDir, torrent.Name)
	abs, err := filepath.Abs(torrentPath)
	if err != nil {
		return err
	}

	stat, err := os.Stat(abs)
	if err != nil {
		return err
	}

	if stat.IsDir() {
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

			err = media.ConvertToHls(path)
			if err != nil {
				return err
			}

			return nil
		})
	} else {
		return media.ConvertToHls(abs)
	}
}
