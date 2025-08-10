package configs

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	TorrentsStatesFile string
	TorrentsDir        string
	PieceCompletionDir string
	OpenAIURL          string
	OpenAIKey          string
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
		OpenAIURL:          os.Getenv("OPENAI_URL"),
		OpenAIKey:          os.Getenv("OPENAI_KEY"),
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
	// Для OpenAI_URL и OpenAI_KEY лучше не задавать дефолты или требовать их
	// Если они критически важны
	if cfg.OpenAIURL == "" {
		return nil, fmt.Errorf("OPENAI_URL is not set")
	}
	if cfg.OpenAIKey == "" {
		return nil, fmt.Errorf("OPENAI_KEY is not set")
	}

	return cfg, nil
}
