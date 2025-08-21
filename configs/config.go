package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	TorrentsStatesFile string
	TorrentsDir        string
	PieceCompletionDir string
}

func LoadConfig() (*Config, error) {
	// Загружаем переменные окружения из .env файла
	// По умолчанию ищет .env в текущей директории
	// Можно указать путь: godotenv.Load("configs/.env")
	err := godotenv.Load()
	if err != nil {
		// Это нормально, если .env файл не найден в продакшене,
		// так как переменные могут быть установлены напрямую в окружении.
		log.Println("No .env file found, falling back to environment variables or defaults.")
		// return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	cfg := &Config{
		Port:               os.Getenv("PORT"),
		TorrentsStatesFile: os.Getenv("TORRENTS_STATES_FILE"),
		TorrentsDir:        os.Getenv("TORRENTS_DIR"),
		PieceCompletionDir: os.Getenv("PIECE_COMPLETION_DIR"),
	}

	// Установка значений по умолчанию, если переменные не заданы
	if cfg.Port == "" {
		cfg.Port = "8081"
	}
	if cfg.TorrentsStatesFile == "" {
		cfg.TorrentsStatesFile = "/app/data/torrent_states.json"
	}
	if cfg.TorrentsDir == "" {
		cfg.TorrentsDir = "/app/data/torrents"
	}
	if cfg.PieceCompletionDir == "" {
		cfg.PieceCompletionDir = "/app/data/torrent_data"
	}

	return cfg, nil
}
